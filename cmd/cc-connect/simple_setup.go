package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/chenhg5/cc-connect/config"
	"github.com/chenhg5/cc-connect/core"
	"github.com/chenhg5/cc-connect/internal/secretstore"
)

func wireSimpleSetup(m *core.ManagementServer, runtimeProjects []config.ProjectConfig, runtimeErrors map[string]string) {
	catalog := buildSetupCatalog()
	store := secretstore.Open("cc-connect")
	running := make(map[string]bool, len(runtimeProjects))
	for _, project := range runtimeProjects {
		running[project.Name] = true
	}

	m.SetSimpleSetupCallbacks(
		func() (core.SetupStatus, error) {
			projects, err := config.ListProjectConfigs()
			if err != nil {
				return core.SetupStatus{}, err
			}
			return core.SetupStatus{
				FirstRun: len(projects) == 0,
				BotCount: len(projects),
				Agents:   detectSetupAgents(catalog.Agents),
			}, nil
		},
		func() core.SetupCatalog { return catalog },
		func() ([]core.BotSummary, error) {
			projects, err := config.ListProjectConfigs()
			if err != nil {
				return nil, err
			}
			bots := make([]core.BotSummary, 0, len(projects))
			for _, project := range projects {
				if project.SimpleMode == nil || !*project.SimpleMode || len(project.Platforms) != 1 {
					continue
				}
				state := "stopped"
				if running[project.Name] {
					state = "running"
				} else if config.ProjectEnabled(project) {
					state = "error"
				}
				bot := projectBotSummary(project, state)
				if runtimeErr := runtimeErrors[project.Name]; runtimeErr != "" {
					bot.RuntimeState = "error"
					bot.RuntimeError = runtimeErr
				}
				bots = append(bots, bot)
			}
			sort.Slice(bots, func(i, j int) bool { return bots[i].DisplayName < bots[j].DisplayName })
			return bots, nil
		},
		func(req core.BotUpsertRequest) (core.BotSummary, error) {
			return saveSimpleBot(req, catalog, store)
		},
		config.SetSimpleBotEnabled,
		config.RemoveSimpleBot,
	)
	m.SetSetupDirectoryPicker(selectSetupDirectory)
	m.SetSetupModelCatalog(listSetupModels)
}

func projectBotSummary(project config.ProjectConfig, state string) core.BotSummary {
	workDir, _ := project.Agent.Options["work_dir"].(string)
	mode, _ := project.Agent.Options["mode"].(string)
	model, _ := project.Agent.Options["model"].(string)
	reasoning, _ := project.Agent.Options["reasoning_effort"].(string)
	replyFooter := project.ReplyFooter == nil || *project.ReplyFooter
	thinkingMessages := true
	toolMessages := true
	if project.Display != nil {
		if project.Display.ThinkingMessages != nil {
			thinkingMessages = *project.Display.ThinkingMessages
		}
		if project.Display.ToolMessages != nil {
			toolMessages = *project.Display.ToolMessages
		}
	}
	return core.BotSummary{
		ID:               project.ID,
		Name:             project.Name,
		DisplayName:      project.DisplayName,
		Enabled:          config.ProjectEnabled(project),
		AgentType:        project.Agent.Type,
		WorkDir:          workDir,
		PermissionMode:   mode,
		Model:            model,
		ReasoningEffort:  reasoning,
		ReplyFooter:      replyFooter,
		ThinkingMessages: thinkingMessages,
		ToolMessages:     toolMessages,
		PlatformType:     project.Platforms[0].Type,
		Configured:       true,
		RuntimeState:     state,
	}
}

