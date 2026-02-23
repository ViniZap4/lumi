package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

// logoLines holds each line of the ASCII art for the diagonal wipe animation.
var logoLines = []string{
	`██╗     ██╗   ██╗███╗   ███╗██╗`,
	` ██║     ██║   ██║████╗ ████║██║`,
	` ██║     ██║   ██║██╔████╔██║██║`,
	` ██║     ██║   ██║██║╚██╔╝██║██║`,
	` ███████╗╚██████╔╝██║ ╚═╝ ██║██║`,
	` ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝`,
}

// logoRuneWidths caches the rune length of each logo line.
var logoRuneWidths []int

func init() {
	logoRuneWidths = make([]int, len(logoLines))
	for i, line := range logoLines {
		logoRuneWidths[i] = len([]rune(line))
	}
}

// logoMaxRunes returns the rune length of the longest logo line.
func logoMaxRunes() int {
	m := 0
	for _, w := range logoRuneWidths {
		if w > m {
			m = w
		}
	}
	return m
}

// logoStagger returns the total diagonal offset across all lines.
// Each line after the first is delayed by 2 rune columns.
func logoStagger() int {
	return (len(logoLines) - 1) * 2
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

	// Diagonal left-to-right wipe: each line is offset by 2 columns.
	// Pad every line to the same rune width so lipgloss centers the
	// block as a whole instead of centering each line independently.
	maxW := logoMaxRunes()

	var artRendered strings.Builder
	for i, line := range logoLines {
		runes := []rune(line)
		color := theme.Current.LogoColors[i%len(theme.Current.LogoColors)]

		visible := m.animCol - i*2 // diagonal offset
		if m.animDone {
			visible = len(runes)
		}
		if visible < 0 {
			visible = 0
		}
		if visible > len(runes) {
			visible = len(runes)
		}

		// Render the visible portion, then pad with spaces to maxW
		visibleStr := ""
		if visible > 0 {
			visibleStr = lipgloss.NewStyle().
				Foreground(color).
				Bold(true).
				Render(string(runes[:visible]))
		}
		pad := maxW - len(runes)
		if pad < 0 {
			pad = 0
		}
		artRendered.WriteString(visibleStr + strings.Repeat(" ", pad))
		if i < len(logoLines)-1 {
			artRendered.WriteString("\n")
		}
	}

	// Center the whole block as one unit
	artBlock := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, artRendered.String())
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
