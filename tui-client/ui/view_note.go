package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/image"
	"github.com/vinizap/lumi/tui-client/theme"
)

func (m Model) renderFullNote() string {
	if m.fullNote == nil {
		return "No note loaded"
	}

	var s strings.Builder

	// --- Header: title + tags (left) + date (right) + separator ---
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(" " + m.fullNote.Title)

	var tagStr string
	if len(m.fullNote.Tags) > 0 {
		var tagParts []string
		for _, tag := range m.fullNote.Tags {
			tagParts = append(tagParts, lipgloss.NewStyle().
				Foreground(theme.Current.Accent).
				Render("#"+tag))
		}
		tagStr = "  " + strings.Join(tagParts, " ")
	}

	dateStyled := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(m.fullNote.UpdatedAt.Format("Jan 2, 2006") + " ")
	tw := lipgloss.Width(titleStyled) + lipgloss.Width(tagStr)
	dw := lipgloss.Width(dateStyled)
	gap := m.width - tw - dw
	if gap < 1 {
		gap = 1
	}
	s.WriteString(titleStyled + tagStr + strings.Repeat(" ", gap) + dateStyled)
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().
		Foreground(theme.Current.Separator).
		Render(strings.Repeat("─", m.width)))
	s.WriteString("\n")

	// --- Content ---
	// Build display lines from raw content, tracking which raw line each
	// display line comes from. Most lines are 1:1; image lines may expand.
	rawLines := m.contentLines
	displayLines := make([]string, 0, len(rawLines))
	rawToDisplay := make([]int, len(rawLines))

	for i, line := range rawLines {
		rawToDisplay[i] = len(displayLines)
		if image.HasImage(line) {
			imgPath := image.GetImagePath(line, m.fullNote.Path)
			if imgPath != "" {
				if _, err := os.Stat(imgPath); err == nil {
					rendered := image.Render(imgPath, m.width-6)
					displayLines = append(displayLines, strings.Split(rendered, "\n")...)
					continue
				}
			}
			displayLines = append(displayLines, lipgloss.NewStyle().
				Foreground(theme.Current.Error).
				Render(fmt.Sprintf("[Image not found: %s]", filepath.Base(image.ExtractImagePath(line)))))
		} else {
			displayLines = append(displayLines, line)
		}
	}

	// Pre-compute which raw lines are inside fenced code blocks.
	codeLines := codeBlockLines(rawLines)

	maxLines := m.viewportHeight()
	totalLines := len(displayLines)

	// Exact cursor position via the map.
	displayCursor := 0
	if m.lineCursor >= 0 && m.lineCursor < len(rawToDisplay) {
		displayCursor = rawToDisplay[m.lineCursor]
	}

	start := displayCursor - maxLines/2
	if start < 0 {
		start = 0
	}
	if start > totalLines-maxLines {
		start = max(0, totalLines-maxLines)
	}
	end := min(start+maxLines, totalLines)

	// Reverse map: display line → owning raw line.
	displayToRaw := func(d int) int {
		raw := 0
		for r, disp := range rawToDisplay {
			if disp <= d {
				raw = r
			}
		}
		return raw
	}

	// Render visible lines
	for i := start; i < end; i++ {
		line := ""
		if i < len(displayLines) {
			line = displayLines[i]
		}

		rawIdx := displayToRaw(i)
		inCode := codeLines[rawIdx]
		inVisual := m.isLineInVisual(rawIdx)
		style := mdLineStyle(line, inCode)

		if i == displayCursor {
			prefix := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Render("> ")
			styledLine := m.renderLineWithCursor(line, style)

			if inVisual {
				line = prefix + lipgloss.NewStyle().
					Background(theme.Current.SelectedBg).
					Render(styledLine)
			} else {
				line = prefix + styledLine
			}
		} else {
			styledLine := style.Render(line)
			if inVisual {
				line = "  " + lipgloss.NewStyle().
					Background(theme.Current.SelectedBg).
					Render(styledLine)
			} else {
				line = "  " + styledLine
			}
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	// --- Footer: separator + status bar + help ---
	s.WriteString(lipgloss.NewStyle().
		Foreground(theme.Current.Separator).
		Render(strings.Repeat("─", m.width)))
	s.WriteString("\n")

	mode := m.modeIndicator()
	status := fmt.Sprintf("Ln %d  Col %d%s", m.lineCursor+1, m.colCursor+1, mode)
	s.WriteString(StatusBarStyle.Width(m.width).Render(status))
	s.WriteString("\n")

	helpKeys := []struct{ key, desc string }{
		{"j/k", "move"},
		{"h/l", "cols"},
		{"w/b", "word"},
		{"g/G", "top/end"},
		{"v/V", "visual"},
		{"y", "yank"},
		{"e", "edit"},
		{"t", "tree"},
		{"/", "search"},
		{"esc", "back"},
	}
	var parts []string
	for _, k := range helpKeys {
		key := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true).Render(k.key)
		desc := lipgloss.NewStyle().Foreground(mutedColor).Render(" " + k.desc)
		parts = append(parts, key+desc)
	}
	s.WriteString(lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(parts, "  ")))

	return s.String()
}

