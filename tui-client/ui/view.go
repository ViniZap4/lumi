package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	// Confirm modal overlay (highest priority)
	if m.showConfirm {
		content = m.renderWithConfirmModal(m.renderBase())
	} else if m.showInput {
		content = m.renderWithInputModal(m.renderBase())
	} else if m.showSearch && !m.inFileSearch {
		// Global search modal overlay
		content = m.renderWithSearchModal(m.renderBase())
	} else {
		switch m.viewMode {
		case ViewHome:
			content = m.renderHome()
		case ViewConfig:
			content = m.renderConfig()
		case ViewFullNote:
			if m.showSearch && m.inFileSearch {
				content = m.renderWithInFileSearch()
			} else if m.showNav {
				content = m.renderWithNavModal(m.renderFullNote())
			} else if m.splitMode != "" && m.splitNote != nil {
				content = m.renderSplitView()
			} else {
				content = m.renderFullNote()
			}
		default:
			content = m.renderTree()
		}
	}

	return m.fillBg(content)
}

// fillBg paints the entire terminal viewport with the theme background color.
//
// Strategy:
//  1. After every SGR full-reset in the content, re-emit the background colour
//     escape so it is never accidentally cleared by a foreground-only style.
//  2. For each line, measure its *visual* width (ANSI stripped), then pad with
//     background-coloured spaces to m.width — without using lipgloss Width()
//     rendering, which would truncate lines that are slightly wider than m.width.
//  3. Fill remaining rows to m.height so the bottom of the screen is covered.
func (m Model) fillBg(content string) string {
	bg := theme.Current.Background
	bgANSI := colorToAnsiBg(bg)
	if bgANSI == "" {
		// No true-color / 256-color bg available; nothing to do.
		return content
	}

	const sgr0 = "\x1b[0m"
	const sgrM = "\x1b[m"

	// Step 1 – make background sticky after every SGR reset.
	content = strings.ReplaceAll(content, sgr0, sgr0+bgANSI)
	content = strings.ReplaceAll(content, sgrM, sgrM+bgANSI)

	// Step 2 – pad each line to m.width (never truncate).
	reset := sgrM
	empty := bgANSI + strings.Repeat(" ", m.width) + reset

	lines := strings.Split(content, "\n")
	for i, l := range lines {
		vw := lipgloss.Width(l) // visual width, ANSI-stripped
		pad := m.width - vw
		if pad < 0 {
			pad = 0
		}
		lines[i] = bgANSI + l + strings.Repeat(" ", pad) + reset
	}

	// Step 3 – fill remaining height.
	for len(lines) < m.height {
		lines = append(lines, empty)
	}
	return strings.Join(lines[:m.height], "\n")
}

// colorToAnsiBg converts a lipgloss.Color to an ANSI SGR background escape.
// Supports hex (#rrggbb) and 256-colour palette (numeric string).
func colorToAnsiBg(c lipgloss.Color) string {
	s := string(c)
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		r, _ := strconv.ParseInt(s[1:3], 16, 64)
		g, _ := strconv.ParseInt(s[3:5], 16, 64)
		b, _ := strconv.ParseInt(s[5:7], 16, 64)
		return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	}
	n, err := strconv.Atoi(s)
	if err == nil {
		return fmt.Sprintf("\x1b[48;5;%dm", n)
	}
	return ""
}

func (m Model) renderBase() string {
	switch m.viewMode {
	case ViewHome:
		return m.renderHome()
	case ViewConfig:
		return m.renderConfig()
	case ViewFullNote:
		return m.renderFullNote()
	default:
		return m.renderTree()
	}
}
