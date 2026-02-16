// tui-client/ui/styles.go
package ui

import "github.com/charmbracelet/lipgloss"

var (
	ActiveBorder   = lipgloss.RoundedBorder()
	InactiveBorder = lipgloss.NormalBorder()

	ActiveStyle = lipgloss.NewStyle().
			Border(ActiveBorder).
			BorderForeground(lipgloss.Color("62"))

	InactiveStyle = lipgloss.NewStyle().
			Border(InactiveBorder).
			BorderForeground(lipgloss.Color("240"))

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
