package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

func (m Model) renderConfig() string {
	t := theme.Current

	sepWidth := 3 // " │ "
	leftWidth := m.width * 38 / 100
	rightWidth := m.width - leftWidth - sepWidth
	colHeight := m.height

	leftCol := m.renderConfigLeft(leftWidth, colHeight)
	rightCol := m.renderConfigPreview(rightWidth, colHeight)

	// Build the separator column as a fixed-height block
	sepChar := lipgloss.NewStyle().Foreground(t.Separator).Render(" │ ")
	sepLines := make([]string, colHeight)
	for i := range sepLines {
		sepLines[i] = sepChar
	}
	sep := strings.Join(sepLines, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, sep, rightCol)
}

// renderConfigLeft renders the left column: title, config items, swatches, help.
func (m Model) renderConfigLeft(width, height int) string {
	var s strings.Builder
	t := theme.Current

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Width(width).
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
				Foreground(t.Primary).
				Render("  " + item.Label)
			if i > 0 {
				s.WriteString("\n")
			}

		case ConfigCycle:
			label := fmt.Sprintf("    %-*s", maxLabelWidth+2, item.Label)
			value := fmt.Sprintf("< %s >", item.Value)

			if i == m.configCursor {
				line = lipgloss.NewStyle().
					Foreground(t.Accent).
					Background(t.SelectedBg).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(t.Secondary).
						Background(t.SelectedBg).
						Bold(true).
						Render(value)
			} else {
				line = lipgloss.NewStyle().
					Foreground(t.Text).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(t.Muted).
						Render(value)
			}

		case ConfigAction:
			label := fmt.Sprintf("    %-*s", maxLabelWidth+2, item.Label)
			arrow := "->"

			if i == m.configCursor {
				line = lipgloss.NewStyle().
					Foreground(t.Accent).
					Background(t.SelectedBg).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(t.Secondary).
						Background(t.SelectedBg).
						Bold(true).
						Render(arrow)
			} else {
				line = lipgloss.NewStyle().
					Foreground(t.Text).
					Render(label) +
					lipgloss.NewStyle().
						Foreground(t.Muted).
						Render(arrow)
			}
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	// Color swatches
	s.WriteString("\n")
	swatchColors := []lipgloss.Color{t.Primary, t.Secondary, t.Accent, t.Muted, t.Text, t.Error, t.Warning, t.Info}
	var swatches strings.Builder
	for i, c := range swatchColors {
		swatches.WriteString(lipgloss.NewStyle().Foreground(c).Render("██"))
		if i < len(swatchColors)-1 {
			swatches.WriteString(" ")
		}
	}
	swatchLine := lipgloss.NewStyle().
		Width(width).
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
		Width(width).
		Align(lipgloss.Center).
		Render(strings.Join(parts, "  "))
	s.WriteString(helpLine)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}

// renderConfigPreview renders the right column: a full sample note preview.
func (m Model) renderConfigPreview(width, height int) string {
	var s strings.Builder
	t := theme.Current

	// --- Header: title + tags (left) + date (right) ---
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Render(" Sample Note")

	tagStr := "  " + lipgloss.NewStyle().
		Foreground(t.Accent).
		Render("#demo") +
		" " + lipgloss.NewStyle().
		Foreground(t.Accent).
		Render("#theme")

	dateStyled := lipgloss.NewStyle().
		Foreground(t.Muted).
		Render("Jan 1, 2026 ")

	tw := lipgloss.Width(titleStyled) + lipgloss.Width(tagStr)
	dw := lipgloss.Width(dateStyled)
	gap := width - tw - dw
	if gap < 1 {
		gap = 1
	}
	s.WriteString(titleStyled + tagStr + strings.Repeat(" ", gap) + dateStyled)
	s.WriteString("\n")

	// Header separator
	s.WriteString(lipgloss.NewStyle().
		Foreground(t.Separator).
		Render(strings.Repeat("─", width)))
	s.WriteString("\n")

	// --- Sample markdown content ---
	previewSamples := []string{
		"# Heading 1",
		"## Heading 2",
		"### Heading 3",
		"",
		"Normal text with **bold** and *italic*.",
		"A `code span` and a [link](url).",
		"",
		"- List item one",
		"- Another with [[wikilink]]",
		"",
		"> Blockquote text here",
		"",
		"```",
		"code block line",
		"```",
		"",
		"---",
	}
	codeLines := codeBlockLines(previewSamples)

	for i, line := range previewSamples {
		inCode := codeLines[i]
		style := mdLineStyle(line, inCode)
		var inlineCls []int
		if shouldClassifyInline(line, inCode) {
			inlineCls = classifyInline(line)
		}
		rendered := m.renderContentLine(line, style, inlineCls, visualRange{}, lipgloss.Color(""), false)
		s.WriteString("  ")
		s.WriteString(rendered)
		s.WriteString("\n")
	}

	// --- Footer: separator + status bar ---
	// Fill remaining space before footer
	usedLines := 2 + len(previewSamples) + 2 // header(2) + content + footer(2)
	remaining := height - usedLines
	for i := 0; i < remaining; i++ {
		s.WriteString("\n")
	}

	s.WriteString(lipgloss.NewStyle().
		Foreground(t.Separator).
		Render(strings.Repeat("─", width)))
	s.WriteString("\n")
	s.WriteString(StatusBarStyle.Width(width).Render("Ln 1  Col 1"))

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}
