package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Cursor movement helpers ---

// clampCursor ensures lineCursor and colCursor are within bounds.
func (m *Model) clampCursor() {
	if len(m.contentLines) == 0 {
		m.lineCursor = 0
		m.colCursor = 0
		return
	}
	if m.lineCursor < 0 {
		m.lineCursor = 0
	}
	if m.lineCursor >= len(m.contentLines) {
		m.lineCursor = len(m.contentLines) - 1
	}
	lineLen := len(m.contentLines[m.lineCursor])
	if lineLen == 0 {
		m.colCursor = 0
	} else if m.colCursor >= lineLen {
		m.colCursor = lineLen - 1
	}
	if m.colCursor < 0 {
		m.colCursor = 0
	}
}

// applyDesiredCol sets colCursor from desiredCol, clamping to line length.
func (m *Model) applyDesiredCol() {
	if len(m.contentLines) == 0 {
		return
	}
	lineLen := len(m.contentLines[m.lineCursor])
	if lineLen == 0 {
		m.colCursor = 0
	} else if m.desiredCol >= lineLen {
		m.colCursor = lineLen - 1
	} else {
		m.colCursor = m.desiredCol
	}
}

// moveDown moves cursor down by n lines.
func (m *Model) moveDown(n int) {
	m.lineCursor += n
	if m.lineCursor >= len(m.contentLines) {
		m.lineCursor = len(m.contentLines) - 1
	}
	m.applyDesiredCol()
	m.updateVisualEnd()
}

// moveUp moves cursor up by n lines.
func (m *Model) moveUp(n int) {
	m.lineCursor -= n
	if m.lineCursor < 0 {
		m.lineCursor = 0
	}
	m.applyDesiredCol()
	m.updateVisualEnd()
}

// moveLeft moves cursor left.
func (m *Model) moveLeft() {
	if m.colCursor > 0 {
		m.colCursor--
		m.desiredCol = m.colCursor
	}
}

// moveRight moves cursor right.
func (m *Model) moveRight() {
	if len(m.contentLines) == 0 {
		return
	}
	lineLen := len(m.contentLines[m.lineCursor])
	if lineLen > 0 && m.colCursor < lineLen-1 {
		m.colCursor++
		m.desiredCol = m.colCursor
	}
}

// moveToLineStart moves to column 0.
func (m *Model) moveToLineStart() {
	m.colCursor = 0
	m.desiredCol = 0
}

// moveToFirstNonBlank moves to the first non-whitespace character.
func (m *Model) moveToFirstNonBlank() {
	if len(m.contentLines) == 0 {
		return
	}
	line := m.contentLines[m.lineCursor]
	for i, ch := range line {
		if !unicode.IsSpace(ch) {
			m.colCursor = i
			m.desiredCol = i
			return
		}
	}
	m.colCursor = 0
	m.desiredCol = 0
}

// moveToLineEnd moves to the last character of the line.
func (m *Model) moveToLineEnd() {
	if len(m.contentLines) == 0 {
		return
	}
	lineLen := len(m.contentLines[m.lineCursor])
	if lineLen > 0 {
		m.colCursor = lineLen - 1
	} else {
		m.colCursor = 0
	}
	m.desiredCol = m.colCursor
}

// moveToFileStart moves to line 0, col 0.
func (m *Model) moveToFileStart() {
	m.lineCursor = 0
	m.colCursor = 0
	m.desiredCol = 0
	m.updateVisualEnd()
}

// moveToFileEnd moves to the last line.
func (m *Model) moveToFileEnd() {
	if len(m.contentLines) > 0 {
		m.lineCursor = len(m.contentLines) - 1
	}
	m.colCursor = 0
	m.desiredCol = 0
	m.updateVisualEnd()
}

// moveWordForward moves to the start of the next word.
func (m *Model) moveWordForward() {
	if len(m.contentLines) == 0 {
		return
	}
	line := m.contentLines[m.lineCursor]

	// Skip current word characters
	i := m.colCursor
	for i < len(line) && !unicode.IsSpace(rune(line[i])) {
		i++
	}
	// Skip whitespace
	for i < len(line) && unicode.IsSpace(rune(line[i])) {
		i++
	}

	if i < len(line) {
		m.colCursor = i
	} else if m.lineCursor < len(m.contentLines)-1 {
		// Move to next line, first non-blank
		m.lineCursor++
		m.moveToFirstNonBlank()
	} else {
		m.moveToLineEnd()
	}
	m.desiredCol = m.colCursor
}

// moveWordBackward moves to the start of the previous word.
func (m *Model) moveWordBackward() {
	if len(m.contentLines) == 0 {
		return
	}
	line := m.contentLines[m.lineCursor]

	i := m.colCursor
	if i > 0 {
		i--
	}

	// Skip whitespace backward
	for i > 0 && unicode.IsSpace(rune(line[i])) {
		i--
	}
	// Skip word characters backward
	for i > 0 && !unicode.IsSpace(rune(line[i-1])) {
		i--
	}

	if i >= 0 {
		m.colCursor = i
	} else if m.lineCursor > 0 {
		m.lineCursor--
		m.moveToLineEnd()
	}
	m.desiredCol = m.colCursor
}

