package ui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

func (m Model) updateTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search modal open in tree view
	if m.showSearch && !m.inFileSearch {
		return m.updateTreeSearch(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.showSearch = true
		m.searchQuery = ""
		m.searchType = "filename"
		m.inFileSearch = false
		return m, func() tea.Msg { return m.performSearch() }
	case "esc":
		m.viewMode = ViewHome
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
	case "l", "enter":
		if m.cursor < len(m.items) {
			item := m.items[m.cursor]
			if item.IsFolder {
				m.currentDir = item.Path
				m.cursor = 0
				return m, m.loadItems
			} else if item.Note != nil {
				m.viewMode = ViewFullNote
				m.openNote(item.Note)
			}
		}
	case "n":
		m.showInput = true
		m.inputMode = "create"
		m.inputValue = ""
		return m, nil
	case "d":
		if m.cursor < len(m.items) && m.items[m.cursor].Note != nil {
			if err := filesystem.DeleteNote(m.items[m.cursor].Note); err == nil {
				return m, m.loadItems
			}
		}
	case "r":
		if m.cursor < len(m.items) && m.items[m.cursor].Note != nil {
			m.showInput = true
			m.inputMode = "rename"
			m.inputValue = m.items[m.cursor].Note.Title
			return m, nil
		}
	case "D":
		if m.cursor < len(m.items) && m.items[m.cursor].Note != nil {
			if _, err := filesystem.DuplicateNote(m.items[m.cursor].Note); err == nil {
				return m, m.loadItems
			}
		}
	}

	return m, nil
}

func (m Model) updateTreeSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if m.cursor < len(m.searchResults) && m.searchResults[m.cursor].Note != nil {
			m.viewMode = ViewFullNote
			m.openNote(m.searchResults[m.cursor].Note)
			m.showSearch = false
			m.cursor = 0
		}
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
