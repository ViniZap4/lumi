package ui

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Input modal overlay (highest priority)
	if m.showInput {
		return m.renderWithInputModal(m.renderBase())
	}

	// Global search modal overlay
	if m.showSearch && !m.inFileSearch {
		return m.renderWithSearchModal(m.renderBase())
	}

	switch m.viewMode {
	case ViewHome:
		return m.renderHome()
	case ViewFullNote:
		if m.showSearch && m.inFileSearch {
			return m.renderWithInFileSearch()
		}
		if m.showTree {
			return m.renderWithTreeModal(m.renderFullNote())
		}
		if m.splitMode != "" && m.splitNote != nil {
			return m.renderSplitView()
		}
		return m.renderFullNote()
	default:
		return m.renderTreeYazi()
	}
}

// renderBase returns the base view for the current mode (used under modals).
func (m Model) renderBase() string {
	switch m.viewMode {
	case ViewHome:
		return m.renderHome()
	case ViewFullNote:
		return m.renderFullNote()
	default:
		return m.renderTreeYazi()
	}
}
