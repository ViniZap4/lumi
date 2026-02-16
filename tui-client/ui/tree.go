// tui-client/ui/tree.go
package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderTreeModal() string {
	// Full screen tree view (not a modal)
	modalWidth := m.width
	modalHeight := m.height

	// Three columns: parent, current, preview
	leftWidth := modalWidth / 4
	centerWidth := modalWidth / 3
	rightWidth := modalWidth - leftWidth - centerWidth - 4

	var content strings.Builder

	// Title bar with path and search
	titleBar := m.renderTreeTitle(modalWidth)
	content.WriteString(titleBar)
	content.WriteString("\n")

	// Three columns
	leftCol := m.renderParentColumn(leftWidth, modalHeight-4)
	centerCol := m.renderCurrentColumn(centerWidth, modalHeight-4)
	rightCol := m.renderPreviewColumn(rightWidth, modalHeight-4)

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftCol,
		lipgloss.NewStyle().Foreground(mutedColor).Render("│"),
		centerCol,
		lipgloss.NewStyle().Foreground(mutedColor).Render("│"),
		rightCol,
	)
	content.WriteString(columns)

	// Help bar
	content.WriteString("\n")
	helpKeys := []string{
		HelpKeyStyle.Render("hjkl") + "=navigate",
		HelpKeyStyle.Render("enter") + "=open",
		HelpKeyStyle.Render("/") + "=search",
		HelpKeyStyle.Render("q") + "=quit",
	}
	helpText := HelpStyle.Render(strings.Join(helpKeys, " | "))
	content.WriteString(helpText)

	return content.String()
}

func (m Model) renderTreeTitle(width int) string {
	// Path
	pathParts := strings.Split(strings.TrimPrefix(m.currentDir, m.rootDir), string(filepath.Separator))
	pathDisplay := "~"
	if len(pathParts) > 0 && pathParts[0] != "" {
		pathDisplay = pathDisplay + "/" + strings.Join(pathParts, "/")
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)
	title := titleStyle.Render("📂 " + pathDisplay)

	// Search box
	var searchBox string
	if m.treeSearch != "" {
		searchBox = lipgloss.NewStyle().
			Foreground(accentColor).
			Render(" 🔍 " + m.treeSearch + "█")
	}

	return title + searchBox
}

func (m Model) renderParentColumn(width, height int) string {
	var content strings.Builder

	// Show parent directory items
	if m.currentDir != m.rootDir {
		content.WriteString(DimItemStyle.Render(".."))
		content.WriteString("\n")

		// Show siblings (simplified)
		for i, item := range m.treeItems[:min(height-2, len(m.treeItems))] {
			if i >= height-2 {
				break
			}
			icon := "📄"
			if item.isFolder {
				icon = "📁"
			}
			line := icon + " " + item.name
			if item.isFolder {
				line += "/"
			}
			content.WriteString(DimItemStyle.Render(line))
			content.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content.String())
}

func (m Model) renderCurrentColumn(width, height int) string {
	var content strings.Builder

	for i, item := range m.treeItems {
		if i >= height {
			break
		}

		var icon string
		var line string
		if item.isFolder {
			icon = "📁"
			line = icon + " " + item.name + "/"
		} else {
			icon = "📄"
			line = icon + " " + item.name
		}

		if i == m.treeCursor {
			line = lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Render("▸ " + line)
		} else {
			line = "  " + line
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	if len(m.treeItems) == 0 {
		content.WriteString(DimItemStyle.Render("  No items"))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content.String())
}

func (m Model) renderPreviewColumn(width, height int) string {
	var content strings.Builder

	if m.treeCursor < len(m.treeItems) {
		item := m.treeItems[m.treeCursor]

		if item.isFolder {
			// Show folder contents preview
			content.WriteString(lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				Render("📁 " + item.name))
			content.WriteString("\n\n")
			content.WriteString(DimItemStyle.Render("Folder contents..."))
		} else if item.note != nil {
			// Show note preview
			content.WriteString(lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				Render("📄 " + item.note.Title))
			content.WriteString("\n\n")

			// Metadata
			meta := fmt.Sprintf("ID: %s\nUpdated: %s\nTags: %s",
				item.note.ID,
				item.note.UpdatedAt.Format("Jan 2, 2006"),
				strings.Join(item.note.Tags, ", "))
			content.WriteString(PreviewMetaStyle.Render(meta))
			content.WriteString("\n\n")

			// Content preview
			lines := strings.Split(item.note.Content, "\n")
			previewLines := min(height-8, len(lines))
			for i := 0; i < previewLines; i++ {
				line := lines[i]
				if len(line) > width-4 {
					line = line[:width-4] + "..."
				}
				content.WriteString(line)
				content.WriteString("\n")
			}
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content.String())
}
