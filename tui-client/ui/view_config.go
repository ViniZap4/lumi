package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

func (m Model) renderConfig() string {
	var s strings.Builder

	// Vertical centering
	contentHeight := len(m.configItems) + 22 // items + title + preview + swatches + help
	topMargin := (m.height - contentHeight) / 2
	if topMargin < 0 {
		topMargin = 0
	}
	for i := 0; i < topMargin; i++ {
		s.WriteString("\n")
	}

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Primary).
		Width(m.width).
		Align(lipgloss.Center).
		Render("Lumi Settings")
	s.WriteString(title)
	s.WriteString("\n\n")

	// Build the item list
	maxLabelWidth := 0
	for _, item := range m.configItems {
		if item.Kind != ConfigHeader && len(item.Label) > maxLabelWidth {
			maxLabelWidth = len(item.Label)
		}
	}

	for i, item := range m.configItems {
		var line string

		switch item.Kind {
		case ConfigHeader:
			line = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Current.Primary).
				Render("    " + item.Label)
			if i > 0 {
				s.WriteString("\n")
			}

		case ConfigCycle:
			label := fmt.Sprintf("      %-*s", maxLabelWidth+2, item.Label)
			value := fmt.Sprintf("< %s >", item.Value)

			if i == m.configCursor {
				line = lipgloss.NewStyle().
					Foreground(theme.Current.Accent).
					Background(theme.Current.SelectedBg).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(theme.Current.Secondary).
						Background(theme.Current.SelectedBg).
						Bold(true).
						Render(value)
			} else {
				line = lipgloss.NewStyle().
					Foreground(theme.Current.Text).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(theme.Current.Muted).
						Render(value)
			}

		case ConfigAction:
			label := fmt.Sprintf("      %-*s", maxLabelWidth+2, item.Label)
			arrow := "->"

			if i == m.configCursor {
				line = lipgloss.NewStyle().
					Foreground(theme.Current.Accent).
					Background(theme.Current.SelectedBg).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(theme.Current.Secondary).
						Background(theme.Current.SelectedBg).
						Bold(true).
						Render(arrow)
			} else {
				line = lipgloss.NewStyle().
					Foreground(theme.Current.Text).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(theme.Current.Muted).
						Render(arrow)
			}
		}

		centered := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Render(line)
		s.WriteString(centered)
		s.WriteString("\n")
	}

	// Note preview – shows how the theme renders markdown content
	s.WriteString("\n")
	previewWidth := 52
	if previewWidth > m.width-8 {
		previewWidth = m.width - 8
	}
	if previewWidth < 20 {
		previewWidth = 20
	}
	t := theme.Current

	sep := lipgloss.NewStyle().Foreground(t.Separator).
		Render(strings.Repeat("─", previewWidth))

	previewSamples := []string{
		"# Heading 1",
		"## Heading 2",
		"",
		"Normal text with **bold** and *italic*.",
		"A `code span` and a [link](url).",
		"",
		"- List item one",
		"> Blockquote text",
	}
	codeLines := codeBlockLines(previewSamples)

	var previewBuf strings.Builder
	previewBuf.WriteString(sep)
	previewBuf.WriteString("\n")
	for i, line := range previewSamples {
		inCode := codeLines[i]
		style := mdLineStyle(line, inCode)
		var inlineCls []int
		if shouldClassifyInline(line, inCode) {
			inlineCls = classifyInline(line)
		}
		rendered := m.renderContentLine(line, style, inlineCls, visualRange{}, lipgloss.Color(""), false)
		previewBuf.WriteString("  ")
		previewBuf.WriteString(rendered)
		previewBuf.WriteString("\n")
	}
	previewBuf.WriteString(sep)

	preview := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(previewBuf.String())
	s.WriteString(preview)
	s.WriteString("\n")

	// Color swatches
	swatchColors := []lipgloss.Color{t.Primary, t.Secondary, t.Accent, t.Muted, t.Text, t.Error, t.Warning, t.Info}
	var swatches strings.Builder
	for i, c := range swatchColors {
		swatches.WriteString(lipgloss.NewStyle().Foreground(c).Render("██"))
		if i < len(swatchColors)-1 {
			swatches.WriteString(" ")
		}
	}
	swatchLine := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(swatches.String())
	s.WriteString(swatchLine)
	s.WriteString("\n\n")

	// Help bar
	helpParts := []struct{ key, desc string }{
		{"j/k", "move"},
		{"h/l", "change"},
		{"enter", "select"},
		{"esc", "back"},
	}
	var parts []string
	for _, h := range helpParts {
		key := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true).Render(h.key)
		desc := lipgloss.NewStyle().Foreground(mutedColor).Render(" " + h.desc)
		parts = append(parts, key+desc)
	}
	helpLine := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(strings.Join(parts, "  "))
	s.WriteString(helpLine)

	return s.String()
}
