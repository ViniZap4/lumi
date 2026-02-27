# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

lumi is a local-first, markdown-based note-taking system with three independent components: a Go TUI client, a Go HTTP/WebSocket server, and a Svelte web client. The filesystem (markdown files with YAML frontmatter) is the single source of truth.

## Build & Run Commands

### TUI Client (Go 1.23+)
```bash
cd tui-client
go build -o lumi                    # Build
./lumi ../notes                     # Run (pass notes dir as arg, or set LUMI_NOTES_DIR)
go run main.go ../notes             # Build + run
```

### Server (Go 1.21+)
```bash
cd server
LUMI_ROOT=../notes LUMI_PASSWORD=dev go run main.go
# Env vars: LUMI_ROOT (notes dir), LUMI_PASSWORD (auth token), LUMI_PORT (default: 8080)
```

### Web Client (Svelte 5 + Vite 7)
```bash
cd web-client
npm install
npm run dev                         # Dev server on http://localhost:5173
npm run build                       # Production build to dist/
```

### Docker Compose (Full Stack)
```bash
cp .env.example .env                # Set LUMI_PASSWORD, ports, notes path
docker compose up -d                # Web on :3000, API on :8080
# VITE_LUMI_SERVER_URL is a build arg (baked at image build, not runtime)
# Rebuild web after changing: docker compose build web
```

### Testing
No automated test suites exist. TUI is tested manually. Server and filesystem packages are tested manually. Format Go code with `gofmt` and `goimports`.

## Architecture

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

- **TUI client reads/writes the filesystem directly** and can optionally connect to the server via WebSocket (`sync/` package) for real-time sync notifications. Configured per-folder in `<notesDir>/.lumi/config.yaml` with `server_url` and `server_token`.
- **Web client connects to the server** via REST for CRUD and WebSocket for real-time updates.
- **Server supports peer-to-peer sync** (`peer/` package) — multiple server instances can federate via `LUMI_PEERS` env var. Peers exchange events over WebSocket with origin tracking to prevent echo loops.
- Both Go components share the same `domain/` package pattern (Note, Folder structs) and `filesystem/` package for YAML frontmatter parsing.

### Component Layout

Each Go component (`server/`, `tui-client/`) follows the same layered structure:
- `domain/` — Core types: `Note` (ID, Title, CreatedAt, UpdatedAt, Tags, Path, Content), `Folder`
- `filesystem/` — Markdown file I/O, YAML frontmatter parsing (`parser.go`), CRUD operations
- TUI-specific: `ui/` (Bubbletea views), `sync/` (WebSocket client), `config/` (global + per-folder config)
- Server-specific: `http/` (REST handlers), `ws/` (WebSocket hub), `auth/` (token middleware), `peer/` (server federation)

### TUI Client (Bubbletea / Elm Architecture)
- Main model and logic in `ui/simple.go`, styles in `ui/styles.go`
- Views: Home (search), Tree (file browser), FullNote (note display with split view)
- Modes: Input modal, Search modal, Tree modal
- External editor integration via `editor/editor.go` (uses `$EDITOR`, falls back to nvim)
- Image rendering via `image/render.go` (fallback chain: timg → chafa → viu)

### Web Client (Svelte 5)
- `App.svelte` is the root component — gates all views behind authentication
- `views/LoginView.svelte` — password login screen with session persistence (localStorage)
- `lib/store.svelte.js` — reactive store with auth state (`authenticated`, `login`, `logout`, `checkAuth`)
- `lib/api.js` — HTTP client with dynamic token (`setToken`, `getToken`, `login`)
- `lib/ws.js` — WebSocket client with auto-reconnect and token auth via query param

## Note Format

```markdown
---
id: example-note
title: Example Note
created_at: 2026-02-16T11:00:00-03:00
updated_at: 2026-02-16T11:05:00-03:00
tags:
  - example
---

# Markdown content here
```

## Conventions

- **Commit messages**: Conventional commits — `feat:`, `fix:`, `docs:`, `refactor:`
- **Go style**: `gofmt`/`goimports`, meaningful package names (`domain`, `filesystem`, not `utils`), explicit error handling
- **Svelte style**: One component per file, reactive declarations over manual updates
- **Auth**: REST requests require `X-Lumi-Token` header matching `LUMI_PASSWORD` env var. WebSocket requires `?token=` query param. `POST /api/auth` validates credentials (no middleware). Web client persists token in localStorage.

## Submodules

Each component is a separate repo linked as a git submodule:

| Path | Repo |
|------|------|
| `tui-client/` | `ViniZap4/lumi-tui` |
| `server/` | `ViniZap4/lumi-server` |
| `web-client/` | `ViniZap4/lumi-web` |
| `site/` | `ViniZap4/lumi-site` |

Clone with submodules: `git clone --recurse-submodules`.
Update submodules: `git submodule update --init --recursive`.
