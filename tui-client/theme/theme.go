package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the complete color palette for the TUI.
type Theme struct {
	Name       string
	IsDark     bool
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Muted      lipgloss.Color
	Background lipgloss.Color
	SelectedBg lipgloss.Color
	OverlayBg  lipgloss.Color
	Text       lipgloss.Color
	TextDim    lipgloss.Color
	Border     lipgloss.Color
	Separator  lipgloss.Color
	Error      lipgloss.Color
	Warning    lipgloss.Color
	Info       lipgloss.Color
	LogoColors [6]lipgloss.Color
}

// Current is the active theme, set by Resolve().
var Current Theme
