package config

import (
	"os"
	"path/filepath"
	"strings"
)

// FolderConfig holds per-folder settings loaded from <rootDir>/.lumi/config.yaml.
type FolderConfig struct {
	ServerURL   string // URL of the lumi server (e.g. "http://localhost:8080")
	ServerToken string // auth token for the server
}

// LoadFolderConfig reads <rootDir>/.lumi/config.yaml and returns the folder config.
// Returns a zero-value FolderConfig if the file doesn't exist.
func LoadFolderConfig(rootDir string) *FolderConfig {
	cfg := &FolderConfig{}

	configPath := filepath.Join(rootDir, ".lumi", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "server_url":
			cfg.ServerURL = value
		case "server_token":
			cfg.ServerToken = value
		}
	}

	return cfg
}

// EnsureLumiDir creates the .lumi/ directory in rootDir if it doesn't exist.
func EnsureLumiDir(rootDir string) error {
	return os.MkdirAll(filepath.Join(rootDir, ".lumi"), 0755)
}
