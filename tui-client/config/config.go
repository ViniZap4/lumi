// tui-client/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Editor      string
	EditorArgs  []string

	// Theme settings
	ThemeMode  string // "dark", "light", "auto"
	DarkTheme  string
	LightTheme string

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
		ThemeMode:       "dark",
		DarkTheme:       "tokyo-night",
		LightTheme:      "catppuccin-latte",
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
			editorParts := strings.Fields(value)
			if len(editorParts) > 0 {
				cfg.Editor = editorParts[0]
				if len(editorParts) > 1 {
					cfg.EditorArgs = editorParts[1:]
				}
			}
		case "theme_mode":
			if value == "dark" || value == "light" || value == "auto" {
				cfg.ThemeMode = value
			}
		case "dark_theme":
			cfg.DarkTheme = value
		case "light_theme":
			cfg.LightTheme = value
		// Legacy: map old "theme" field to theme_mode
		case "theme":
			if value == "dark" || value == "light" || value == "auto" {
				cfg.ThemeMode = value
			}
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

// Save writes the config back to ~/.config/lumi/config.yaml.
func (c *Config) Save() error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "lumi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	editorCmd := c.Editor
	if len(c.EditorArgs) > 0 {
		editorCmd += " " + strings.Join(c.EditorArgs, " ")
	}

	showLn := "false"
	if c.ShowLineNumbers {
		showLn = "true"
	}

	content := fmt.Sprintf(`# Lumi Configuration

# Editor command with args
editor: %s

# Theme
theme_mode: %s
dark_theme: %s
light_theme: %s

# TUI Settings
show_line_numbers: %s
cursor_style: %s
preview_lines: %d
default_search_type: %s
`, editorCmd, c.ThemeMode, c.DarkTheme, c.LightTheme, showLn, c.CursorStyle, c.PreviewLines, c.SearchType)

	return os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0644)
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
