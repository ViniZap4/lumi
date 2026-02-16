// tui-client/ui/main.go
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/atotto/clipboard"
	"github.com/vinizap/lumi/tui-client/domain"
	"github.com/vinizap/lumi/tui-client/editor"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

// Simple model - home, tree (3-column), and full view with cursor
type SimpleModel struct {
	rootDir      string
	currentDir   string
	items        []Item
	cursor       int
	search       string
	width        int
	height       int
	viewMode     ViewMode
	fullNote     *domain.Note
	contentLines []string
	lineCursor   int
	colCursor    int
	err          error
	renderer     *glamour.TermRenderer
	searchMode   bool // true = recursive search
	
	// Enhanced modes
	visualMode   bool
	visualStart  int
	visualEnd    int
	showTree     bool // tree modal overlay
	splitMode    string // "", "horizontal", "vertical"
	splitNote    *domain.Note
	
	// Search modal
	showSearch   bool
	searchQuery  string
	searchType   string // "content" or "filename"
	searchResults []Item
}

type ViewMode int

const (
	ViewHome ViewMode = iota
	ViewTree
	ViewFullNote
)

type Item struct {
	Name     string
	IsFolder bool
	Path     string
	Note     *domain.Note
}

func NewSimpleModel(rootDir string) SimpleModel {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	return SimpleModel{
		rootDir:    rootDir,
		currentDir: rootDir,
		items:      []Item{},
		viewMode:   ViewHome,
		renderer:   renderer,
	}
}

func (m SimpleModel) Init() tea.Cmd {
	return m.loadItems
}

// Recursive search across all subdirectories
func (m SimpleModel) searchRecursive(query string) []Item {
	var results []Item
	query = strings.ToLower(query)
	
	filepath.Walk(m.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		
		// Check filename match
		if strings.Contains(strings.ToLower(info.Name()), query) {
			relPath, _ := filepath.Rel(m.rootDir, path)
			note, _ := filesystem.ReadNote(path)
			results = append(results, Item{
				Name:     relPath,
				IsFolder: false,
				Path:     path,
				Note:     note,
			})
		}
		return nil
	})
	
	return results
}

