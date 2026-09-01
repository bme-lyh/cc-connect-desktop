//go:build windows

package main

import (
	"os"
	"os/exec"
)

func restartProcess(execPath string) error {
	return restartCommand(execPath).Start()
}

func restartCommand(execPath string) *exec.Cmd {
	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	configureBackgroundCommand(cmd)
	return cmd
}
