package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/filesystem"
	"github.com/vinizap/lumi/tui-client/theme"
)

func (m Model) renderTree() string {
	leftWidth := m.width / 4
	centerWidth := m.width / 3
	rightWidth := m.width - leftWidth - centerWidth - 6

	var s strings.Builder

	// Header bar
	pathDisplay := m.displayPath()
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(theme.Current.Background).
		Padding(0, 1).
		Render("lumi")
	pathLabel := lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(theme.Current.Background).
		Render("  " + pathDisplay)
	s.WriteString(header + pathLabel)
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Separator).Render(strings.Repeat("-", m.width)))
	s.WriteString("\n")

	// Three columns
	colHeight := m.height - 4
	leftCol := m.renderParentCol(leftWidth, colHeight)
	centerCol := m.renderCenterCol(centerWidth, colHeight)
	rightCol := m.renderPreviewCol(rightWidth, colHeight)

	sep := lipgloss.NewStyle().Foreground(theme.Current.Separator).Render(" | ")

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
			if f.Path == m.currentDir {
				s.WriteString(lipgloss.NewStyle().
					Foreground(accentColor).
					Bold(true).
					Render(" > " + f.Name + "/"))
			} else {
				s.WriteString(lipgloss.NewStyle().
					Foreground(secondaryColor).
					Render("   " + f.Name + "/"))
			}
			s.WriteString("\n")
			count++
		}

		for _, n := range parentNotes {
			if count >= maxItems {
				break
			}
			s.WriteString(lipgloss.NewStyle().
				Foreground(theme.Current.TextDim).
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
	start := 0
	if m.cursor >= maxItems {
		start = m.cursor - maxItems + 1
	}

	for i := start; i < len(m.items) && i < start+maxItems; i++ {
		item := m.items[i]

		if i == m.cursor {
			name := item.Name
			if item.IsFolder {
				name += "/"
			}
			line := lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Width(width - 1).
				Render(" > " + name)
			s.WriteString(line)
		} else if item.IsFolder {
			name := item.Name + "/"
			s.WriteString(lipgloss.NewStyle().
				Foreground(secondaryColor).
				Width(width - 1).
				Render("   " + name))
		} else {
			s.WriteString(lipgloss.NewStyle().
				Foreground(theme.Current.Text).
				Width(width - 1).
				Render("   " + item.Name))
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
		// Folder header
		s.WriteString(lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Render(" " + item.Name + "/"))
		s.WriteString("\n\n")

		subFolders, _ := filesystem.ListFolders(item.Path)
		folderNotes, _ := filesystem.ListNotes(item.Path)
		total := len(subFolders) + len(folderNotes)

		if total > 0 {
			s.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render(fmt.Sprintf(" %d item", total)))
			if total != 1 {
				s.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("s"))
			}
			s.WriteString("\n\n")

			maxPreview := height - 5
			count := 0
			for _, f := range subFolders {
				if count >= maxPreview {
					break
				}
				s.WriteString(lipgloss.NewStyle().
					Foreground(secondaryColor).
					Render("   " + f.Name + "/"))
				s.WriteString("\n")
				count++
			}
			for _, n := range folderNotes {
				if count >= maxPreview {
					break
				}
				s.WriteString(lipgloss.NewStyle().
					Foreground(theme.Current.TextDim).
					Render("   " + n.Title))
				s.WriteString("\n")
				count++
			}
			if len(subFolders)+len(folderNotes) > maxPreview {
				s.WriteString(lipgloss.NewStyle().
					Foreground(mutedColor).
					Italic(true).
					Render(fmt.Sprintf("   … %d more", total-maxPreview)))
				s.WriteString("\n")
			}
		} else {
			s.WriteString(DimItemStyle.Render(" (empty)"))
		}

	} else if item.Note != nil {
		// Note preview
		s.WriteString(lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render(" " + item.Note.Title))
		s.WriteString("\n")

		meta := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(fmt.Sprintf(" %s", item.Note.UpdatedAt.Format("Jan 2, 2006")))
		s.WriteString(meta)
		s.WriteString("\n")

		if width > 3 {
			s.WriteString(lipgloss.NewStyle().
				Foreground(theme.Current.Separator).
				Render(" " + strings.Repeat("─", width-3)))
		}
		s.WriteString("\n")

		// Tags
		if len(item.Note.Tags) > 0 {
			var tagParts []string
			for _, tag := range item.Note.Tags {
				tagParts = append(tagParts, lipgloss.NewStyle().
					Foreground(theme.Current.Accent).
					Render("#"+tag))
			}
			s.WriteString(" " + strings.Join(tagParts, " "))
			s.WriteString("\n\n")
		} else {
			s.WriteString("\n")
		}

		// Content preview
		lines := strings.Split(item.Note.Content, "\n")
		maxPrev := height - 7
		if maxPrev < 1 {
			maxPrev = 1
		}
		shown := 0
		for _, line := range lines {
			if shown >= maxPrev {
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if len([]rune(line)) > width-3 {
				line = string([]rune(line)[:width-6]) + "..."
			}
			s.WriteString(lipgloss.NewStyle().
				Foreground(theme.Current.TextDim).
				Render(" " + line))
			s.WriteString("\n")
			shown++
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}
