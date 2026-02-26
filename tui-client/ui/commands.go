package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/vinizap/lumi/tui-client/domain"
	"github.com/vinizap/lumi/tui-client/filesystem"
	"github.com/vinizap/lumi/tui-client/sync"
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

// syncEventMsg wraps a sync event from the WebSocket connection.
type syncEventMsg struct {
	event sync.Event
}

// editorDoneMsg is sent after the editor exits so the note is re-read from disk.
type editorDoneMsg struct {
	notePath string
}

// --- Commands ---

// waitForSyncEvent blocks until a sync event is received from the server.
func (m Model) waitForSyncEvent() tea.Msg {
	if m.syncClient == nil {
		return nil
	}
	evt, ok := <-m.syncClient.Events()
	if !ok {
		return nil
	}
	return syncEventMsg{event: evt}
}

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

func (m *Model) followLinkAtCursor() tea.Cmd {
	if m.lineCursor >= len(m.contentLines) {
		return nil
	}

	line := m.contentLines[m.lineCursor]

	// 1. Checkbox toggle takes priority
	if m.toggleCheckbox() {
		return nil
	}

	// 2. Wikilinks [[target]]
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

	// 3. Standard links [text](url)
	if url := m.linkURLAtCursor(); url != "" {
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			openBrowser(url)
			m.statusMsg = "Opened in browser"
			return nil
		}
		// Treat as internal note reference
		return m.openNoteByLink(url)
	}

	return nil
}

// toggleCheckbox checks if the current line has a checkbox and toggles it.
// Returns true if a checkbox was found and toggled.
func (m *Model) toggleCheckbox() bool {
	if m.lineCursor >= len(m.contentLines) || m.fullNote == nil {
		return false
	}

	line := m.contentLines[m.lineCursor]
	trimmed := strings.TrimLeft(line, " \t")
	prefix := line[:len(line)-len(trimmed)]

	// Match list marker + checkbox: "- [ ] " or "- [x] " (also + and *)
	var marker string
	rest := trimmed
	if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '+' || rest[0] == '*') && rest[1] == ' ' {
		marker = rest[:2]
		rest = rest[2:]
	} else {
		return false
	}

	if len(rest) < 3 || rest[0] != '[' || rest[2] != ']' {
		return false
	}

	var newBox string
	switch rest[1] {
	case ' ':
		newBox = "[x]"
	case 'x', 'X':
		newBox = "[ ]"
	default:
		return false
	}

	// Rebuild the line with toggled checkbox
	m.contentLines[m.lineCursor] = prefix + marker + newBox + rest[3:]

	// Persist to disk
	m.fullNote.Content = strings.Join(m.contentLines, "\n")
	filesystem.WriteNote(m.fullNote)

	return true
}

// linkURLAtCursor returns the URL from a standard [text](url) link if the
// cursor is positioned within its bounds, or empty string otherwise.
func (m *Model) linkURLAtCursor() string {
	if m.lineCursor >= len(m.contentLines) {
		return ""
	}
	line := m.contentLines[m.lineCursor]
	runes := []rune(line)
	n := len(runes)

	for i := 0; i < n; i++ {
		if runes[i] != '[' {
			continue
		}
		// Skip wikilinks
		if i+1 < n && runes[i+1] == '[' {
			continue
		}
		j := i + 1
		for j < n && runes[j] != ']' {
			j++
		}
		if j >= n || j+1 >= n || runes[j+1] != '(' {
			continue
		}
		k := j + 2
		for k < n && runes[k] != ')' {
			k++
		}
		if k >= n {
			continue
		}
		// We found [text](url) spanning i..k
		if m.colCursor >= i && m.colCursor <= k {
			return string(runes[j+2 : k])
		}
	}
	return ""
}

// wikiLinkTargetAtCursor returns the wikilink target if the cursor is on one.
func (m *Model) wikiLinkTargetAtCursor() string {
	if m.lineCursor >= len(m.contentLines) {
		return ""
	}
	line := m.contentLines[m.lineCursor]
	for i := 0; i < len(line)-1; i++ {
		if line[i:i+2] == "[[" {
			end := strings.Index(line[i+2:], "]]")
			if end != -1 {
				linkStart := i
				linkEnd := i + 2 + end + 2
				if m.colCursor >= linkStart && m.colCursor < linkEnd {
					return line[i+2 : i+2+end]
				}
			}
		}
	}
	return ""
}

// openLinkInSplit opens the link under the cursor in a split view.
// Returns true if a link was found and opened.
func (m *Model) openLinkInSplit(mode string) bool {
	// Try wikilink first
	if target := m.wikiLinkTargetAtCursor(); target != "" {
		note := m.findNoteByLink(target)
		if note != nil {
			m.splitMode = mode
			m.splitNote = note
			return true
		}
	}
	// Try standard link to internal note
	if url := m.linkURLAtCursor(); url != "" {
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			note := m.findNoteByLink(url)
			if note != nil {
				m.splitMode = mode
				m.splitNote = note
				return true
			}
		}
	}
	return false
}

// findNoteByLink searches for a note matching the given target (ID or path).
func (m *Model) findNoteByLink(target string) *domain.Note {
	for _, item := range m.items {
		if item.Note != nil && (item.Note.ID == target || strings.Contains(item.Note.Path, target)) {
			return item.Note
		}
	}

	var found *domain.Note
	filepath.Walk(m.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		note, err := filesystem.ReadNote(path)
		if err != nil {
			return nil
		}
		if note.ID == target || strings.TrimSuffix(info.Name(), ".md") == target {
			found = note
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// openExternalLink opens the URL under the cursor in the system browser.
// Returns true if a URL was found and opened.
func (m *Model) openExternalLink() bool {
	// Check standard link URL
	if url := m.linkURLAtCursor(); url != "" {
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			openBrowser(url)
			m.statusMsg = "Opened in browser"
			return true
		}
	}
	return false
}

// openBrowser opens a URL in the default system browser.
func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

func (m *Model) openNoteByLink(target string) tea.Cmd {
	note := m.findNoteByLink(target)
	if note != nil {
		m.openNote(note)
	}
	return nil
}

// openNote sets up the model to view a note with rendered markdown.
func (m *Model) openNote(note *domain.Note) {
	m.fullNote = note
	m.contentLines = strings.Split(note.Content, "\n")
	preprocessTableBlocks(m.contentLines)
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
