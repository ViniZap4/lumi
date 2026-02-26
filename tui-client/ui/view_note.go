package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/image"
	"github.com/vinizap/lumi/tui-client/theme"
)

// visualRange describes which columns of a line fall inside the visual selection.
type visualRange struct {
	active   bool
	full     bool // entire line is selected (VisualLine or middle line in VisualChar)
	startCol int  // first selected column (0-based)
	endCol   int  // last selected column, or -1 for "to end of line"
}

func (m Model) renderFullNote() string {
	if m.fullNote == nil {
		return "No note loaded"
	}

	var s strings.Builder

	// --- Header: title + tags (left) + date (right) + separator ---
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(" " + m.fullNote.Title)

	var tagStr string
	if len(m.fullNote.Tags) > 0 {
		var tagParts []string
		for _, tag := range m.fullNote.Tags {
			tagParts = append(tagParts, lipgloss.NewStyle().
				Foreground(theme.Current.Accent).
				Render("#"+tag))
		}
		tagStr = "  " + strings.Join(tagParts, " ")
	}

	dateStyled := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(m.fullNote.UpdatedAt.Format("Jan 2, 2006") + " ")
	tw := lipgloss.Width(titleStyled) + lipgloss.Width(tagStr)
	dw := lipgloss.Width(dateStyled)
	gap := m.width - tw - dw
	if gap < 1 {
		gap = 1
	}
	s.WriteString(titleStyled + tagStr + strings.Repeat(" ", gap) + dateStyled)
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().
		Foreground(theme.Current.Separator).
		Render(strings.Repeat("─", m.width)))
	s.WriteString("\n")

	// --- Content ---
	rawLines := m.contentLines
	displayLines := make([]string, 0, len(rawLines))
	rawToDisplay := make([]int, len(rawLines))

	for i, line := range rawLines {
		rawToDisplay[i] = len(displayLines)
		if image.HasImage(line) {
			imgPath := image.GetImagePath(line, m.fullNote.Path)
			if imgPath != "" {
				if _, err := os.Stat(imgPath); err == nil {
					rendered := image.Render(imgPath, m.width-6)
					displayLines = append(displayLines, strings.Split(rendered, "\n")...)
					continue
				}
			}
			displayLines = append(displayLines, lipgloss.NewStyle().
				Foreground(theme.Current.Error).
				Render(fmt.Sprintf("[Image not found: %s]", filepath.Base(image.ExtractImagePath(line)))))
		} else {
			displayLines = append(displayLines, line)
		}
	}

	codeLines := codeBlockLines(rawLines)
	tableCtx := buildTableLineCtx(rawLines)

	maxLines := m.viewportHeight()
	totalLines := len(displayLines)

	displayCursor := 0
	if m.lineCursor >= 0 && m.lineCursor < len(rawToDisplay) {
		displayCursor = rawToDisplay[m.lineCursor]
	}

	start := displayCursor - maxLines/2
	if start < 0 {
		start = 0
	}
	if start > totalLines-maxLines {
		start = max(0, totalLines-maxLines)
	}
	end := min(start+maxLines, totalLines)

	displayToRaw := func(d int) int {
		raw := 0
		for r, disp := range rawToDisplay {
			if disp <= d {
				raw = r
			}
		}
		return raw
	}

	// Render visible lines
	for i := start; i < end; i++ {
		line := ""
		if i < len(displayLines) {
			line = displayLines[i]
		}

		rawIdx := displayToRaw(i)
		inCode := codeLines[rawIdx]
		style := mdLineStyle(line, inCode)
		vr := m.visualRangeForLine(rawIdx)
		isCursorLine := (i == displayCursor)

		// Merge visual selection and yank flash ranges
		yr := m.yankRangeForLine(rawIdx)
		activeRange := vr
		selBg := visualSelBg
		if yr.active {
			activeRange = yr
			selBg = yankFlashBg
		}

		var inlineCls []int
		if shouldClassifyInline(line, inCode) {
			if tctx, ok := tableCtx[rawIdx]; ok {
				inlineCls = classifyInlineWithCtx(line, tctx)
			} else {
				inlineCls = classifyInline(line)
			}
		}
		styledLine := m.renderContentLine(line, style, inlineCls, activeRange, selBg, isCursorLine)

		// Pad code block lines with background to full width
		if inCode && !isCursorLine && !activeRange.active {
			visWidth := lipgloss.Width(styledLine)
			pad := m.width - 2 - visWidth
			if pad > 0 {
				styledLine += lipgloss.NewStyle().Background(theme.Current.SelectedBg).Render(strings.Repeat(" ", pad))
			}
		}

		// Pad visual-selected lines to full width so the highlight spans the entire row.
		if activeRange.active && activeRange.full && !isCursorLine {
			visWidth := lipgloss.Width(styledLine)
			pad := m.width - 2 - visWidth // 2 for prefix
			if pad > 0 {
				styledLine += lipgloss.NewStyle().Background(selBg).Render(strings.Repeat(" ", pad))
			}
		}

		if isCursorLine {
			prefix := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Render("> ")
			line = prefix + styledLine
		} else if activeRange.active && activeRange.full {
			prefix := lipgloss.NewStyle().Background(selBg).Render("  ")
			line = prefix + styledLine
		} else {
			line = "  " + styledLine
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	// --- Footer: separator + status bar + help ---
	s.WriteString(lipgloss.NewStyle().
		Foreground(theme.Current.Separator).
		Render(strings.Repeat("─", m.width)))
	s.WriteString("\n")

	if m.statusMsg != "" {
		s.WriteString(StatusBarStyle.Width(m.width).Render(" " + m.statusMsg))
	} else {
		mode := m.modeIndicator()
		status := fmt.Sprintf("Ln %d  Col %d%s", m.lineCursor+1, m.colCursor+1, mode)
		s.WriteString(StatusBarStyle.Width(m.width).Render(status))
	}
	s.WriteString("\n")

	helpKeys := []struct{ key, desc string }{
		{"j/k", "move"},
		{"h/l", "cols"},
		{"w/b", "word"},
		{"g/G", "top/end"},
		{"v/V", "visual"},
		{"y", "yank"},
		{"e", "edit"},
		{"t", "tree"},
		{"x", "open url"},
		{"/", "search"},
		{"esc", "back"},
	}
	var parts []string
	for _, k := range helpKeys {
		key := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true).Render(k.key)
		desc := lipgloss.NewStyle().Foreground(mutedColor).Render(" " + k.desc)
		parts = append(parts, key+desc)
	}
	s.WriteString(lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(parts, "  ")))

	return s.String()
}

