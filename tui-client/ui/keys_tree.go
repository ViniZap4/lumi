package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/config"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

func (m Model) updateTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "c":
		return m, m.openConfig()
	case "g":
		m.cursor = 0
	case "G":
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
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

func (m Model) openConfig() tea.Cmd {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "lumi")
	configPath := filepath.Join(configDir, "config.yaml")

	os.MkdirAll(configDir, 0755)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := strings.Join([]string{
			"# Lumi Configuration",
			"",
			"# Editor command with args",
			"editor: nvim",
			"",
			"# Theme (dark or light)",
			"theme: dark",
			"",
			"# TUI Settings",
			"show_line_numbers: false",
			"cursor_style: block",
			"preview_lines: 10",
			"default_search_type: filename",
			"",
		}, "\n")
		os.WriteFile(configPath, []byte(defaultConfig), 0644)
	}

	cfg := config.Load()
	args := append(cfg.EditorArgs, configPath)
	return tea.ExecProcess(exec.Command(cfg.Editor, args...), nil)
}