func saveSimpleBot(req core.BotUpsertRequest, catalog core.SetupCatalog, store secretstore.Store) (core.BotSummary, error) {
	workDir, err := filepath.Abs(strings.TrimSpace(req.WorkDir))
	if err != nil {
		return core.BotSummary{}, fmt.Errorf("resolve work_dir: %w", err)
	}
	info, err := os.Stat(workDir)
	if err != nil || !info.IsDir() {
		return core.BotSummary{}, fmt.Errorf("work_dir must be an existing directory: %s", workDir)
	}
	if !contains(core.ListRegisteredAgents(), req.AgentType) {
		return core.BotSummary{}, fmt.Errorf("unknown agent %q", req.AgentType)
	}

	projects, err := config.ListProjectConfigs()
	if err != nil {
		return core.BotSummary{}, err
	}
	var existing *config.ProjectConfig
	for i := range projects {
		if req.ID != "" && projects[i].ID == req.ID {
			existing = &projects[i]
			break
		}
	}
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	compactID := strings.ReplaceAll(req.ID, "-", "")
	if len(compactID) < 12 {
		return core.BotSummary{}, fmt.Errorf("invalid bot id")
	}
	name := "bot-" + compactID[:12]
	if existing != nil {
		name = existing.Name
	}

	options := make(map[string]any, len(req.Options))
	for key, value := range req.Options {
		options[key] = value
	}
	platform, ok := findSetupPlatform(catalog, req.PlatformType)
	if !ok {
		return core.BotSummary{}, fmt.Errorf("unknown platform %q", req.PlatformType)
	}
	var oldOptions map[string]any
	if existing != nil && len(existing.Platforms) == 1 && existing.Platforms[0].Type == req.PlatformType {
		oldOptions = existing.Platforms[0].Options
		for key, value := range oldOptions {
			if _, supplied := options[key]; !supplied {
				options[key] = value
			}
		}
	}
	req.Options = options
	if err := core.ValidateBotRequest(req, catalog); err != nil {
		return core.BotSummary{}, err
	}
	for _, field := range platform.Fields {
		if !field.Secret {
			continue
		}
		value, supplied := options[field.Key]
		text := strings.TrimSpace(fmt.Sprint(value))
		if !supplied || text == "" {
			if old, exists := oldOptions[field.Key]; exists {
				options[field.Key] = old
			}
			continue
		}
		if _, isRef := secretstore.ParseReference(text); isRef {
			options[field.Key] = text
			continue
		}
		key := req.ID + "/" + req.PlatformType + "/" + field.Key
		if err := store.Set(key, text); err != nil {
			return core.BotSummary{}, fmt.Errorf("store credential %q: %w", field.Key, err)
		}
		options[field.Key] = secretstore.Reference(key)
	}

	enabled := true
	if existing != nil {
		enabled = config.ProjectEnabled(*existing)
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	simple := true
	agentOptions := map[string]any{"work_dir": workDir, "mode": safePermissionMode(req.AgentType, req.PermissionMode)}
	if strings.TrimSpace(req.Model) != "" {
		agentOptions["model"] = strings.TrimSpace(req.Model)
	}
	if strings.TrimSpace(req.ReasoningEffort) != "" {
		agentOptions["reasoning_effort"] = strings.TrimSpace(req.ReasoningEffort)
	}
	var replyFooter *bool
	if existing != nil && existing.ReplyFooter != nil {
		value := *existing.ReplyFooter
		replyFooter = &value
	} else if existing == nil {
		// Simple desktop bots default to clean answers without diagnostic
		// metadata. Advanced projects retain the upstream default (enabled).
		value := false
		replyFooter = &value
	}
	if req.ReplyFooter != nil {
		value := *req.ReplyFooter
		replyFooter = &value
	}
	var display *config.DisplayConfig
	if existing != nil && existing.Display != nil {
		value := *existing.Display
		display = &value
	}
	if req.ThinkingMessages != nil || req.ToolMessages != nil {
		if display == nil {
			display = &config.DisplayConfig{}
		}
		if req.ThinkingMessages != nil {
			value := *req.ThinkingMessages
			display.ThinkingMessages = &value
		}
		if req.ToolMessages != nil {
			value := *req.ToolMessages
			display.ToolMessages = &value
		}
	}
	bot := config.ProjectConfig{
		ID:          req.ID,
		Name:        name,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Enabled:     &enabled,
		SimpleMode:  &simple,
		ReplyFooter: replyFooter,
		Display:     display,
		Agent: config.AgentConfig{
			Type:    req.AgentType,
			Options: agentOptions,
		},
		Platforms: []config.PlatformConfig{{Type: req.PlatformType, Options: options}},
	}
	if err := config.UpsertSimpleBot(bot); err != nil {
		return core.BotSummary{}, err
	}
	return projectBotSummary(bot, "applying"), nil
}

func safePermissionMode(agentType, mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" || strings.EqualFold(mode, "yolo") || strings.EqualFold(mode, "bypassPermissions") {
		if agentType == "codex" {
			return "suggest"
		}
		return "default"
	}
	if agentType == "codex" && mode != "suggest" && mode != "auto-edit" && mode != "full-auto" {
		return "suggest"
	}
	if agentType == "pi" && mode != "default" {
		return "default"
	}
	return mode
}

