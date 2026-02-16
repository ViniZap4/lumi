# lumi

A local-first, Markdown-based note-taking ecosystem with terminal and web clients.

## Quick Start

### TUI Client (Terminal)

```bash
cd tui-client
go build -o lumi
./lumi
```

**Keybindings:**
- `j/k` - Move down/up
- `h/l` - Navigate folders / switch panels
- `e` or `Enter` - Edit note in $EDITOR
- `n` - Create new note
- `d` - Delete note
- `g/G` - Jump to top/bottom
- `tab` - Switch panels
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
