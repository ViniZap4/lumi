// tui-client/ui/styles.go
package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("99")
	secondaryColor = lipgloss.Color("141")
	accentColor    = lipgloss.Color("212")
	mutedColor     = lipgloss.Color("241")
	bgColor        = lipgloss.Color("235")
	selectedBg     = lipgloss.Color("237")

	// Borders
	ActiveBorder   = lipgloss.RoundedBorder()
	InactiveBorder = lipgloss.Border{}

	// Panel styles
	ActivePanelStyle = lipgloss.NewStyle().
				Border(ActiveBorder).
				BorderForeground(primaryColor).
				Padding(0, 1)

	InactivePanelStyle = lipgloss.NewStyle().
				Border(InactiveBorder).
				BorderForeground(mutedColor).
				Padding(0, 1)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	// Item styles
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Padding(0, 1)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	DimItemStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// Preview styles
	PreviewTitleStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Underline(true).
				MarginBottom(1)

	PreviewMetaStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true)

	PreviewContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	PreviewLinkStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Underline(true)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)
)
