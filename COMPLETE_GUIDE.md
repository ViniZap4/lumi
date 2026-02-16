# Lumi TUI - Complete Feature Guide

## ✨ All Features Working

### 1. Glamour Rendering + Cursor Navigation
**Both work together!**
- Beautiful markdown rendering with syntax highlighting
- Cursor visible as █ block character
- Navigate with hjkl between characters
- Cursor position: Ln X, Col Y in status bar

### 2. Visual Mode + System Clipboard
- Press `v` to enter visual mode
- `j/k` to extend selection
- Selected lines highlighted (gray background)
- Press `y` to **copy to system clipboard**
- Paste anywhere with Cmd+V (Mac) or Ctrl+V (Linux/Windows)
- Press `esc` to exit visual mode

### 3. Tree Modal (File Browser)
- Press `t` in note view
- Centered overlay (same as main tree navigation)
- Navigate with `hjkl`
- Type to search/filter
- `enter` to open note
- `esc` to close

### 4. Search Modal (Telescope-style)
- Press `/` in note view
- **Centered modal overlay**
- Type to search in real-time
- **Ctrl+F** to toggle search type:
  - `[Filename]` - Search note filenames
  - `[Content]` - Search inside note content
- **Recursive** - searches all subdirectories
- Shows preview snippet of selected result
- `hjkl` to navigate results
- `enter` to open note
- `esc` to close
- **No key conflicts** - separate modal mode

### 5. Split Views
- `s` = horizontal split (top/bottom)
- `S` = vertical split (left/right)
- View two notes simultaneously
- Navigate between splits

### 6. Link Following
- Move cursor to [[wiki-link]]
- Press `enter` to follow link
- Opens the linked note

## 🎮 Complete Keybindings

### Home View
- `enter` or `t` - Go to tree
- Type anything - Search and go to tree
- `q` - Quit

### Tree Navigation
- `j/k` - Move cursor
- `h` - Go back to parent
- `l` or `enter` - Open folder/note
- Type - Search/filter
- `esc` - Clear search
- `q` - Quit

### Full Note View (Normal Mode)
- `h/j/k/l` - Move cursor (char by char)
- `0` - Start of line
- `$` - End of line
- `g` - Top of file
- `G` - Bottom of file
- `enter` - Follow [[link]] at cursor
- `v` - Enter visual mode
- `t` - Open tree modal
- `/` - Open search modal
- `s` - Horizontal split
- `S` - Vertical split
- `e` - Edit in external editor
- `esc` - Go back to tree
- `q` - Quit

### Visual Mode
- `j/k` - Extend selection
- `y` - Copy to clipboard
- `esc` - Exit visual mode

### Tree Modal (when t is pressed)
- `hjkl` - Navigate
- Type - Search
- `enter` - Open note
- `esc` - Close modal

### Search Modal (when / is pressed)
- Type - Search query
- `Ctrl+F` - Toggle Filename ↔ Content
- `j/k` - Navigate results
- `enter` - Open selected note
- `backspace` - Delete char
- `esc` - Close modal

## 🎨 Visual Features

### Status Bar
```
Ln 42, Col 15 [VISUAL] | my-note.md
```
Shows:
- Current line and column
- Mode indicators: `[VISUAL]`, `[TREE]`
- Current filename

### Glamour Rendering
- **Headers**: Bold, colored (yellow/blue/green)
- **Code blocks**: Syntax highlighted
- **Lists**: Properly indented
- **Links**: Highlighted
- **Quotes**: Left border, muted color
- **Bold/Italic**: Styled

### Visual Selection
- Gray background on selected lines
- Cursor visible as █ block
- Clear visual feedback

### Search Modal
```
┌────────────────────────────────────┐
│  🔍 Search Notes                   │
│                                    │
│  [Filename] search query█          │
│                                    │
│  ▸ 📄 example-note.md              │
│      This is a preview snippet...  │
│    📄 another-note.md              │
│    📄 third-note.md                │
│                                    │
│  ctrl+f=toggle | enter=open | esc=close │
└────────────────────────────────────┘
```

