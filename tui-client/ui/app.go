// tui-client/ui/app.go
package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
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
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeNewNote
	modeDelete
)

type Model struct {
	rootDir      string
	currentDir   string
	folders      list.Model
	notes        list.Model
	focus        focusPanel
	mode         inputMode
	input        string
	width        int
	height       int
	err          error
}

func NewModel(rootDir string) Model {
	folders := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	folders.Title = "Folders"
	folders.SetShowHelp(false)

	notes := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	notes.Title = "Notes"
	notes.SetShowHelp(false)

	return Model{
		rootDir:    rootDir,
		currentDir: rootDir,
		folders:    folders,
		notes:      notes,
		focus:      focusFolders,
		mode:       modeNormal,
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

		// Normal mode
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.focus == focusFolders {
				m.focus = focusNotes
			} else {
				m.focus = focusFolders
			}
			return m, nil
		case "h":
			if m.focus == focusFolders {
				return m.goUpFolder()
			}
			m.focus = focusFolders
			return m, nil
		case "l":
			if m.focus == focusFolders {
				return m.enterFolder()
			}
			m.focus = focusNotes
			return m, nil
		case "j":
			return m.handleDown()
		case "k":
			return m.handleUp()
		case "g":
			return m.handleTop()
		case "G":
			return m.handleBottom()
		case "enter":
			if m.focus == focusFolders {
				return m.enterFolder()
			}
			return m.editNote()
		case "e":
			return m.editNote()
		case "n":
			m.mode = modeNewNote
			m.input = ""
			return m, nil
		case "d":
			if m.focus == focusNotes && len(m.notes.Items()) > 0 {
				m.mode = modeDelete
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case foldersLoadedMsg:
		items := make([]list.Item, len(msg.folders))
		for i, f := range msg.folders {
			items[i] = folderItem{f}
		}
		m.folders.SetItems(items)
		return m, nil

	case notesLoadedMsg:
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			items[i] = noteItem{n}
		}
		m.notes.SetItems(items)
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
		m.folders.CursorDown()
	} else {
		m.notes.CursorDown()
	}
	return m, nil
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	if m.focus == focusFolders {
		m.folders.CursorUp()
	} else {
		m.notes.CursorUp()
	}
	return m, nil
}

func (m Model) handleTop() (tea.Model, tea.Cmd) {
	if m.focus == focusFolders {
		m.folders.Select(0)
	} else {
		m.notes.Select(0)
	}
	return m, nil
}

func (m Model) handleBottom() (tea.Model, tea.Cmd) {
	if m.focus == focusFolders {
		m.folders.Select(len(m.folders.Items()) - 1)
	} else {
		m.notes.Select(len(m.notes.Items()) - 1)
	}
	return m, nil
}

func (m *Model) updateSizes() {
	panelWidth := m.width/2 - 2
	panelHeight := m.height - 4

	m.folders.SetSize(panelWidth, panelHeight)
	m.notes.SetSize(panelWidth, panelHeight)
}

func (m Model) enterFolder() (tea.Model, tea.Cmd) {
	if len(m.folders.Items()) == 0 {
		return m, nil
	}

	item := m.folders.SelectedItem()
	if item == nil {
		return m, nil
	}

	folder := item.(folderItem).folder
	m.currentDir = folder.Path

	return m, tea.Batch(m.loadFolders, m.loadNotes)
}

func (m Model) goUpFolder() (tea.Model, tea.Cmd) {
	if m.currentDir == m.rootDir {
		return m, nil
	}

	m.currentDir = filepath.Dir(m.currentDir)
	return m, tea.Batch(m.loadFolders, m.loadNotes)
}

func (m Model) editNote() (tea.Model, tea.Cmd) {
	if len(m.notes.Items()) == 0 {
		return m, nil
	}

	item := m.notes.SelectedItem()
	if item == nil {
		return m, nil
	}

	note := item.(noteItem).note
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
	if len(m.notes.Items()) == 0 {
		m.mode = modeNormal
		return m, nil
	}

	item := m.notes.SelectedItem()
	if item == nil {
		m.mode = modeNormal
		return m, nil
	}

	note := item.(noteItem).note
	if err := filesystem.DeleteNote(note.Path); err != nil {
		m.err = err
	}

	m.mode = modeNormal
	return m, tea.Batch(m.loadNotes, m.loadFolders)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	foldersView := m.folders.View()
	notesView := m.notes.View()

	if m.focus == focusFolders {
		foldersView = ActiveStyle.Render(foldersView)
		notesView = InactiveStyle.Render(notesView)
	} else {
		foldersView = InactiveStyle.Render(foldersView)
		notesView = ActiveStyle.Render(notesView)
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top, foldersView, notesView)
	
	currentPath := lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Render(m.currentDir)
	
	var help string
	switch m.mode {
	case modeNewNote:
		help = fmt.Sprintf("New note title: %s█ (enter=create, esc=cancel)", m.input)
	case modeDelete:
		help = "Delete note? (y/n)"
	default:
		help = HelpStyle.Render("q=quit | tab=switch | h/l=folder nav | j/k=move | e/enter=edit | n=new | d=delete | g/G=top/bottom")
	}
	
	return lipgloss.JoinVertical(lipgloss.Left, currentPath, panels, help)
}

type folderItem struct {
	folder *domain.Folder
}

func (f folderItem) FilterValue() string { return f.folder.Name }
func (f folderItem) Title() string       { return f.folder.Name }
func (f folderItem) Description() string { return f.folder.Path }

type noteItem struct {
	note *domain.Note
}

func (n noteItem) FilterValue() string { return n.note.Title }
func (n noteItem) Title() string       { return n.note.Title }
func (n noteItem) Description() string { return n.note.ID }
