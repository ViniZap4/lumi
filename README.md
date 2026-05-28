# lumi

A local-first, markdown-based note-taking system. **v2** is multi-tenant and collaborative: an Obsidian-style vault is either fully local or bound to a self-hosted server with live Yjs CRDT sync. The filesystem (markdown + YAML frontmatter) stays the source of truth on disk; the server projects it through a CRDT for real-time collaboration.

> Full v2 spec: [`SPEC.md`](./SPEC.md) · Docs site: [lumi-note.vercel.app/#/docs](https://lumi-note.vercel.app/#/docs)

## Architecture

```
┌─────────────────┐  ┌──────────────┐  ┌──────────────────┐
│   TUI Client    │  │  Web Client  │  │   Apple Client   │
│ (Go + Bubbletea)│  │  (Svelte 5)  │  │    (SwiftUI)     │
└────────┬────────┘  └──────┬───────┘  └────────┬─────────┘
         │                  │                   │
  snapshot+diff       Yjs WS + REST       Yjs WS + REST
  REST sync           (live awareness)    (yswift + presence)
         │                  │                   │
         └──────────────────┼───────────────────┘
                            │
                  ┌─────────▼──────────┐
                  │  lumi-server (v2)  │
                  │  Fiber + cgo yrs   │
                  └─────────┬──────────┘
                            │
              ┌─────────────┴────────────┐
              │                          │
       ┌──────▼──────┐         ┌─────────▼─────────┐
       │  Postgres   │         │   Filesystem      │
       │ users/vaults│         │  <vault>/notes/   │
       │ roles/audit │         │  Markdown + YAML  │
       │ Yjs state   │         │ (source of truth) │
       └─────────────┘         └───────────────────┘
```

- **Vault = unit of work.** Portable directory containing notes + a `.lumi/` metadata folder. Local-only or bound to a server. Sync is opt-in per vault.
- **Multi-tenant server.** Fiber + Postgres host many vaults with per-vault custom roles, invite-link signup, session auth.
- **Yjs CRDT** drives live collaboration. Web and Apple clients exchange Yjs updates + awareness/presence over WebSocket. TUI uses snapshot+diff REST (no live cursors — `$EDITOR` is opaque).
- **FS-CRDT bridge.** External markdown edits are detected via fsnotify and diff-merged into the CRDT, then broadcast to all subscribers.

## Repository

Each component is a separate repo, linked here as a git submodule:

| Component | Path | Repo | Stack |
|-----------|------|------|-------|
| Server | `server/` | [ViniZap4/lumi-server](https://github.com/ViniZap4/lumi-server) | Go 1.25, Fiber v2, pgx/v5, yrs (cgo) |
| Web client | `web-client/` | [ViniZap4/lumi-web](https://github.com/ViniZap4/lumi-web) | Svelte 5, Vite 7, CodeMirror 6, Yjs |
| TUI client | `tui-client/` | [ViniZap4/lumi-tui](https://github.com/ViniZap4/lumi-tui) | Go 1.23+, Bubbletea |
| Apple client | `apple-client/` | [ViniZap4/lumi-apple](https://github.com/ViniZap4/lumi-apple) | SwiftUI, yswift (iOS / iPadOS / macOS / visionOS) |
| Site | `site/` | [ViniZap4/lumi-site](https://github.com/ViniZap4/lumi-site) | Svelte 5, Tailwind 4 |

### Clone

```bash
git clone --recurse-submodules git@github.com:ViniZap4/lumi.git
```

If already cloned without submodules:

```bash
git submodule update --init --recursive
```

The server depends on `third_party/y-crdt` (a submodule of `lumi-server`); the recursive flag is required.

## Prerequisites

| Dependency | Required by | Install |
|------------|-------------|---------|
| Docker + Compose | Full-stack deployment | [docker.com](https://www.docker.com/) |
| Go 1.25 | Server (standalone) | [go.dev/dl](https://go.dev/dl/) |
| Rust (stable) | Server (cgo yrs build) | [rustup.rs](https://rustup.rs/) |
| Postgres 16 | Server (standalone) | bundled in compose |
| Go 1.23+ | TUI client | [go.dev/dl](https://go.dev/dl/) |
| Node 20+ / npm | Web client, Site | [nodejs.org](https://nodejs.org/) |
| Xcode 16, Swift 6 | Apple client | macOS only |

Optional TUI media: `timg`, `chafa`, `viu` for images; `ffmpeg` for video thumbnails.

## Quick Start

### Docker Compose (recommended)

```bash
cp .env.example .env                       # edit POSTGRES_PASSWORD, LUMI_ADMIN_*
git submodule update --init --recursive
docker compose up -d

# Web UI: http://localhost:3000
# API:    http://localhost:8080
```

On first boot, if `LUMI_ADMIN_USERNAME` + `LUMI_ADMIN_PASSWORD` are set and the users table is empty, the server seeds a single admin account. Otherwise, sign up via an invite link.

> `VITE_LUMI_SERVER_URL` is baked at image build time. After changing `LUMI_SERVER_URL`, rebuild: `docker compose build web`.

### Server (standalone)

```bash
cd server
git submodule update --init --recursive    # pulls third_party/y-crdt
make libyrs                                 # builds libyrs.a via cargo
docker compose up -d postgres               # local Postgres on :5432
make migrate                                # apply migrations
make run                                    # http://localhost:8080
```

### Web client (standalone)

```bash
cd web-client
npm install
npm run dev                                 # http://localhost:5173
```

### TUI client

```bash
cd tui-client
go build -o lumi
./lumi                                      # vault picker if vaults registered
./lumi ../vaults/personal                   # open a local vault
./lumi login https://lumi.example.com       # sign in to a server
./lumi vault clone https://lumi.example.com/personal
./lumi vault manage                         # interactive vault TUI
```

### Apple client

```bash
cd apple-client
make bootstrap
make open                                   # opens Xcode
```

## Environment Variables

### Server (`docker compose` / `lumi-server`)

| Variable | Default | Description |
|----------|---------|-------------|
| `LUMI_DATABASE_URL` | — (required) | Postgres connection string |
| `LUMI_ROOT` | — (required) | Vault root directory |
| `LUMI_PORT` | `8080` | HTTP listen port |
| `LUMI_BIND_ADDR` | `0.0.0.0` | Listen address |
| `LUMI_REQUIRE_TLS` | `true` | Refuse non-loopback bind without upstream TLS |
| `LUMI_ALLOWED_ORIGINS` | — | CSV of allowed CORS origins |
| `LUMI_REGISTRATION` | `invite-only` | `invite-only` or `open` |
| `LUMI_AUTO_MIGRATE` | `false` | Apply pending migrations at startup |
| `LUMI_ADMIN_USERNAME` | — | First-boot admin bootstrap (with password) |
| `LUMI_ADMIN_PASSWORD` | — | First-boot admin bootstrap (with username) |
| `LUMI_PUBLIC_BASE_URL` | — | Public URL for invite links |
| `LUMI_AUDIT_RETENTION_DAYS` | `90` | LGPD audit retention |
| `LUMI_TOS_VERSION` | — | Records consent at signup if set with privacy version |
| `LUMI_PRIVACY_VERSION` | — | Records consent at signup if set with ToS version |
| `LUMI_LOG_FORMAT` | `json` | `json` or `console` |
| `LUMI_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

### Web client

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_LUMI_SERVER_URL` | `http://localhost:8080` | API server URL (build-time only) |

### Compose

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | `lumi` / `lumi` / `lumi` | Bundled Postgres credentials |
| `LUMI_PORT` | `8080` | Host port for the API |
| `WEB_PORT` | `3000` | Host port for the web UI |
| `VAULTS_PATH` | `./vaults` | Host path mounted at `/vaults` |
| `LUMI_SERVER_URL` | `http://localhost:8080` | URL baked into the web bundle |

See `.env.example` for the complete list.

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

Markdown body with wiki-links: `[[other-note-id]]`.
```

Plain markdown files (no frontmatter) are accepted — lumi preserves the on-disk format on save unless metadata is explicitly added through the UI.

## API

REST + WebSocket served by `lumi-server`. Full surface and protocol live in [`SPEC.md`](./SPEC.md) (API surface section).

Auth is session-based: `POST /api/auth/login` returns a session cookie + token; subsequent calls send `X-Lumi-Token` (REST) or `?token=` (WS). Per-vault custom roles gate every mutation; the seed roles are Admin / Editor / Viewer / Commenter.

## Pillars

Applied to every design decision, in order: Security/LGPD → performance → DX → scale → UX → UI → QA. LGPD compliance is a hard constraint.

## License

MIT

## Credits

Built with [Fiber](https://gofiber.io), [Bubbletea](https://github.com/charmbracelet/bubbletea), [Svelte](https://svelte.dev), [CodeMirror](https://codemirror.net), [Yjs](https://yjs.dev) / [y-crdt](https://github.com/y-crdt/y-crdt), and [Go](https://golang.org).
