// tui-client/ui/app.go
package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/domain"
	"github.com/vinizap/lumi/tui-client/editor"
	"github.com/vinizap/lumi/tui-client/filesystem"
)

type focusPanel int

const (
	focusFolders focusPanel = iota
	focusNotes
	focusPreview
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeNewNote
	modeDelete
	modeLinks
)

type Model struct {
	rootDir       string
	currentDir    string
	folders       []folderItem
	notes         []noteItem
	folderCursor  int
	noteCursor    int
	previewScroll int
	linkCursor    int
	focus         focusPanel
	mode          inputMode
	previewMode   PreviewMode
	input         string
	width         int
	height        int
	err           error
	links         []string
}

func NewModel(rootDir string) Model {
	return Model{
		rootDir:     rootDir,
		currentDir:  rootDir,
		folders:     []folderItem{},
		notes:       []noteItem{},
		focus:       focusFolders,
		mode:        modeNormal,
		previewMode: PreviewPartial,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadFolders,
		m.loadNotes,
	)
}

func (m Model) loadFolders() tea.Msg {
	folders, err := filesystem.ListFolders(m.currentDir)
	if err != nil {
		return errMsg{err}
	}
	return foldersLoadedMsg{folders}
}

func (m Model) loadNotes() tea.Msg {
	notes, err := filesystem.ListNotes(m.currentDir)
	if err != nil {
		return errMsg{err}
	}
	return notesLoadedMsg{notes}
}

type foldersLoadedMsg struct {
	folders []*domain.Folder
}

type notesLoadedMsg struct {
	notes []*domain.Note
}

type errMsg struct {
	err error
}

type editorFinishedMsg struct{}

func openEditorCmd(note *domain.Note) tea.Cmd {
	return tea.ExecProcess(editor.OpenCmd(note), func(err error) tea.Msg {
		if err != nil {
			return errMsg{err}
		}
		return editorFinishedMsg{}
	})
}

type folderItem struct {
	folder *domain.Folder
}

type noteItem struct {
	note *domain.Note
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input modes
		if m.mode == modeNewNote {
			switch msg.String() {
			case "enter":
				return m.createNote()
			case "esc":
				m.mode = modeNormal
				m.input = ""
				return m, nil
			case "backspace":
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.input += msg.String()
				}
				return m, nil
			}
		}

		if m.mode == modeDelete {
			switch msg.String() {
			case "y":
				return m.deleteNote()
			case "n", "esc":
				m.mode = modeNormal
				return m, nil
			}
			return m, nil
		}

		if m.mode == modeLinks {
			switch msg.String() {
			case "j":
				if m.linkCursor < len(m.links)-1 {
					m.linkCursor++
				}
				return m, nil
			case "k":
				if m.linkCursor > 0 {
					m.linkCursor--
				}
				return m, nil
			case "enter":
				return m.followLink()
			case "esc", "q":
				m.mode = modeNormal
				return m, nil
			}
			return m, nil
		}

		// Normal mode
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "h":
			// Vim motion: move left (folders or up directory)
			if m.focus == focusNotes {
				m.focus = focusFolders
			} else if m.focus == focusPreview {
				m.focus = focusNotes
			} else if m.focus == focusFolders {
				return m.goUpFolder()
			}
			return m, nil
		case "l":
			// Vim motion: move right (notes or preview or enter folder)
			if m.focus == focusFolders {
				if len(m.folders) > 0 {
					return m.enterFolder()
				}
				m.focus = focusNotes
			} else if m.focus == focusNotes {
				if m.previewMode != PreviewOff {
					m.focus = focusPreview
					m.updateLinks()
				}
			}
			return m, nil
		case "j":
			return m.handleDown()
		case "k":
			return m.handleUp()
		case "g":
			return m.handleTop()
		case "G":
			return m.handleBottom()
		case "v":
			// Toggle preview mode
			switch m.previewMode {
			case PreviewOff:
				m.previewMode = PreviewPartial
			case PreviewPartial:
				m.previewMode = PreviewFull
			case PreviewFull:
				m.previewMode = PreviewOff
				if m.focus == focusPreview {
					m.focus = focusNotes
				}
			}
			m.previewScroll = 0
			return m, nil
		case "enter":
			if m.focus == focusFolders {
				return m.enterFolder()
			} else if m.focus == focusPreview {
				return m.followLinkAtCursor()
			}
			return m.editNote()
		case "e":
			return m.editNote()
		case "n":
			m.mode = modeNewNote
			m.input = ""
			return m, nil
		case "d":
			if m.focus == focusNotes && len(m.notes) > 0 {
				m.mode = modeDelete
			}
			return m, nil
		case "L":
			// Show links modal
			m.updateLinks()
			if len(m.links) > 0 {
				m.mode = modeLinks
				m.linkCursor = 0
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case foldersLoadedMsg:
		m.folders = make([]folderItem, len(msg.folders))
		for i, f := range msg.folders {
			m.folders[i] = folderItem{f}
		}
		return m, nil

	case notesLoadedMsg:
		m.notes = make([]noteItem, len(msg.notes))
		for i, n := range msg.notes {
			m.notes[i] = noteItem{n}
		}
		m.previewScroll = 0
		return m, nil

	case editorFinishedMsg:
		return m, tea.Batch(m.loadNotes, m.loadFolders)

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m Model) handleDown() (tea.Model, tea.Cmd) {
	if m.focus == focusFolders {
		if m.folderCursor < len(m.folders)-1 {
			m.folderCursor++
		}
	} else if m.focus == focusNotes {
		if m.noteCursor < len(m.notes)-1 {
			m.noteCursor++
		}
	} else if m.focus == focusPreview {
		m.previewScroll++
	}
	return m, nil
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	if m.focus == focusFolders {
		if m.folderCursor > 0 {
			m.folderCursor--
		}
	} else if m.focus == focusNotes {
		if m.noteCursor > 0 {
			m.noteCursor--
		}
	} else if m.focus == focusPreview {
		if m.previewScroll > 0 {
			m.previewScroll--
		}
	}
	return m, nil
}

