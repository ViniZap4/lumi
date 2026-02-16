#!/bin/bash
# Apply TUI fixes for glamour rendering and better navigation

cd "$(dirname "$0")/tui-client/ui"

# Backup
cp simple.go simple.go.before-fixes

# Fix renderFullNote to use glamour (line ~375)
# Replace the entire function with glamour-based rendering
cat > /tmp/render_patch.txt << 'PATCH'
func (m SimpleModel) renderFullNote() string {
	if m.renderer == nil || m.fullNote == nil {
		return "Error: No note loaded"
	}

	// Render markdown with glamour
	rendered, err := m.renderer.Render(m.fullNote.Content)
	if err != nil {
		rendered = m.fullNote.Content
	}

	lines := strings.Split(rendered, "\n")
	var s strings.Builder

	// Scrollable content
	maxLines := m.height - 4
	start := m.lineCursor
	if start > len(lines)-maxLines && len(lines) > maxLines {
		start = len(lines) - maxLines
	}
	if start < 0 {
		start = 0
	}

	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i < end; i++ {
		s.WriteString(lines[i])
		s.WriteString("\n")
	}

	s.WriteString("\n")
	help := HelpStyle.Render(fmt.Sprintf("j/k=scroll | e=edit | esc=back | Line %d/%d", m.lineCursor+1, len(lines)))
	s.WriteString(help)

	return s.String()
}
PATCH

echo "Patch file created at /tmp/render_patch.txt"
echo "Manually replace renderFullNote function (line ~375) with the content above"
echo ""
echo "Then rebuild:"
echo "  cd tui-client && go build -o lumi"
