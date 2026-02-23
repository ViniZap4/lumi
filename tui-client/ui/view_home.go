package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHome() string {
	var s strings.Builder

	// Vertical centering
	contentHeight := 18
	topMargin := (m.height - contentHeight) / 2
	if topMargin < 0 {
		topMargin = 0
	}
	for i := 0; i < topMargin; i++ {
		s.WriteString("\n")
	}

	// ASCII art logo
	art := `
  ██╗     ██╗   ██╗███╗   ███╗██╗
  ██║     ██║   ██║████╗ ████║██║
  ██║     ██║   ██║██╔████╔██║██║
  ██║     ██║   ██║██║╚██╔╝██║██║
  ███████╗╚██████╔╝██║ ╚═╝ ██║██║
  ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝
`
	artStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center)
	s.WriteString(artStyle.Render(art))
	s.WriteString("\n")

	// Subtitle
	subtitle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(m.width).
		Align(lipgloss.Center).
		Render("Local-first Markdown notes")
	s.WriteString(subtitle)
	s.WriteString("\n\n\n")

	// Keybindings
	keys := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Width(m.width).
		Align(lipgloss.Center).
		Render(
			"/  Search notes\n" +
				"t  Browse tree\n" +
				"c  Edit config\n" +
				"q  Quit",
		)
	s.WriteString(keys)

	return s.String()
}
