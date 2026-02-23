package theme

import "github.com/charmbracelet/lipgloss"

// Resolve selects the active theme based on mode and theme names, and sets Current.
func Resolve(mode, darkName, lightName string) Theme {
	custom := loadCustomThemes()

	var name string
	switch mode {
	case "light":
		name = lightName
	case "auto":
		if lipgloss.HasDarkBackground() {
			name = darkName
		} else {
			name = lightName
		}
	default: // "dark" or anything else
		name = darkName
	}

	// Lookup order: builtins first, then custom
	if t, ok := builtins[name]; ok {
		Current = t
		return t
	}
	if t, ok := custom[name]; ok {
		Current = t
		return t
	}

	// Fallback
	Current = builtins["tokyo-night"]
	return Current
}

// ThemeNamesForMode returns theme names appropriate for the given mode.
// For "dark" returns dark themes, for "light" returns light themes, for "auto" returns all.
func ThemeNamesForMode(mode string) []string {
	custom := loadCustomThemes()
	var names []string

	switch mode {
	case "light":
		names = LightThemeNames()
		for n, t := range custom {
			if !t.IsDark {
				names = append(names, n)
			}
		}
	case "dark":
		names = DarkThemeNames()
		for n, t := range custom {
			if t.IsDark {
				names = append(names, n)
			}
		}
	default:
		names = AllThemeNames()
		for n := range custom {
			names = append(names, n)
		}
	}

	return names
}
