package ui

import (
	"github.com/charmbracelet/glamour"
	"github.com/vinizap/lumi/tui-client/config"
	"github.com/vinizap/lumi/tui-client/domain"
	"github.com/vinizap/lumi/tui-client/theme"
)

// ViewMode represents the current screen.
type ViewMode int

const (
	ViewHome     ViewMode = iota // animated splash
	ViewTree                     // file browser (3-column)
	ViewFullNote                 // reading a note
	ViewConfig                   // in-app settings
)

// VisualModeType represents the type of visual selection.
type VisualModeType int

const (
	VisualNone VisualModeType = iota
	VisualChar               // v - character-wise selection
	VisualLine               // V - line-wise selection
)

// ConfigItemKind represents the kind of config item.
type ConfigItemKind int

const (
	ConfigHeader ConfigItemKind = iota // section label, not selectable
	ConfigCycle                        // h/l cycles options
	ConfigAction                       // enter triggers action
)

// ConfigItem represents a single row in the config view.
type ConfigItem struct {
	Label   string
	Kind    ConfigItemKind
	Key     string   // config field key
	Value   string
	Options []string // for Cycle kind
}

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

	// Home animation (diagonal left-to-right wipe)
	animCol  int  // how many rune columns have been revealed
	animDone bool // animation finished

	// Note view state
	fullNote      *domain.Note
	contentLines  []string
	renderedView  string
	renderedLines []string
	lineCursor    int
	colCursor     int
	desiredCol    int

	// Visual mode
	visualMode     VisualModeType
	visualStart    int
	visualEnd      int
	visualStartCol int
	visualEndCol   int

	// Modals
	showNav    bool
	showSearch bool
	showInput  bool

	// Navigation modal state (own cursor, dir, items)
	navCursor int
	navDir    string
	navItems  []Item

	// Split view
	splitMode string
	splitNote *domain.Note

	// Search state
	searchQuery   string
	searchType    string
	searchResults []Item
	inFileSearch  bool

	// Input modal
	inputMode  string
	inputValue string

	// Config view state
	configCursor int
	configItems  []ConfigItem
	configCfg    *config.Config
	previousView ViewMode
}

// NewModel creates and returns a new Model.
func NewModel(rootDir string) Model {
	cfg := config.Load()

	// Resolve the theme based on config
	theme.Resolve(cfg.ThemeMode, cfg.DarkTheme, cfg.LightTheme)
	ApplyTheme()

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath(glamourStyle()),
		glamour.WithWordWrap(100),
	)

	return Model{
		rootDir:    rootDir,
		currentDir: rootDir,
		navDir:     rootDir,
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
