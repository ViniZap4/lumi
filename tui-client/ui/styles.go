package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

var (
	// Color aliases (updated by ApplyTheme)
	primaryColor   lipgloss.Color
	secondaryColor lipgloss.Color
	accentColor    lipgloss.Color
	mutedColor     lipgloss.Color
	bgColor        lipgloss.Color
	selectedBg     lipgloss.Color

	// Computed highlight colors (updated by ApplyTheme)
	visualSelBg lipgloss.Color // visual selection background (blended)
	yankFlashBg lipgloss.Color // yank flash background (blended)

	// Borders
	ActiveBorder   = lipgloss.RoundedBorder()
	InactiveBorder = lipgloss.Border{}

	// Panel styles
	ActivePanelStyle   lipgloss.Style
	InactivePanelStyle lipgloss.Style

	// Title styles
	TitleStyle lipgloss.Style

	// Item styles
	SelectedItemStyle lipgloss.Style
	NormalItemStyle   lipgloss.Style
	DimItemStyle      lipgloss.Style

	// Help styles
	HelpStyle    lipgloss.Style
	HelpKeyStyle lipgloss.Style

	// Preview styles
	PreviewTitleStyle   lipgloss.Style
	PreviewMetaStyle    lipgloss.Style
	PreviewContentStyle lipgloss.Style
	PreviewLinkStyle    lipgloss.Style

	// Status bar
	StatusBarStyle lipgloss.Style
)

// ApplyTheme rebuilds all package-level style vars from theme.Current.
func ApplyTheme() {
	t := theme.Current

	primaryColor = t.Primary
	secondaryColor = t.Secondary
	accentColor = t.Accent
	mutedColor = t.Muted
	bgColor = t.Background
	selectedBg = t.SelectedBg

	// Compute highlight backgrounds by blending theme colors.
	visualSelBg = blendHex(t.Background, t.Primary, 0.28)
	yankFlashBg = blendHex(t.Background, t.Warning, 0.35)

	ActivePanelStyle = lipgloss.NewStyle().
		Border(ActiveBorder).
		BorderForeground(primaryColor).
		Padding(0, 1)

	InactivePanelStyle = lipgloss.NewStyle().
		Border(InactiveBorder).
		BorderForeground(mutedColor).
		Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Padding(0, 1)

	SelectedItemStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Background(selectedBg).
		Bold(true).
		Padding(0, 1)

	NormalItemStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Padding(0, 1)

	DimItemStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)

	PreviewTitleStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Underline(true).
		MarginBottom(1)

	PreviewMetaStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)

	PreviewContentStyle = lipgloss.NewStyle().
		Foreground(t.Text)

	PreviewLinkStyle = lipgloss.NewStyle().
		Foreground(secondaryColor).
		Underline(true)

	StatusBarStyle = lipgloss.NewStyle().
		Background(t.SelectedBg).
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)
}

// blendHex linearly blends two hex lipgloss.Colors. ratio=0 returns c1, ratio=1 returns c2.
// Falls back to c2 if either color isn't a valid #rrggbb hex string.
func blendHex(c1, c2 lipgloss.Color, ratio float64) lipgloss.Color {
	r1, g1, b1, ok1 := parseHex(string(c1))
	r2, g2, b2, ok2 := parseHex(string(c2))
	if !ok1 || !ok2 {
		return c2
	}
	r := uint8(float64(r1) + ratio*(float64(r2)-float64(r1)))
	g := uint8(float64(g1) + ratio*(float64(g2)-float64(g1)))
	b := uint8(float64(b1) + ratio*(float64(b2)-float64(b1)))
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func parseHex(s string) (uint8, uint8, uint8, bool) {
	if !strings.HasPrefix(s, "#") || len(s) != 7 {
		return 0, 0, 0, false
	}
	r, err1 := strconv.ParseUint(s[1:3], 16, 8)
	g, err2 := strconv.ParseUint(s[3:5], 16, 8)
	b, err3 := strconv.ParseUint(s[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return uint8(r), uint8(g), uint8(b), true
}

func init() {
	// Set defaults before theme is resolved, so styles are never nil
	theme.Current = theme.Theme{
		Primary:    lipgloss.Color("99"),
		Secondary:  lipgloss.Color("141"),
		Accent:     lipgloss.Color("212"),
		Muted:      lipgloss.Color("241"),
		Background: lipgloss.Color("235"),
		SelectedBg: lipgloss.Color("237"),
		OverlayBg:  lipgloss.Color("0"),
		Text:       lipgloss.Color("252"),
		TextDim:    lipgloss.Color("240"),
		Border:     lipgloss.Color("62"),
		Separator:  lipgloss.Color("236"),
		Error:      lipgloss.Color("196"),
		Warning:    lipgloss.Color("226"),
		Info:       lipgloss.Color("81"),
		LogoColors: [6]lipgloss.Color{"99", "105", "111", "141", "147", "183"},
	}
	ApplyTheme()
}