func resolveSecretReferences(cfg *config.Config) map[string]error {
	store := secretstore.Open("cc-connect")
	errorsByProject := make(map[string]error)
	for i := range cfg.Projects {
		for j := range cfg.Projects[i].Platforms {
			for key, raw := range cfg.Projects[i].Platforms[j].Options {
				value, ok := raw.(string)
				if !ok {
					continue
				}
				secretKey, ok := secretstore.ParseReference(value)
				if !ok {
					continue
				}
				secret, err := store.Get(secretKey)
				if err != nil {
					errorsByProject[cfg.Projects[i].Name] = fmt.Errorf("credential %s: %w", key, err)
					continue
				}
				cfg.Projects[i].Platforms[j].Options[key] = secret
			}
		}
	}
	return errorsByProject
}

func detectSetupAgents(agents []core.SetupAgent) []core.AgentHealth {
	result := make([]core.AgentHealth, 0, len(agents))
	for _, agent := range agents {
		if agent.Key != "codex" && agent.Key != "pi" {
			continue
		}
		result = append(result, detectSetupAgent(agent))
	}
	return result
}

func detectSetupAgent(agent core.SetupAgent) core.AgentHealth {
	command := agent.Key
	path, err := exec.LookPath(command)
	health := core.AgentHealth{Key: agent.Key, Label: agent.Label, Installed: err == nil, Path: path}
	if err != nil {
		health.Problem = command + " was not found"
		health.Guide = agentInstallGuide(agent.Key)
		return health
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	versionCmd := setupAgentCommand(ctx, path, "--version")
	out, versionErr := versionCmd.CombinedOutput()
	health.Version = strings.TrimSpace(string(out))
	if versionErr != nil {
		health.Problem = "version check failed"
		return health
	}
	health.LoggedIn = true
	if agent.Key == "codex" {
		loginCtx, loginCancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer loginCancel()
		loginCmd := setupAgentCommand(loginCtx, path, "login", "status")
		loginOut, loginErr := loginCmd.CombinedOutput()
		health.LoggedIn = loginErr == nil
		if loginErr != nil {
			health.Problem = strings.TrimSpace(string(loginOut))
			health.Guide = "Run: codex login"
		}
	}
	return health
}

func setupAgentCommand(ctx context.Context, path string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if strings.HasSuffix(strings.ToLower(path), ".ps1") {
		powerShellArgs := []string{"-NoProfile", "-NonInteractive", "-File", path}
		cmd = exec.CommandContext(ctx, "powershell.exe", append(powerShellArgs, args...)...)
	} else {
		cmd = exec.CommandContext(ctx, path, args...)
	}
	configureBackgroundCommand(cmd)
	return cmd
}

func listSetupModels(agentType string) (core.SetupModelCatalog, error) {
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		return core.SetupModelCatalog{}, fmt.Errorf("agent is required")
	}
	agent, err := core.CreateAgent(agentType, map[string]any{"work_dir": "."})
	if err != nil {
		return core.SetupModelCatalog{}, fmt.Errorf("create %s agent for model discovery: %w", agentType, err)
	}
	defer func() {
		if err := agent.Stop(); err != nil {
			slog.Debug("stop model discovery agent", "agent", agentType, "error", err)
		}
	}()

	switcher, ok := agent.(core.ModelSwitcher)
	if !ok {
		return core.SetupModelCatalog{}, fmt.Errorf("agent %q does not provide a model catalog", agentType)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	available := switcher.AvailableModels(ctx)
	if len(available) == 0 {
		return core.SetupModelCatalog{}, fmt.Errorf("%s did not report any available models; check its local model configuration and sign-in status", agentType)
	}

	models := make([]core.SetupModel, 0, len(available))
	seen := make(map[string]struct{}, len(available))
	for _, model := range available {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, core.SetupModel{
			Name:        name,
			Description: strings.TrimSpace(model.Desc),
			Alias:       strings.TrimSpace(model.Alias),
		})
	}
	if len(models) == 0 {
		return core.SetupModelCatalog{}, fmt.Errorf("%s returned an empty model catalog", agentType)
	}
	sort.SliceStable(models, func(i, j int) bool { return models[i].Name < models[j].Name })
	return core.SetupModelCatalog{Models: models, Current: strings.TrimSpace(switcher.GetModel())}, nil
}

