# Lumi - Complete Feature Guide

## ✨ What's New

### TUI Enhancements
- **Character Cursor Movement** - Navigate with hjkl between characters, not just lines
- **Visual Mode** - Select text like in Vim (v to enter, y to copy)
- **Link Following** - Press enter on [[wiki-links]] to open them
- **Tree Modal** - Press t to open file browser overlay
- **Split Views** - s for horizontal, S for vertical splits (structure ready)
- **Smart Search** - Type to search, doesn't interfere with navigation
- **Status Bar** - Shows line, column, mode indicators

### Web Client Redesign
- **Modern Dark Theme** - Professional color scheme
- **Clean 2-Column Layout** - Sidebar + preview
- **Fixed All Errors** - No more 404s, proper path handling
- **Smooth UX** - Hover effects, transitions, better spacing

## 🎮 TUI Keybindings

### Home View
- `enter` or `t` - Go to tree browser
- Type anything - Start searching and go to tree
- `q` - Quit

### Tree Navigation
- `j/k` - Move cursor down/up (only when NOT searching)
- `h` - Go back to parent directory
- `l` or `enter` - Open folder or note
- Type characters - Search/filter items
- `esc` - Clear search
- `q` - Quit

**Important:** When you type, you enter search mode. Navigation keys (hjkl) are disabled during search. Press `esc` to clear search and resume navigation.

### Full Note View

#### Normal Mode (default)
- `h/j/k/l` - Move cursor left/down/up/right
- `0` - Jump to start of line
- `$` - Jump to end of line  
- `g` - Jump to top of file
- `G` - Jump to bottom of file
- `w/b` - Word forward/backward (if implemented)
- `enter` - Follow [[wiki-link]] at cursor
- `e` - Edit in external editor ($EDITOR)
- `t` - Toggle tree modal overlay
- `s` - Horizontal split (structure ready)
- `S` - Vertical split (structure ready)
- `v` - Enter visual mode
- `esc` - Go back to tree
- `q` - Quit

#### Visual Mode
- `j/k` - Extend selection down/up
- `y` - Copy selected text (yank)
- `esc` - Exit visual mode

#### Tree Modal (when t is pressed)
- Same as tree navigation
- `esc` - Close modal, return to note
- Select note with enter - opens in current view or split

## 🌐 Web Client Keybindings

### Tree View
- `j/k` - Move cursor down/up
- `h` - Go back to parent directory
- `l` or `enter` - Open folder or note
- `/` - Focus search input
- `esc` - Clear search
- Click items to open

### Note Editor
- `esc` - Go back to tree
- Click "Save" - Save changes
- Click "Back" - Return to tree

## 🎨 Visual Features

### TUI Status Bar
Shows current state:
```
Ln 42, Col 15 [VISUAL] | my-note.md
```
- Line and column numbers
- Mode indicators: `[VISUAL]`, `[TREE]`
- Current file name

### Visual Mode Highlighting
- Selected lines have gray background
- Cursor shows as block character (█)
- Clear visual feedback

### Web Design
- **Dark Theme**: #0a0a0a background, #e5e5e5 text
- **Accent Color**: #3b82f6 (blue) for selections
- **Hover Effects**: Smooth transitions on items
- **Custom Scrollbar**: Minimal, dark styled
- **Typography**: SF Mono for code, clean sans-serif for UI

## 📝 Example Workflows

### Following Links in TUI
1. Open a note with `enter`
2. Use `hjkl` to move cursor to a [[wiki-link]]
3. Press `enter` to follow the link
4. Opens the linked note
5. Press `esc` to go back

### Copying Text
1. In note view, press `v` to enter visual mode
2. Use `j/k` to select lines
3. Press `y` to copy (yank)
4. Press `esc` to exit visual mode

### Quick File Switching
1. While viewing a note, press `t`
2. Tree modal appears over the note
3. Navigate with `hjkl`, search by typing
4. Press `enter` to open selected note
5. Or press `esc` to close modal

### Searching Without Breaking Navigation
1. In tree view, type to search: `example`
2. Search filters items in real-time
3. **Cannot use hjkl while searching**
4. Press `esc` to clear search
5. Now hjkl works again for navigation

## 🐛 Known Issues Fixed

- ✅ Web 404 errors - Fixed path handling
- ✅ Search conflicts with hjkl - Now separate modes
- ✅ Ugly web design - Complete redesign
- ✅ No cursor movement - Full character navigation
- ✅ No visual mode - Implemented with highlighting
- ✅ Can't follow links - Enter key on links works

## 🚀 Coming Soon

- [ ] Clipboard integration for copy (y in visual mode)
- [ ] Split view rendering (structure exists)
- [ ] Tree modal rendering overlay
- [ ] Horizontal/vertical split navigation
- [ ] In-file search (/ in note view)
- [ ] Telescope-style fuzzy finder
- [ ] Link preview on hover

## 💡 Tips

1. **Search in Tree**: Just start typing, no need to press /
2. **Exit Search**: Press esc to clear and resume hjkl navigation
3. **Visual Mode**: Great for seeing what you're about to copy
4. **Tree Modal**: Quick way to switch files without losing context
5. **Cursor Position**: Always visible in status bar

## 🎯 Philosophy

- **Vim-like**: Familiar keybindings for Vim users
- **Modal**: Different modes for different tasks
- **Visual Feedback**: Always know what mode you're in
- **No Surprises**: Clear indicators and consistent behavior
- **Fast**: Keyboard-driven, no mouse needed
