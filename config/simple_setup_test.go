package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPermissive_AllowsZeroProjects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[management]\nenabled = true\nport = 9820\ntoken = \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPermissive(path); err != nil {
		t.Fatalf("LoadPermissive() error = %v", err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("Load() error = %v, want strict zero-project rejection", err)
	}
}

func TestProjectEnabled_DefaultAndExplicit(t *testing.T) {
	if !ProjectEnabled(ProjectConfig{}) {
		t.Fatal("legacy project should be enabled by default")
	}
	disabled := false
	if ProjectEnabled(ProjectConfig{Enabled: &disabled}) {
		t.Fatal("explicitly disabled project should not be enabled")
	}
}

func TestSimpleSetup_UpsertWritesBackupAndPendingMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	original := "[management]\nenabled = true\nport = 9820\ntoken = \"test\"\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	previousPath := ConfigPath
	ConfigPath = path
	t.Cleanup(func() { ConfigPath = previousPath })
	enabled, simple := true, true
	err := UpsertSimpleBot(ProjectConfig{
		ID: "11111111-1111-1111-1111-111111111111", Name: "bot-111111111111", DisplayName: "Test", Enabled: &enabled, SimpleMode: &simple,
		Agent:     AgentConfig{Type: "codex", Options: map[string]any{"work_dir": dir}},
		Platforms: []PlatformConfig{{Type: "telegram", Options: map[string]any{"token": "secret://cc-connect/id/telegram/token"}}},
	})
	if err != nil {
		t.Fatalf("UpsertSimpleBot() error = %v", err)
	}
	backup, err := os.ReadFile(path + ".previous")
	if err != nil || string(backup) != original {
		t.Fatalf("backup = %q, err = %v", backup, err)
	}
	marker, err := applyMarkerPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("pending marker missing: %v", err)
	}
}