### Tree Modal
```
┌────────────────────────────────────┐
│  📂 Select Note                    │
│                                    │
│  🔍 filter█                        │
│                                    │
│  ▸ 📄 current-note                 │
│    📄 other-note                   │
│    📁 folder/                      │
│                                    │
│  hjkl=move | enter=open | esc=close │
└────────────────────────────────────┘
```

### Split View (Horizontal)
```
┌────────────────────────────────────┐
│  Note 1 Content                    │
│  Beautiful glamour rendering...    │
│                                    │
├────────────────────────────────────┤
│  Note 2 Content                    │
│  Also rendered beautifully...      │
│                                    │
└────────────────────────────────────┘
```

## 📝 Example Workflows

### 1. Copy Text to Clipboard
```
1. Open note with enter
2. Press v (enter visual mode)
3. Press j/k to select lines
4. Press y (copy to clipboard)
5. Switch to any app
6. Cmd+V to paste
```

### 2. Search Across All Notes
```
1. In note view, press /
2. Type search query: "example"
3. Press Ctrl+F to search in content (not just filenames)
4. Use j/k to navigate results
5. See preview snippets
6. Press enter to open selected note
```

### 3. Quick File Switching
```
1. While viewing note, press t
2. Tree modal appears
3. Type to filter: "arch"
4. Press enter to open
5. Modal closes, note opens
```

### 4. Follow Links
```
1. Move cursor with hjkl to [[wiki-link]]
2. Press enter
3. Linked note opens
4. Press esc to go back
```

### 5. Split View Comparison
```
1. Open first note
2. Press s (horizontal split)
3. Navigate to second note in tree modal
4. Both notes visible
5. Compare side by side
```

## 🔍 Search Modal Details

### Filename Search (Default)
- Searches note filenames
- Fast, instant results
- Good for finding specific notes

### Content Search (Ctrl+F)
- Searches inside note content
- Finds text anywhere in notes
- Shows preview of matching line
- Slower but more thorough

### Recursive
- Searches all subdirectories
- No need to navigate folders
- Finds notes anywhere in tree

### No Key Conflicts
- Search modal is separate mode
- hjkl only work in modal
- Type to search, no interference
- Esc exits cleanly

## 💡 Pro Tips

1. **Quick Copy**: `v` → `j/k` → `y` → paste anywhere
2. **Fast Search**: `/` → type → `enter` (Telescope-style)
3. **File Switch**: `t` → type → `enter` (no need to navigate tree)
4. **Link Jump**: Move cursor to link → `enter`
5. **Split Compare**: `s` → open second note → compare
6. **Content Search**: `/` → `Ctrl+F` → search text inside notes

## 🎯 Key Differences from Before

### Before
- Glamour OR cursor (not both)
- No clipboard integration
- Search conflicted with hjkl
- No search modal
- Tree modal didn't work
- Splits not implemented

### After
- ✅ Glamour AND cursor together
- ✅ System clipboard copy
- ✅ Search modal (no conflicts)
- ✅ Tree modal works perfectly
- ✅ Splits fully implemented
- ✅ All features integrated

## 🚀 Test Everything

```bash
cd tui-client
./lumi ../notes
```

**Try in order:**
1. Open a note → see glamour rendering
2. Press `hjkl` → cursor moves
3. Press `v` → visual mode
4. Press `j/k` → selection extends
5. Press `y` → copied to clipboard
6. Press `esc` → exit visual
7. Press `t` → tree modal opens
8. Type to filter → results update
9. Press `esc` → modal closes
10. Press `/` → search modal opens
11. Type query → results appear
12. Press `Ctrl+F` → toggle to content search
13. Press `enter` → open result
14. Move cursor to [[link]] → press `enter`
15. Press `s` → horizontal split

All features work together seamlessly!