// codeBlockLines returns which raw line indices are inside fenced code blocks.
// The fence lines themselves (```) are included.
func codeBlockLines(lines []string) map[int]bool {
	result := map[int]bool{}
	inside := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			result[i] = true
			inside = !inside
			continue
		}
		if inside {
			result[i] = true
		}
	}
	return result
}

// mdLineStyle returns the theme-aware style for a markdown line.
func mdLineStyle(line string, inCodeBlock bool) lipgloss.Style {
	if inCodeBlock {
		return lipgloss.NewStyle().Foreground(theme.Current.TextDim)
	}
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "# "):
		return lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	case strings.HasPrefix(trimmed, "## "):
		return lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	case strings.HasPrefix(trimmed, "### "),
		strings.HasPrefix(trimmed, "#### "),
		strings.HasPrefix(trimmed, "##### "),
		strings.HasPrefix(trimmed, "###### "):
		return lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	case strings.HasPrefix(trimmed, "> "):
		return lipgloss.NewStyle().Italic(true).Foreground(mutedColor)
	case trimmed == "---" || trimmed == "***" || trimmed == "___":
		return lipgloss.NewStyle().Foreground(theme.Current.Separator)
	default:
		return lipgloss.NewStyle().Foreground(theme.Current.Text)
	}
}

// renderLineWithCursor renders a line with a visible vim-style block cursor
// at colCursor. The before/after parts keep the line's markdown style; the
// cursor character gets an inverted highlight.
func (m Model) renderLineWithCursor(line string, style lipgloss.Style) string {
	runes := []rune(line)
	col := m.colCursor
	if col < 0 {
		col = 0
	}

	cursorStyle := lipgloss.NewStyle().
		Background(primaryColor).
		Foreground(theme.Current.Background)

	// Empty line: show a single-space block cursor.
	if len(runes) == 0 {
		return cursorStyle.Render(" ")
	}
	if col >= len(runes) {
		col = len(runes) - 1
	}

	var result strings.Builder
	if col > 0 {
		result.WriteString(style.Render(string(runes[:col])))
	}
	result.WriteString(cursorStyle.Render(string(runes[col : col+1])))
	if col+1 < len(runes) {
		result.WriteString(style.Render(string(runes[col+1:])))
	}
	return result.String()
}

// modeIndicator returns a string showing the current mode.
func (m Model) modeIndicator() string {
	parts := []string{}
	switch m.visualMode {
	case VisualChar:
		parts = append(parts, " [VISUAL]")
	case VisualLine:
		parts = append(parts, " [V-LINE]")
	}
	if m.splitMode != "" {
		parts = append(parts, " [SPLIT]")
	}
	return strings.Join(parts, "")
}
