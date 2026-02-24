package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/vinizap/lumi/tui-client/config"
	"github.com/vinizap/lumi/tui-client/theme"
)

// enterConfig sets up the config view and transitions to it.
func (m *Model) enterConfig() {
	cfg := config.Load()
	m.configCfg = cfg
	m.configCursor = 1 // skip first header
	m.previousView = m.viewMode
	m.viewMode = ViewConfig
	m.configItems = m.buildConfigItems(cfg)
}

func (m Model) buildConfigItems(cfg *config.Config) []ConfigItem {
	darkNames := theme.ThemeNamesForMode("dark")
	lightNames := theme.ThemeNamesForMode("light")

	editorFull := cfg.Editor
	if len(cfg.EditorArgs) > 0 {
		editorFull += " " + strings.Join(cfg.EditorArgs, " ")
	}

	serverURL := ""
	serverToken := ""
	if m.folderConfig != nil {
		serverURL = m.folderConfig.ServerURL
		serverToken = m.folderConfig.ServerToken
	}
	if serverURL == "" {
		serverURL = "(none)"
	}
	tokenDisplay := "(none)"
	if serverToken != "" {
		tokenDisplay = strings.Repeat("*", len(serverToken))
	}

	return []ConfigItem{
		{Label: "Theme", Kind: ConfigHeader},
		{Label: "Mode", Kind: ConfigCycle, Key: "theme_mode", Value: cfg.ThemeMode, Options: []string{"dark", "light", "auto"}},
		{Label: "Dark theme", Kind: ConfigCycle, Key: "dark_theme", Value: cfg.DarkTheme, Options: darkNames},
		{Label: "Light theme", Kind: ConfigCycle, Key: "light_theme", Value: cfg.LightTheme, Options: lightNames},

		{Label: "Editor", Kind: ConfigHeader},
		{Label: "Command", Kind: ConfigCycle, Key: "editor", Value: editorFull, Options: []string{editorFull}},
		{Label: "Open in editor", Kind: ConfigAction, Key: "open_editor"},

		{Label: "Display", Kind: ConfigHeader},
		{Label: "Line numbers", Kind: ConfigCycle, Key: "show_line_numbers", Value: boolStr(cfg.ShowLineNumbers), Options: []string{"off", "on"}},
		{Label: "Cursor style", Kind: ConfigCycle, Key: "cursor_style", Value: cfg.CursorStyle, Options: []string{"block", "underline", "bar"}},
		{Label: "Preview lines", Kind: ConfigCycle, Key: "preview_lines", Value: fmt.Sprintf("%d", cfg.PreviewLines), Options: []string{"5", "10", "15", "20"}},

		{Label: "Search", Kind: ConfigHeader},
		{Label: "Default type", Kind: ConfigCycle, Key: "default_search_type", Value: cfg.SearchType, Options: []string{"filename", "content"}},

		{Label: "Server (.lumi)", Kind: ConfigHeader},
		{Label: "Server URL", Kind: ConfigInput, Key: "server_url", Value: serverURL},
		{Label: "Server token", Kind: ConfigInput, Key: "server_token", Value: tokenDisplay},
	}
}

func (m Model) updateConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = m.previousView
		return m, nil
	case "q":
		return m, tea.Quit
	case "j", "down":
		m.configMoveDown()
	case "k", "up":
		m.configMoveUp()
	case "l", "right", "enter":
		return m.configCycleForward()
	case "h", "left":
		return m.configCycleBackward()
	}
	return m, nil
}

func (m *Model) configMoveDown() {
	for i := m.configCursor + 1; i < len(m.configItems); i++ {
		if m.configItems[i].Kind != ConfigHeader {
			m.configCursor = i
			return
		}
	}
}

func (m *Model) configMoveUp() {
	for i := m.configCursor - 1; i >= 0; i-- {
		if m.configItems[i].Kind != ConfigHeader {
			m.configCursor = i
			return
		}
	}
}

func (m Model) configCycleForward() (Model, tea.Cmd) {
	if m.configCursor >= len(m.configItems) {
		return m, nil
	}
	item := &m.configItems[m.configCursor]

	switch item.Kind {
	case ConfigCycle:
		item.Value = cycleOption(item.Options, item.Value, 1)
		m.applyConfigChange(item.Key, item.Value)
	case ConfigAction:
		if item.Key == "open_editor" {
			return m, m.openConfig()
		}
	case ConfigInput:
		m.showInput = true
		m.inputMode = "config_" + item.Key
		if item.Value == "(none)" || strings.HasPrefix(item.Value, "***") {
			m.inputValue = ""
		} else {
			m.inputValue = item.Value
		}
		// For server_token, show actual value in input
		if item.Key == "server_token" && m.folderConfig != nil {
			m.inputValue = m.folderConfig.ServerToken
		}
		return m, nil
	}
	return m, nil
}

func (m Model) configCycleBackward() (Model, tea.Cmd) {
	if m.configCursor >= len(m.configItems) {
		return m, nil
	}
	item := &m.configItems[m.configCursor]

	if item.Kind == ConfigCycle {
		item.Value = cycleOption(item.Options, item.Value, -1)
		m.applyConfigChange(item.Key, item.Value)
	}
	return m, nil
}

func (m *Model) applyConfigChange(key, value string) {
	cfg := m.configCfg
	themeChanged := false

	switch key {
	case "theme_mode":
		cfg.ThemeMode = value
		themeChanged = true
		// Rebuild items to update available theme lists
		m.configItems = m.buildConfigItems(cfg)
	case "dark_theme":
		cfg.DarkTheme = value
		themeChanged = true
	case "light_theme":
		cfg.LightTheme = value
		themeChanged = true
	case "show_line_numbers":
		cfg.ShowLineNumbers = value == "on"
	case "cursor_style":
		cfg.CursorStyle = value
	case "preview_lines":
		cfg.PreviewLines = parseInt(value)
	case "default_search_type":
		cfg.SearchType = value
		m.searchType = value
	case "server_url":
		if m.folderConfig != nil {
			if value == "(none)" {
				value = ""
			}
			m.folderConfig.ServerURL = value
			config.SaveFolderConfig(m.rootDir, m.folderConfig)
			m.configItems = m.buildConfigItems(cfg)
		}
		return
	case "server_token":
		if m.folderConfig != nil {
			m.folderConfig.ServerToken = value
			config.SaveFolderConfig(m.rootDir, m.folderConfig)
			m.configItems = m.buildConfigItems(cfg)
		}
		return
	}

	cfg.Save()

	if themeChanged {
		theme.Resolve(cfg.ThemeMode, cfg.DarkTheme, cfg.LightTheme)
		ApplyTheme()
		// Rebuild the glamour renderer with the new theme style
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStylePath(glamourStyle()),
			glamour.WithWordWrap(100),
		)
		if err == nil {
			m.renderer = renderer
		}
	}
}

func cycleOption(options []string, current string, direction int) string {
	if len(options) == 0 {
		return current
	}
	idx := 0
	for i, o := range options {
		if o == current {
			idx = i
			break
		}
	}
	idx += direction
	if idx < 0 {
		idx = len(options) - 1
	}
	if idx >= len(options) {
		idx = 0
	}
	return options[idx]
}

func parseInt(s string) int {
	switch s {
	case "5":
		return 5
	case "10":
		return 10
	case "15":
		return 15
	case "20":
		return 20
	default:
		return 10
	}
}

func boolStr(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

