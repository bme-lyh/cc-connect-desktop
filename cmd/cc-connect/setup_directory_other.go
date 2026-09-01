//go:build !windows

package main

import "fmt"

func selectSetupDirectory() (string, bool, error) {
	return "", false, fmt.Errorf("native directory selection is currently available only on Windows")
}
