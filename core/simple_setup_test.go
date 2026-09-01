package core

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
