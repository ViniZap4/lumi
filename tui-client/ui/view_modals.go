package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/domain"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

// renderWithNavModal renders the navigation overlay on top of the note view.
func (m Model) renderWithNavModal(base string) string {
	modalWidth := min(m.width-6, 90)
	modalHeight := min(m.height-4, 30)

	var modal strings.Builder

	// Path header
	navPath := strings.TrimPrefix(m.navDir, m.rootDir)
	if navPath == "" {
		navPath = "~"
	} else {
		navPath = "~" + navPath
	}

	parentInfo := ""
	if m.navDir != m.rootDir {
		parentName := filepath.Base(filepath.Dir(m.navDir))
		if filepath.Dir(m.navDir) == m.rootDir {
			parentName = "~"
		}
		parentInfo = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("  <- " + parentName)
	}

	modal.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(" " + navPath + parentInfo))
	modal.WriteString("\n")
	modal.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("236")).
		Render(strings.Repeat("-", modalWidth-6)))
	modal.WriteString("\n")

	// Items list with scrolling
	listHeight := modalHeight - 8
	if listHeight < 4 {
		listHeight = 4
	}

	items := m.navItems
	start := 0
	if m.navCursor >= listHeight {
		start = m.navCursor - listHeight + 1
	}

	count := 0
	for i := start; i < len(items) && count < listHeight; i++ {
		item := items[i]
		name := item.Name
		if item.IsFolder {
			name += "/"
		}

		if i == m.navCursor {
			line := lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Width(modalWidth - 8).
				Render(" > " + name)
			modal.WriteString(line)
		} else {
			modal.WriteString(lipgloss.NewStyle().
				Width(modalWidth - 8).
				Render("   " + name))
		}
		modal.WriteString("\n")
		count++
	}

	if len(items) == 0 {
		modal.WriteString(DimItemStyle.Render("   (empty)"))
		modal.WriteString("\n")
	}

	// Preview for selected note
	if m.navCursor >= 0 && m.navCursor < len(items) {
		item := items[m.navCursor]
		if item.Note != nil {
			modal.WriteString("\n")
			modal.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("236")).
				Render(strings.Repeat("-", modalWidth-6)))
			modal.WriteString("\n")

			modal.WriteString(lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				Render(" " + item.Note.Title))
			modal.WriteString("\n")

			previewLines := strings.Split(item.Note.Content, "\n")
			maxPreview := 4
			for i := 0; i < min(len(previewLines), maxPreview); i++ {
				line := previewLines[i]
				if len(line) > modalWidth-8 {
					line = line[:modalWidth-11] + "..."
				}
				modal.WriteString(lipgloss.NewStyle().
					Foreground(mutedColor).
					Render(" " + line))
				modal.WriteString("\n")
			}
		} else if item.IsFolder {
			modal.WriteString("\n")
			notes, _ := filesystem.ListNotes(item.Path)
			folders, _ := filesystem.ListFolders(item.Path)
			modal.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render(lipgloss.NewStyle().
					Foreground(lipgloss.Color("236")).
					Render(strings.Repeat("-", modalWidth-6)) + "\n"))
			modal.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render(renderItemCount(len(folders), len(notes))))
			modal.WriteString("\n")
		}
	}

	// Help
	modal.WriteString("\n")
	helpParts := []struct{ key, desc string }{
		{"hjkl", "navigate"},
		{"enter", "open"},
		{"s/S", "split"},
		{"esc", "close"},
	}
	var parts []string
	for _, h := range helpParts {
		key := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true).Render(h.key)
		desc := lipgloss.NewStyle().Foreground(mutedColor).Render(" " + h.desc)
		parts = append(parts, key+desc)
	}
	modal.WriteString(strings.Join(parts, "  "))

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(modalWidth).
		Render(modal.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

func renderItemCount(folders, notes int) string {
	var parts []string
	if folders > 0 {
		s := "folder"
		if folders > 1 {
			s = "folders"
		}
		parts = append(parts, fmt.Sprintf(" %d %s", folders, s))
	}
	if notes > 0 {
		s := "note"
		if notes > 1 {
			s = "notes"
		}
		parts = append(parts, fmt.Sprintf(" %d %s", notes, s))
	}
	if len(parts) == 0 {
		return " (empty)"
	}
	return strings.Join(parts, ",")
}

func (m Model) renderWithInputModal(base string) string {
	title := "Create Note"
	if m.inputMode == "rename" {
		title = "Rename Note"
	}

	modalWidth := 60

	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")).Render(title))
	s.WriteString("\n\n")
	s.WriteString("Title: ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Render(m.inputValue + "_"))
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Enter to confirm  Esc to cancel"))

	modal := lipgloss.NewStyle().
		Width(modalWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(s.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

func (m Model) renderSplitView() string {
	var s strings.Builder

	if m.splitMode == "horizontal" {
		topHeight := m.height / 2
		bottomHeight := m.height - topHeight - 1

		s.WriteString(m.renderNoteInBox(m.fullNote, m.width, topHeight))
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(strings.Repeat("-", m.width)))
		s.WriteString("\n")
		s.WriteString(m.renderNoteInBox(m.splitNote, m.width, bottomHeight))
	} else {
		leftWidth := m.width / 2
		rightWidth := m.width - leftWidth - 1

		left := m.renderNoteInBox(m.fullNote, leftWidth, m.height)
		right := m.renderNoteInBox(m.splitNote, rightWidth, m.height)

		s.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render("|"),
			right,
		))
	}

	return s.String()
}

func (m Model) renderNoteInBox(note *domain.Note, width, height int) string {
	if note == nil {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Render("No note")
	}

	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(note.Title))
	s.WriteString("\n\n")

	boxWidth := width - 4
	if boxWidth < 20 {
		boxWidth = 20
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(boxWidth),
	)

	var lines []string
	if err == nil {
		rendered, err := renderer.Render(note.Content)
		if err == nil {
			lines = strings.Split(rendered, "\n")
		}
	}
	if lines == nil {
		lines = strings.Split(note.Content, "\n")
	}

	maxLines := height - 4
	for i := 0; i < min(len(lines), maxLines); i++ {
		line := lines[i]
		if len(line) > width-2 {
			line = line[:width-2] + "..."
		}
		s.WriteString(line)
		s.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}
