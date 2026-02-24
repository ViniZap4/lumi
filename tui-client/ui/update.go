package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/filesystem"
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
	cmds := []tea.Cmd{m.loadItems, animTick()}
	if m.syncClient != nil {
		m.syncClient.Start()
		cmds = append(cmds, m.waitForSyncEvent)
	}
	return tea.Batch(cmds...)
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

	case syncEventMsg:
		// A note was changed on the server — reload items.
		// If viewing a note that was updated, re-read it.
		if m.fullNote != nil && msg.event.Type == "note_updated" {
			reloaded, err := filesystem.ReadNote(m.fullNote.Path)
			if err == nil {
				m.openNote(reloaded)
			}
		}
		return m, tea.Batch(m.loadItems, m.waitForSyncEvent)

	case tea.KeyMsg:
		if m.showConfirm {
			return m.updateConfirm(msg)
		}
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