func (m Model) handleTop() (tea.Model, tea.Cmd) {
	if m.focus == focusFolders {
		m.folderCursor = 0
	} else if m.focus == focusNotes {
		m.noteCursor = 0
	} else if m.focus == focusPreview {
		m.previewScroll = 0
	}
	return m, nil
}

func (m Model) handleBottom() (tea.Model, tea.Cmd) {
	if m.focus == focusFolders {
		if len(m.folders) > 0 {
			m.folderCursor = len(m.folders) - 1
		}
	} else if m.focus == focusNotes {
		if len(m.notes) > 0 {
			m.noteCursor = len(m.notes) - 1
		}
	} else if m.focus == focusPreview {
		m.previewScroll = 9999 // Will be clamped in render
	}
	return m, nil
}

func (m *Model) updateSizes() {
	// No longer needed with custom rendering
}

func (m Model) enterFolder() (tea.Model, tea.Cmd) {
	if len(m.folders) == 0 {
		return m, nil
	}

	if m.folderCursor >= len(m.folders) {
		return m, nil
	}

	folder := m.folders[m.folderCursor].folder
	m.currentDir = folder.Path
	m.folderCursor = 0
	m.noteCursor = 0
	m.previewScroll = 0

	return m, tea.Batch(m.loadFolders, m.loadNotes)
}

func (m Model) goUpFolder() (tea.Model, tea.Cmd) {
	if m.currentDir == m.rootDir {
		return m, nil
	}

	m.currentDir = filepath.Dir(m.currentDir)
	m.folderCursor = 0
	m.noteCursor = 0
	m.previewScroll = 0
	return m, tea.Batch(m.loadFolders, m.loadNotes)
}

func (m Model) editNote() (tea.Model, tea.Cmd) {
	if len(m.notes) == 0 {
		return m, nil
	}

	if m.noteCursor >= len(m.notes) {
		return m, nil
	}

	note := m.notes[m.noteCursor].note
	return m, openEditorCmd(note)
}

func (m Model) createNote() (tea.Model, tea.Cmd) {
	if m.input == "" {
		m.mode = modeNormal
		return m, nil
	}

	id := time.Now().Format("2006-01-02-") + m.input
	note, err := filesystem.CreateNote(m.currentDir, id, m.input)
	if err != nil {
		m.err = err
		m.mode = modeNormal
		m.input = ""
		return m, nil
	}

	m.mode = modeNormal
	m.input = ""

	return m, tea.Sequence(
		tea.Batch(m.loadNotes, m.loadFolders),
		openEditorCmd(note),
	)
}

