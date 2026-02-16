# lumi

A local-first, Markdown-based note-taking ecosystem with terminal and web clients.

## Quick Start

### Docker Compose (Recommended)

```bash
# Copy environment file
cp .env.example .env

# Edit .env with your settings (optional)
# Then start everything:
docker-compose up -d

# Access:
# - Web UI: http://localhost:3000
# - API: http://localhost:8080
```

**Environment variables:**
- `SERVER_PORT` - Server port (default: 8080)
- `WEB_PORT` - Web UI port (default: 3000)
- `LUMI_PASSWORD` - Authentication token (default: dev)
- `NOTES_PATH` - Path to notes directory (default: ./notes)

### TUI Client (Terminal)

```bash
cd tui-client
go build -o lumi

# Run with default (current directory)
./lumi

# Run with specific path
./lumi ../notes
./lumi /path/to/your/notes
```

**Keybindings:**
- `h/l` - Navigate left/right (folders ↔ notes ↔ preview)
- `j/k` - Move down/up (or scroll in preview)
- `enter` - Open folder / Edit note / Follow link
- `e` - Edit note in $EDITOR
- `n` - Create new note
- `d` - Delete note
- `g/G` - Jump to top/bottom
- `v` - Toggle preview (off/partial/full)
- `L` - Show all links in current note
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

**API Endpoints:**
- `GET /api/folders` - List folders
- `GET /api/notes?path=` - List notes
- `GET /api/notes/:id` - Get note
- `POST /api/notes` - Create note
- `PUT /api/notes/:id` - Update note
- `DELETE /api/notes/:id` - Delete note
- `WS /ws` - WebSocket for realtime updates

**Authentication:** Include `X-Lumi-Token: <password>` header

## Documentation

- [Developer Wiki](wiki/DEV.md) - Architecture, tech stack, development guide
- [User Guide](wiki/USER.md) - Installation, usage, configuration

## Project Structure

```
lumi/
├── wiki/           # Documentation
├── tui-client/     # Go terminal client (Bubbletea)
├── server/         # Go API server (HTTP + WebSocket)
├── web-client/     # Svelte web app (coming soon)
└── notes/          # Sample notes
```

## Features

- **Local-first** - Notes are plain Markdown files on disk
- **Vim-like navigation** - Keyboard-driven TUI
- **External editor** - Edit in nvim/vim/emacs/VS Code
- **Realtime sync** - WebSocket updates across clients
- **Simple auth** - Token-based authentication
- **Docker-ready** - Easy deployment

## License

MIT
