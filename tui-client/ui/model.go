package ui

import (
	"github.com/charmbracelet/glamour"
	"github.com/vinizap/lumi/tui-client/config"
	"github.com/vinizap/lumi/tui-client/domain"
)

// ViewMode represents the current screen.
type ViewMode int

const (
	ViewHome ViewMode = iota
	ViewTree
	ViewFullNote
)

// VisualModeType represents the type of visual selection.
type VisualModeType int

const (
	VisualNone VisualModeType = iota
	VisualChar               // v - character-wise selection
	VisualLine               // V - line-wise selection
)

// Item represents a folder or note in listings.
type Item struct {
	Name     string
	IsFolder bool
	Path     string
	Note     *domain.Note
}

// Model is the main Bubbletea model for the TUI.
type Model struct {
	// Core state
	rootDir    string
	currentDir string
	items      []Item
	cursor     int
	width      int
	height     int
	viewMode   ViewMode
	renderer   *glamour.TermRenderer

	// Note view state
	fullNote     *domain.Note
	contentLines []string // raw markdown lines
	renderedView string   // glamour-rendered output
	renderedLines []string // rendered output split into lines
	lineCursor   int
	colCursor    int
	desiredCol   int // sticky column for vertical movement (like vim)

	// Visual mode
	visualMode  VisualModeType
	visualStart int // anchor line (or char position for VisualChar)
	visualEnd   int
	visualStartCol int // anchor column for VisualChar
	visualEndCol   int

	// Modals
	showTree  bool
	showSearch bool
	showInput  bool

	// Split view
	splitMode string // "", "horizontal", "vertical"
	splitNote *domain.Note

	// Search state
	searchQuery   string
	searchType    string // "content" or "filename"
	searchResults []Item
	inFileSearch  bool

	// Input modal
	inputMode  string // "create", "rename"
	inputValue string

	// Half-page scroll amount
	scrollAmount int
}

// NewModel creates and returns a new Model.
func NewModel(rootDir string) Model {
	cfg := config.Load()

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	return Model{
		rootDir:    rootDir,
		currentDir: rootDir,
		items:      []Item{},
		viewMode:   ViewHome,
		renderer:   renderer,
		searchType: cfg.SearchType,
	}
}

// NewSimpleModel is an alias for backward compatibility with main.go.
func NewSimpleModel(rootDir string) Model {
	return NewModel(rootDir)
}
