package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/domain"
)

func (m Model) renderWithTreeModal(base string) string {
	modalWidth := min(m.width-10, 90)

	var modal strings.Builder

	// Path header
	pathDisplay := m.displayPath()
	parentInfo := ""
	if m.currentDir != m.rootDir {
		parentDir := filepath.Dir(m.currentDir)
		parentName := filepath.Base(parentDir)
		if parentName == filepath.Base(m.rootDir) {
			parentName = "~"
		}
		parentInfo = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(" <- " + parentName)
	}

	modal.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render("  " + pathDisplay + parentInfo))
	modal.WriteString("\n\n")

	// Items
	maxItems := 12
	for i, item := range m.items {
		if i >= maxItems {
			break
		}

		name := item.Name
		if item.IsFolder {
			name += "/"
		}

		if i == m.cursor {
			line := lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Render("> " + name)
			modal.WriteString(line)
		} else {
			modal.WriteString("  " + name)
		}
		modal.WriteString("\n")
	}

	if len(m.items) == 0 {
		modal.WriteString(DimItemStyle.Render("  (empty)"))
		modal.WriteString("\n")
	}

	// Preview for selected note
	if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].Note != nil {
		modal.WriteString("\n")
		modal.WriteString(strings.Repeat("-", modalWidth-4))
		modal.WriteString("\n")
		modal.WriteString(lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render("Preview"))
		modal.WriteString("\n\n")

		note := m.items[m.cursor].Note
		previewLines := strings.Split(note.Content, "\n")
		maxPreview := 5
		for i := 0; i < min(len(previewLines), maxPreview); i++ {
			line := previewLines[i]
			if len(line) > modalWidth-6 {
				line = line[:modalWidth-6] + "..."
			}
			modal.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render("  " + line))
			modal.WriteString("\n")
		}
	}

	modal.WriteString("\n")
	modal.WriteString(HelpStyle.Render("hjkl=navigate | enter=open | s=split-h | S=split-v | esc=close"))

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
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
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
		s.WriteString(strings.Repeat("-", m.width))
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
			lipgloss.NewStyle().Foreground(mutedColor).Render("|"),
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

	// Use glamour for split note rendering too
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
