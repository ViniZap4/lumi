package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// animTickMsg drives the home screen animation.
type animTickMsg time.Time

// yankFlashMsg clears the yank highlight after a brief delay.
type yankFlashMsg time.Time

func animTick() tea.Cmd {
	return tea.Tick(60*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadItems, animTick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.fullNote != nil {
			m.renderMarkdown()
		}
		return m, nil

	case yankFlashMsg:
		m.yankHighlight = false
		return m, nil

	case animTickMsg:
		if m.viewMode == ViewHome && !m.animDone {
			m.animCol += 5
			if m.animCol >= logoMaxRunes()+logoStagger() {
				m.animCol = logoMaxRunes() + logoStagger()
				m.animDone = true
				return m, nil
			}
			return m, animTick()
		}
		return m, nil

	case itemsLoadedMsg:
		m.items = msg.items
		return m, nil

	case navItemsLoadedMsg:
		m.navItems = msg.items
		return m, nil

	case searchResultsMsg:
		m.searchResults = msg.results
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		if m.showInput {
			return m.updateInput(msg)
		}

		switch m.viewMode {
		case ViewHome:
			return m.updateHome(msg)
		case ViewTree:
			return m.updateTree(msg)
		case ViewFullNote:
			return m.updateNote(msg)
		case ViewConfig:
			return m.updateConfig(msg)
		}
	}

	return m, nil
}