func (m Model) deleteNote() (tea.Model, tea.Cmd) {
	if len(m.notes) == 0 {
		m.mode = modeNormal
		return m, nil
	}

	if m.noteCursor >= len(m.notes) {
		m.mode = modeNormal
		return m, nil
	}

	note := m.notes[m.noteCursor].note
	if err := filesystem.DeleteNote(note.Path); err != nil {
		m.err = err
	}

	m.mode = modeNormal
	if m.noteCursor > 0 {
		m.noteCursor--
	}
	return m, tea.Batch(m.loadNotes, m.loadFolders)
}

func (m *Model) updateLinks() {
	if len(m.notes) == 0 || m.noteCursor >= len(m.notes) {
		m.links = []string{}
		return
	}
	note := m.notes[m.noteCursor].note
	m.links = extractLinks(note.Content)
}

func (m Model) followLink() (tea.Model, tea.Cmd) {
	if m.linkCursor >= len(m.links) {
		m.mode = modeNormal
		return m, nil
	}

	link := m.links[m.linkCursor]
	m.mode = modeNormal

	// Try to find note by ID
	for i, item := range m.notes {
		if item.note.ID == link || strings.Contains(item.note.Path, link) {
			m.noteCursor = i
			m.previewScroll = 0
			return m, nil
		}
	}

	return m, nil
}

func (m Model) followLinkAtCursor() (tea.Model, tea.Cmd) {
	m.updateLinks()
	if len(m.links) == 0 {
		return m.editNote()
	}

	// For now, just open links modal
	m.mode = modeLinks
	m.linkCursor = 0
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate dimensions
	var foldersWidth, notesWidth, previewWidth int
	if m.previewMode == PreviewOff {
		foldersWidth = m.width / 3
		notesWidth = m.width - foldersWidth - 4
		previewWidth = 0
	} else {
		foldersWidth = m.width / 5
		notesWidth = m.width / 4
		previewWidth = m.width - foldersWidth - notesWidth - 6
	}

	panelHeight := m.height - 4

	// Render folders panel
	foldersView := m.renderFolders(foldersWidth, panelHeight)
	if m.focus == focusFolders {
		foldersView = ActivePanelStyle.Width(foldersWidth).Height(panelHeight).Render(foldersView)
	} else {
		foldersView = InactivePanelStyle.Width(foldersWidth).Height(panelHeight).Render(foldersView)
	}

	// Render notes panel
	notesView := m.renderNotes(notesWidth, panelHeight)
	if m.focus == focusNotes {
		notesView = ActivePanelStyle.Width(notesWidth).Height(panelHeight).Render(notesView)
	} else {
		notesView = InactivePanelStyle.Width(notesWidth).Height(panelHeight).Render(notesView)
	}

	// Render preview panel
	var panels string
	if m.previewMode != PreviewOff {
		var selectedNote *domain.Note
		if len(m.notes) > 0 && m.noteCursor < len(m.notes) {
			selectedNote = m.notes[m.noteCursor].note
		}
		previewView := m.renderPreview(selectedNote, previewWidth, panelHeight)
		if m.focus == focusPreview {
			previewView = ActivePanelStyle.Width(previewWidth).Height(panelHeight).Render(previewView)
		} else {
			previewView = InactivePanelStyle.Width(previewWidth).Height(panelHeight).Render(previewView)
		}
		panels = lipgloss.JoinHorizontal(lipgloss.Top, foldersView, notesView, previewView)
	} else {
		panels = lipgloss.JoinHorizontal(lipgloss.Top, foldersView, notesView)
	}

	// Status bar
	statusLeft := StatusBarStyle.Render(m.currentDir)
	var statusRight string
	switch m.previewMode {
	case PreviewOff:
		statusRight = DimItemStyle.Render("Preview: off")
	case PreviewPartial:
		statusRight = DimItemStyle.Render("Preview: partial")
	case PreviewFull:
		statusRight = DimItemStyle.Render("Preview: full")
	}
	statusBar := lipgloss.JoinHorizontal(
		lipgloss.Top,
		statusLeft,
		strings.Repeat(" ", m.width-lipgloss.Width(statusLeft)-lipgloss.Width(statusRight)),
		statusRight,
	)

	// Help bar
	var help string
	switch m.mode {
	case modeNewNote:
		help = fmt.Sprintf("New note title: %s█ (enter=create, esc=cancel)", m.input)
	case modeDelete:
		help = "Delete note? (y/n)"
	case modeLinks:
		help = "Links: " + HelpKeyStyle.Render("j/k") + "=navigate | " + HelpKeyStyle.Render("enter") + "=follow | " + HelpKeyStyle.Render("esc") + "=close"
	default:
		helpKeys := []string{
			HelpKeyStyle.Render("q") + "=quit",
			HelpKeyStyle.Render("h/l") + "=navigate",
			HelpKeyStyle.Render("j/k") + "=move",
			HelpKeyStyle.Render("e") + "=edit",
			HelpKeyStyle.Render("n") + "=new",
			HelpKeyStyle.Render("d") + "=delete",
			HelpKeyStyle.Render("v") + "=preview",
			HelpKeyStyle.Render("L") + "=links",
		}
		help = HelpStyle.Render(strings.Join(helpKeys, " | "))
	}

	// Show links modal if active
	view := lipgloss.JoinVertical(lipgloss.Left, statusBar, panels, help)
	if m.mode == modeLinks {
		modal := m.renderLinksModal()
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "))
	}

	return view
}