// --- Helpers ---

// codeBlockLines returns which raw line indices are inside fenced code blocks.
func codeBlockLines(lines []string) map[int]bool {
	result := map[int]bool{}
	inside := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			result[i] = true
			inside = !inside
			continue
		}
		if inside {
			result[i] = true
		}
	}
	return result
}

// mdLineStyle returns the theme-aware style for a markdown line.
func mdLineStyle(line string, inCodeBlock bool) lipgloss.Style {
	t := theme.Current
	if inCodeBlock {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			// Fence lines: dim with background
			return lipgloss.NewStyle().Foreground(t.Muted).Background(t.SelectedBg)
		}
		// Code content: accent color with background
		return lipgloss.NewStyle().Foreground(t.Accent).Background(t.SelectedBg)
	}
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "# "):
		return lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	case strings.HasPrefix(trimmed, "## "):
		return lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	case strings.HasPrefix(trimmed, "### "),
		strings.HasPrefix(trimmed, "#### "),
		strings.HasPrefix(trimmed, "##### "),
		strings.HasPrefix(trimmed, "###### "):
		return lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	case strings.HasPrefix(trimmed, "> "):
		return lipgloss.NewStyle().Italic(true).Foreground(mutedColor)
	case trimmed == "---" || trimmed == "***" || trimmed == "___":
		return lipgloss.NewStyle().Foreground(t.Separator)
	case isTableLine(trimmed):
		return lipgloss.NewStyle().Foreground(t.Text)
	default:
		return lipgloss.NewStyle().Foreground(t.Text)
	}
}

// isTableLine detects markdown table rows (e.g. "| a | b |" or "| --- | --- |").
func isTableLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") && len(trimmed) > 2
}

// visualRangeForLine computes the column range selected on a given raw line.
func (m Model) visualRangeForLine(rawLine int) visualRange {
	if m.visualMode == VisualNone {
		return visualRange{}
	}

	sLine, sCol := m.visualStart, m.visualStartCol
	eLine, eCol := m.visualEnd, m.visualEndCol

	// Normalize so s is before e.
	if sLine > eLine || (sLine == eLine && sCol > eCol) {
		sLine, eLine = eLine, sLine
		sCol, eCol = eCol, sCol
	}

	if rawLine < sLine || rawLine > eLine {
		return visualRange{}
	}

	if m.visualMode == VisualLine {
		return visualRange{active: true, full: true, startCol: 0, endCol: -1}
	}

	// VisualChar
	if sLine == eLine {
		return visualRange{active: true, startCol: sCol, endCol: eCol}
	}
	if rawLine == sLine {
		return visualRange{active: true, startCol: sCol, endCol: -1}
	}
	if rawLine == eLine {
		return visualRange{active: true, startCol: 0, endCol: eCol}
	}
	// Middle line: fully selected.
	return visualRange{active: true, full: true, startCol: 0, endCol: -1}
}

