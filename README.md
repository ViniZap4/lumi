# lumi

A local-first, Markdown-based note-taking system with a Go TUI client, Go HTTP/WebSocket server, and Svelte 5 web client. The filesystem is the single source of truth — no database, just markdown files with YAML frontmatter.

> **Documentation**: [lumi-note.vercel.app/#/docs](https://lumi-note.vercel.app/#/docs)

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

## Features

- **Local-first**: Notes are plain markdown files on disk — yours forever
- **TUI client**: Vim keybindings, inline image rendering, 12 built-in themes, external editor integration
- **Web client**: CodeMirror editor with vim mode, real-time sync, media embeds (video, PDF, YouTube, Vimeo)
- **Real-time sync**: WebSocket-based live updates between server and clients
- **Peer federation**: Multiple server instances can sync via `LUMI_PEERS`
- **Wiki-links**: `[[note-id]]` linking between notes
- **Media support**: Images, video thumbnails (ffmpeg), PDF, and embed rendering

## Repository Structure

Each component lives in its own repo, linked here as git submodules:

| Component | Path | Repo | Stack |
|-----------|------|------|-------|
| TUI Client | `tui-client/` | [ViniZap4/lumi-tui](https://github.com/ViniZap4/lumi-tui) | Go 1.23+, Bubbletea, Lipgloss |
| Server | `server/` | [ViniZap4/lumi-server](https://github.com/ViniZap4/lumi-server) | Go 1.21+, Gorilla WebSocket |
| Web Client | `web-client/` | [ViniZap4/lumi-web](https://github.com/ViniZap4/lumi-web) | Svelte 5, Vite 7, CodeMirror 6 |
| Site | `site/` | [ViniZap4/lumi-site](https://github.com/ViniZap4/lumi-site) | Svelte 5, Vite 7, Tailwind 4 |

### Clone

```bash
git clone --recurse-submodules git@github.com:ViniZap4/lumi.git
```

If you already cloned without `--recurse-submodules`:

```bash
git submodule update --init --recursive
```

## Prerequisites

| Dependency | Required by | Install |
|------------|-------------|---------|
| Go 1.23+ | TUI client | [go.dev/dl](https://go.dev/dl/) |
| Go 1.21+ | Server | [go.dev/dl](https://go.dev/dl/) |
| Node 18+ / npm | Web client, Site | [nodejs.org](https://nodejs.org/) |
| Docker + Compose | Docker deployment | [docker.com](https://www.docker.com/) |

**Optional (TUI media rendering):**

| Tool | Purpose | Install (macOS) |
|------|---------|-----------------|
| timg | Terminal image display | `brew install timg` |
| chafa | Terminal image display (fallback) | `brew install chafa` |
| viu | Terminal image display (fallback) | `brew install viu` |
| ffmpeg | Video thumbnail extraction | `brew install ffmpeg` |

The TUI tries renderers in order: Kitty protocol > iTerm2 protocol > timg > chafa > viu. Install at least one for inline image previews.

## Quick Start

### Docker Compose (recommended)

```bash
cp .env.example .env    # edit LUMI_PASSWORD before starting
docker compose up -d

# Web UI: http://localhost:3000
# API:    http://localhost:8080
```

> `VITE_LUMI_SERVER_URL` is baked at build time. After changing it, rebuild: `docker compose build web`

### TUI Client (standalone)

```bash
cd tui-client
go build -o lumi
./lumi ../notes          # pass notes dir as arg, or set LUMI_NOTES_DIR
```

### Server (standalone)

```bash
cd server
LUMI_ROOT=../notes LUMI_PASSWORD=secret go run main.go
```

### Web Client (standalone)

```bash
cd web-client
npm install
npm run dev              # http://localhost:5173
```

## Environment Variables

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `LUMI_ROOT` | `./notes` | Path to the notes directory |
| `LUMI_PASSWORD` | `dev` | Auth token for REST and WebSocket |
| `LUMI_PORT` | `8080` | HTTP listen port |
| `LUMI_SERVER_ID` | auto | Unique ID for peer sync |
| `LUMI_PEERS` | — | Comma-separated peer server URLs |

### Web Client

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_LUMI_SERVER_URL` | `http://localhost:8080` | API server URL (build-time only) |

### Docker Compose

| Variable | Default | Description |
|----------|---------|-------------|
| `LUMI_PASSWORD` | `dev` | Shared auth token |
| `LUMI_PORT` | `8080` | Host port for the API |
| `NOTES_PATH` | `./notes` | Host path to notes directory |
| `WEB_PORT` | `3000` | Host port for web UI |
| `LUMI_SERVER_URL` | `http://localhost:8080` | API URL baked into web bundle |

## Note Format

```markdown
---
id: my-note-id
title: My Note Title
created_at: 2026-02-16T10:00:00Z
updated_at: 2026-02-16T10:00:00Z
tags:
  - example
---

# My Note Title

Your content here with **markdown** formatting.

Link to other notes: [[other-note-id]]
```

Notes are auto-generated with an `id` derived from the title (lowercase, non-alphanumeric chars replaced with hyphens).

## API Overview

All REST endpoints require `X-Lumi-Token` header (or `?token=` query param). WebSocket uses `?token=` query param.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth` | Validate token (login) |
| `GET` | `/api/folders` | List folders |
| `GET` | `/api/notes` | List all notes |
| `GET` | `/api/notes/:id` | Get note by ID |
| `POST` | `/api/notes` | Create note |
| `PUT` | `/api/notes/:id` | Update note |
| `DELETE` | `/api/notes/:id` | Delete note |
| `POST` | `/api/notes/:id/move` | Move note |
| `GET` | `/api/files/` | Serve static files (images) |
| `WS` | `/ws?token=` | Real-time updates |
| `WS` | `/ws/peer?server_id=` | Peer synchronization |

## License

MIT

## Credits

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Svelte](https://svelte.dev), [CodeMirror](https://codemirror.net), and [Go](https://golang.org).
