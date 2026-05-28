# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

lumi is a local-first, markdown-based note-taking system. **v2** is multi-tenant and collaborative: vaults are portable directories that can be local-only or bound to a self-hosted server with live Yjs CRDT sync. The filesystem (markdown + YAML frontmatter) remains the source of truth on disk; the server projects it through a CRDT for live collaboration. Five independent components live in this monorepo: a Go server (Fiber + Postgres + yrs cgo), a Svelte 5 web client, a Go TUI client, a SwiftUI Apple client, and a Svelte docs site.

The full v2 spec lives at [`SPEC.md`](./SPEC.md) — vault model, data model, capability vocabulary, API surface, phased rollout.

## Build & Run Commands

### Full stack via Docker Compose
```bash
cp .env.example .env                       # edit POSTGRES_PASSWORD, LUMI_ADMIN_*
git submodule update --init --recursive    # required for server cgo build
docker compose up -d
# Web UI: http://localhost:3000   API: http://localhost:8080
# Rebuild web after changing LUMI_SERVER_URL: docker compose build web
```

### Server (Go 1.25, cgo yrs)
```bash
cd server
git submodule update --init --recursive    # pulls third_party/y-crdt
make libyrs                                 # cargo build yffi static lib
docker compose up -d postgres               # local Postgres
make migrate                                # apply migrations
make run                                    # http://localhost:8080
# Tests
make test                                   # unit
make test-integration                       # testcontainers-go Postgres
make smoke                                  # docker compose smoke
# Subcommands
go run ./cmd/lumi-server migrate up|down N|status
go run ./cmd/lumi-server version
```

### TUI Client (Go 1.23+)
```bash
cd tui-client
go build -o lumi
./lumi                                      # vault picker if any vaults registered
./lumi <dir>                                # open a local vault or markdown dir
./lumi <file.md>                            # open a single note
./lumi login <server-url>                   # sign in to a lumi-server v2
./lumi accounts [--verify]                  # list signed-in servers
./lumi vaults                               # list registered vaults
./lumi vault link <path>                    # register dir as vault
./lumi vault clone <url>/<slug> [<path>]    # pull a server vault locally
./lumi vault unlink <slug>                  # remove from registry
./lumi vault sync <slug>                    # snapshot+diff round-trip
./lumi vault manage                         # interactive vault TUI
```

### Web Client (Svelte 5 + Vite 7)
```bash
cd web-client
npm install
npm run dev                                 # http://localhost:5173
npm run build                               # production build to dist/
npm test                                    # vitest suites
```

### Apple Client (SwiftUI, Swift 6)
```bash
cd apple-client
make bootstrap                              # installs xcodegen
make open                                   # generates project + opens Xcode
make test                                   # swift test
```

### Site (Svelte 5 + Tailwind 4)
```bash
cd site
npm install
npm run dev                                 # http://localhost:5173
```

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

### Vault model

- A **vault** is a portable directory: notes + a `.lumi/` metadata folder.
- Vaults are **local-only** or **server-bound**. Clients can track many vaults across many servers; each binding has its own account.
- The server is **multi-tenant**: it hosts many vaults with per-vault custom roles. Identity is server-scoped (no central SSO).
- **FS-CRDT contract**: external markdown edits are detected via fsnotify and diff-merged into the CRDT (`server/internal/fswatch`), then broadcast to live subscribers.

### Auth

- Session-based: `POST /api/auth/login` issues a session cookie + token; subsequent REST sends `X-Lumi-Token`, WebSocket sends `?token=`.
- Sessions in Postgres, bcrypt cost 12, 30-day TTL.
- First-boot bootstrap via `LUMI_ADMIN_USERNAME` + `LUMI_ADMIN_PASSWORD` (when users table is empty). Otherwise via invite-link signup.
- Per-vault custom roles gate every mutation; seed roles: Admin / Editor / Viewer / Commenter.

### Server (Go 1.25, Fiber, pgx/v5)

Layout (`server/`):
- `cmd/lumi-server/` — binary entrypoint, env config, Fiber app wiring
- `internal/auth/` — sessions, login, password hashing, middleware, bootstrap
- `internal/users/`, `vaults/`, `roles/`, `members/`, `invites/`, `notes/`, `audit/` — domain services + handlers (each registers Fiber routes)
- `internal/crdt/` — yrs cgo wrapper, doc registry, snapshot/update persistence
- `internal/wsync/` — Yjs WebSocket hub, awareness fan-out, debounced FS mirror
- `internal/fswatch/` — fsnotify watcher, diff-merges external edits into the CRDT
- `internal/storage/fs/` — SafeJoin, atomic writes, vault.yaml
- `internal/storage/pg/` — sqlc-generated queries
- `internal/domain/` — canonical types, errors, capability vocabulary
- `migrations/` — SQL migrations (golang-migrate)
- `third_party/y-crdt/` — submodule pinned to yrs v0.26.0; `make libyrs` builds `libyrs.a` for cgo linking