// moveWordEnd moves to the end of the current/next word.
func (m *Model) moveWordEnd() {
	if len(m.contentLines) == 0 {
		return
	}
	line := m.contentLines[m.lineCursor]

	i := m.colCursor
	if i < len(line)-1 {
		i++
	}

	// Skip whitespace
	for i < len(line) && unicode.IsSpace(rune(line[i])) {
		i++
	}
	// Move to end of word
	for i < len(line)-1 && !unicode.IsSpace(rune(line[i+1])) {
		i++
	}

	if i < len(line) {
		m.colCursor = i
	} else if m.lineCursor < len(m.contentLines)-1 {
		m.lineCursor++
		line = m.contentLines[m.lineCursor]
		m.colCursor = 0
		// Skip whitespace then to end of word
		for m.colCursor < len(line) && unicode.IsSpace(rune(line[m.colCursor])) {
			m.colCursor++
		}
		for m.colCursor < len(line)-1 && !unicode.IsSpace(rune(line[m.colCursor+1])) {
			m.colCursor++
		}
	}
	m.desiredCol = m.colCursor
}

// halfPageDown scrolls half a page down.
func (m *Model) halfPageDown() {
	amount := m.viewportHeight() / 2
	if amount < 1 {
		amount = 1
	}
	m.moveDown(amount)
}

// halfPageUp scrolls half a page up.
func (m *Model) halfPageUp() {
	amount := m.viewportHeight() / 2
	if amount < 1 {
		amount = 1
	}
	m.moveUp(amount)
}

// viewportHeight returns usable lines for note content.
func (m *Model) viewportHeight() int {
	h := m.height - 5 // status bar + help + padding
	if h < 1 {
		h = 1
	}
	return h
}

// --- Visual mode ---

// startVisualLine enters visual line mode.
func (m *Model) startVisualLine() {
	m.visualMode = VisualLine
	m.visualStart = m.lineCursor
	m.visualEnd = m.lineCursor
}

// startVisualChar enters visual character mode.
func (m *Model) startVisualChar() {
	m.visualMode = VisualChar
	m.visualStart = m.lineCursor
	m.visualEnd = m.lineCursor
	m.visualStartCol = m.colCursor
	m.visualEndCol = m.colCursor
}

// updateVisualEnd updates the visual selection endpoint.
func (m *Model) updateVisualEnd() {
	if m.visualMode != VisualNone {
		m.visualEnd = m.lineCursor
		m.visualEndCol = m.colCursor
	}
}

// isLineInVisual returns true if line i is part of the visual selection.
func (m *Model) isLineInVisual(i int) bool {
	if m.visualMode == VisualNone {
		return false
	}
	startLine := min(m.visualStart, m.visualEnd)
	endLine := max(m.visualStart, m.visualEnd)
	return i >= startLine && i <= endLine
}

// yankSelection copies the visual selection to clipboard, starts a yank flash,
// and exits visual mode. Returns a tea.Cmd for the flash timer.
func (m *Model) yankSelection() tea.Cmd {
	if m.visualMode == VisualNone {
		return nil
	}

	startLine := min(m.visualStart, m.visualEnd)
	endLine := max(m.visualStart, m.visualEnd)

	if m.visualMode == VisualLine {
		var selected []string
		for i := startLine; i <= endLine && i < len(m.contentLines); i++ {
			selected = append(selected, m.contentLines[i])
		}
		clipboard.WriteAll(strings.Join(selected, "\n"))
		n := endLine - startLine + 1
		m.statusMsg = fmt.Sprintf("%d line%s yanked", n, pluralS(n))
	} else if m.visualMode == VisualChar {
		// Character-wise selection
		sLine, sCol := m.visualStart, m.visualStartCol
		eLine, eCol := m.visualEnd, m.visualEndCol
		// Normalize: ensure start is before end
		if sLine > eLine || (sLine == eLine && sCol > eCol) {
			sLine, eLine = eLine, sLine
			sCol, eCol = eCol, sCol
		}

		if sLine == eLine {
			line := m.contentLines[sLine]
			if eCol >= len(line) {
				eCol = len(line) - 1
			}
			if sCol < len(line) {
				clipboard.WriteAll(line[sCol : eCol+1])
			}
		} else {
			var parts []string
			// First line from sCol to end
			if sCol < len(m.contentLines[sLine]) {
				parts = append(parts, m.contentLines[sLine][sCol:])
			}
			// Middle lines in full
			for i := sLine + 1; i < eLine && i < len(m.contentLines); i++ {
				parts = append(parts, m.contentLines[i])
			}
			// Last line from start to eCol
			if eLine < len(m.contentLines) {
				lastLine := m.contentLines[eLine]
				if eCol >= len(lastLine) {
					eCol = len(lastLine) - 1
				}
				if eCol >= 0 {
					parts = append(parts, lastLine[:eCol+1])
				}
			}
			clipboard.WriteAll(strings.Join(parts, "\n"))
		}
		m.statusMsg = "selection yanked"
	}

	// Save range for yank flash highlight
	m.yankHighlight = true
	m.yankMode = m.visualMode
	m.yankStartLine = m.visualStart
	m.yankEndLine = m.visualEnd
	m.yankStartCol = m.visualStartCol
	m.yankEndCol = m.visualEndCol

	m.visualMode = VisualNone

	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return yankFlashMsg(t)
	})
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// scrollOffset computes the scroll start line for the viewport.
func (m *Model) scrollOffset(totalLines int) int {
	maxLines := m.viewportHeight()
	if totalLines <= maxLines {
		return 0
	}

	start := m.lineCursor - maxLines/2
	if start < 0 {
		start = 0
	}
	if start > totalLines-maxLines {
		start = totalLines - maxLines
	}
	return start
}
