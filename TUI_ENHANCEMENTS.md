# TUI Enhancement Roadmap

## Current Status
✅ Home view with ASCII art
✅ Yazi-style 3-column layout
✅ Cursor navigation with vim motions
✅ Link following with [[wiki-links]]
✅ External editor support ($EDITOR)

## Planned Enhancements

### 1. Recursive Search
- [ ] Search across all subdirectories
- [ ] Show full path in results
- [ ] Fast fuzzy matching (fzf-style)

### 2. Improved Note Rendering (Glow-style)
- [ ] Syntax highlighting for code blocks
- [ ] Better markdown rendering (headers, lists, quotes)
- [ ] Color scheme for different elements
- [ ] Image support (Kitty/iTerm2 protocols)

### 3. Vim-like Insert Mode
- [ ] Normal mode (current - navigation)
- [ ] Insert mode (i key - edit inline)
- [ ] Visual mode (v key - select text)
- [ ] Commands: :w (save), :q (quit), :wq (save & quit)

### 4. Easy Link Insertion
- [ ] Ctrl+L to open link picker
- [ ] Fuzzy search all notes recursively
- [ ] Insert [[note-id]] at cursor
- [ ] Preview note while selecting

### 5. Configuration System
- [ ] Config file: ~/.config/lumi/config.yaml
- [ ] Settings:
  - editor: "nvim" | "vim" | "emacs" | "code"
  - theme: "default" | "gruvbox" | "nord"
  - search_recursive: true | false
  - image_protocol: "kitty" | "iterm2" | "none"
- [ ] Edit config from home view (c key)

### 6. Enhanced External Editor
- [ ] Keep both inline and external editing
- [ ] e key: open in $EDITOR
- [ ] i key: enter insert mode (inline)
- [ ] Auto-reload on external save

## Implementation Priority

### Phase 1 (Critical)
1. Recursive search
2. Config system
3. Link insertion with fzf

### Phase 2 (Polish)
4. Glow-style rendering
5. Vim insert mode
6. Image support

### Phase 3 (Advanced)
7. Visual mode
8. Advanced vim commands
9. Custom themes

## Technical Notes

### Recursive Search
```go
func searchRecursive(root, query string) []Item {
    var results []Item
    filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if strings.HasSuffix(path, ".md") {
            // Check filename and content
            if fuzzyMatch(query, path) {
                results = append(results, ...)
            }
        }
        return nil
    })
    return results
}
```

### Glow-style Rendering
- Use `github.com/charmbracelet/glamour` for markdown rendering
- Custom styles with lipgloss
- Code highlighting with `github.com/alecthomas/chroma`

### Image Support
- Detect terminal capabilities
- Use `github.com/srwiley/oksvg` for SVG
- Kitty protocol for raster images
- Fallback to ASCII art

### Config System
```yaml
# ~/.config/lumi/config.yaml
editor: nvim
theme: default
search:
  recursive: true
  fuzzy: true
rendering:
  images: true
  image_protocol: kitty
  syntax_highlighting: true
keybindings:
  insert_link: "ctrl+l"
  edit_external: "e"
  edit_inline: "i"
```

## Web Client Fixes
✅ Add folder navigation
✅ Fix note ID handling
✅ Better error messages
- [ ] Add home view with ASCII
- [ ] Add 3-column preview
- [ ] Match TUI keybindings exactly
