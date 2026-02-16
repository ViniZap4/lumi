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
	
	// TUI settings
	ShowLineNumbers bool
	CursorStyle     string // "block", "underline", "bar"
	PreviewLines    int
	SearchType      string // "filename" or "content"
}

func Load() *Config {
	cfg := &Config{
		Editor:          "nvim",
		EditorArgs:      []string{},
		Theme:           "dark",
		ShowLineNumbers: false,
		CursorStyle:     "block",
		PreviewLines:    10,
		SearchType:      "filename",
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
		case "show_line_numbers":
			cfg.ShowLineNumbers = value == "true"
		case "cursor_style":
			cfg.CursorStyle = value
		case "preview_lines":
			if value == "5" || value == "10" || value == "15" || value == "20" {
				cfg.PreviewLines = parseInt(value)
			}
		case "default_search_type":
			if value == "filename" || value == "content" {
				cfg.SearchType = value
			}
		}
	}
	
	return cfg
}

func parseInt(s string) int {
	switch s {
	case "5":
		return 5
	case "10":
		return 10
	case "15":
		return 15
	case "20":
		return 20
	default:
		return 10
	}
}
