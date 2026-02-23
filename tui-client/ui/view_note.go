package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/image"
)

func (m Model) renderFullNote() string {
	if m.fullNote == nil {
		return "No note loaded"
	}

	var s strings.Builder

	// Use glamour-rendered lines for display
	displayLines := m.renderedLines
	if len(displayLines) == 0 {
		displayLines = m.contentLines
	}

	// Process image lines — replace markdown image syntax with rendered images
	displayLines = m.processImages(displayLines)

	// Viewport scrolling centered on cursor
	maxLines := m.viewportHeight()
	totalLines := len(displayLines)

	// Map lineCursor (raw content position) to display position
	displayCursor := m.mapCursorToDisplay(totalLines)

	start := displayCursor - maxLines/2
	if start < 0 {
		start = 0
	}
	if start > totalLines-maxLines {
		start = max(0, totalLines-maxLines)
	}
	end := min(start+maxLines, totalLines)

	// Render visible lines
	for i := start; i < end; i++ {
		line := ""
		if i < len(displayLines) {
			line = displayLines[i]
		}

		// Visual mode highlighting
		inVisual := m.isDisplayLineInVisual(i, totalLines)

		if i == displayCursor {
			prefix := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Render("> ")

			if inVisual {
				line = prefix + lipgloss.NewStyle().
					Background(lipgloss.Color("237")).
					Render(line)
			} else {
				line = prefix + line
			}
		} else {
			if inVisual {
				line = "  " + lipgloss.NewStyle().
					Background(lipgloss.Color("237")).
					Render(line)
			} else {
				line = "  " + line
			}
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	// Status bar
	s.WriteString("\n")
	mode := m.modeIndicator()
	status := fmt.Sprintf("Ln %d, Col %d%s | %s", m.lineCursor+1, m.colCursor+1, mode, m.fullNote.Title)
	s.WriteString(StatusBarStyle.Render(status))
	s.WriteString("\n")

	helpParts := []string{"hjkl=move", "w/b=word", "v=visual", "V=vline", "y=yank"}
	helpParts = append(helpParts, "e=edit", "t=tree", "/=search", "esc=back")
	s.WriteString(HelpStyle.Render(strings.Join(helpParts, " | ")))

	return s.String()
}

// processImages replaces image markdown lines with rendered terminal images.
func (m Model) processImages(lines []string) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if image.HasImage(line) {
			imgPath := image.GetImagePath(line, m.fullNote.Path)
			if imgPath != "" {
				if _, err := os.Stat(imgPath); err == nil {
					rendered := image.Render(imgPath, m.width-6)
					// Image might be multi-line
					result = append(result, strings.Split(rendered, "\n")...)
					continue
				}
			}
			result = append(result, lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Render(fmt.Sprintf("[Image not found: %s]", filepath.Base(image.ExtractImagePath(line)))))
		} else {
			result = append(result, line)
		}
	}
	return result
}

// mapCursorToDisplay maps the raw content lineCursor to a display line index.
// When using glamour, display lines may differ from raw content lines,
// so we use a proportional mapping.
func (m Model) mapCursorToDisplay(totalDisplayLines int) int {
	if len(m.contentLines) == 0 || totalDisplayLines == 0 {
		return 0
	}
	if len(m.contentLines) == 1 {
		return 0
	}
	// Proportional mapping
	ratio := float64(m.lineCursor) / float64(len(m.contentLines)-1)
	mapped := int(ratio * float64(totalDisplayLines-1))
	if mapped < 0 {
		mapped = 0
	}
	if mapped >= totalDisplayLines {
		mapped = totalDisplayLines - 1
	}
	return mapped
}

// isDisplayLineInVisual checks if a display line falls in the visual selection range.
func (m Model) isDisplayLineInVisual(displayLine, totalDisplayLines int) bool {
	if m.visualMode == VisualNone {
		return false
	}
	startLine := min(m.visualStart, m.visualEnd)
	endLine := max(m.visualStart, m.visualEnd)

	startDisplay := m.mapRawToDisplay(startLine, totalDisplayLines)
	endDisplay := m.mapRawToDisplay(endLine, totalDisplayLines)

	return displayLine >= startDisplay && displayLine <= endDisplay
}

func (m Model) mapRawToDisplay(rawLine, totalDisplayLines int) int {
	if len(m.contentLines) <= 1 || totalDisplayLines == 0 {
		return 0
	}
	ratio := float64(rawLine) / float64(len(m.contentLines)-1)
	mapped := int(ratio * float64(totalDisplayLines-1))
	if mapped < 0 {
		return 0
	}
	if mapped >= totalDisplayLines {
		return totalDisplayLines - 1
	}
	return mapped
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
