package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.pendingDeleteNote != nil {
			if err := filesystem.DeleteNote(m.pendingDeleteNote); err == nil {
				m.showConfirm = false
				m.pendingDeleteNote = nil
				m.confirmMsg = ""
				return m, m.loadItems
			}
		}
		m.showConfirm = false
		m.pendingDeleteNote = nil
		m.confirmMsg = ""
		return m, nil
	case "n", "esc":
		m.showConfirm = false
		m.pendingDeleteNote = nil
		m.confirmMsg = ""
		return m, nil
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showInput = false
		m.inputValue = ""
		return m, nil

	case "enter":
		if m.inputValue == "" {
			m.showInput = false
			return m, nil
		}
		switch m.inputMode {
		case "create":
			if _, err := filesystem.CreateNote(m.currentDir, m.inputValue); err == nil {
				m.showInput = false
				m.inputValue = ""
				return m, m.loadItems
			}
		case "rename":
			if m.cursor < len(m.items) && m.items[m.cursor].Note != nil {
				if err := filesystem.RenameNote(m.items[m.cursor].Note, m.inputValue); err == nil {
					m.showInput = false
					m.inputValue = ""
					return m, m.loadItems
				}
			}
		}
		m.showInput = false
		m.inputValue = ""
		return m, nil

	case "backspace":
		if len(m.inputValue) > 0 {
			m.inputValue = m.inputValue[:len(m.inputValue)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
			m.inputValue += msg.String()
		}
		return m, nil
	}
}