### Web Client (Svelte 5 + TypeScript)

- `src/App.svelte` — auth gate, vault selection, route to vault home
- `src/lib/auth.svelte.ts` — session + token state
- `src/lib/vaults.svelte.ts` — vault list + selection
- `src/lib/vaultmembers.svelte.ts` — member/role admin
- `src/lib/capabilities.ts` — capability matching
- `src/lib/notes.svelte.ts` — note CRUD
- `src/lib/editor-session.svelte.ts` — lazy-loaded Yjs + CodeMirror editor + vim
- `src/lib/uistate.svelte.ts` — UI state ($state runes)
- `src/lib/api.ts` — REST client (auth-aware)
- `src/lib/ws.ts` — Yjs WebSocket sync (awareness/presence)
- `src/lib/markdown.ts` — markdown renderer (DOMPurify-sanitised)
- `src/views/{LoginView,VaultsView,VaultHomeView}.svelte` — entry views

CSP enforced via meta tag + nginx headers; suites under `*.test.ts` run via vitest.

### TUI Client (Bubbletea / Elm)

- Top-level CLI dispatch in `main.go`: subcommands `login`, `accounts`, `vaults`, `vault link|clone|unlink|sync|manage`. Default action is the vault picker or directory open.
- `cmd_*.go` — subcommand handlers + tests
- `account/` — accounts.yaml + vaults.yaml readers/writers, last-opened bumps
- `sync/` — REST snapshot/diff client (no live CRDT — `$EDITOR` is opaque)
- `ui/` — Bubbletea Model/Update/View, vim cursor, theme styles
- `config/`, `theme/`, `editor/`, `image/`, `filesystem/`, `domain/` — as in v1
- `cmd_vault_manage.go` — interactive vault TUI, hands off to main TUI via `syscall.Exec`

### Apple Client (SwiftUI, Swift 6)

- `Sources/LumiKit/` — platform-agnostic core: Domain, Filesystem, Network
- `Sources/LumiUI/` — shared SwiftUI: theme, editor, markdown render
- `App/` — `@main` app target with `AppState` + `RootView`
- CRDT via [yswift](https://github.com/y-crdt/yswift) (Y.Doc actor wrapper)
- Live awareness/presence with self-echo filter + reconnect-stable client ID
- TextKit 2 native vim engine

### Site (Svelte 5 + Tailwind 4)

- SPA with path routing, 12 doc pages, 12-theme catalog, Vercel deploy
- Note: site is still on v1 docs as of 2026-05-28; a v2 docs rewrite is pending

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

Plain markdown files (no frontmatter) are accepted by both Go clients; on-disk format is preserved on save unless metadata is explicitly added through the UI.

## Conventions

- **Commit messages**: Conventional commits with module scope — `feat(server, notes): ...`, `fix(web, phase-4.x): ...`, `docs(...)`, `refactor(...)`. Push immediately after every commit.
- **Go style**: `gofmt`/`goimports`, meaningful package names (`auth`, `crdt`, `wsync`, not `utils`), explicit error handling, `domain.ErrValidation` for input errors.
- **Svelte style**: Svelte 5 runes (`$state`, `$derived`, `$effect`); one component per file; lazy-load heavy chunks (Yjs, CodeMirror) on first edit.
- **Auth**: session token via `X-Lumi-Token` (REST) or `?token=` (WS). Per-vault capabilities checked at every handler.
- **Tests**: server has unit + integration (testcontainers-go) + smoke; web client has vitest suites. v1's "no automated tests" rule is superseded for security-critical and CRDT paths.
- **Pillars** (applied to every decision): Security/LGPD → performance → DX → scale → UX → UI → QA.

## Submodules

Each component is a separate repo linked as a git submodule:

| Path | Repo |
|------|------|
| `server/` | `ViniZap4/lumi-server` |
| `web-client/` | `ViniZap4/lumi-web` |
| `tui-client/` | `ViniZap4/lumi-tui` |
| `apple-client/` | `ViniZap4/lumi-apple` |
| `site/` | `ViniZap4/lumi-site` |

All components track `main` as of the v2 cutover (2026-05-28).

Clone with submodules: `git clone --recurse-submodules`. The server has its own nested submodule (`third_party/y-crdt`) required for cgo; the `--recursive` flag covers it.
