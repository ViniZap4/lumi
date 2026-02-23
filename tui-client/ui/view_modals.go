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

// renderWithNavModal renders a 3-column navigation modal (parent | current | preview)
// overlaid on top of the note view — the same layout as the main tree browser.
func (m Model) renderWithNavModal(base string) string {
	modalWidth := min(m.width-4, 110)
	modalInner := modalWidth - 6 // padding + border

	// Path header
	navPath := strings.TrimPrefix(m.navDir, m.rootDir)
	if navPath == "" {
		navPath = "~"
	} else {
		navPath = "~" + navPath
	}

	var header strings.Builder
	header.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(" " + navPath))
	if m.navDir != m.rootDir {
		parentName := filepath.Base(filepath.Dir(m.navDir))
		if filepath.Dir(m.navDir) == m.rootDir {
			parentName = "~"
		}
		header.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("  <- " + parentName))
	}

	// Column dimensions
	leftW := modalInner / 4
	centerW := modalInner / 3
	rightW := modalInner - leftW - centerW - 6 // separators
	colHeight := min(m.height-10, 20)
	if colHeight < 6 {
		colHeight = 6
	}

	// Build the three columns
	leftCol := m.navParentCol(leftW, colHeight)
	centerCol := m.navCenterCol(centerW, colHeight)
	rightCol := m.navPreviewCol(rightW, colHeight)

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(" | ")

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftCol,
		sep,
		centerCol,
		sep,
		rightCol,
	)

	// Help bar
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
	helpLine := strings.Join(parts, "  ")

	// Assemble modal content
	var modal strings.Builder
	modal.WriteString(header.String())
	modal.WriteString("\n")
	modal.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("236")).
		Render(strings.Repeat("-", modalInner)))
	modal.WriteString("\n")
	modal.WriteString(columns)
	modal.WriteString("\n")
	modal.WriteString(helpLine)

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

// navParentCol renders the left column showing items in the parent directory.
func (m Model) navParentCol(width, height int) string {
	var s strings.Builder

	if m.navDir != m.rootDir {
		parentDir := filepath.Dir(m.navDir)
		parentFolders, _ := filesystem.ListFolders(parentDir)
		parentNotes, _ := filesystem.ListNotes(parentDir)

		maxItems := height
		count := 0

		for _, f := range parentFolders {
			if count >= maxItems {
				break
			}
			name := f.Name
			if f.Path == m.navDir {
				s.WriteString(lipgloss.NewStyle().
					Foreground(accentColor).
					Bold(true).
					Render(" > " + name))
			} else {
				s.WriteString(lipgloss.NewStyle().
					Foreground(mutedColor).
					Render("   " + name))
			}
			s.WriteString("\n")
			count++
		}

		for _, n := range parentNotes {
			if count >= maxItems {
				break
			}
			s.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render("   " + n.Title))
			s.WriteString("\n")
			count++
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}

// navCenterCol renders the center column with the current navItems and navCursor.
func (m Model) navCenterCol(width, height int) string {
	var s strings.Builder

	maxItems := height
	start := 0
	if m.navCursor >= maxItems {
		start = m.navCursor - maxItems + 1
	}

	for i := start; i < len(m.navItems) && i < start+maxItems; i++ {
		item := m.navItems[i]
		name := item.Name
		if item.IsFolder {
			name += "/"
		}

		if i == m.navCursor {
			line := lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Width(width - 1).
				Render(" > " + name)
			s.WriteString(line)
		} else {
			s.WriteString(lipgloss.NewStyle().
				Width(width - 1).
				Render("   " + name))
		}
		s.WriteString("\n")
	}

	if len(m.navItems) == 0 {
		s.WriteString(DimItemStyle.Render("   (empty)"))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}

// navPreviewCol renders the right column showing a preview of the selected item.
func (m Model) navPreviewCol(width, height int) string {
	var s strings.Builder

	if m.navCursor >= len(m.navItems) {
		return lipgloss.NewStyle().Width(width).Height(height).Render("")
	}

	item := m.navItems[m.navCursor]

	if item.IsFolder {
		s.WriteString(lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Render(" " + item.Name + "/"))
		s.WriteString("\n\n")

		subFolders, _ := filesystem.ListFolders(item.Path)
		notes, _ := filesystem.ListNotes(item.Path)

		if len(subFolders) > 0 || len(notes) > 0 {
			s.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render(fmt.Sprintf(" %d items", len(subFolders)+len(notes))))
			s.WriteString("\n\n")

			count := 0
			for _, f := range subFolders {
				if count >= height-5 {
					break
				}
				s.WriteString(lipgloss.NewStyle().
					Foreground(mutedColor).
					Render("   " + f.Name + "/"))
				s.WriteString("\n")
				count++
			}
			for _, n := range notes {
				if count >= height-5 {
					break
				}
				s.WriteString("   " + n.Title)
				s.WriteString("\n")
				count++
			}
		} else {
			s.WriteString(DimItemStyle.Render(" (empty folder)"))
		}
	} else if item.Note != nil {
		s.WriteString(lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Render(" " + item.Note.Title))
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(fmt.Sprintf(" %s", item.Note.UpdatedAt.Format("Jan 2, 2006"))))
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("236")).
			Render(" " + strings.Repeat("-", width-3)))
		s.WriteString("\n")

		lines := strings.Split(item.Note.Content, "\n")
		previewLines := min(height-4, len(lines))
		for i := 0; i < previewLines; i++ {
			line := lines[i]
			if len(line) > width-3 {
				line = line[:width-6] + "..."
			}
			s.WriteString(" " + line)
			s.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
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
