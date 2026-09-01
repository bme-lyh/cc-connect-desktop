//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func startDesktopProcess(execPath string, args []string, logPath string) error {
	cmd := exec.Command(execPath, args...)
	cmd.Env = append(os.Environ(), "CC_LOG_FILE="+logPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008 | 0x00000200, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	}
	return cmd.Start()
}
