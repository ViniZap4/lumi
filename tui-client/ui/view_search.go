package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

func (m Model) renderWithSearchModal(base string) string {
	modalWidth := min(m.width-10, 100)

	var modal strings.Builder

	// Title
	modal.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render("  Find Notes"))
	modal.WriteString("\n\n")

	// Search input
	typeLabel := "Filename"
	if m.searchType == "content" {
		typeLabel = "Content"
	}

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1).
		Width(modalWidth - 8).
		Render(fmt.Sprintf("[%s] %s", typeLabel, m.searchQuery+"_"))

	modal.WriteString(inputBox)
	modal.WriteString("\n\n")

	if len(m.searchResults) == 0 {
		modal.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("  No results"))
		modal.WriteString("\n")
	} else {
		modal.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(fmt.Sprintf("  Results (%d)", len(m.searchResults))))
		modal.WriteString("\n\n")

		// Split: results left, preview right
		leftWidth := (modalWidth - 8) / 2
		rightWidth := modalWidth - 8 - leftWidth - 3

		var resultsList strings.Builder
		maxResults := 12
		for i, item := range m.searchResults {
			if i >= maxResults {
				break
			}

			name := item.Name
			if len(name) > leftWidth-6 {
				name = name[:leftWidth-9] + "..."
			}

			if i == m.cursor {
				line := lipgloss.NewStyle().
					Foreground(accentColor).
					Background(selectedBg).
					Width(leftWidth).
					Render(fmt.Sprintf(" > %s", name))
				resultsList.WriteString(line)
			} else {
				resultsList.WriteString(lipgloss.NewStyle().
					Width(leftWidth).
					Render(fmt.Sprintf("   %s", name)))
			}
			resultsList.WriteString("\n")
		}

		// Preview
		var previewBox strings.Builder
		if m.cursor >= 0 && m.cursor < len(m.searchResults) && m.searchResults[m.cursor].Note != nil {
			note := m.searchResults[m.cursor].Note

			previewBox.WriteString(lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Render(note.Title))
			previewBox.WriteString("\n")
			previewBox.WriteString(strings.Repeat("-", rightWidth))
			previewBox.WriteString("\n")

			previewLines := strings.Split(note.Content, "\n")
			maxPreview := 10
			for i := 0; i < min(len(previewLines), maxPreview); i++ {
				line := previewLines[i]
				if len(line) > rightWidth {
					line = line[:rightWidth-3] + "..."
				}
				previewBox.WriteString(lipgloss.NewStyle().
					Foreground(mutedColor).
					Render(line))
				previewBox.WriteString("\n")
			}
		} else {
			previewBox.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				Render("No preview available"))
		}

		// Combine side by side
		resultsLines := strings.Split(resultsList.String(), "\n")
		previewLines := strings.Split(previewBox.String(), "\n")
		maxLines := max(len(resultsLines), len(previewLines))

		for i := 0; i < maxLines; i++ {
			if i < len(resultsLines) {
				modal.WriteString(resultsLines[i])
			} else {
				modal.WriteString(strings.Repeat(" ", leftWidth))
			}
			modal.WriteString(" | ")
			if i < len(previewLines) {
				modal.WriteString(previewLines[i])
			}
			modal.WriteString("\n")
		}
	}

	modal.WriteString("\n")
	modal.WriteString(HelpStyle.Render("ctrl+f=toggle | ↑↓=navigate | enter=open | esc=close"))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(modalWidth).
		Render(modal.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
	)
}

func (m Model) renderWithInFileSearch() string {
	matches := m.findInFileMatches()

	var s strings.Builder

	// Title
	s.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(m.fullNote.Title))
	s.WriteString("\n\n")

	// Search bar
	s.WriteString(lipgloss.NewStyle().
		Foreground(accentColor).
		Render(fmt.Sprintf("  Search: %s_ (%d matches)", m.searchQuery, len(matches))))
	s.WriteString("\n\n")

	// Show matches with context
	maxLines := m.height - 10
	for i, lineNum := range matches {
		if i >= maxLines {
			break
		}

		line := m.contentLines[lineNum]
		if len(line) > m.width-10 {
			line = line[:m.width-10] + "..."
		}

		// Highlight the match
		if m.searchQuery != "" {
			lowerLine := strings.ToLower(line)
			query := strings.ToLower(m.searchQuery)
			idx := strings.Index(lowerLine, query)
			if idx >= 0 {
				before := line[:idx]
				match := lipgloss.NewStyle().
					Background(accentColor).
					Foreground(theme.Current.OverlayBg).
					Render(line[idx : idx+len(m.searchQuery)])
				after := line[idx+len(m.searchQuery):]
				line = before + match + after
			}
		}

		lineStr := fmt.Sprintf("%4d: %s", lineNum+1, line)
		if i == m.cursor && m.cursor < len(matches) {
			lineStr = lipgloss.NewStyle().
				Foreground(accentColor).
				Render("> " + lineStr)
		} else {
			lineStr = "  " + lineStr
		}

		s.WriteString(lineStr)
		s.WriteString("\n")
	}

	if len(matches) == 0 {
		s.WriteString(DimItemStyle.Render("  No matches found"))
	}

	s.WriteString("\n\n")
	s.WriteString(HelpStyle.Render("↑↓=navigate | enter=jump to line | esc=close"))

	return s.String()
}
