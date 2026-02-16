// tui-client/config/config.go
package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Editor      string
	EditorArgs  []string
	Theme       string
}

func Load() *Config {
	cfg := &Config{
		Editor:     "nvim",
		EditorArgs: []string{},
		Theme:      "dark",
	}
	
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "lumi", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}
	
	// Simple YAML parsing
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
		case "editor":
			// Parse "nvim --no-dashboard" into command and args
			editorParts := strings.Fields(value)
			if len(editorParts) > 0 {
				cfg.Editor = editorParts[0]
				if len(editorParts) > 1 {
					cfg.EditorArgs = editorParts[1:]
				}
			}
		case "theme":
			cfg.Theme = value
		}
	}
	
	return cfg
}
