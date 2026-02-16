# TUI Improvements Applied

## Changes Made

### 1. Added Glamour for Markdown Rendering
- Import: `github.com/charmbracelet/glamour`
- Added `renderer *glamour.TermRenderer` field to SimpleModel
- Initialize in NewSimpleModel with auto-style and word wrap

### 2. Recursive Search
- Added `searchMode bool` field to toggle recursive vs local search
- Added `searchRecursive()` function that walks entire directory tree
- Press `/` in tree view to toggle between local and recursive search
- Shows full relative path when in recursive mode

### 3. Improved Note Rendering
- Use glamour.Render() for markdown content
- Automatic syntax highlighting for code blocks
- Better formatting for headers, lists, quotes
- Respects terminal colors and theme

## To Apply

Replace the renderFullNote function around line 330 with:

```go
func (m SimpleModel) renderFullNote() string {
	var s strings.Builder

	// Render markdown with glamour
	rendered, err := m.renderer.Render(m.fullNote.Content)
	if err != nil {
		rendered = m.fullNote.Content // Fallback
	}
	
	lines := strings.Split(rendered, "\n")
	maxLines := m.height - 4
	start := m.lineCursor
	if start > len(lines)-maxLines {
		start = max(0, len(lines)-maxLines)
	}
	
	for i := start; i < min(len(lines), start+maxLines); i++ {
		s.WriteString(lines[i])
		s.WriteString("\n")
	}

	s.WriteString("\n")
	status := fmt.Sprintf("Line %d/%d | %s", m.lineCursor+1, len(lines), m.fullNote.ID)
	s.WriteString(HelpStyle.Render(status))
	s.WriteString("\n")
	help := HelpStyle.Render("hjkl=scroll | /=search | e=edit | esc=back | q=quit")
	s.WriteString(help)

	return s.String()
}
```

## Status Bar Update

Update the tree view status to show search mode:

```go
// In renderTreeYazi or renderTree, add to help text:
searchIndicator := ""
if m.searchMode {
	searchIndicator = " [RECURSIVE]"
}
help := HelpStyle.Render(fmt.Sprintf("hjkl=move | /=toggle search%s | enter=open | q=quit", searchIndicator))
```

## Build and Test

```bash
cd tui-client
go build -o lumi
./lumi ../notes
```

Test:
1. Press `/` in tree view - should show [RECURSIVE] indicator
2. Type to search - finds files in all subdirectories
3. Open a note - should see beautiful markdown rendering with colors
4. Code blocks should have syntax highlighting
