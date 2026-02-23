package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

// logoLines holds each line of the ASCII art separately for line-by-line animation.
var logoLines = []string{
	`  ██╗     ██╗   ██╗███╗   ███╗██╗`,
	`  ██║     ██║   ██║████╗ ████║██║`,
	`  ██║     ██║   ██║██╔████╔██║██║`,
	`  ██║     ██║   ██║██║╚██╔╝██║██║`,
	`  ███████╗╚██████╔╝██║ ╚═╝ ██║██║`,
	`  ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝`,
}

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

	// Animated logo: reveal lines progressively
	visibleCount := len(logoLines)
	if !m.animDone && m.animLine < len(logoLines) {
		visibleCount = m.animLine
	}

	// Color each visible line with theme gradient
	var artRendered strings.Builder
	for i := 0; i < visibleCount; i++ {
		color := theme.Current.LogoColors[i%len(theme.Current.LogoColors)]
		artRendered.WriteString(lipgloss.NewStyle().
			Foreground(color).
			Bold(true).
			Render(logoLines[i]))
		if i < visibleCount-1 {
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
			{"c", "Settings"},
			{"q", "Quit"},
		}

		var keysBlock strings.Builder
		for _, k := range keys {
			keyStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(k.key)
			descStyle := lipgloss.NewStyle().Foreground(theme.Current.Text).Render("  " + k.desc)
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
