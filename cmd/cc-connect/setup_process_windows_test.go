//go:build windows

package main

import (
	"context"
	"testing"
)

func TestSimpleSetup_AgentDetectionCommandsAreHiddenOnWindows(t *testing.T) {
	cmd := setupAgentCommand(context.Background(), "codex.exe", "--version")
	if cmd.SysProcAttr == nil {
		t.Fatal("setup agent command is missing Windows process attributes")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("setup agent command must hide its console window")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("creation flags = %#x, want CREATE_NO_WINDOW", cmd.SysProcAttr.CreationFlags)
	}
}

func TestRestartCommandIsHiddenOnWindows(t *testing.T) {
	cmd := restartCommand("cc-connect.exe")
	if cmd.SysProcAttr == nil {
		t.Fatal("restart command is missing Windows process attributes")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("restart command must hide its console window")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("creation flags = %#x, want CREATE_NO_WINDOW", cmd.SysProcAttr.CreationFlags)
	}
}