func (m Model) renderFolders(width, height int) string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("📁 Folders"))
	content.WriteString("\n\n")

	if len(m.folders) == 0 {
		content.WriteString(DimItemStyle.Render("No folders"))
		return content.String()
	}

	start := max(0, m.folderCursor-height+5)
	end := min(len(m.folders), start+height-3)

	for i := start; i < end; i++ {
		folder := m.folders[i].folder
		line := folder.Name

		if i == m.folderCursor {
			line = SelectedItemStyle.Render("▸ " + line)
		} else {
			line = NormalItemStyle.Render("  " + line)
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	return content.String()
}

func (m Model) renderNotes(width, height int) string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("📝 Notes"))
	content.WriteString("\n\n")

	if len(m.notes) == 0 {
		content.WriteString(DimItemStyle.Render("No notes"))
		return content.String()
	}

	start := max(0, m.noteCursor-height+5)
	end := min(len(m.notes), start+height-3)

	for i := start; i < end; i++ {
		note := m.notes[i].note
		line := note.Title

		if i == m.noteCursor {
			line = SelectedItemStyle.Render("▸ " + line)
			if len(note.Tags) > 0 {
				line += "\n" + DimItemStyle.Render("  "+strings.Join(note.Tags, ", "))
			}
		} else {
			line = NormalItemStyle.Render("  " + line)
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	return content.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) renderPreview(note *domain.Note, width, height int) string {
	if note == nil {
		return DimItemStyle.Render("No note selected")
	}

	var content strings.Builder

	// Title
	content.WriteString(PreviewTitleStyle.Render(note.Title))
	content.WriteString("\n\n")

	// Metadata
	meta := PreviewMetaStyle.Render(
		"ID: " + note.ID + " | " +
			"Created: " + note.CreatedAt.Format("2006-01-02") + " | " +
			"Tags: " + strings.Join(note.Tags, ", "),
	)
	content.WriteString(meta)
	content.WriteString("\n\n")

	// Content with scroll
	noteContent := note.Content
	lines := strings.Split(noteContent, "\n")

	// Apply scroll
	if m.previewScroll > len(lines) {
		m.previewScroll = max(0, len(lines)-1)
	}

	visibleLines := height - 8
	start := m.previewScroll
	end := min(len(lines), start+visibleLines)

	if start < len(lines) {
		visibleContent := strings.Join(lines[start:end], "\n")
		visibleContent = highlightLinks(visibleContent)
		content.WriteString(PreviewContentStyle.Width(width - 4).Render(visibleContent))

		if end < len(lines) {
			content.WriteString("\n" + DimItemStyle.Render("... (more below)"))
		}
	}

	// Show scroll indicator
	if m.focus == focusPreview {
		scrollInfo := fmt.Sprintf("\n\n%s Line %d/%d", DimItemStyle.Render("↕"), m.previewScroll+1, len(lines))
		content.WriteString(scrollInfo)
	}

	return content.String()
}

func (m Model) renderLinksModal() string {
	modalWidth := min(60, m.width-4)
	modalHeight := min(20, m.height-4)

	var content strings.Builder
	content.WriteString(TitleStyle.Render("🔗 Links in Note"))
	content.WriteString("\n\n")

	if len(m.links) == 0 {
		content.WriteString(DimItemStyle.Render("No links found"))
	} else {
		for i, link := range m.links {
			if i == m.linkCursor {
				content.WriteString(SelectedItemStyle.Render("▸ " + link))
			} else {
				content.WriteString(NormalItemStyle.Render("  " + link))
			}
			content.WriteString("\n")
		}
	}

	modal := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		Background(bgColor).
		Render(content.String())

	return modal
}
