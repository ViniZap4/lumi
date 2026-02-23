package ui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/editor"
)

func (m Model) updateNote(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search modal open
	if m.showSearch {
		if m.inFileSearch {
			return m.updateInFileSearch(msg)
		}
		return m.updateNoteGlobalSearch(msg)
	}

	// Tree modal open
	if m.showTree {
		return m.updateTreeModal(msg)
	}

	// Normal note navigation
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
		m.yankSelection()

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
			// In visual mode, 'e' moves to end of word
			m.moveWordEnd()
			m.updateVisualEnd()
		} else {
			// Not in visual mode: open in editor
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

	// Modals
	case "t":
		m.showTree = !m.showTree
		if m.showTree {
			return m, m.loadItems
		}
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
			m.showTree = true
			return m, m.loadItems
		}
	case "S":
		if m.visualMode == VisualNone {
			m.splitMode = "vertical"
			m.showTree = true
			return m, m.loadItems
		}

	// Follow link
	case "enter":
		return m, m.followLinkAtCursor()
	}

	return m, nil
}

func (m Model) updateTreeModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "h":
		if m.currentDir != m.rootDir {
			m.currentDir = filepath.Dir(m.currentDir)
			m.cursor = 0
			return m, m.loadItems
		}
	case "enter", "l":
		if m.cursor < len(m.items) {
			item := m.items[m.cursor]
			if item.IsFolder {
				m.currentDir = item.Path
				m.cursor = 0
				return m, m.loadItems
			} else if item.Note != nil {
				m.openNote(item.Note)
				m.showTree = false
				m.cursor = 0
			}
		}
	case "s":
		if m.cursor < len(m.items) && m.items[m.cursor].Note != nil {
			m.splitMode = "horizontal"
			m.splitNote = m.items[m.cursor].Note
			m.showTree = false
			m.cursor = 0
		}
	case "S":
		if m.cursor < len(m.items) && m.items[m.cursor].Note != nil {
			m.splitMode = "vertical"
			m.splitNote = m.items[m.cursor].Note
			m.showTree = false
			m.cursor = 0
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
	case "j", "down":
		if m.cursor < len(m.searchResults)-1 {
			m.cursor++
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "h", "l":
		return m, nil
	case "enter":
		if m.cursor < len(m.searchResults) && m.searchResults[m.cursor].Note != nil {
			m.openNote(m.searchResults[m.cursor].Note)
			m.showSearch = false
			m.cursor = 0
		}
		return m, nil
	case "s":
		if m.cursor < len(m.searchResults) && m.searchResults[m.cursor].Note != nil {
			m.splitMode = "horizontal"
			m.splitNote = m.searchResults[m.cursor].Note
			m.showSearch = false
			m.cursor = 0
		}
		return m, nil
	case "S":
		if m.cursor < len(m.searchResults) && m.searchResults[m.cursor].Note != nil {
			m.splitMode = "vertical"
			m.splitNote = m.searchResults[m.cursor].Note
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
	case "j", "down":
		matches := m.findInFileMatches()
		if m.cursor < len(matches)-1 {
			m.cursor++
		}
	case "k", "up":
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
