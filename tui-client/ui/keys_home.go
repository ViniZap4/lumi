package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/config"
)

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key during animation skips to the end
	if !m.animDone {
		m.animDone = true
		m.animPos = len(logoFull)
		return m, nil
	}

	// Search modal open
	if m.showSearch {
		return m.updateHomeSearch(msg)
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
	case "c":
		return m, m.openConfig()
	case "t", "enter":
		m.viewMode = ViewTree
		return m, m.loadItems
	}

	return m, nil
}

func (m Model) updateHomeSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			return m, func() tea.Msg { return m.performSearch() }
		}
		return m, nil
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
