package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SetupField struct {
	Key         string              `json:"key"`
	Label       string              `json:"label"`
	Required    bool                `json:"required,omitempty"`
	Type        string              `json:"type,omitempty"`
	Secret      bool                `json:"secret,omitempty"`
	Placeholder string              `json:"placeholder,omitempty"`
	Hint        string              `json:"hint,omitempty"`
	Group       string              `json:"group,omitempty"`
	Options     []string            `json:"options,omitempty"`
	ShowWhen    map[string][]string `json:"show_when,omitempty"`
}

type SetupPlatform struct {
	Key    string       `json:"key"`
	Label  string       `json:"label"`
	QR     bool         `json:"qr,omitempty"`
	Fields []SetupField `json:"fields"`
}

type SetupAgent struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Recommended bool     `json:"recommended,omitempty"`
	Modes       []string `json:"modes,omitempty"`
}

type SetupCatalog struct {
	Agents    []SetupAgent    `json:"agents"`
	Platforms []SetupPlatform `json:"platforms"`
}

type AgentHealth struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Installed bool   `json:"installed"`
	LoggedIn  bool   `json:"logged_in"`
	Version   string `json:"version,omitempty"`
	Path      string `json:"path,omitempty"`
	Problem   string `json:"problem,omitempty"`
	Guide     string `json:"guide,omitempty"`
}

type SetupStatus struct {
	FirstRun bool          `json:"first_run"`
	BotCount int           `json:"bot_count"`
	Agents   []AgentHealth `json:"agents"`
}

type BotSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Enabled         bool   `json:"enabled"`
	AgentType       string `json:"agent_type"`
	WorkDir         string `json:"work_dir"`
	PermissionMode  string `json:"permission_mode,omitempty"`
	Model           string `json:"model,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	PlatformType    string `json:"platform_type"`
	Configured      bool   `json:"configured"`
	RuntimeState    string `json:"runtime_state"`
	RuntimeError    string `json:"runtime_error,omitempty"`
}

type BotUpsertRequest struct {
	ID              string         `json:"id,omitempty"`
	Name            string         `json:"name"`
	DisplayName     string         `json:"display_name"`
	Enabled         *bool          `json:"enabled,omitempty"`
	AgentType       string         `json:"agent_type"`
	WorkDir         string         `json:"work_dir"`
	PermissionMode  string         `json:"permission_mode,omitempty"`
	Model           string         `json:"model,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	PlatformType    string         `json:"platform_type"`
	Options         map[string]any `json:"options"`
}

func (m *ManagementServer) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		mgmtError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	mgmtJSON(w, http.StatusOK, map[string]any{"ready": true})
}

func (m *ManagementServer) handleSimpleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		mgmtError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	if m.getSetupStatus == nil {
		mgmtError(w, http.StatusNotImplemented, "setup status is not configured")
		return
	}
	status, err := m.getSetupStatus()
	if err != nil {
		mgmtError(w, http.StatusInternalServerError, err.Error())
		return
	}
	mgmtJSON(w, http.StatusOK, status)
}

func (m *ManagementServer) handleSimpleSetupCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		mgmtError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	if m.getSetupCatalog == nil {
		mgmtError(w, http.StatusNotImplemented, "setup catalog is not configured")
		return
	}
	mgmtJSON(w, http.StatusOK, m.getSetupCatalog())
}

func (m *ManagementServer) handleBots(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if m.listBots == nil {
			mgmtError(w, http.StatusNotImplemented, "bot listing is not configured")
			return
		}
		bots, err := m.listBots()
		if err != nil {
			mgmtError(w, http.StatusInternalServerError, err.Error())
			return
		}
		mgmtJSON(w, http.StatusOK, map[string]any{"bots": bots})
	case http.MethodPost:
		m.upsertBotRequest(w, r, "")
	default:
		mgmtError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

func (m *ManagementServer) handleBotRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/bots/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		mgmtError(w, http.StatusBadRequest, "bot id required")
		return
	}
	id := parts[0]
	if len(parts) == 2 && parts[1] == "state" {
		if r.Method != http.MethodPatch {
			mgmtError(w, http.StatusMethodNotAllowed, "PATCH only")
			return
		}
		var body struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Enabled == nil {
			mgmtError(w, http.StatusBadRequest, "enabled is required")
			return
		}
		if m.setBotEnabled == nil {
			mgmtError(w, http.StatusNotImplemented, "bot state is not configured")
			return
		}
		if err := m.setBotEnabled(id, *body.Enabled); err != nil {
			mgmtError(w, http.StatusBadRequest, err.Error())
			return
		}
		mgmtJSON(w, http.StatusOK, map[string]any{"configured": true, "applying": true})
		m.scheduleRestart()
		return
	}
	if len(parts) != 1 {
		mgmtError(w, http.StatusNotFound, "not found")
		return
	}
	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		m.upsertBotRequest(w, r, id)
	case http.MethodDelete:
		if m.removeBot == nil {
			mgmtError(w, http.StatusNotImplemented, "bot removal is not configured")
			return
		}
		if err := m.removeBot(id); err != nil {
			mgmtError(w, http.StatusBadRequest, err.Error())
			return
		}
		mgmtJSON(w, http.StatusOK, map[string]any{"configured": true, "applying": true})
		m.scheduleRestart()
	default:
		mgmtError(w, http.StatusMethodNotAllowed, "PUT, PATCH or DELETE only")
	}
}

func (m *ManagementServer) upsertBotRequest(w http.ResponseWriter, r *http.Request, id string) {
	if m.upsertBot == nil {
		mgmtError(w, http.StatusNotImplemented, "bot setup is not configured")
		return
	}
	var req BotUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mgmtError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if id != "" {
		req.ID = id
	}
	bot, err := m.upsertBot(req)
	if err != nil {
		mgmtError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Never echo request options: they may contain freshly entered secrets.
	mgmtJSON(w, http.StatusOK, map[string]any{"bot": bot, "configured": true, "applying": true})
	m.scheduleRestart()
}

func (m *ManagementServer) scheduleRestart() {
	go func() {
		time.Sleep(350 * time.Millisecond)
		select {
		case RestartCh <- RestartRequest{}:
		default:
		}
	}()
}

func ValidateBotRequest(req BotUpsertRequest, catalog SetupCatalog) error {
	if strings.TrimSpace(req.DisplayName) == "" {
		return fmt.Errorf("display_name is required")
	}
	if strings.TrimSpace(req.AgentType) == "" || strings.TrimSpace(req.PlatformType) == "" {
		return fmt.Errorf("agent_type and platform_type are required")
	}
	if strings.TrimSpace(req.WorkDir) == "" {
		return fmt.Errorf("work_dir is required")
	}
	for _, p := range catalog.Platforms {
		if p.Key != req.PlatformType {
			continue
		}
		for _, field := range p.Fields {
			if !field.Required {
				continue
			}
			if value, ok := req.Options[field.Key]; !ok || strings.TrimSpace(fmt.Sprint(value)) == "" {
				return fmt.Errorf("platform field %q is required", field.Key)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown platform %q", req.PlatformType)
}
