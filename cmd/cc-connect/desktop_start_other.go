//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func startDesktopProcess(execPath string, args []string, logPath string) error {
	cmd := exec.Command(execPath, args...)
	cmd.Env = append(os.Environ(), "CC_LOG_FILE="+logPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}