// yankRangeForLine computes the column range for the yank flash highlight on a given raw line.
func (m Model) yankRangeForLine(rawLine int) visualRange {
	if !m.yankHighlight {
		return visualRange{}
	}

	sLine, sCol := m.yankStartLine, m.yankStartCol
	eLine, eCol := m.yankEndLine, m.yankEndCol

	if sLine > eLine || (sLine == eLine && sCol > eCol) {
		sLine, eLine = eLine, sLine
		sCol, eCol = eCol, sCol
	}

	if rawLine < sLine || rawLine > eLine {
		return visualRange{}
	}

	if m.yankMode == VisualLine {
		return visualRange{active: true, full: true, startCol: 0, endCol: -1}
	}

	// VisualChar
	if sLine == eLine {
		return visualRange{active: true, startCol: sCol, endCol: eCol}
	}
	if rawLine == sLine {
		return visualRange{active: true, startCol: sCol, endCol: -1}
	}
	if rawLine == eLine {
		return visualRange{active: true, startCol: 0, endCol: eCol}
	}
	return visualRange{active: true, full: true, startCol: 0, endCol: -1}
}

// renderContentLine renders a single content line with the correct combination
// of inline markdown highlighting, visual-selection background, and block cursor.
// selBg is the background color for selected/highlighted regions.
// It batches consecutive characters that share the same (zone, inlineClass) pair
// into segments so the output stays compact.
func (m Model) renderContentLine(line string, baseStyle lipgloss.Style, inlineCls []int, vr visualRange, selBg lipgloss.Color, isCursorLine bool) string {
	runes := []rune(line)

	// Empty line with cursor: show a visible block.
	if len(runes) == 0 {
		if isCursorLine {
			return lipgloss.NewStyle().
				Background(primaryColor).
				Foreground(theme.Current.Background).
				Render(" ")
		}
		return ""
	}

	// Check if any inline class is non-normal.
	hasInline := false
	if inlineCls != nil {
		for _, c := range inlineCls {
			if c != clsNormal {
				hasInline = true
				break
			}
		}
	}

	// Fast path: no visual, no cursor, no inline → plain styled line.
	if !vr.active && !isCursorLine && !hasInline {
		return baseStyle.Render(line)
	}

	// Fast path: full-line visual, no cursor, no inline → style with selection bg.
	if vr.active && vr.full && !isCursorLine && !hasInline {
		return baseStyle.Background(selBg).Render(line)
	}

	// --- Segment-based rendering ---
	col := m.colCursor
	if col < 0 {
		col = 0
	}
	if col >= len(runes) {
		col = len(runes) - 1
	}

	sc, ec := -1, -1
	if vr.active {
		sc = vr.startCol
		if sc < 0 {
			sc = 0
		}
		ec = vr.endCol
		if ec < 0 || ec >= len(runes) {
			ec = len(runes) - 1
		}
	}

	cursorStyle := lipgloss.NewStyle().
		Background(primaryColor).
		Foreground(theme.Current.Background)

	// zone: 0=normal  1=selected  2=cursor
	type seg struct {
		text string
		zone int
		cls  int
	}
	var segs []seg
	for i, r := range runes {
		zone := 0
		if isCursorLine && i == col {
			zone = 2
		} else if vr.active && i >= sc && i <= ec {
			zone = 1
		}
		c := clsNormal
		if hasInline && i < len(inlineCls) {
			c = inlineCls[i]
		}
		ch := string(r)
		if len(segs) > 0 && segs[len(segs)-1].zone == zone && segs[len(segs)-1].cls == c {
			segs[len(segs)-1].text += ch
		} else {
			segs = append(segs, seg{text: ch, zone: zone, cls: c})
		}
	}

	var result strings.Builder
	for _, sg := range segs {
		st := resolveInlineStyle(sg.cls, baseStyle)
		switch sg.zone {
		case 2:
			result.WriteString(cursorStyle.Render(sg.text))
		case 1:
			result.WriteString(st.Background(selBg).Render(sg.text))
		default:
			result.WriteString(st.Render(sg.text))
		}
	}
	return result.String()
}

// modeIndicator returns a string showing the current mode.
func (m Model) modeIndicator() string {
	parts := []string{}
	switch m.visualMode {
	case VisualChar:
		parts = append(parts, " [VISUAL]")
	case VisualLine:
		parts = append(parts, " [V-LINE]")
	}
	if m.splitMode != "" {
		parts = append(parts, " [SPLIT]")
	}
	return strings.Join(parts, "")
}
