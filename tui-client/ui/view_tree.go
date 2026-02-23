package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

func (m Model) renderTreeYazi() string {
	leftWidth := m.width / 4
	centerWidth := m.width / 3
	rightWidth := m.width - leftWidth - centerWidth - 4

	var s strings.Builder

	// Title bar with path
	pathDisplay := m.displayPath()
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render("  " + pathDisplay)
	s.WriteString(title)
	s.WriteString("\n\n")

	// Three columns
	leftCol := m.renderParentCol(leftWidth, m.height-4)
	centerCol := m.renderCenterCol(centerWidth, m.height-4)
	rightCol := m.renderPreviewCol(rightWidth, m.height-4)

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftCol,
		lipgloss.NewStyle().Foreground(mutedColor).Render("|"),
		centerCol,
		lipgloss.NewStyle().Foreground(mutedColor).Render("|"),
		rightCol,
	)
	s.WriteString(columns)

	// Help bar
	s.WriteString("\n")
	s.WriteString(HelpStyle.Render("hjkl=move | enter=open | n=new | r=rename | d=delete | D=duplicate | /=search | esc=back | q=quit"))

	return s.String()
}

func (m Model) displayPath() string {
	pathDisplay := strings.TrimPrefix(m.currentDir, m.rootDir)
	if pathDisplay == "" {
		return "~"
	}
	return "~" + pathDisplay
}

func (m Model) renderParentCol(width, height int) string {
	var s strings.Builder

	if m.currentDir != m.rootDir {
		parentDir := filepath.Dir(m.currentDir)
		parentItems, _ := filesystem.ListFolders(parentDir)
		parentNotes, _ := filesystem.ListNotes(parentDir)

		maxItems := height - 2
		count := 0

		for _, f := range parentItems {
			if count >= maxItems {
				break
			}
			name := f.Name
			if f.Path == m.currentDir {
				name = lipgloss.NewStyle().
					Foreground(accentColor).
					Render("> " + name)
			}
			s.WriteString("  " + name)
			s.WriteString("\n")
			count++
		}

		for _, n := range parentNotes {
			if count >= maxItems {
				break
			}
			s.WriteString("  " + n.Title)
			s.WriteString("\n")
			count++
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}

func (m Model) renderCenterCol(width, height int) string {
	var s strings.Builder

	for i, item := range m.items {
		if i >= height {
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
			s.WriteString(line)
		} else {
			s.WriteString("  " + name)
		}
		s.WriteString("\n")
	}

	if len(m.items) == 0 {
		s.WriteString(DimItemStyle.Render("  No items"))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}

func (m Model) renderPreviewCol(width, height int) string {
	var s strings.Builder

	if m.cursor >= len(m.items) {
		return lipgloss.NewStyle().Width(width).Height(height).Render("")
	}

	item := m.items[m.cursor]

	if item.IsFolder {
		s.WriteString(lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Render("  " + item.Name))
		s.WriteString("\n\n")

		folderNotes, _ := filesystem.ListNotes(item.Path)
		if len(folderNotes) > 0 {
			s.WriteString(DimItemStyle.Render(fmt.Sprintf("%d notes:", len(folderNotes))))
			s.WriteString("\n")
			for i, note := range folderNotes {
				if i >= height-4 {
					break
				}
				s.WriteString(fmt.Sprintf("  %s\n", note.Title))
			}
		} else {
			s.WriteString(DimItemStyle.Render("(empty folder)"))
		}
	} else if item.Note != nil {
		s.WriteString(lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Render("  " + item.Note.Title))
		s.WriteString("\n\n")

		meta := fmt.Sprintf("%s  %s",
			item.Note.ID,
			item.Note.UpdatedAt.Format("Jan 2"))
		s.WriteString(PreviewMetaStyle.Render(meta))
		s.WriteString("\n\n")

		lines := strings.Split(item.Note.Content, "\n")
		previewLines := min(height-6, len(lines))
		for i := 0; i < previewLines; i++ {
			line := lines[i]
			if len(line) > width-2 {
				line = line[:width-2] + "..."
			}
			s.WriteString(line)
			s.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}
