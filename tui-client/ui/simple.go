// tui-client/ui/main.go
package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/domain"
	"github.com/vinizap/lumi/tui-client/editor"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

// Simple model - only tree view and full view
type SimpleModel struct {
	rootDir    string
	currentDir string
	items      []Item
	cursor     int
	search     string
	width      int
	height     int
	fullView   bool
	fullNote   *domain.Note
	err        error
}

type Item struct {
	Name     string
	IsFolder bool
	Path     string
	Note     *domain.Note
}

func NewSimpleModel(rootDir string) SimpleModel {
	return SimpleModel{
		rootDir:    rootDir,
		currentDir: rootDir,
		items:      []Item{},
		fullView:   false,
	}
}

func (m SimpleModel) Init() tea.Cmd {
	return m.loadItems
}

func (m SimpleModel) loadItems() tea.Msg {
	var items []Item

	// Load folders
	folders, _ := filesystem.ListFolders(m.currentDir)
	for _, f := range folders {
		items = append(items, Item{
			Name:     f.Name,
			IsFolder: true,
			Path:     f.Path,
		})
	}

	// Load notes
	notes, _ := filesystem.ListNotes(m.currentDir)
	for _, n := range notes {
		items = append(items, Item{
			Name:     n.Title,
			IsFolder: false,
			Path:     n.Path,
			Note:     n,
		})
	}

	// Filter by search
	if m.search != "" {
		var filtered []Item
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Name), strings.ToLower(m.search)) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	return itemsLoadedMsg{items}
}

type itemsLoadedMsg struct {
	items []Item
}

func (m SimpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Full view mode
		if m.fullView {
			switch msg.String() {
			case "q", "esc":
				m.fullView = false
				return m, nil
			case "e":
				if m.fullNote != nil {
					return m, tea.ExecProcess(editor.OpenCmd(m.fullNote), func(err error) tea.Msg {
						return m.loadItems()
					})
				}
			}
			return m, nil
		}

		// Tree view mode
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "h":
			// Go up directory
			if m.currentDir != m.rootDir {
				m.currentDir = filepath.Dir(m.currentDir)
				m.cursor = 0
				m.search = ""
				return m, m.loadItems
			}
		case "l", "enter":
			// Open folder or note
			if m.cursor < len(m.items) {
				item := m.items[m.cursor]
				if item.IsFolder {
					m.currentDir = item.Path
					m.cursor = 0
					m.search = ""
					return m, m.loadItems
				} else if item.Note != nil {
					m.fullView = true
					m.fullNote = item.Note
				}
			}
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.cursor = 0
				return m, m.loadItems
			}
		default:
			// Add to search
			if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
				m.search += msg.String()
				m.cursor = 0
				return m, m.loadItems
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case itemsLoadedMsg:
		m.items = msg.items
	}

	return m, nil
}

func (m SimpleModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.fullView && m.fullNote != nil {
		return m.renderFullNote()
	}

	return m.renderTree()
}

func (m SimpleModel) renderTree() string {
	var s strings.Builder

	// Title with path
	pathDisplay := strings.TrimPrefix(m.currentDir, m.rootDir)
	if pathDisplay == "" {
		pathDisplay = "~"
	} else {
		pathDisplay = "~" + pathDisplay
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render("📂 " + pathDisplay)

	s.WriteString(title)
	s.WriteString("\n\n")

	// Search bar
	if m.search != "" {
		searchBar := lipgloss.NewStyle().
			Foreground(accentColor).
			Render("🔍 " + m.search + "█")
		s.WriteString(searchBar)
		s.WriteString("\n\n")
	}

	// Items list
	maxItems := m.height - 6
	for i, item := range m.items {
		if i >= maxItems {
			break
		}

		icon := "📄"
		name := item.Name
		if item.IsFolder {
			icon = "📁"
			name += "/"
		}

		line := icon + " " + name

		if i == m.cursor {
			line = lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Render("▸ " + line)
		} else {
			line = "  " + line
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	if len(m.items) == 0 {
		s.WriteString(DimItemStyle.Render("  No items"))
	}

	// Help
	s.WriteString("\n")
	help := HelpStyle.Render("hjkl=move | enter=open | type=search | q=quit")
	s.WriteString(help)

	return s.String()
}

func (m SimpleModel) renderFullNote() string {
	var s strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Width(m.width).
		Align(lipgloss.Center).
		Render(m.fullNote.Title)
	s.WriteString(title)
	s.WriteString("\n\n")

	// Metadata
	meta := fmt.Sprintf("%s • %s",
		m.fullNote.ID,
		m.fullNote.UpdatedAt.Format("Jan 2, 2006"))
	if len(m.fullNote.Tags) > 0 {
		meta += " • " + strings.Join(m.fullNote.Tags, ", ")
	}
	metaStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(m.width).
		Align(lipgloss.Center)
	s.WriteString(metaStyle.Render(meta))
	s.WriteString("\n\n")

	// Content
	lines := strings.Split(m.fullNote.Content, "\n")
	maxLines := m.height - 8
	for i := 0; i < min(len(lines), maxLines); i++ {
		s.WriteString(lines[i])
		s.WriteString("\n")
	}

	// Help
	s.WriteString("\n")
	help := HelpStyle.Render("e=edit | esc=back | q=quit")
	s.WriteString(help)

	return s.String()
}
