package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chenhg5/cc-connect/config"
	"github.com/chenhg5/cc-connect/core"
)

func runWebCommand(args []string) {
	if err := runWeb(args); err != nil {
		fmt.Fprintf(os.Stderr, "cc-connect desktop: %v\n", err)
		os.Exit(1)
	}
}

func runWeb(args []string) error {
	configPath := resolveConfigPath("")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := bootstrapConfig(configPath); err != nil {
			return fmt.Errorf("create config: %w", err)
		}
		fmt.Printf("Created minimal config at %s\n", configPath)
	}

	// Use LoadPermissive so `cc-connect web` works even before any platforms
	// are configured (e.g. during initial setup via the Web Admin UI).
	cfg, err := config.LoadPermissive(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	config.ConfigPath = configPath

	mgmtEnabled := cfg.Management.Enabled != nil && *cfg.Management.Enabled
	port := cfg.Management.Port
	if port == 0 {
		port = 9820
	}
	token := cfg.Management.Token

	if !mgmtEnabled || token == "" {
		fmt.Println("Configuring local management UI...")
		configuredPort, configuredToken, err := config.EnsureManagement(core.GenerateToken(24))
		if err != nil {
			return fmt.Errorf("enable web admin: %w", err)
		}
		port = configuredPort
		token = configuredToken
	}

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if !managementReady(baseURL, token) {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate cc-connect executable: %w", err)
		}
		logDir := filepath.Join(filepath.Dir(configPath), "logs")
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return fmt.Errorf("create log directory: %w", err)
		}
		logPath := filepath.Join(logDir, "desktop.log")
		if err := startDesktopProcess(execPath, []string{"--config", configPath}, logPath); err != nil {
			return fmt.Errorf("start cc-connect: %w", err)
		}
		if err := waitForManagement(baseURL, token, 20*time.Second); err != nil {
			return fmt.Errorf("cc-connect did not become ready: %w (log: %s)", err, logPath)
		}
	}

	noBrowser := false
	for _, a := range args {
		if a == "--no-browser" || a == "-n" {
			noBrowser = true
		}
	}

	if noBrowser {
		fmt.Printf("URL:   %s\n", baseURL)
		fmt.Printf("Token: %s\n", token)
		return nil
	}

	loginURL := fmt.Sprintf("%s/login?token=%s",
		baseURL, url.QueryEscape(token))

	fmt.Printf("Opening: %s\n", baseURL)
	if err := openBrowser(loginURL); err != nil {
		fmt.Printf("\nCould not open browser automatically.\n")
		fmt.Printf("Open this URL in your browser:\n")
		fmt.Printf("  %s/login?token=%s\n", baseURL, token)
		fmt.Printf("\nNote: make sure cc-connect is running (it hosts the web admin on port %d).\n", port)
	}
	return nil
}

func managementReady(baseURL, token string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/ready", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func waitForManagement(baseURL, token string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if managementReady(baseURL, token) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", baseURL)
}

func openBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start()
	case "linux":
		if isWSL() {
			return exec.Command("cmd.exe", "/c", "start", rawURL).Start()
		}
		// On headless Linux, xdg-open is often unavailable.
		// Check early and return a clear error so the caller can print the URL.
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return fmt.Errorf("xdg-open not found (headless server?): %w", err)
		}
		return exec.Command("xdg-open", rawURL).Start()
	case "windows":
		// Avoid cmd.exe's command-line parsing here. It can drop or reinterpret
		// URL query characters, which leaves the desktop flow on the manual
		// token screen instead of completing the one-click login.
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", rawURL).Start()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}
