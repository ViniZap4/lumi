package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// logoFull is the complete ASCII art that gets animated character-by-character.
var logoFull = strings.Join([]string{
	`  ██╗     ██╗   ██╗███╗   ███╗██╗`,
	`  ██║     ██║   ██║████╗ ████║██║`,
	`  ██║     ██║   ██║██╔████╔██║██║`,
	`  ██║     ██║   ██║██║╚██╔╝██║██║`,
	`  ███████╗╚██████╔╝██║ ╚═╝ ██║██║`,
	`  ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝`,
}, "\n")

func (m Model) renderHome() string {
	var s strings.Builder

	// Vertical centering
	contentHeight := 20
	topMargin := (m.height - contentHeight) / 2
	if topMargin < 0 {
		topMargin = 0
	}
	for i := 0; i < topMargin; i++ {
		s.WriteString("\n")
	}

	// Animated logo: reveal characters progressively
	visible := logoFull
	if !m.animDone && m.animPos < len(logoFull) {
		visible = logoFull[:m.animPos]
	}

	// Color the visible portion with a gradient effect
	artLines := strings.Split(visible, "\n")
	colors := []lipgloss.Color{"99", "105", "111", "141", "147", "183"}

	var artRendered strings.Builder
	for i, line := range artLines {
		color := colors[i%len(colors)]
		artRendered.WriteString(lipgloss.NewStyle().
			Foreground(color).
			Bold(true).
			Render(line))
		if i < len(artLines)-1 {
			artRendered.WriteString("\n")
		}
	}

	// Center the art
	artBlock := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(artRendered.String())
	s.WriteString(artBlock)
	s.WriteString("\n\n")

	// Only show the rest after animation completes
	if m.animDone {
		// Subtitle
		subtitle := lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(m.width).
			Align(lipgloss.Center).
			Render("Local-first Markdown notes")
		s.WriteString(subtitle)
		s.WriteString("\n\n\n")

		// Keybindings
		keys := []struct{ key, desc string }{
			{"/", "Search notes"},
			{"t", "Browse tree"},
			{"c", "Edit config"},
			{"q", "Quit"},
		}

		var keysBlock strings.Builder
		for _, k := range keys {
			keyStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(k.key)
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("  " + k.desc)
			keysBlock.WriteString(keyStyle + descStyle + "\n")
		}

		centered := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Render(keysBlock.String())
		s.WriteString(centered)
	}

	return s.String()
}
