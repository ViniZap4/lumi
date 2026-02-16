# TUI Redesign Plan - Yazi/lf Style

## Current Issues
1. Navigation is confusing (home → tree modal → full view)
2. No persistent file browser
3. Search only in tree modal
4. Web client can't fetch notes (CORS issue - FIXED)

## New Design (Like Yazi/lf)

### Layout (Always Visible)
```
┌─────────────┬──────────────┬─────────────────────┐
│   Parent    │   Current    │      Preview        │
│  (folders)  │ (files/dirs) │   (note content)    │
│             │              │                     │
│  ../        │ ▸ note1.md   │ # Note Title        │
│  folder1/   │   note2.md   │                     │
│  folder2/   │   folder/    │ Content preview...  │
│             │              │                     │
└─────────────┴──────────────┴─────────────────────┘
 Search: query_  |  hjkl=move  enter=open  /=search
```

### Key Features
1. **3-Panel Layout** (always visible)
   - Left: Parent directory context
   - Center: Current directory (folders + notes)
   - Right: Live preview of selected note

2. **Search** (like fzf)
   - Press `/` to start search
   - Type to filter in real-time
   - Shows matches in center panel
   - ESC to clear search

3. **Navigation**
   - `h` - Go to parent directory
   - `l` - Enter folder or open note in full view
   - `j/k` - Move up/down in list
   - `gg/G` - Top/bottom
   - `/` - Start search
   - `enter` - Open note in full view
   - `V` - Toggle full view (hides panels)

4. **Home View**
   - Shows on startup
   - ASCII art + recent notes
   - Press any key to enter browser
   - Search bar at bottom

## Implementation Steps
1. ✅ Fix CORS in server
2. Remove home/tree modal modes
3. Make 3-panel layout default
4. Add persistent search bar
5. Simplify navigation logic

## Files to Modify
- `tui-client/ui/app.go` - Main layout
- `tui-client/ui/home.go` - Simplify home
- `tui-client/ui/tree.go` - Remove (integrate into main)
- `server/main.go` - ✅ CORS added
