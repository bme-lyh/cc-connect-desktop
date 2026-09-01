//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const setupDirectoryPickerScript = `$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.Application]::EnableVisualStyles()
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class CCConnectWindow {
    [DllImport("user32.dll")]
    public static extern bool SetForegroundWindow(IntPtr hWnd);
}
'@
$screen = [System.Windows.Forms.Screen]::FromPoint([System.Windows.Forms.Cursor]::Position)
$owner = New-Object System.Windows.Forms.Form
$owner.Text = 'CC-Connect'
$owner.ShowInTaskbar = $false
$owner.TopMost = $true
$owner.StartPosition = [System.Windows.Forms.FormStartPosition]::Manual
$owner.Size = New-Object System.Drawing.Size(1, 1)
$owner.Location = New-Object System.Drawing.Point(
    ($screen.WorkingArea.Left + [Math]::Floor($screen.WorkingArea.Width / 2)),
    ($screen.WorkingArea.Top + [Math]::Floor($screen.WorkingArea.Height / 2))
)
$owner.Opacity = 0.01
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = 'Select the working directory'
$dialog.ShowNewFolderButton = $true
try {
    $owner.Show()
    $owner.Activate() | Out-Null
    $owner.BringToFront() | Out-Null
    [CCConnectWindow]::SetForegroundWindow($owner.Handle) | Out-Null
    $result = $dialog.ShowDialog($owner)
    if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
        [Console]::Out.Write($dialog.SelectedPath)
        exit 0
    }
    exit 2
} finally {
    $dialog.Dispose()
    $owner.Close()
    $owner.Dispose()
}
`

func selectSetupDirectory() (string, bool, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-WindowStyle", "Hidden", "-Command", setupDirectoryPickerScript)
	configureBackgroundCommand(cmd)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			return "", true, nil
		}
		return "", false, fmt.Errorf("open Windows directory picker: %w", err)
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", true, nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false, fmt.Errorf("directory picker returned an invalid path: %s", path)
	}
	return path, false, nil
}
