# lumi Developer Wiki

## Architecture Overview

lumi is a **local-first note-taking ecosystem** with three main components:

```
┌─────────────┐
│  TUI Client │ (Go + Bubbletea)
│  (Terminal) │ ─┐
└─────────────┘  │
                 │    ┌──────────┐      ┌─────────────┐
                 ├───→│  Server  │◄────→│ Filesystem  │
                 │    │ (Go API) │      │ (Markdown)  │
┌─────────────┐  │    └──────────┘      └─────────────┘
│ Web Client  │  │         ▲
│  (Svelte)   │ ─┘         │
└─────────────┘      WebSocket (realtime)
```

### Source of Truth

**The filesystem is the source of truth.** All notes are plain Markdown files with YAML frontmatter stored in a configurable root directory.

### Components

1. **TUI Client** (`tui-client/`)
   - Terminal interface using Bubbletea
   - Reads/writes directly to filesystem
   - Opens notes in $EDITOR (nvim fallback)
   - Vim-like keybindings

2. **Server** (`server/`)
   - HTTP API for CRUD operations
   - WebSocket for realtime sync
   - Docker-first deployment
   - Simple token authentication

3. **Web Client** (`web-client/`)
   - Svelte-based UI
   - Connects to server via HTTP + WebSocket
   - Docker-ready static build

## Tech Stack

### TUI Client
- **Language**: Go 1.21+
- **Libraries**:
  - `bubbletea` - TUI framework (Elm architecture)
  - `bubbles` - Reusable components (lists, inputs, etc)
  - `lipgloss` - Styling and layout
  - `goldmark` - Markdown parsing (if needed)

### Server
- **Language**: Go 1.21+
- **Libraries**:
  - `net/http` - HTTP server
  - `gorilla/websocket` - WebSocket support
  - `gopkg.in/yaml.v3` - Frontmatter parsing
- **Deployment**: Docker with volume mounts

### Web Client
- **Framework**: Svelte + Vite
- **Libraries**:
  - Native WebSocket API
  - Markdown editor (TBD: CodeMirror or similar)
- **Deployment**: Docker (Nginx or static server)

## Data Model

### Note Structure (Filesystem)

```markdown
---
id: 2026-example-note
title: Example Note
created_at: 2026-02-16T11:00:00-03:00
updated_at: 2026-02-16T11:05:00-03:00
tags:
  - example
  - markdown
---

# Content

This is the note content in **Markdown**.

Links: [[2026-another-note]] or [relative](./other.md)
```

### Note Structure (Go Domain)

```go
type Note struct {
    ID        string    `yaml:"id"`
    Title     string    `yaml:"title"`
    CreatedAt time.Time `yaml:"created_at"`
    UpdatedAt time.Time `yaml:"updated_at"`
    Tags      []string  `yaml:"tags"`
    Path      string    `yaml:"-"` // Filesystem path
    Content   string    `yaml:"-"` // Body after frontmatter
}
```

## Project Structure

```
lumi/
├── wiki/                    # Documentation
│   ├── DEV.md              # This file
│   └── USER.md             # User guide
├── tui-client/             # Terminal client
│   ├── main.go
│   ├── go.mod
│   ├── domain/             # Core types (Note, Folder)
│   ├── filesystem/         # File I/O, frontmatter parsing
│   ├── ui/                 # Bubbletea models and views
│   │   ├── app.go          # Main app model
│   │   ├── folders.go      # Folder panel
│   │   ├── notes.go        # Notes list panel
│   │   └── styles.go       # Lipgloss styles
│   └── editor/             # External editor integration
├── server/                 # API server
│   ├── main.go
│   ├── go.mod
│   ├── Dockerfile
│   ├── domain/             # Shared types
│   ├── filesystem/         # File operations
│   ├── http/               # HTTP handlers
│   ├── ws/                 # WebSocket hub
│   └── auth/               # Token authentication
└── web-client/             # Web interface
    ├── package.json
    ├── Dockerfile
    ├── src/
    │   ├── lib/
    │   │   ├── api.js      # HTTP client
    │   │   └── ws.js       # WebSocket client
    │   └── components/
    └── vite.config.js
```

## Development Workflow

### TUI Client Development

```bash
cd tui-client
go mod init github.com/vinizap/lumi/tui-client
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go run main.go
```

### Server Development

```bash
cd server
go mod init github.com/vinizap/lumi/server
go run main.go
```

Environment variables:
- `LUMI_ROOT` - Notes directory (default: `./notes`)
- `LUMI_PASSWORD` - Auth token (default: `dev`)
- `LUMI_PORT` - Server port (default: `8080`)

### Web Client Development

```bash
cd web-client
npm install
npm run dev
```

Environment variables:
- `VITE_LUMI_SERVER_URL` - Server URL (default: `http://localhost:8080`)

## API Design

### HTTP Endpoints

```
GET    /api/folders              # List all folders
GET    /api/folders/:path/notes  # List notes in folder
GET    /api/notes/:id            # Get note by ID
POST   /api/notes                # Create note
PUT    /api/notes/:id            # Update note
DELETE /api/notes/:id            # Delete note
```

### WebSocket Protocol

```
Client → Server:
{
  "type": "subscribe",
  "path": "/folder/subfolder"  // Optional: subscribe to specific path
}

Server → Client:
{
  "type": "note_created" | "note_updated" | "note_deleted",
  "note": { ... }
}
```

### Authentication

All requests require header:
```
X-Lumi-Token: <LUMI_PASSWORD>
```

## Coding Standards

### Go
- Use `gofmt` and `goimports`
- Keep functions small and focused
- Prefer composition over inheritance
- Use meaningful package names (`domain`, `filesystem`, not `utils`)
- Handle errors explicitly, don't ignore them
- Use context for cancellation and timeouts

### Svelte
- One component per file
- Keep components under 200 lines
- Use stores for shared state
- Prefer reactive declarations over manual updates

### General
- Write self-documenting code
- Add comments for "why", not "what"
- Keep commits small and atomic
- Write commit messages in imperative mood

## Git Workflow

### Commit Structure

```
feat: add folder navigation to TUI
fix: handle missing frontmatter gracefully
docs: update API endpoints in dev wiki
refactor: extract frontmatter parsing to separate package
```

### Suggested Commit Boundaries

1. Domain models and types
2. Filesystem operations
3. UI components (one panel at a time)
4. API endpoints (one resource at a time)
5. WebSocket implementation
6. Docker configuration

## Testing Strategy

- **TUI**: Manual testing in terminal (automated TUI testing is complex)
- **Server**: Unit tests for handlers, integration tests for API
- **Filesystem**: Unit tests for parsing and file operations
- **Web**: Component tests with Vitest

## Future Enhancements

- [ ] Image display in terminal (Kitty protocol)
- [ ] Full-text search
- [ ] Note templates
- [ ] Backlinks (notes that link to current note)
- [ ] Graph view of note connections
- [ ] Mobile clients (Swift, Kotlin)
- [ ] Encryption at rest
- [ ] Git integration for versioning
