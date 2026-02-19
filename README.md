# lumi

A local-first, Markdown-based note-taking system with beautiful TUI and web clients.

## ✨ Features

### TUI (Terminal)
- **Glamour Rendering** - Beautiful markdown with syntax highlighting
- **Image Support** - Inline images via sixel/Unicode (timg/chafa/viu)
- **Cursor Navigation** - Character-by-character movement (hjkl, 0/$, g/G)
- **Visual Mode** - Select text and copy to system clipboard (v, y)
- **Link Following** - Press enter on [[wiki-links]] to open
- **Tree Modal** - Quick file switcher (t key)
- **Search Modal** - Telescope-style search (/ key)
  - Search by filename or content (Ctrl+F to toggle)
  - Recursive across all folders
  - Live preview of results
- **Split Views** - Horizontal (s) and vertical (S) splits
- **External Editor** - Edit in $EDITOR (e key)

### Web Client
- **Modern Dark Theme** - Clean, professional interface
- **Folders & Notes** - Browse your note hierarchy
- **Live Preview** - Rendered markdown with syntax highlighting
- **Vim Keybindings** - j/k navigation, / for search
- **Real-time Sync** - WebSocket updates

### Server
- **HTTP API** - RESTful endpoints for notes and folders
- **WebSocket** - Real-time updates across clients
- **Token Auth** - Simple X-Lumi-Token header
- **CORS Enabled** - Works with web client

## 🚀 Quick Start

### Docker Compose (Recommended)

```bash
cp .env.example .env
docker-compose up -d

# Access:
# - Web UI: http://localhost:3000
# - API: http://localhost:8080
```

### TUI Client

```bash
cd tui-client
go build -o lumi
./lumi ../notes
```

**Dependencies (optional for image support):**
```bash
brew install timg  # or chafa, or viu
```

**Key Bindings:**
- `hjkl` - Navigate / move cursor
- `enter` - Open note / follow link
- `n` - Create new note
- `r` - Rename note
- `d` - Delete note
- `D` - Duplicate note
- `v` - Visual mode
- `y` - Copy (in visual mode)
- `t` - Tree modal (file switcher)
- `/` - Search modal
- `e` - Edit in external editor
- `s/S` - Horizontal/vertical split
- `esc` - Go back / exit mode
- `q` - Quit

### Server

```bash
cd server
LUMI_ROOT=../notes LUMI_PASSWORD=dev go run main.go
```

Or with Docker:

```bash
cd server
docker build -t lumi-server .
docker run -p 8080:8080 -v $(pwd)/../notes:/notes -e LUMI_PASSWORD=dev lumi-server
```

### Web Client

```bash
cd web-client
npm install
npm run dev
# Open http://localhost:5173
```

**Features:**
- Browse folders and notes
- Click or use j/k to navigate
- Enter to open notes
- / to search
- Live markdown preview
- Save with button or auto-save

## 📚 Documentation

- [Complete Feature Guide](COMPLETE_GUIDE.md) - All TUI features and workflows
- [Quick Test Guide](QUICKTEST.md) - How to test everything
- [Features Overview](FEATURES.md) - Detailed feature list
- [Developer Wiki](wiki/DEV.md) - Architecture and development
- [User Guide](wiki/USER.md) - Installation and usage

## 🎮 TUI Keybindings

### Home View
- Type to search
- `enter` - Go to tree

### Tree Navigation
- `j/k` - Move cursor
- `h` - Go back
- `l/enter` - Open folder/note
- Type - Search/filter
- `esc` - Clear search

### Full Note View
- `hjkl` - Move cursor
- `0/$` - Start/end of line
- `g/G` - Top/bottom
- `v` - Visual mode
- `y` - Copy (in visual)
- `enter` - Follow [[link]]
- `t` - Tree modal
- `/` - Search modal
- `s/S` - Splits
- `e` - External editor
- `esc` - Back

### Search Modal (/)
- Type - Search query
- `Ctrl+F` - Toggle filename/content search
- `j/k` - Navigate results
- `enter` - Open note
- `esc` - Close

### Tree Modal (t)
- `hjkl` - Navigate
- Type - Filter
- `enter` - Open note
- `esc` - Close

## 🌐 Web Client

**Navigation:**
- `j/k` - Move cursor
- `enter` - Open note
- `/` - Focus search
- `esc` - Clear search
- Click items to open

**Features:**
- Folders shown with 📁
- Notes shown with 📄
- Live markdown preview
- Syntax highlighted code blocks
- Colored headers (H1=yellow, H2=blue, H3=green)

## 🔧 API

### Endpoints

```
GET  /api/folders          - List all folders
GET  /api/notes            - List all notes
GET  /api/notes/:id        - Get note by ID
POST /api/notes            - Create note
PUT  /api/notes/:id        - Update note
DELETE /api/notes/:id      - Delete note
WS   /ws                   - WebSocket for updates
```

### Authentication

Include header: `X-Lumi-Token: <your-token>`

## 📁 Project Structure

```
lumi/
├── tui-client/     # Go TUI with Bubbletea
├── server/         # Go HTTP + WebSocket server
├── web-client/     # Svelte web app
├── wiki/           # Documentation
└── notes/          # Your notes (markdown files)
```

## 🎨 Note Format

```markdown
---
id: my-note-id
title: My Note Title
tags: [tag1, tag2]
created_at: 2026-02-16T10:00:00Z
updated_at: 2026-02-16T10:00:00Z
---

# My Note Title

Your content here with **markdown** formatting.

Link to other notes: [[other-note-id]]
```

## 💡 Tips

1. **Quick Copy**: `v` → `j/k` → `y` → paste anywhere
2. **Fast Search**: `/` → type → `enter`
3. **File Switch**: `t` → type → `enter`
4. **Link Jump**: Move cursor to [[link]] → `enter`
5. **Split View**: `s` → navigate to second note

## 🐛 Known Issues

- Split view structure exists but needs polish
- Clipboard copy works on Mac/Linux (needs testing on Windows)

## 🚧 Roadmap

- [ ] Mobile app
- [ ] End-to-end encryption
- [ ] Git sync
- [ ] Plugin system
- [ ] Themes

## 📄 License

MIT

## 🙏 Credits

Built with:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Svelte](https://svelte.dev) - Web framework
- [Go](https://golang.org) - Backend language
