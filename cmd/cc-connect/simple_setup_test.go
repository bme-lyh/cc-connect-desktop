package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhg5/cc-connect/config"
	"github.com/chenhg5/cc-connect/core"
	"github.com/chenhg5/cc-connect/internal/secretstore"
)

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