// Search in content or filename
func (m SimpleModel) performSearch() tea.Msg {
	if m.searchQuery == "" {
		return itemsLoadedMsg{m.searchResults}
	}
	
	var results []Item
	query := strings.ToLower(m.searchQuery)
	
	filepath.Walk(m.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		
		note, err := filesystem.ReadNote(path)
		if err != nil {
			return nil
		}
		
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
	
	m.searchResults = results
	return itemsLoadedMsg{results}
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
		// Home view
		if m.viewMode == ViewHome {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter", "t":
				m.viewMode = ViewTree
				return m, m.loadItems
			default:
				// Start search from home
				if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
					m.search = msg.String()
					m.viewMode = ViewTree
					return m, m.loadItems
				}
			}
			return m, nil
		}

		// Full view mode with cursor
		if m.viewMode == ViewFullNote {
			// Search modal is open
			if m.showSearch {
				switch msg.String() {
				case "esc":
					m.showSearch = false
					m.searchQuery = ""
					return m, nil
				case "ctrl+f":
					// Toggle search type
					if m.searchType == "filename" {
						m.searchType = "content"
					} else {
						m.searchType = "filename"
					}
					return m, func() tea.Msg { return m.performSearch() }
				case "j", "down":
					if m.cursor < len(m.searchResults)-1 {
						m.cursor++
					}
				case "k", "up":
					if m.cursor > 0 {
						m.cursor--
					}
				case "enter":
					// Open selected result
					if m.cursor < len(m.searchResults) {
						item := m.searchResults[m.cursor]
						if item.Note != nil {
							m.fullNote = item.Note
							m.contentLines = strings.Split(item.Note.Content, "\n")
							m.lineCursor = 0
							m.colCursor = 0
							m.showSearch = false
						}
					}
				case "backspace":
					if len(m.searchQuery) > 0 {
						m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
						return m, func() tea.Msg { return m.performSearch() }
					}
				default:
					// Add to search query
					if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
						m.searchQuery += msg.String()
						return m, func() tea.Msg { return m.performSearch() }
					}
				}
				return m, nil
			}
			
			// Tree modal is open - handle tree navigation
			if m.showTree {
				switch msg.String() {
				case "esc":
					m.showTree = false
					return m, nil
				case "j", "down":
					if m.cursor < len(m.items)-1 {
						m.cursor++
					}
				case "k", "up":
					if m.cursor > 0 {
						m.cursor--
					}
				case "enter", "l":
					// Open selected note
					if m.cursor < len(m.items) {
						item := m.items[m.cursor]
						if !item.IsFolder && item.Note != nil {
							m.fullNote = item.Note
							m.contentLines = strings.Split(item.Note.Content, "\n")
							m.lineCursor = 0
							m.colCursor = 0
							m.showTree = false
						}
					}
				case "backspace":
					if len(m.search) > 0 {
						m.search = m.search[:len(m.search)-1]
						return m, m.loadItems
					}
				default:
					// Add to search
					if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
						m.search += msg.String()
						return m, m.loadItems
					}
				}
				return m, nil
			}
			
			// Normal full view navigation
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "esc":
				if m.visualMode {
					m.visualMode = false
				} else if m.showTree {
					m.showTree = false
				} else {
					m.viewMode = ViewTree
				}
				return m, nil
			case "v":
				// Toggle visual mode
				m.visualMode = !m.visualMode
				if m.visualMode {
					m.visualStart = m.lineCursor
					m.visualEnd = m.lineCursor
				}
			case "y":
				// Copy selected lines to clipboard
				if m.visualMode {
					start := min(m.visualStart, m.visualEnd)
					end := max(m.visualStart, m.visualEnd)
					var selected []string
					for i := start; i <= end && i < len(m.contentLines); i++ {
						selected = append(selected, m.contentLines[i])
					}
					text := strings.Join(selected, "\n")
					clipboard.WriteAll(text)
					m.visualMode = false
				}
			case "t":
				// Toggle tree modal
				m.showTree = !m.showTree
				if m.showTree {
					return m, m.loadItems
				}
			case "/":
				// Open search modal
				m.showSearch = !m.showSearch
				m.searchQuery = ""
				m.searchType = "filename"
				if m.showSearch {
					return m, func() tea.Msg { return m.performSearch() }
				}
			case "s":
				// Horizontal split
				m.splitMode = "horizontal"
			case "S":
				// Vertical split
				m.splitMode = "vertical"
			case "h":
				if m.colCursor > 0 {
					m.colCursor--
				}
			case "l":
				if m.lineCursor < len(m.contentLines) && m.colCursor < len(m.contentLines[m.lineCursor]) {
					m.colCursor++
				}
			case "j":
				if m.visualMode {
					m.visualEnd = m.lineCursor
				}
				if m.lineCursor < len(m.contentLines)-1 {
					m.lineCursor++
					if m.lineCursor < len(m.contentLines) && m.colCursor > len(m.contentLines[m.lineCursor]) {
						m.colCursor = len(m.contentLines[m.lineCursor])
					}
				}
				if m.visualMode {
					m.visualEnd = m.lineCursor
				}
			case "k":
				if m.visualMode {
					m.visualEnd = m.lineCursor
				}
				if m.lineCursor > 0 {
					m.lineCursor--
					if m.colCursor > len(m.contentLines[m.lineCursor]) {
						m.colCursor = len(m.contentLines[m.lineCursor])
					}
				}
				if m.visualMode {
					m.visualEnd = m.lineCursor
				}
			case "0":
				m.colCursor = 0
			case "$":
				if m.lineCursor < len(m.contentLines) {
					m.colCursor = len(m.contentLines[m.lineCursor])
				}
			case "g":
				m.lineCursor = 0
				m.colCursor = 0
			case "G":
				m.lineCursor = len(m.contentLines) - 1
				m.colCursor = 0
			case "enter":
				// Follow link if cursor is on one
				return m, m.followLinkAtCursor()
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
		case "/":
			// Start search mode - next chars will be added to search
			// (search is already being added in default case)
			if m.search == "" {
				// Just pressed /, wait for input
			}
		case "esc":
			// Clear search
			m.search = ""
			m.searchMode = false
			return m, m.loadItems
		case "j", "down":
			if m.search == "" { // Only navigate if not searching
				if m.cursor < len(m.items)-1 {
					m.cursor++
				}
			}
		case "k", "up":
			if m.search == "" {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "h":
			// Go up directory (only if not searching)
			if m.search == "" && m.currentDir != m.rootDir {
				m.currentDir = filepath.Dir(m.currentDir)
				m.cursor = 0
				m.search = ""
				return m, m.loadItems
			}
		case "l", "enter":
			// Open folder or note (only if not searching)
			if m.search == "" && m.cursor < len(m.items) {
				item := m.items[m.cursor]
				if item.IsFolder {
					m.currentDir = item.Path
					m.cursor = 0
					m.search = ""
					return m, m.loadItems
				} else if item.Note != nil {
					m.viewMode = ViewFullNote
					m.fullNote = item.Note
					m.contentLines = strings.Split(item.Note.Content, "\n")
					m.lineCursor = 0
					m.colCursor = 0
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

	switch m.viewMode {
	case ViewHome:
		return m.renderHome()
	case ViewFullNote:
		base := m.renderFullNote()
		// Overlay search modal if active
		if m.showSearch {
			return m.renderWithSearchModal(base)
		}
		// Overlay tree modal if active
		if m.showTree {
			return m.renderWithTreeModal(base)
		}
		// Render split view if active
		if m.splitMode != "" && m.splitNote != nil {
			return m.renderSplitView()
		}
		return base
	default:
		return m.renderTreeYazi()
	}
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
	if m.fullNote == nil {
		return "Error: No note loaded"
	}

	var s strings.Builder

	// Render with glamour for beautiful display
	rendered := m.fullNote.Content
	if m.renderer != nil {
		glamourRendered, err := m.renderer.Render(m.fullNote.Content)
		if err == nil {
			rendered = glamourRendered
		}
	}

	// Split into lines for cursor navigation
	lines := strings.Split(rendered, "\n")
	
	// Keep contentLines in sync for link detection
	if len(m.contentLines) == 0 {
		m.contentLines = strings.Split(m.fullNote.Content, "\n")
	}

	// Scrollable view centered on cursor
	maxLines := m.height - 5
	start := m.lineCursor - maxLines/2
	if start < 0 {
		start = 0
	}
	if start > len(lines)-maxLines {
		start = max(0, len(lines)-maxLines)
	}
	end := min(start+maxLines, len(lines))

	// Render lines with cursor and visual selection
	for i := start; i < end; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}

		// Visual mode highlighting
		inVisual := m.visualMode && i >= min(m.visualStart, m.visualEnd) && i <= max(m.visualStart, m.visualEnd)
		
		// Show cursor on current line
		if i == m.lineCursor {
			// Find cursor position in rendered line (approximate)
			cursorPos := m.colCursor
			if cursorPos > len(line) {
				cursorPos = len(line)
			}
			
			if cursorPos <= len(line) {
				before := ""
				if cursorPos > 0 {
					before = line[:cursorPos]
				}
				cursor := "█"
				after := ""
				if cursorPos < len(line) {
					cursor = lipgloss.NewStyle().
						Background(accentColor).
						Foreground(lipgloss.Color("0")).
						Render(string(line[cursorPos]))
					after = line[cursorPos+1:]
				}
				line = before + cursor + after
			}
		}

		if inVisual {
			line = lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(line)
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	// Status bar
	s.WriteString("\n")
	mode := ""
	if m.visualMode {
		mode = " [VISUAL]"
	}
	if m.showTree {
		mode += " [TREE]"
	}
	status := fmt.Sprintf("Ln %d, Col %d%s | %s", m.lineCursor+1, m.colCursor+1, mode, m.fullNote.ID)
	s.WriteString(HelpStyle.Render(status))
	s.WriteString("\n")
	help := HelpStyle.Render("hjkl=move | v=visual | y=copy | enter=link | t=tree | /=search | e=edit | esc=back")
	s.WriteString(help)

	return s.String()
}

func (m SimpleModel) renderWithTreeModal(base string) string {
	// Render tree as centered modal overlay
	modalWidth := min(m.width-10, 80)
	modalHeight := min(m.height-6, 30)
	
	var modal strings.Builder
	modal.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render("📂 Select Note"))
	modal.WriteString("\n\n")
	
	// Search bar
	if m.search != "" {
		modal.WriteString(lipgloss.NewStyle().
			Foreground(accentColor).
			Render("🔍 " + m.search + "█"))
		modal.WriteString("\n\n")
	}
	
	// Items
	maxItems := modalHeight - 8
	for i, item := range m.items {
		if i >= maxItems {
			break
		}
		
		icon := "📄"
		if item.IsFolder {
			icon = "📁"
		}
		line := icon + " " + item.Name
		
		if i == m.cursor {
			line = lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Render("▸ " + line)
		} else {
			line = "  " + line
		}
		
		modal.WriteString(line)
		modal.WriteString("\n")
	}
	
	modal.WriteString("\n")
	modal.WriteString(HelpStyle.Render("hjkl=move | enter=open | esc=close"))
	
	// Style modal
	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(modalWidth).
		Render(modal.String())
	
	// Center on screen
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

func (m SimpleModel) followLinkAtCursor() tea.Cmd {
	if m.lineCursor >= len(m.contentLines) {
		return nil
	}

	line := m.contentLines[m.lineCursor]
	
	// Check if cursor is on a [[wiki-link]]
	// Simple check: find all links in line and see if cursor is within one
	for i := 0; i < len(line)-1; i++ {
		if line[i:i+2] == "[[" {
			end := strings.Index(line[i+2:], "]]")
			if end != -1 {
				linkStart := i
				linkEnd := i + 2 + end + 2
				if m.colCursor >= linkStart && m.colCursor < linkEnd {
					// Extract link target
					target := line[i+2 : i+2+end]
					// Try to find and open this note
					for _, item := range m.items {
						if item.Note != nil && (item.Note.ID == target || strings.Contains(item.Note.Path, target)) {
							m.fullNote = item.Note
							m.contentLines = strings.Split(item.Note.Content, "\n")
							m.lineCursor = 0
							m.colCursor = 0
							return nil
						}
					}
				}
			}
		}
	}
	
	return nil
}

func (m SimpleModel) renderHome() string {
	var s strings.Builder

	// ASCII art centered
	art := `
  ██╗     ██╗   ██╗███╗   ███╗██╗
  ██║     ██║   ██║████╗ ████║██║
  ██║     ██║   ██║██╔████╔██║██║
  ██║     ██║   ██║██║╚██╔╝██║██║
  ███████╗╚██████╔╝██║ ╚═╝ ██║██║
  ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝
`
	artStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(3)
	s.WriteString(artStyle.Render(art))
	s.WriteString("\n\n")

	// Subtitle
	subtitle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(m.width).
		Align(lipgloss.Center).
		Render("Local-first Markdown notes with vim motions")
	s.WriteString(subtitle)
	s.WriteString("\n\n\n")

	// Search prompt
	searchPrompt := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Width(m.width).
		Align(lipgloss.Center).
		Render("Type to search or press Enter to browse")
	s.WriteString(searchPrompt)
	s.WriteString("\n\n")

	// Help
	help := HelpStyle.Render("enter=browse | type=search | q=quit")
	helpCentered := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(help)
	s.WriteString(helpCentered)

	return s.String()
}

func (m SimpleModel) renderTreeYazi() string {
	// 3-column Yazi-style layout
	leftWidth := m.width / 4
	centerWidth := m.width / 3
	rightWidth := m.width - leftWidth - centerWidth - 4

	var s strings.Builder

	// Title
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

	if m.search != "" {
		searchBar := lipgloss.NewStyle().
			Foreground(accentColor).
			Render(" 🔍 " + m.search + "█")
		s.WriteString(searchBar)
	}
	s.WriteString("\n\n")

	// Three columns
	leftCol := m.renderParentCol(leftWidth, m.height-4)
	centerCol := m.renderCenterCol(centerWidth, m.height-4)
	rightCol := m.renderPreviewCol(rightWidth, m.height-4)

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftCol,
		lipgloss.NewStyle().Foreground(mutedColor).Render("│"),
		centerCol,
		lipgloss.NewStyle().Foreground(mutedColor).Render("│"),
		rightCol,
	)
	s.WriteString(columns)

	// Help
	s.WriteString("\n")
	help := HelpStyle.Render("hjkl=move | enter=open | type=search | q=quit")
	s.WriteString(help)

	return s.String()
}

func (m SimpleModel) renderParentCol(width, height int) string {
	var s strings.Builder
	
	if m.currentDir != m.rootDir {
		s.WriteString(DimItemStyle.Render(".."))
		s.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}

func (m SimpleModel) renderCenterCol(width, height int) string {
	var s strings.Builder

	for i, item := range m.items {
		if i >= height {
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

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}

func (m SimpleModel) renderPreviewCol(width, height int) string {
	var s strings.Builder

	if m.cursor < len(m.items) {
		item := m.items[m.cursor]

		if item.IsFolder {
			s.WriteString(lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				Render("📁 " + item.Name))
			s.WriteString("\n\n")
			
			// Show folder contents
			folderNotes, _ := filesystem.ListNotes(item.Path)
			if len(folderNotes) > 0 {
				s.WriteString(DimItemStyle.Render(fmt.Sprintf("%d notes:", len(folderNotes))))
				s.WriteString("\n")
				for i, note := range folderNotes {
					if i >= height-4 {
						break
					}
					s.WriteString(fmt.Sprintf("  📄 %s\n", note.Title))
				}
			} else {
				s.WriteString(DimItemStyle.Render("(empty folder)"))
			}
		} else if item.Note != nil {
			s.WriteString(lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				Render("📄 " + item.Note.Title))
			s.WriteString("\n\n")

			// Metadata
			meta := fmt.Sprintf("%s • %s",
				item.Note.ID,
				item.Note.UpdatedAt.Format("Jan 2"))
			s.WriteString(PreviewMetaStyle.Render(meta))
			s.WriteString("\n\n")

			// Content preview
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
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(s.String())
}


func (m SimpleModel) renderWithSearchModal(base string) string {
	// Telescope-style centered search modal
	modalWidth := min(m.width-10, 100)
	modalHeight := min(m.height-6, 35)
	
	var modal strings.Builder
	
	// Title
	modal.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render("🔍 Search Notes"))
	modal.WriteString("\n\n")
	
	// Search type indicator
	typeIndicator := "Filename"
	if m.searchType == "content" {
		typeIndicator = "Content"
	}
	modal.WriteString(lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(fmt.Sprintf("[%s] ", typeIndicator)))
	
	// Search input
	modal.WriteString(lipgloss.NewStyle().
		Foreground(accentColor).
		Render(m.searchQuery + "█"))
	modal.WriteString("\n\n")
	
	// Results
	maxResults := modalHeight - 10
	for i, item := range m.searchResults {
		if i >= maxResults {
			break
		}
		
		line := "📄 " + item.Name
		
		if i == m.cursor {
			line = lipgloss.NewStyle().
				Foreground(accentColor).
				Background(selectedBg).
				Bold(true).
				Render("▸ " + line)
		} else {
			line = "  " + line
		}
		
		modal.WriteString(line)
		modal.WriteString("\n")
		
		// Show preview snippet
		if i == m.cursor && item.Note != nil {
			preview := strings.Split(item.Note.Content, "\n")[0]
			if len(preview) > modalWidth-6 {
				preview = preview[:modalWidth-6] + "..."
			}
			modal.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render("    " + preview))
			modal.WriteString("\n")
		}
	}
	
	if len(m.searchResults) == 0 && m.searchQuery != "" {
		modal.WriteString(DimItemStyle.Render("  No results found"))
		modal.WriteString("\n")
	}
	
	modal.WriteString("\n")
	modal.WriteString(HelpStyle.Render("ctrl+f=toggle type | enter=open | esc=close"))
	
	// Style modal
	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(modalWidth).
		Render(modal.String())
	
	// Center on screen
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
	)
}

func (m SimpleModel) renderSplitView() string {
	var s strings.Builder
	
	if m.splitMode == "horizontal" {
		// Top and bottom
		topHeight := m.height / 2
		bottomHeight := m.height - topHeight - 1
		
		// Render main note
		s.WriteString(m.renderNoteInBox(m.fullNote, m.width, topHeight))
		s.WriteString("\n")
		s.WriteString(strings.Repeat("─", m.width))
		s.WriteString("\n")
		
		// Render split note
		s.WriteString(m.renderNoteInBox(m.splitNote, m.width, bottomHeight))
	} else {
		// Left and right
		leftWidth := m.width / 2
		rightWidth := m.width - leftWidth - 1
		
		left := m.renderNoteInBox(m.fullNote, leftWidth, m.height)
		right := m.renderNoteInBox(m.splitNote, rightWidth, m.height)
		
		s.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().Foreground(mutedColor).Render("│"),
			right,
		))
	}
	
	return s.String()
}

func (m SimpleModel) renderNoteInBox(note *domain.Note, width, height int) string {
	if note == nil {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Render("No note")
	}
	
	var s strings.Builder
	
	// Title
	s.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(note.Title))
	s.WriteString("\n\n")
	
	// Content
	lines := strings.Split(note.Content, "\n")
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
