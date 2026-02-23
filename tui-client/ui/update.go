package ui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) Init() tea.Cmd {
	return m.loadItems
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Re-render markdown with new width
		if m.fullNote != nil {
			m.renderMarkdown()
		}
		return m, nil

	case itemsLoadedMsg:
		m.items = msg.items
		return m, nil

	case searchResultsMsg:
		m.searchResults = msg.results
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		// Input modal has highest priority
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
		}
	}

	return m, nil
}
