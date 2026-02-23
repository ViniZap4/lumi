package theme

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// loadCustomThemes reads .yaml files from ~/.config/lumi/themes/ and returns them keyed by name.
func loadCustomThemes() map[string]Theme {
	themes := make(map[string]Theme)

	dir := filepath.Join(os.Getenv("HOME"), ".config", "lumi", "themes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return themes
	}

	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		t, ok := parseThemeYAML(data)
		if ok && t.Name != "" {
			themes[t.Name] = t
		}
	}

	return themes
}

func parseThemeYAML(data []byte) (Theme, bool) {
	t := Theme{}
	var logoColors []string

	lines := strings.Split(string(data), "\n")
	inLogo := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			if inLogo && trimmed == "" {
				inLogo = false
			}
			continue
		}

		// Handle logo_colors list items
		if inLogo {
			if strings.HasPrefix(trimmed, "- ") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				val = strings.Trim(val, "\"'")
				logoColors = append(logoColors, val)
				continue
			}
			inLogo = false
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")

		switch key {
		case "name":
			t.Name = value
		case "is_dark":
			t.IsDark = value == "true"
		case "primary":
			t.Primary = lipgloss.Color(value)
		case "secondary":
			t.Secondary = lipgloss.Color(value)
		case "accent":
			t.Accent = lipgloss.Color(value)
		case "muted":
			t.Muted = lipgloss.Color(value)
		case "background":
			t.Background = lipgloss.Color(value)
		case "selected_bg":
			t.SelectedBg = lipgloss.Color(value)
		case "overlay_bg":
			t.OverlayBg = lipgloss.Color(value)
		case "text":
			t.Text = lipgloss.Color(value)
		case "text_dim":
			t.TextDim = lipgloss.Color(value)
		case "border":
			t.Border = lipgloss.Color(value)
		case "separator":
			t.Separator = lipgloss.Color(value)
		case "error":
			t.Error = lipgloss.Color(value)
		case "warning":
			t.Warning = lipgloss.Color(value)
		case "info":
			t.Info = lipgloss.Color(value)
		case "logo_colors":
			inLogo = true
			logoColors = nil
		}
	}

	// Fill LogoColors array
	if len(logoColors) > 0 {
		for i := 0; i < 6; i++ {
			t.LogoColors[i] = lipgloss.Color(logoColors[i%len(logoColors)])
		}
	} else {
		// Default: cycle primary, secondary, accent
		t.LogoColors = [6]lipgloss.Color{t.Primary, t.Secondary, t.Accent, t.Primary, t.Secondary, t.Accent}
	}

	return t, t.Name != ""
}
