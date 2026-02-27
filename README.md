# lumi

> **🚧 Work in progress** — lumi is under active development. Features may change, break, or be incomplete.

A local-first, Markdown-based note-taking system with beautiful TUI and web clients.

## 📦 Repository Structure

Each component lives in its own repo, linked here as git submodules:

| Component | Path | Repo |
|-----------|------|------|
| TUI Client | [`tui-client/`](tui-client/) | [ViniZap4/lumi-tui](https://github.com/ViniZap4/lumi-tui) |
| Server | [`server/`](server/) | [ViniZap4/lumi-server](https://github.com/ViniZap4/lumi-server) |
| Web Client | [`web-client/`](web-client/) | [ViniZap4/lumi-web](https://github.com/ViniZap4/lumi-web) |

### Clone with All Submodules

```bash
git clone --recurse-submodules git@github.com:ViniZap4/lumi.git
```

If you already cloned without `--recurse-submodules`:

```bash
git submodule update --init --recursive
```

### Clone Individual Components

```bash
git clone git@github.com:ViniZap4/lumi-tui.git
git clone git@github.com:ViniZap4/lumi-server.git
git clone git@github.com:ViniZap4/lumi-web.git
```

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
- **Settings & Themes** - Config view (c key) with live full-page note preview
  - Multiple dark/light themes with real-time switching
  - Split layout: settings on the left, themed note preview on the right

### Web Client
- **Login Screen** - Password-gated access with session persistence
- **Modern Dark Theme** - Clean, professional interface
- **Folders & Notes** - Browse your note hierarchy
- **Live Preview** - Rendered markdown with syntax highlighting
- **Vim Keybindings** - j/k navigation, / for search
- **Real-time Sync** - Authenticated WebSocket updates

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
- `c` - Settings (theme, editor, display options)
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
- [Submodules Guide](wiki/SUBMODULES.md) - How to work with the monorepo
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
- `c` - Settings
- `e` - External editor
- `esc` - Back

### Settings View (c)
- `j/k` - Move between options
- `h/l` - Cycle option values (themes, toggles)
- `enter` - Select action items
- `esc` - Return to previous view
- Live note preview on the right updates as you change themes

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
POST /api/auth             - Validate token (no middleware)
GET  /api/folders          - List all folders
GET  /api/notes            - List all notes
GET  /api/notes/:id        - Get note by ID
POST /api/notes            - Create note
PUT  /api/notes/:id        - Update note
DELETE /api/notes/:id      - Delete note
WS   /ws?token=<token>     - WebSocket for updates (token required)
```

### Authentication

- **REST**: Include header `X-Lumi-Token: <your-token>` on all requests
- **WebSocket**: Pass token as `?token=` query parameter
- **Login**: `POST /api/auth` with `X-Lumi-Token` header to validate credentials

## 📁 Project Structure

```
lumi/
├── tui-client/     # Go TUI — direct FS + optional server sync
├── server/         # Go HTTP + WebSocket server + peer federation
├── web-client/     # Svelte 5 web app
├── site/           # Landing page (Svelte 5 + Tailwind)
├── wiki/           # Documentation
└── notes/          # Your notes (markdown files)
```

### Architecture

```
┌─────────────────┐          ┌──────────────┐
│   TUI Client    │          │  Web Client  │
│ (Go + Bubbletea)│          │  (Svelte 5)  │
└────────┬────────┘          └──────┬───────┘
         │                          │
         │ direct R/W               │ HTTP + WebSocket
         │ + optional WS            │
         │                          │
         │        ┌─────────────────┘
         │        │
         │  ┌─────▼───────────┐     ┌─────────────┐
         │  │   Go Server     │◄───►│ Peer Servers │
         │  │  REST + WS Hub  │     │  (optional)  │
         │  └─────────┬───────┘     └─────────────┘
         │            │
         └──────┬─────┘
                │
       ┌────────▼──────────┐
       │    Filesystem     │
       │  Markdown + YAML  │
       │   frontmatter     │
       └───────────────────┘
```

- **TUI** reads/writes files directly and can optionally sync with the server via WebSocket
- **Web client** uses REST for CRUD and WebSocket for live updates
- **Servers** can peer with each other for multi-instance federation

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
- [x] Themes

## 📄 License

MIT

## 🙏 Credits

Built with:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Svelte](https://svelte.dev) - Web framework
- [Go](https://golang.org) - Backend language
