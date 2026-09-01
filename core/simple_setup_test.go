package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSimpleSetup_DirectoryPickerEndpoint(t *testing.T) {
	server := NewManagementServer(0, "", nil)
	server.SetSetupDirectoryPicker(func() (string, bool, error) {
		return `D:\projects\example`, false, nil
	})
	handler := server.buildHandler(http.NewServeMux())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/setup/select-directory", nil)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Path      string `json:"path"`
			Cancelled bool   `json:"cancelled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Data.Path != `D:\projects\example` || response.Data.Cancelled {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestSimpleSetup_ModelCatalogEndpoint(t *testing.T) {
	server := NewManagementServer(0, "", nil)
	server.SetSetupModelCatalog(func(agentType string) (SetupModelCatalog, error) {
		if agentType != "codex" {
			t.Fatalf("agent = %q, want codex", agentType)
		}
		return SetupModelCatalog{
			Models:  []SetupModel{{Name: "gpt-test", Description: "Test model"}},
			Current: "gpt-test",
		}, nil
	})
	handler := server.buildHandler(http.NewServeMux())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/setup/models?agent=codex", nil)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"name":"gpt-test"`) || !strings.Contains(body, `"current":"gpt-test"`) {
		t.Fatalf("unexpected model response: %s", body)
	}
}

func TestSimpleSetup_ValidateBotRequest(t *testing.T) {
	catalog := SetupCatalog{Platforms: []SetupPlatform{{
		Key:    "telegram",
		Fields: []SetupField{{Key: "token", Required: true, Secret: true}},
	}}}
	req := BotUpsertRequest{DisplayName: "Bot", AgentType: "codex", WorkDir: t.TempDir(), PlatformType: "telegram", Options: map[string]any{}}
	if err := ValidateBotRequest(req, catalog); err == nil {
		t.Fatal("missing required platform token should fail")
	}
	req.Options["token"] = "test-token"
	if err := ValidateBotRequest(req, catalog); err != nil {
		t.Fatalf("valid request failed: %v", err)
	}
}

func TestSimpleSetup_BotResponseDoesNotEchoSecret(t *testing.T) {
	server := NewManagementServer(0, "", nil)
	server.SetSimpleSetupCallbacks(nil, nil, nil, func(req BotUpsertRequest) (BotSummary, error) {
		return BotSummary{ID: "id", DisplayName: req.DisplayName, Configured: true}, nil
	}, nil, nil)
	handler := server.buildHandler(http.NewServeMux())
	body := []byte(`{"display_name":"Bot","agent_type":"codex","work_dir":"C:/work","platform_type":"telegram","options":{"token":"never-return-this"}}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/bots", bytes.NewReader(body))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "never-return-this") {
		t.Fatalf("response leaked secret: %s", recorder.Body.String())
	}
}

func TestSimpleSetup_RedactConfigText(t *testing.T) {
	input := "token = \"plain\"\napp_secret = \"value\"\nwork_dir = \"C:/safe\"\n"
	got := redactConfigText(input)
	if strings.Contains(got, "plain") || strings.Contains(got, "value") {
		t.Fatalf("redacted config leaked secret: %s", got)
	}
	if !strings.Contains(got, "C:/safe") {
		t.Fatalf("redaction removed non-secret value: %s", got)
	}
}