func agentInstallGuide(key string) string {
	if key == "codex" {
		return "Install with npm install -g @openai/codex, then run codex login"
	}
	return "Install with npm install -g @mariozechner/pi-coding-agent"
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func findSetupPlatform(catalog core.SetupCatalog, key string) (core.SetupPlatform, bool) {
	for _, platform := range catalog.Platforms {
		if platform.Key == key {
			return platform, true
		}
	}
	return core.SetupPlatform{}, false
}

func buildSetupCatalog() core.SetupCatalog {
	agentLabels := map[string]string{
		"codex": "Codex", "pi": "Pi", "claudecode": "Claude Code", "gemini": "Gemini CLI",
		"cursor": "Cursor", "opencode": "OpenCode", "copilot": "GitHub Copilot", "qoder": "Qoder",
	}
	agents := make([]core.SetupAgent, 0)
	for _, key := range core.ListRegisteredAgents() {
		label := agentLabels[key]
		if label == "" {
			label = key
		}
		modes := []string{"default", "plan", "accept-edits"}
		switch key {
		case "codex":
			modes = []string{"suggest", "auto-edit", "full-auto"}
		case "pi":
			modes = []string{"default"}
		}
		agents = append(agents, core.SetupAgent{Key: key, Label: label, Recommended: key == "codex" || key == "pi", Modes: modes})
	}
	sort.SliceStable(agents, func(i, j int) bool {
		if agents[i].Recommended != agents[j].Recommended {
			return agents[i].Recommended
		}
		return agents[i].Label < agents[j].Label
	})

	definitions := setupPlatformDefinitions()
	platforms := make([]core.SetupPlatform, 0)
	for _, key := range core.ListRegisteredPlatforms() {
		definition, ok := definitions[key]
		if !ok {
			slog.Warn("registered platform missing setup catalog", "platform", key)
			definition = core.SetupPlatform{Key: key, Label: key, Fields: []core.SetupField{}}
		}
		platforms = append(platforms, definition)
	}
	sort.Slice(platforms, func(i, j int) bool { return platforms[i].Label < platforms[j].Label })
	return core.SetupCatalog{Agents: agents, Platforms: platforms}
}

func sf(key, label string, required, secret bool) core.SetupField {
	typ := "text"
	if secret {
		typ = "password"
	}
	return core.SetupField{Key: key, Label: label, Required: required, Secret: secret, Type: typ}
}

func setupPlatformDefinitions() map[string]core.SetupPlatform {
	allow := core.SetupField{Key: "allow_from", Label: "fields.allowFrom", Group: "advanced", Placeholder: "*"}
	return map[string]core.SetupPlatform{
		"feishu":         {Key: "feishu", Label: "Feishu", QR: true, Fields: []core.SetupField{sf("app_id", "fields.appId", true, false), sf("app_secret", "fields.appSecret", true, true)}},
		"lark":           {Key: "lark", Label: "Lark", QR: true, Fields: []core.SetupField{sf("app_id", "fields.appId", true, false), sf("app_secret", "fields.appSecret", true, true)}},
		"weixin":         {Key: "weixin", Label: "Weixin", QR: true, Fields: []core.SetupField{sf("token", "fields.botToken", true, true), sf("base_url", "fields.apiBaseUrl", false, false)}},
		"telegram":       {Key: "telegram", Label: "Telegram", Fields: []core.SetupField{sf("token", "fields.botToken", true, true), allow}},
		"discord":        {Key: "discord", Label: "Discord", Fields: []core.SetupField{sf("token", "fields.botToken", true, true), allow}},
		"slack":          {Key: "slack", Label: "Slack", Fields: []core.SetupField{sf("bot_token", "fields.botToken", true, true), sf("app_token", "fields.appToken", true, true), allow}},
		"dingtalk":       {Key: "dingtalk", Label: "DingTalk", Fields: []core.SetupField{sf("client_id", "fields.clientId", true, false), sf("client_secret", "fields.clientSecret", true, true), allow}},
		"wecom":          {Key: "wecom", Label: "WeCom", Fields: []core.SetupField{sf("corp_id", "fields.corpId", true, false), sf("corp_secret", "fields.corpSecret", true, true), sf("agent_id", "fields.agentId", true, false), sf("callback_token", "fields.callbackToken", true, true), sf("callback_aes_key", "fields.callbackAesKey", true, true)}},
		"qq":             {Key: "qq", Label: "QQ (OneBot)", Fields: []core.SetupField{sf("ws_url", "fields.wsUrl", true, false), sf("token", "fields.accessToken", false, true), allow}},
		"qqbot":          {Key: "qqbot", Label: "QQ Bot", Fields: []core.SetupField{sf("app_id", "fields.appId", true, false), sf("app_secret", "fields.appSecret", true, true), allow}},
		"yuanbao":        {Key: "yuanbao", Label: "Tencent Yuanbao", Fields: []core.SetupField{sf("bot_token", "fields.botToken", true, true), allow}},
		"line":           {Key: "line", Label: "LINE", Fields: []core.SetupField{sf("channel_secret", "fields.channelSecret", true, true), sf("channel_token", "fields.channelToken", true, true), allow}},
		"weibo":          {Key: "weibo", Label: "Weibo", Fields: []core.SetupField{sf("app_id", "fields.appId", true, false), sf("app_secret", "fields.appSecret", true, true), allow}},
		"tuitui":         {Key: "tuitui", Label: "TuiTui", Fields: []core.SetupField{sf("app_id", "fields.appId", true, false), sf("app_secret", "fields.appSecret", true, true), allow}},
		"cloud_web":      {Key: "cloud_web", Label: "Cloud Web", Fields: []core.SetupField{{Key: "transport", Label: "fields.transport", Required: true, Type: "select", Options: []string{"websocket", "long_poll", "gateway"}}, sf("token", "fields.accessToken", true, true), sf("base_url", "fields.apiBaseUrl", true, false)}},
		"matrix":         {Key: "matrix", Label: "Matrix", Fields: []core.SetupField{sf("homeserver", "fields.apiBaseUrl", true, false), sf("access_token", "fields.accessToken", true, true), sf("user_id", "fields.userId", false, false), allow}},
		"webex":          {Key: "webex", Label: "Webex", Fields: []core.SetupField{sf("token", "fields.botToken", true, true), allow}},
		"max":            {Key: "max", Label: "MAX", Fields: []core.SetupField{sf("token", "fields.botToken", true, true), allow}},
		"googlechat":     {Key: "googlechat", Label: "Google Chat", Fields: []core.SetupField{sf("subscription", "fields.subscription", true, false), sf("credentials_file", "fields.credentialsFile", true, false), allow}},
		"wps-xiezuo":     {Key: "wps-xiezuo", Label: "WPS Xiezuo", Fields: []core.SetupField{sf("app_id", "fields.appId", true, false), sf("app_secret", "fields.appSecret", true, true), allow}},
		"wps-agentspace": {Key: "wps-agentspace", Label: "WPS Agentspace", Fields: []core.SetupField{sf("app_id", "fields.appId", true, false), sf("wps_sid", "fields.wpsSid", true, true)}},
	}
}
