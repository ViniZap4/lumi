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
cp .env.example .env
docker-compose up -d                # Web on :3000, API on :8080
```

### Testing
No automated test suites exist. TUI is tested manually. Server and filesystem packages are tested manually. Format Go code with `gofmt` and `goimports`.

## Architecture

```
┌─────────────────┐          ┌──────────────┐
│   TUI Client    │          │  Web Client  │
│ (Go + Bubbletea)│          │  (Svelte 5)  │
│ direct FS R/W   │          │  HTTP + WS   │
└────────┬────────┘          └──────┬───────┘
         │                          │
         │ filesystem               │ HTTP/WebSocket
         │                          │
         │                  ┌───────▼─────────┐
         │                  │   Go Server      │
         │                  │  REST + WS Hub   │
         │                  └────────┬─────────┘
         │                           │
         └───────────┬───────────────┘
                     │
            ┌────────▼──────────┐
            │   Filesystem      │
            │ Markdown + YAML   │
            │   frontmatter     │
            └───────────────────┘
```

- **TUI client reads/writes the filesystem directly** — it does not go through the server.
- **Web client connects to the server** which provides REST endpoints and a WebSocket hub for real-time sync.
- Both components share the same `domain/` package pattern (Note, Folder structs) and `filesystem/` package for YAML frontmatter parsing.

### Component Layout

Each Go component (`server/`, `tui-client/`) follows the same layered structure:
- `domain/` — Core types: `Note` (ID, Title, CreatedAt, UpdatedAt, Tags, Path, Content), `Folder`
- `filesystem/` — Markdown file I/O, YAML frontmatter parsing (`parser.go`), CRUD operations
- Component-specific: `ui/` (TUI views), `http/` + `ws/` + `auth/` (server)

### TUI Client (Bubbletea / Elm Architecture)
- Main model and logic in `ui/simple.go`, styles in `ui/styles.go`
- Views: Home (search), Tree (file browser), FullNote (note display with split view)
- Modes: Input modal, Search modal, Tree modal
- External editor integration via `editor/editor.go` (uses `$EDITOR`, falls back to nvim)
- Image rendering via `image/render.go` (fallback chain: timg → chafa → viu)

### Web Client (Svelte 5)
- `AppFinal.svelte` is the active app component (3-panel layout)
- `lib/api.js` — HTTP client wrapping fetch
- `lib/ws.js` — WebSocket client with auto-reconnect

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
- **Auth**: All server requests require `X-Lumi-Token` header matching `LUMI_PASSWORD` env var

## Subtree Remotes

Each component is published as a standalone repo via `git subtree`. The monorepo (`ViniZap4/lumi`) is the primary repo.

| Remote | Repo | Prefix |
|--------|------|--------|
| `lumi-tui` | `ViniZap4/lumi-tui` | `tui-client/` |
| `lumi-server` | `ViniZap4/lumi-server` | `server/` |
| `lumi-web` | `ViniZap4/lumi-web` | `web-client/` |

### Push to standalone repos
```bash
git subtree push --prefix=tui-client lumi-tui main
git subtree push --prefix=server lumi-server main
git subtree push --prefix=web-client lumi-web main
```

### Pull from standalone repos
```bash
git subtree pull --prefix=tui-client lumi-tui main --squash
git subtree pull --prefix=server lumi-server main --squash
git subtree pull --prefix=web-client lumi-web main --squash
```
