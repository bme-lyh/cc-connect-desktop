//go:build windows

package main

import (
	"strings"
	"testing"
)

func TestSimpleSetup_DirectoryPickerUsesForegroundOwner(t *testing.T) {
	for _, expected := range []string{
		"$owner.TopMost = $true",
		"SetForegroundWindow($owner.Handle)",
		"$dialog.ShowDialog($owner)",
	} {
		if !strings.Contains(setupDirectoryPickerScript, expected) {
			t.Fatalf("directory picker script is missing %q", expected)
		}
	}
}
