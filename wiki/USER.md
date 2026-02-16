# lumi User Guide

## What is lumi?

**lumi** is a local-first note-taking system that keeps your notes as plain Markdown files on your computer. You can access and edit your notes through:

- A **terminal interface** (TUI) with vim-like keyboard shortcuts
- A **web interface** that syncs in realtime
- Your **favorite text editor** (vim, emacs, VS Code, etc)

Your notes never leave your control - they're just files on your disk.

## Installation

### TUI Client (Terminal)

```bash
# From source
cd tui-client
go build -o lumi
./lumi

# Or run directly
go run main.go
```

### Server (for Web Client)

```bash
# Using Docker
docker run -d \
  -p 8080:8080 \
  -v /path/to/your/notes:/notes \
  -e LUMI_ROOT=/notes \
  -e LUMI_PASSWORD=your-secret-token \
  lumi-server

# From source
cd server
LUMI_ROOT=/path/to/notes LUMI_PASSWORD=secret go run main.go
```

### Web Client

```bash
# Using Docker
docker run -d \
  -p 3000:80 \
  -e VITE_LUMI_SERVER_URL=http://localhost:8080 \
  lumi-web

# From source
cd web-client
npm install
VITE_LUMI_SERVER_URL=http://localhost:8080 npm run dev
```

## Using the TUI (Terminal Interface)

### Layout

```
┌─────────────────┬──────────────────────────────┐
│   Folders       │   Notes                      │
│                 │                              │
│ > Projects      │ > 2026-lumi-architecture     │
│   Personal      │   2026-meeting-notes         │
│   Archive       │   2026-ideas                 │
│                 │                              │
└─────────────────┴──────────────────────────────┘
┌──────────────────────────────────────────────────┐
│ Help: q=quit | e=edit | n=new | d=delete | ?=help│
└──────────────────────────────────────────────────┘
```

### Keyboard Shortcuts

#### Navigation
- `j` / `k` - Move down/up in current panel
- `h` / `l` - Switch between panels (folders ↔ notes)
- `gg` - Jump to top
- `G` - Jump to bottom
- `Tab` - Switch focus between panels
- `/` - Search/filter current panel

#### Actions
- `e` or `Enter` - Edit selected note in $EDITOR
- `n` - Create new note
- `d` - Delete selected note (with confirmation)
- `r` - Rename note
- `?` - Show help
- `q` - Quit

#### Folder Navigation
- `Enter` - Open selected folder
- `Backspace` - Go up one folder level
- `h` - Go up (alternative to Backspace)
- `l` - Open folder (alternative to Enter)

### Creating a Note

1. Press `n` to create a new note
2. Enter the note title
3. The note opens in your editor with frontmatter pre-filled
4. Save and close the editor
5. The note appears in the list

### Editing a Note

1. Navigate to the note with `j`/`k`
2. Press `e` or `Enter`
3. Your editor opens (uses `$EDITOR` env var, defaults to `nvim`)
4. Make changes and save
5. Close the editor to return to lumi

### Note Format

Notes are Markdown files with YAML frontmatter:

```markdown
---
id: 2026-my-note
title: My Note Title
created_at: 2026-02-16T11:00:00-03:00
updated_at: 2026-02-16T11:00:00-03:00
tags:
  - personal
  - ideas
---

# My Note

This is the content in **Markdown**.

## Linking to Other Notes

Use double brackets: [[2026-another-note]]

Or regular Markdown links: [Another Note](./another-note.md)
```

### Organizing Notes

- Notes are organized in folders on your filesystem
- Use `h`/`l` to navigate folder hierarchy in the TUI
- Create folders directly in your file manager or terminal
- lumi automatically detects new folders and notes

## Using the Web Interface

### Connecting

1. Make sure the server is running
2. Open your browser to `http://localhost:3000` (or your configured URL)
3. Enter your authentication token (set via `LUMI_PASSWORD`)

### Features

- **Folder tree** on the left
- **Note list** in the middle
- **Editor** on the right
- **Realtime sync** - changes appear instantly across all clients

### Keyboard Shortcuts (Web)

- `Ctrl+S` - Save note
- `Ctrl+N` - New note
- `Ctrl+F` - Search notes
- `Esc` - Close dialogs

## Configuration

### TUI Client

The TUI reads notes from the current directory by default. You can specify a different location:

```bash
# Set notes directory
export LUMI_NOTES_DIR=/path/to/notes
./lumi
```

### Server

Configure via environment variables:

- `LUMI_ROOT` - Path to notes directory (default: `./notes`)
- `LUMI_PASSWORD` - Authentication token (default: `dev`)
- `LUMI_PORT` - Server port (default: `8080`)

### Editor

The TUI uses your `$EDITOR` environment variable:

```bash
# Use vim
export EDITOR=vim

# Use VS Code (wait for window to close)
export EDITOR="code --wait"

# Use emacs
export EDITOR=emacs
```

If `$EDITOR` is not set, lumi defaults to `nvim`.

## Tips & Tricks

### Quick Note Creation

Create a note template in your notes directory:

```bash
# Create a template
cat > notes/_template.md << 'EOF'
---
id: REPLACE_ME
title: New Note
created_at: REPLACE_ME
updated_at: REPLACE_ME
tags: []
---

# New Note

Start writing here...
EOF
```

### Folder Structure

Organize by project, date, or topic:

```
notes/
├── projects/
│   ├── lumi/
│   └── work/
├── journal/
│   ├── 2026-02/
│   └── 2026-01/
└── reference/
    ├── code-snippets/
    └── bookmarks/
```

### Linking Notes

Use consistent ID patterns for easy linking:

- Date-based: `2026-02-16-meeting-notes`
- Topic-based: `go-concurrency-patterns`
- Project-based: `lumi-architecture`

### Backup

Since notes are plain files, backup is simple:

```bash
# Git
cd notes
git init
git add .
git commit -m "Backup notes"

# Rsync
rsync -av notes/ backup/notes/

# Cloud sync
# Just point Dropbox/iCloud/etc to your notes folder
```

## Troubleshooting

### TUI won't start

- Check that you're in the correct directory
- Verify Go is installed: `go version`
- Check terminal compatibility (needs ANSI color support)

### Editor doesn't open

- Verify `$EDITOR` is set: `echo $EDITOR`
- Try setting explicitly: `export EDITOR=vim`
- Check that the editor is in your PATH

### Server connection fails

- Verify server is running: `curl http://localhost:8080/api/folders`
- Check authentication token matches
- Check firewall settings

### Notes not syncing

- Verify WebSocket connection in browser console
- Check server logs for errors
- Ensure `LUMI_ROOT` points to correct directory

## FAQ

**Q: Can I use lumi without the server?**  
A: Yes! The TUI works completely offline with local files.

**Q: What happens if I edit a file outside of lumi?**  
A: Changes are detected automatically. The TUI refreshes on focus, and the web client receives realtime updates.

**Q: Can multiple people use the same notes?**  
A: Yes, if they share the same `LUMI_ROOT` directory (via network mount or sync). The server broadcasts changes to all connected clients.

**Q: Is there mobile support?**  
A: Not yet, but it's planned. The web client works on mobile browsers.

**Q: Can I encrypt my notes?**  
A: Not built-in yet. For now, use filesystem encryption (FileVault, LUKS, etc).

**Q: How do I export my notes?**  
A: They're already plain Markdown files! Just copy the folder.

## Getting Help

- Check the [Developer Wiki](./DEV.md) for technical details
- Open an issue on GitHub
- Read the source code - it's designed to be readable!
