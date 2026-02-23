package ui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/editor"
)

func (m Model) updateNote(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""      // clear transient message on any keypress
	m.yankHighlight = false // clear yank flash on any keypress

	if m.showSearch {
		if m.inFileSearch {
			return m.updateInFileSearch(msg)
		}
		return m.updateNoteGlobalSearch(msg)
	}

	if m.showNav {
		return m.updateNavModal(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		if m.visualMode != VisualNone {
			m.visualMode = VisualNone
		} else if m.splitMode != "" {
			m.splitMode = ""
			m.splitNote = nil
		} else {
			m.viewMode = ViewTree
		}
		return m, nil

	// Visual modes
	case "v":
		if m.visualMode == VisualChar {
			m.visualMode = VisualNone
		} else {
			m.startVisualChar()
		}
	case "V":
		if m.visualMode == VisualLine {
			m.visualMode = VisualNone
		} else {
			m.startVisualLine()
		}

	// Yank
	case "y":
		return m, m.yankSelection()

	// Cursor movement
	case "h", "left":
		m.moveLeft()
		m.updateVisualEnd()
	case "l", "right":
		m.moveRight()
		m.updateVisualEnd()
	case "j", "down":
		m.moveDown(1)
	case "k", "up":
		m.moveUp(1)

	// Word motions
	case "w":
		m.moveWordForward()
		m.updateVisualEnd()
	case "b":
		m.moveWordBackward()
		m.updateVisualEnd()
	case "e":
		if m.visualMode != VisualNone {
			m.moveWordEnd()
			m.updateVisualEnd()
		} else {
			if m.fullNote != nil {
				return m, tea.ExecProcess(editor.OpenCmd(m.fullNote), func(err error) tea.Msg {
					return m.loadItems()
				})
			}
		}

	// Line navigation
	case "0":
		m.moveToLineStart()
		m.updateVisualEnd()
	case "^":
		m.moveToFirstNonBlank()
		m.updateVisualEnd()
	case "$":
		m.moveToLineEnd()
		m.updateVisualEnd()

	// File navigation
	case "g":
		m.moveToFileStart()
	case "G":
		m.moveToFileEnd()

	// Half-page scroll
	case "ctrl+d":
		m.halfPageDown()
	case "ctrl+u":
		m.halfPageUp()

	// Navigation modal
	case "t":
		m.showNav = true
		m.navDir = m.currentDir
		m.navCursor = 0
		return m, m.loadNavItems

	// Search
	case "/":
		m.showSearch = true
		m.searchQuery = ""
		m.inFileSearch = true
		m.searchType = "content"
	case "ctrl+/":
		m.showSearch = true
		m.searchQuery = ""
		m.inFileSearch = false
		m.searchType = "filename"
		return m, func() tea.Msg { return m.performSearch() }

	// Split
	case "s":
		if m.visualMode == VisualNone {
			m.splitMode = "horizontal"
			m.showNav = true
			m.navDir = m.currentDir
			m.navCursor = 0
			return m, m.loadNavItems
		}
	case "S":
		if m.visualMode == VisualNone {
			m.splitMode = "vertical"
			m.showNav = true
			m.navDir = m.currentDir
			m.navCursor = 0
			return m, m.loadNavItems
		}

	// Follow link
	case "enter":
		return m, m.followLinkAtCursor()
	}

	return m, nil
}

// updateNavModal handles keys when the navigation modal is open inside note view.
func (m Model) updateNavModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showNav = false
		// If we opened nav for split and didn't pick, cancel split
		if m.splitNote == nil {
			m.splitMode = ""
		}
		return m, nil
	case "j", "down":
		if m.navCursor < len(m.navItems)-1 {
			m.navCursor++
		}
	case "k", "up":
		if m.navCursor > 0 {
			m.navCursor--
		}
	case "h":
		if m.navDir != m.rootDir {
			m.navDir = filepath.Dir(m.navDir)
			m.navCursor = 0
			return m, m.loadNavItems
		}
	case "enter", "l":
		if m.navCursor < len(m.navItems) {
			item := m.navItems[m.navCursor]
			if item.IsFolder {
				m.navDir = item.Path
				m.navCursor = 0
				return m, m.loadNavItems
			} else if item.Note != nil {
				if m.splitMode != "" {
					// We're picking a note for a split
					m.splitNote = item.Note
					m.showNav = false
					m.navCursor = 0
				} else {
					// Switch to this note
					m.openNote(item.Note)
					m.showNav = false
					m.navCursor = 0
				}
			}
		}
	case "s":
		if m.navCursor < len(m.navItems) && m.navItems[m.navCursor].Note != nil {
			m.splitMode = "horizontal"
			m.splitNote = m.navItems[m.navCursor].Note
			m.showNav = false
			m.navCursor = 0
		}
	case "S":
		if m.navCursor < len(m.navItems) && m.navItems[m.navCursor].Note != nil {
			m.splitMode = "vertical"
			m.splitNote = m.navItems[m.navCursor].Note
			m.showNav = false
			m.navCursor = 0
		}
	case "g":
		m.navCursor = 0
	case "G":
		if len(m.navItems) > 0 {
			m.navCursor = len(m.navItems) - 1
		}
	}

	return m, nil
}

func (m Model) updateNoteGlobalSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showSearch = false
		m.searchQuery = ""
		m.cursor = 0
		return m, nil
	case "ctrl+f":
		if m.searchType == "filename" {
			m.searchType = "content"
		} else {
			m.searchType = "filename"
		}
		return m, func() tea.Msg { return m.performSearch() }
	case "down":
		if m.cursor < len(m.searchResults)-1 {
			m.cursor++
		}
		return m, nil
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "enter":
		if m.cursor < len(m.searchResults) && m.searchResults[m.cursor].Note != nil {
			m.openNote(m.searchResults[m.cursor].Note)
			m.showSearch = false
			m.cursor = 0
		}
		return m, nil
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			return m, func() tea.Msg { return m.performSearch() }
		}
	default:
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
			m.searchQuery += msg.String()
			return m, func() tea.Msg { return m.performSearch() }
		}
	}

	return m, nil
}

func (m Model) updateInFileSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showSearch = false
		m.searchQuery = ""
		m.cursor = 0
		return m, nil
	case "down":
		matches := m.findInFileMatches()
		if m.cursor < len(matches)-1 {
			m.cursor++
		}
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		matches := m.findInFileMatches()
		if m.cursor < len(matches) {
			m.lineCursor = matches[m.cursor]
			m.colCursor = 0
			m.desiredCol = 0
			m.showSearch = false
			m.cursor = 0
		}
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.cursor = 0
		}
	default:
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
			m.searchQuery += msg.String()
			m.cursor = 0
		}
	}

	return m, nil
}

func (m Model) findInFileMatches() []int {
	var matches []int
	if m.searchQuery == "" {
		return matches
	}
	query := strings.ToLower(m.searchQuery)
	for i, line := range m.contentLines {
		if strings.Contains(strings.ToLower(line), query) {
			matches = append(matches, i)
		}
	}
	return matches
}
