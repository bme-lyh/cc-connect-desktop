package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhg5/cc-connect/config"
	"github.com/chenhg5/cc-connect/core"
	"github.com/chenhg5/cc-connect/internal/secretstore"
)

type setupModelTestAgent struct {
	stubMainAgent
	model  string
	models []core.ModelOption
}

func (a *setupModelTestAgent) SetModel(model string) { a.model = model }
func (a *setupModelTestAgent) GetModel() string      { return a.model }
func (a *setupModelTestAgent) AvailableModels(context.Context) []core.ModelOption {
	return a.models
}

func TestSimpleSetup_CatalogCoversEveryRegisteredPlatform(t *testing.T) {
	definitions := setupPlatformDefinitions()
	for _, platform := range core.ListRegisteredPlatforms() {
		if _, ok := definitions[platform]; !ok {
			t.Errorf("registered platform %q is missing setup metadata", platform)
		}
	}
}

func TestSimpleSetup_DefaultPermissionRejectsUnrestrictedModes(t *testing.T) {
	for _, mode := range []string{"", "yolo", "bypassPermissions"} {
		if got := safePermissionMode("pi", mode); got != "default" {
			t.Errorf("safePermissionMode(pi, %q) = %q", mode, got)
		}
		if got := safePermissionMode("codex", mode); got != "suggest" {
			t.Errorf("safePermissionMode(codex, %q) = %q", mode, got)
		}
	}
}

func TestSimpleSetup_ModelCatalogUsesRegisteredAgent(t *testing.T) {
	const agentName = "simple-setup-model-catalog-test"
	core.RegisterAgent(agentName, func(map[string]any) (core.Agent, error) {
		return &setupModelTestAgent{
			model: "model-b",
			models: []core.ModelOption{
				{Name: "model-b", Desc: "Second"},
				{Name: "model-a", Alias: "a"},
				{Name: "model-b", Desc: "Duplicate"},
			},
		}, nil
	})

	catalog, err := listSetupModels(agentName)
	if err != nil {
		t.Fatalf("listSetupModels() error = %v", err)
	}
	if catalog.Current != "model-b" {
		t.Fatalf("current = %q, want model-b", catalog.Current)
	}
	if len(catalog.Models) != 2 || catalog.Models[0].Name != "model-a" || catalog.Models[1].Name != "model-b" {
		t.Fatalf("models = %#v", catalog.Models)
	}
}

func TestSimpleSetup_UpdatePreservesDisabledStateAndMigratesPlaintext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[management]\nenabled = true\nport = 9820\ntoken = \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	previousPath := config.ConfigPath
	config.ConfigPath = path
	t.Cleanup(func() { config.ConfigPath = previousPath })

	disabled, simple := false, true
	botID := "11111111-1111-1111-1111-111111111111"
	if err := config.UpsertSimpleBot(config.ProjectConfig{
		ID: botID, Name: "bot-111111111111", DisplayName: "Legacy", Enabled: &disabled, SimpleMode: &simple,
		Agent:     config.AgentConfig{Type: "codex", Options: map[string]any{"work_dir": dir, "mode": "default"}},
		Platforms: []config.PlatformConfig{{Type: "telegram", Options: map[string]any{"token": "legacy-plaintext"}}},
	}); err != nil {
		t.Fatal(err)
	}

	store := secretstore.NewMemory()
	updated, err := saveSimpleBot(core.BotUpsertRequest{
		ID: botID, DisplayName: "Updated", AgentType: "codex", WorkDir: dir,
		PermissionMode: "default", PlatformType: "telegram", Options: map[string]any{},
	}, buildSetupCatalog(), store)
	if err != nil {
		t.Fatalf("saveSimpleBot() error = %v", err)
	}
	if updated.Enabled {
		t.Fatal("editing a stopped bot must not enable it")
	}
	if !updated.ReplyFooter {
		t.Fatal("editing a legacy bot must preserve the upstream reply footer default")
	}
	key := botID + "/telegram/token"
	if value, err := store.Get(key); err != nil || value != "legacy-plaintext" {
		t.Fatalf("migrated credential = %q, err = %v", value, err)
	}
	projects, err := config.ListProjectConfigs()
	if err != nil || len(projects) != 1 {
		t.Fatalf("projects = %d, err = %v", len(projects), err)
	}
	wantRef := secretstore.Reference(key)
	if got := projects[0].Platforms[0].Options["token"]; got != wantRef {
		t.Fatalf("persisted token = %q, want %q", got, wantRef)
	}
	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(persisted) == "" || strings.Contains(string(persisted), "legacy-plaintext") {
		t.Fatalf("persisted config leaked plaintext credential: %s", persisted)
	}
}

func TestSimpleSetup_NewBotDefaultsToCleanReplies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[management]\nenabled = true\nport = 9820\ntoken = \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	previousPath := config.ConfigPath
	config.ConfigPath = path
	t.Cleanup(func() { config.ConfigPath = previousPath })

	created, err := saveSimpleBot(core.BotUpsertRequest{
		DisplayName: "Clean", AgentType: "codex", WorkDir: dir,
		PermissionMode: "suggest", PlatformType: "telegram",
		Options: map[string]any{"token": "secret://cc-connect/test/telegram/token"},
	}, buildSetupCatalog(), secretstore.NewMemory())
	if err != nil {
		t.Fatalf("saveSimpleBot() error = %v", err)
	}
	if created.ReplyFooter {
		t.Fatal("new simple bot should hide answer diagnostics by default")
	}
	projects, err := config.ListProjectConfigs()
	if err != nil || len(projects) != 1 {
		t.Fatalf("projects = %d, err = %v", len(projects), err)
	}
	if projects[0].ReplyFooter == nil || *projects[0].ReplyFooter {
		t.Fatalf("persisted reply_footer = %v, want false", projects[0].ReplyFooter)
	}
}
