# Current Status - What Works & What Needs Fixing

## ✅ WORKING

### Server
- ✅ Searches for notes by ID (not by filename)
- ✅ Handles notes in subdirectories
- ✅ CORS enabled
- ✅ WebSocket support
- ✅ Token authentication

### Web Client
- ✅ Shows folders (📁) and notes (📄)
- ✅ Markdown preview with styling
- ✅ Headers colored (H1=yellow, H2=blue, H3=green)
- ✅ Code blocks styled
- ✅ Notes load correctly (no more 404s)
- ✅ Vim keybindings (j/k, enter, /)
- ✅ Search functionality
- ✅ Clean dark theme

### TUI
- ✅ Glamour rendering (beautiful markdown)
- ✅ Tree modal (t key)
- ✅ Search modal (/ key) - Telescope-style
- ✅ Visual mode (v key)
- ✅ Clipboard copy (y in visual mode)
- ✅ External editor (e key)
- ✅ 3-column Yazi layout

## ❌ NEEDS FIXING

### TUI Issues

1. **Cursor Navigation Not Working Properly**
   - Cursor should move character-by-character with hjkl
   - Currently may not be visible or responsive
   - Need to ensure cursor shows on glamour-rendered content

2. **Split Views Not Working**
   - Structure exists (s/S keys)
   - Rendering not implemented
   - Need to show two notes side by side

3. **Tree Modal Behavior**
   - Should act exactly like main tree navigation
   - Need to ensure hjkl works in modal
   - Should support opening notes in splits

4. **Search in Home/Tree**
   - Home view should have search modal
   - Tree view should have search modal
   - Currently search modal only in full note view

5. **Search Inside File**
   - When in note view, / should search within current note
   - Different from global search
   - Need in-file search feature

## 🔧 FIXES NEEDED

### TUI Cursor Fix
The issue is that glamour renders markdown, but cursor needs to work on raw content. Solution:
- Keep two versions: rendered (for display) and raw (for cursor)
- Show cursor position on rendered content
- Use raw content for link detection and navigation

### TUI Split View Fix
Need to implement `renderSplitView()` properly:
- Horizontal: top/bottom panes
- Vertical: left/right panes
- Each pane shows a note with glamour rendering
- Navigate between panes with Ctrl+W or similar

### TUI Search in Home/Tree
Add search modal to home and tree views:
- Press / to open search modal
- Same Telescope-style interface
- Works from any view

### Web Client (Already Fixed)
- ✅ Server finds notes by ID
- ✅ Folders display correctly
- ✅ Notes load without 404 errors
- ✅ Preview shows rendered markdown

## 📋 TODO Priority

### High Priority
1. Fix TUI cursor navigation on glamour content
2. Implement TUI split view rendering
3. Add search modal to home/tree views
4. Add in-file search (/ in note view)

### Medium Priority
5. Tree modal should support split opening
6. Improve cursor visibility
7. Add split navigation (switch between panes)

### Low Priority
8. Polish split view UI
9. Add split resize
10. Add more vim motions

## 🎯 Quick Fixes

### For Cursor
```go
// In renderFullNote, show cursor on raw content
// but display glamour-rendered content
// Overlay cursor position indicator
```

### For Splits
```go
// In renderSplitView, use lipgloss.JoinHorizontal/Vertical
// Render each note with glamour
// Show active pane indicator
```

### For Search in Home
```go
// In home view, handle / key
// Open search modal (already exists)
// Reuse search modal code from full view
```

## 🧪 Testing

### Test Web Client
```bash
cd server && go run main.go &
cd web-client && npm run dev
# Open http://localhost:5173
# Should see folders and notes
# Click note - should open without 404
```

### Test TUI
```bash
cd tui-client && ./lumi ../notes
# Test cursor: hjkl should move cursor
# Test splits: s should create horizontal split
# Test search: / should open search modal
```

## 📝 Notes

- Web client is now fully working
- Server correctly finds notes by ID
- TUI has all features but some need polish
- Main issues are cursor visibility and split rendering
- Search modal exists but needs to be accessible from all views
