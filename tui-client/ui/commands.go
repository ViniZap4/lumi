package ui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/vinizap/lumi/tui-client/domain"
	"github.com/vinizap/lumi/tui-client/filesystem"
	"github.com/vinizap/lumi/tui-client/theme"
)

// glamourStyle returns the glamour style path based on the current theme.
func glamourStyle() string {
	if theme.Current.IsDark {
		return "dark"
	}
	return "light"
}

// --- Messages ---

type itemsLoadedMsg struct {
	items []Item
}

type navItemsLoadedMsg struct {
	items []Item
}

type searchResultsMsg struct {
	results []Item
}

// --- Commands ---

func (m Model) loadItems() tea.Msg {
	var items []Item

	folders, _ := filesystem.ListFolders(m.currentDir)
	for _, f := range folders {
		items = append(items, Item{
			Name:     f.Name,
			IsFolder: true,
			Path:     f.Path,
		})
	}

	notes, _ := filesystem.ListNotes(m.currentDir)
	for _, n := range notes {
		items = append(items, Item{
			Name:     n.Title,
			IsFolder: false,
			Path:     n.Path,
			Note:     n,
		})
	}

	return itemsLoadedMsg{items}
}

func (m Model) loadNavItems() tea.Msg {
	var items []Item

	folders, _ := filesystem.ListFolders(m.navDir)
	for _, f := range folders {
		items = append(items, Item{
			Name:     f.Name,
			IsFolder: true,
			Path:     f.Path,
		})
	}

	notes, _ := filesystem.ListNotes(m.navDir)
	for _, n := range notes {
		items = append(items, Item{
			Name:     n.Title,
			IsFolder: false,
			Path:     n.Path,
			Note:     n,
		})
	}

	return navItemsLoadedMsg{items}
}

func (m Model) performSearch() tea.Msg {
	var results []Item

	filepath.Walk(m.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		note, err := filesystem.ReadNote(path)
		if err != nil {
			return nil
		}

		if m.searchQuery == "" {
			relPath, _ := filepath.Rel(m.rootDir, path)
			results = append(results, Item{
				Name:     relPath,
				IsFolder: false,
				Path:     path,
				Note:     note,
			})
			return nil
		}

		query := strings.ToLower(m.searchQuery)
		match := false
		if m.searchType == "filename" {
			match = strings.Contains(strings.ToLower(info.Name()), query)
		} else {
			match = strings.Contains(strings.ToLower(note.Content), query)
		}

		if match {
			relPath, _ := filepath.Rel(m.rootDir, path)
			results = append(results, Item{
				Name:     relPath,
				IsFolder: false,
				Path:     path,
				Note:     note,
			})
		}
		return nil
	})

	return searchResultsMsg{results}
}

func (m Model) followLinkAtCursor() tea.Cmd {
	if m.lineCursor >= len(m.contentLines) {
		return nil
	}

	line := m.contentLines[m.lineCursor]

	for i := 0; i < len(line)-1; i++ {
		if line[i:i+2] == "[[" {
			end := strings.Index(line[i+2:], "]]")
			if end != -1 {
				linkStart := i
				linkEnd := i + 2 + end + 2
				if m.colCursor >= linkStart && m.colCursor < linkEnd {
					target := line[i+2 : i+2+end]
					return m.openNoteByLink(target)
				}
			}
		}
	}

	return nil
}

func (m *Model) openNoteByLink(target string) tea.Cmd {
	for _, item := range m.items {
		if item.Note != nil && (item.Note.ID == target || strings.Contains(item.Note.Path, target)) {
			m.openNote(item.Note)
			return nil
		}
	}

	var found *Item
	filepath.Walk(m.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		note, err := filesystem.ReadNote(path)
		if err != nil {
			return nil
		}
		if note.ID == target || strings.TrimSuffix(info.Name(), ".md") == target {
			found = &Item{Note: note, Path: path}
			return filepath.SkipAll
		}
		return nil
	})

	if found != nil && found.Note != nil {
		m.openNote(found.Note)
	}
	return nil
}

// openNote sets up the model to view a note with rendered markdown.
func (m *Model) openNote(note *domain.Note) {
	m.fullNote = note
	m.contentLines = strings.Split(note.Content, "\n")
	m.lineCursor = 0
	m.colCursor = 0
	m.desiredCol = 0
	m.visualMode = VisualNone
	m.viewMode = ViewFullNote
	m.renderMarkdown()
}

// renderMarkdown uses glamour to render the note content.
func (m *Model) renderMarkdown() {
	if m.fullNote == nil || m.renderer == nil {
		return
	}

	width := m.width - 4
	if width < 40 {
		width = 40
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath(glamourStyle()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		m.renderedView = m.fullNote.Content
		m.renderedLines = m.contentLines
		return
	}

	rendered, err := renderer.Render(m.fullNote.Content)
	if err != nil {
		m.renderedView = m.fullNote.Content
		m.renderedLines = m.contentLines
		return
	}

	m.renderedView = rendered
	m.renderedLines = strings.Split(rendered, "\n")

	for len(m.renderedLines) > 0 && strings.TrimSpace(m.renderedLines[len(m.renderedLines)-1]) == "" {
		m.renderedLines = m.renderedLines[:len(m.renderedLines)-1]
	}
}
