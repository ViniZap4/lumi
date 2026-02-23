package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

var (
	// Color aliases (updated by ApplyTheme)
	primaryColor   lipgloss.Color
	secondaryColor lipgloss.Color
	accentColor    lipgloss.Color
	mutedColor     lipgloss.Color
	bgColor        lipgloss.Color
	selectedBg     lipgloss.Color

	// Borders
	ActiveBorder   = lipgloss.RoundedBorder()
	InactiveBorder = lipgloss.Border{}

	// Panel styles
	ActivePanelStyle   lipgloss.Style
	InactivePanelStyle lipgloss.Style

	// Title styles
	TitleStyle lipgloss.Style

	// Item styles
	SelectedItemStyle lipgloss.Style
	NormalItemStyle   lipgloss.Style
	DimItemStyle      lipgloss.Style

	// Help styles
	HelpStyle    lipgloss.Style
	HelpKeyStyle lipgloss.Style

	// Preview styles
	PreviewTitleStyle   lipgloss.Style
	PreviewMetaStyle    lipgloss.Style
	PreviewContentStyle lipgloss.Style
	PreviewLinkStyle    lipgloss.Style

	// Status bar
	StatusBarStyle lipgloss.Style
)

// ApplyTheme rebuilds all package-level style vars from theme.Current.
func ApplyTheme() {
	t := theme.Current

	primaryColor = t.Primary
	secondaryColor = t.Secondary
	accentColor = t.Accent
	mutedColor = t.Muted
	bgColor = t.Background
	selectedBg = t.SelectedBg

	ActivePanelStyle = lipgloss.NewStyle().
		Border(ActiveBorder).
		BorderForeground(primaryColor).
		Padding(0, 1)

	InactivePanelStyle = lipgloss.NewStyle().
		Border(InactiveBorder).
		BorderForeground(mutedColor).
		Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Padding(0, 1)

	SelectedItemStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Background(selectedBg).
		Bold(true).
		Padding(0, 1)

	NormalItemStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Padding(0, 1)

	DimItemStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)

	PreviewTitleStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Underline(true).
		MarginBottom(1)

	PreviewMetaStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)

	PreviewContentStyle = lipgloss.NewStyle().
		Foreground(t.Text)

	PreviewLinkStyle = lipgloss.NewStyle().
		Foreground(secondaryColor).
		Underline(true)

	StatusBarStyle = lipgloss.NewStyle().
		Background(bgColor).
		Foreground(t.Text).
		Padding(0, 1)
}

func init() {
	// Set defaults before theme is resolved, so styles are never nil
	theme.Current = theme.Theme{
		Primary:    lipgloss.Color("99"),
		Secondary:  lipgloss.Color("141"),
		Accent:     lipgloss.Color("212"),
		Muted:      lipgloss.Color("241"),
		Background: lipgloss.Color("235"),
		SelectedBg: lipgloss.Color("237"),
		OverlayBg:  lipgloss.Color("0"),
		Text:       lipgloss.Color("252"),
		TextDim:    lipgloss.Color("240"),
		Border:     lipgloss.Color("62"),
		Separator:  lipgloss.Color("236"),
		Error:      lipgloss.Color("196"),
		Warning:    lipgloss.Color("226"),
		Info:       lipgloss.Color("81"),
		LogoColors: [6]lipgloss.Color{"99", "105", "111", "141", "147", "183"},
	}
	ApplyTheme()
}
