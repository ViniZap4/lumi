package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

func (m Model) renderTree() string {
	leftWidth := m.width / 4
	centerWidth := m.width / 3
	rightWidth := m.width - leftWidth - centerWidth - 4

	var s strings.Builder

	// Header bar
	pathDisplay := m.displayPath()
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Padding(0, 1).
		Render("lumi")
	pathLabel := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("  " + pathDisplay)
	s.WriteString(header + pathLabel)
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(strings.Repeat("-", m.width)))
	s.WriteString("\n")

	// Three columns
	colHeight := m.height - 4
	leftCol := m.renderParentCol(leftWidth, colHeight)
	centerCol := m.renderCenterCol(centerWidth, colHeight)
	rightCol := m.renderPreviewCol(rightWidth, colHeight)

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(" | ")

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftCol,
		sep,
		centerCol,
		sep,
		rightCol,
	)
	s.WriteString(columns)

	// Bottom bar
	s.WriteString("\n")
	s.WriteString(m.renderTreeHelp())

	return s.String()
}

func (m Model) displayPath() string {
	pathDisplay := strings.TrimPrefix(m.currentDir, m.rootDir)
	if pathDisplay == "" {
		return "~"
	}
	return "~" + pathDisplay
}

func (m Model) renderTreeHelp() string {
	keys := []struct{ key, desc string }{
		{"j/k", "move"},
		{"l/enter", "open"},
		{"h", "back"},
		{"n", "new"},
		{"r", "rename"},
		{"d", "delete"},
		{"/", "search"},
		{"c", "config"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		key := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true).Render(k.key)
		desc := lipgloss.NewStyle().Foreground(mutedColor).Render(" " + k.desc)
		parts = append(parts, key+desc)
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(parts, "  "))
}

func (m Model) renderParentCol(width, height int) string {
	var s strings.Builder

	if m.currentDir != m.rootDir {
		parentDir := filepath.Dir(m.currentDir)
		parentItems, _ := filesystem.ListFolders(parentDir)
		parentNotes, _ := filesystem.ListNotes(parentDir)

		maxItems := height - 1
		count := 0

		for _, f := range parentItems {
			if count >= maxItems {
				break
			}
			name := f.Name
			if f.Path == m.currentDir {
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

func (m Model) renderCenterCol(width, height int) string {
	var s strings.Builder

	maxItems := height - 1
	// Scroll if needed
	start := 0
	if m.cursor >= maxItems {
		start = m.cursor - maxItems + 1
	}

	for i := start; i < len(m.items) && i < start+maxItems; i++ {
		item := m.items[i]
		name := item.Name
		if item.IsFolder {
			name += "/"
		}

		if i == m.cursor {
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

	if len(m.items) == 0 {
		s.WriteString(DimItemStyle.Render("   (empty)"))
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
			Render(" " + item.Name + "/"))
		s.WriteString("\n\n")

		folderNotes, _ := filesystem.ListNotes(item.Path)
		subFolders, _ := filesystem.ListFolders(item.Path)

		if len(subFolders) > 0 || len(folderNotes) > 0 {
			s.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render(fmt.Sprintf(" %d items", len(subFolders)+len(folderNotes))))
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
			for _, n := range folderNotes {
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
		// Title
		s.WriteString(lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Render(" " + item.Note.Title))
		s.WriteString("\n")

		// Metadata
		meta := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(fmt.Sprintf(" %s", item.Note.UpdatedAt.Format("Jan 2, 2006")))
		s.WriteString(meta)
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(" " + strings.Repeat("-", width-3)))
		s.WriteString("\n")

		// Content preview
		lines := strings.Split(item.Note.Content, "\n")
		previewLines := min(height-5, len(lines))
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
