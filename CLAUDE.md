# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

lumi is a local-first, markdown-based note-taking system. Notes live in **books** ("livros" in the pt-BR UI; `vault` in code and API). A book is a portable directory that is either local-only or bound to a self-hosted server with live Yjs CRDT sync. The filesystem (markdown + YAML frontmatter) stays the source of truth on disk; the server projects it through a CRDT for live collaboration. A client can hold accounts on many servers, and servers can federate books with each other.

Five components live in this monorepo as git submodules: a Go server (Fiber + Postgres + yrs cgo) with an admin API, a Svelte 5 web client (plus a SvelteKit admin panel), a Go TUI, a SwiftUI Apple client, and a Svelte docs site.

Specs and contracts (read before touching sync or books):
- [`SPEC.md`](./SPEC.md): v2 vault model, data model, capabilities, API surface.
- [`docs/CRDT-CLIENT-PROTOCOL.md`](./docs/CRDT-CLIENT-PROTOCOL.md): **normative** client protocol. It covers the join rules (never seed locally, join empty and wait for SyncStep2), ranged ops only, the 3-way `/diff` with `base_state`, WS tickets, offline replicas, `crdt_epoch` lineage (§12) and the conformance vectors. Its status table tracks what each server and client implements.
- [`docs/SPEC-ESPELHO-SELADO.md`](./docs/SPEC-ESPELHO-SELADO.md): one-way mirrors (M0–M1 shipped) and sealed E2E books (design, owner decisions at the end).
- [`docs/DESIGN-LANGUAGE.md`](./docs/DESIGN-LANGUAGE.md): visual and motion language (named springs, choreography, the event→motion porting table for web/Mac/TUI).
- [`docs/SPEC-PARTE2.md`](./docs/SPEC-PARTE2.md): part 2 UX and format contract (floating explorer, note splits, hover preview, Kanban, JSON Canvas, `lumi://` + `lumi open`, admin user login).
- [`docs/SPEC-LIVROS.md`](./docs/SPEC-LIVROS.md): the books model. It covers the account key `(server_url, username)`, the shared `~/.config/lumi/books.yaml` (byte-identical between Apple and TUI), the SSE feed `GET /api/events`, `GET /api/vaults/:id/changes?after=N` and the 7-state sync status. It ends with the owner's decisions.
- `server/docs/ADMIN.md` (pt-BR): admin API and `lpa_` tokens.
- `deploy/README.md` (pt-BR): production deploy.

## Build & Run Commands

### Full stack via Docker Compose (local)
```bash
cp .env.example .env                       # edit POSTGRES_PASSWORD, LUMI_ADMIN_*
git submodule update --init --recursive    # required for server cgo build
docker compose up -d
# Web UI: http://localhost:3000   API: http://localhost:8080
```

### Server (Go 1.25, cgo yrs)
```bash
cd server
make libyrs                                 # cargo build yffi static lib → internal/crdt/{lib,include} (gitignored)
docker compose up -d postgres
make migrate && make run                    # http://localhost:8080
GOTOOLCHAIN=go1.25.5 go test ./...          # see "Local tooling" — newer Go can't link libyrs
make test-integration                       # testcontainers (two-server federation suite), needs Docker
go run ./cmd/lumi-server migrate up|down N|status
go run ./cmd/lumi-server token create <name> [--ttl 90d]   # admin lpa_ token (CLI only)
go run ./cmd/lumi-server token revoke <id>
go run ./cmd/lumi-server mirror list|status|resync|pause|resume <id>   # one-way mirrors (docs/SPEC-ESPELHO-SELADO.md)
go run ./cmd/lumi-server mirror promote <vault> --confirm               # CLI only
go run ./cmd/lumi-server admin grant|revoke|list <user>   # server_admin flag: panel username+password login (lps_ sessions, 12 h)
```

### Web client (Svelte 5 + Vite 7) and admin panel
```bash
cd web-client
npm install && npm run dev                  # http://localhost:5173
npm test                                    # vitest
npm run build && npx svelte-check
cd admin && npm install && npm run dev      # SvelteKit admin panel, :5174 (needs LUMI admin API + lpa_ token)
npm test && npm run check && npm run build
```

### TUI client (Go)
```bash
cd tui-client
go build -o lumi . && ./lumi                # book picker / explorer
./lumi login <server-url>                   # sign in (accounts keyed by server+username)
./lumi accounts [--verify] | ./lumi vaults | ./lumi vault link|clone|unlink|sync|manage ...
go test ./... && go vet ./...
go test ./ui -update                        # regenerate golden renders in ui/testdata/golden
```
Tests must never touch the real `~/.config/lumi`: `ui/main_test.go` points HOME/XDG at a temp dir.

### Apple client (SwiftUI, Swift 6)
```bash
cd apple-client
make bootstrap                              # xcodegen
make open                                   # generate project + open Xcode
swift test                                  # LumiKit + LumiUI suites
# Unsigned Release build for local install / release:
xcodegen generate && xcodebuild -project Lumi.xcodeproj -scheme Lumi -configuration Release \
  -destination 'generic/platform=macOS' -derivedDataPath <tmp> ARCHS="arm64 x86_64" ONLY_ACTIVE_ARCH=NO \
  CODE_SIGN_IDENTITY=- CODE_SIGN_STYLE=Manual DEVELOPMENT_TEAM= build
```
`xcodebuild -quiet` prints spurious "failed with exit code 0" lines; judge by `BUILD SUCCEEDED`. The generated `Lumi.xcodeproj` is not committed.

### Site (Svelte 5 + Tailwind 4)
```bash
cd site && npm install && npm run dev
```

## Architecture

```
┌─────────────────┐  ┌──────────────┐  ┌──────────────────┐  ┌──────────────┐
│   TUI Client    │  │  Web Client  │  │   Apple Client   │  │ Admin panel  │
│ (Go + Bubbletea)│  │  (Svelte 5)  │  │    (SwiftUI)     │  │ (SvelteKit   │
└────────┬────────┘  └──────┬───────┘  └────────┬─────────┘  │  BFF, node)  │
  3-way /diff REST    Yjs WS (ticket)     Yjs WS (ticket)     └──────┬───────┘
  + SSE feed          + REST + SSE        + REST + SSE               │ /admin/v1
         └──────────────────┼───────────────────┘                    │ (lpa_ token)
                  ┌─────────▼──────────────────────────────────────────▼┐
                  │  lumi-server: public API :8080 │ admin API :8081     │
                  │  Fiber + cgo yrs, federation relay to other servers │
                  └─────────┬────────────────────────────────────────────┘
              ┌─────────────┴────────────┐
       ┌──────▼──────┐         ┌─────────▼─────────┐
       │  Postgres   │         │   Filesystem      │
       │ users/vaults│         │  <vault>/notes/   │
       │ roles/audit │         │  Markdown + YAML  │
       │ Yjs state   │         │ (source of truth) │
       └─────────────┘         └───────────────────┘
```

### Books, sync and offline
- **Accounts and books:** a client holds many accounts (one per `server_url` + `username`), each with many books. A federated book binds to its home server by default. Renaming a bound book renames it for everyone (needs `vault.manage`).
- **Feed:** each account opens `GET /api/events` (SSE, Bearer only). Events are `hello`, `vault.head`, `vault.access`, `vault.updated`, `vault.revoked`, `session.ended` and `reconnect`, with a ping every 25 s. On a head change the client catches up with `/changes?after=N`; a `410 resync_required` means a full resync.
- **Live editing (web, Apple):**
  - `POST /api/vaults/:vault/notes/:id/ws-ticket` returns `{ticket, expires_at, crdt_epoch}`. The ticket is single-use, valid 30 s, and goes in the WS URL as `?ticket=`. `?token=` on the WS is deprecated.
  - Every edit reaches Yjs as minimal ranged ops, never replace-all. Undo only reverts the user's own edits.
- **Offline (web, Apple):**
  - After the first SyncStep2, each note keeps a persisted replica: IndexedDB `lumi-offline` on the web, `Application Support/lumi/replicas/` on Apple. Replicas are keyed by server, account, vault and note.
  - A cold offline open is editable, and edits merge on reconnect.
  - "Pendente" tracks unsent edits. Sign-out, `session.ended` or `vault.revoked` wipes the replicas (LGPD) after offering an export.
  - The web also has a service worker (`/sw.js`) that precaches the app shell and never `/api/`.
- **Lineage:**
  - `notes.crdt_epoch` (migration 0008) changes only when a note starts a new CRDT history: create, delete and recreate, init from file, or vault copy.
  - Clients never merge a replica or base into a different epoch. The local text becomes a "<title> (conflito offline)" copy instead.
  - `GET .../state-vector` is the cheap ack and lineage probe. Client close code 4409 means the lineage changed; 4401 `ticket_stale` means the epoch changed between minting the ticket and the upgrade.
- **TUI:** uses REST only. It pushes with a 3-way `/diff` from a base stored in `<UserCacheDir>/lumi/books/<id>.bases/`, and checks the epoch before every push. Editing happens in `$EDITOR`.
- **WS Origin:** a present `Origin` must be on `LUMI_ALLOWED_ORIGINS`. An absent `Origin` (native clients) is admitted and gated by the ticket. Don't route Origin checks through the WS library: it rejects an absent Origin.

### Auth
- **Shared login on a Mac:** the Lumi app and the TUI share `~/.config/lumi/accounts.yaml` v3, which holds no secrets. Each client keeps its own session: the app in the Keychain, the TUI in `tokens.yaml` (0600). They link through one-time PKCE device links (`POST /api/auth/device-links`, `/redeem`), and the app always asks before approving. `logout {"scope":"device"}` signs out the whole Mac (SPEC-LIVROS §8).
- Sessions: `POST /api/auth/login` issues a session token. REST sends `X-Lumi-Token` (or Bearer). Sessions are stored hashed in Postgres, with bcrypt cost 12 and a 30-day TTL.
- An account locks after 10 wrong passwords (`failed_logins`, `locked_until`). Admins can disable or unlock it.
- First-boot bootstrap uses `LUMI_ADMIN_USERNAME` + `LUMI_ADMIN_PASSWORD` and only runs while the users table is empty. In production these are created for the first deploy and then deleted from the Gitea org secrets. After that, signup is by invite link.
- Per-vault custom roles gate every mutation. Seed roles: Admin / Editor / Viewer / Commenter.
- **Admin API:**
  - It lives at `/admin/v1` on a separate listener (`LUMI_ADMIN_ADDR`), with `lpa_` tokens minted only via the CLI.
  - The panel (`web-client/admin`, routes in pt-BR) is a BFF: the token stays server-side, and the browser holds the cookie `__Host-lumi_admin` (HttpOnly, SameSite=Strict, ≤12 h).
  - The panel uses CSP nonces and an Origin check on every POST.

### Server layout (`server/`)
- `cmd/lumi-server/`: entrypoint, env config, Fiber wiring, `migrate` and `token` subcommands.
- `internal/auth`, `users`, `vaults`, `roles`, `members`, `invites`, `notes`, `audit`, `capguard`: domain services and handlers.
- `internal/crdt`: yrs cgo wrapper, doc registry, snapshot/update persistence, `crdt_epoch` stamping.
- `internal/wsync`: Yjs WebSocket hub, tickets, awareness, debounced FS mirror.
- `internal/feed`: SSE feed and `/changes`.
- `internal/federation`: server↔server relay.
  - Manifests and announces carry the epoch, and frontmatter travels in `noteMeta2`.
  - Home enforces a per-link capability ceiling on everything a follower sends.
  - One-way mirrors: the mirror is read-only and home only sends.
- `internal/fswatch`: fsnotify, diff-merges external edits into the CRDT.
- `internal/admin`: admin API.
- `internal/storage/{fs,pg}` and `internal/domain`.
- `migrations/`: up to `0011_device_links` (0009 admin login, 0010 mirror + tombstone epochs, 0011 device links for the shared Mac app/TUI login).
- `third_party/y-crdt/`: submodule, yrs v0.26.0.

### Web client (`web-client/src`)
- **Views:** `App.svelte` (auth gate), `views/{LoginView,VaultsView,VaultHomeView}.svelte`; the settings view and modals are in `components/`.
- **Books and sync:**
  - `lib/accounts`, `books*`, `bookstatus.ts`: books and status. The status reducer shares `src/test/fixtures/bookstatus.cases.json` with the other clients.
  - `sse.ts`, `feed.ts`, `sync.svelte.ts`: the feed and sync.
- **Editor:**
  - `editor-session.svelte.ts` and `editor-boot.ts`: the Yjs provider plus CodeMirror + `@replit/codemirror-vim`, lazy-loaded.
  - `live-preview.ts`: Live Preview decorations.
  - `vim-config.ts`, `vimrc.ts`: escape sequences, `.vimrc` safe-subset parser (never evaluated).
  - `yank-flash.ts`, `editor-keys.ts`: yank highlight and editor key bindings.
  - `link-helper.ts`: link helper.
- **Offline:** `replica.ts`, `offline-store.ts`, `lineage.ts`, `ws-ticket.ts`.
- **Navigation:**
  - `components/FloatingExplorer.svelte`, `shell-keymap.ts`, `notetree.ts`, `fileops.ts`, `book-home.ts`: floating explorer (the only navigation) and the book home.
  - `pane-tree.ts`, `link-open.ts`, `hover-preview.ts`, `session-pool.ts`, `note-views.ts`, `deeplink.ts`: splits, link routing, hover card, shared sessions, view registry, deep links.
  - `kanban/`, `canvas/`, `text-diff.ts`, `live-text*.ts`: Kanban and JSON Canvas views over minimal ranged edits.
  - `motion.ts`, `remote-flash.ts`: spring motion and the remote-edit flash.
  - `pane-nav.ts`, `preview-keys.ts`: Ctrl-hjkl panes and keyboard reading of the split preview.
  - `legend.ts`, `hints.ts`, `palette.ts`, `context-actions.ts`: legend, hint mode, palette, context menu.
- **Rendering:**
  - `markdown.ts`: renderer, sanitized by DOMPurify. Only disabled checkbox inputs and allow-listed iframes survive.
  - `scroll-sync.ts`: split preview.
  - `app.css`: one shared prose stylesheet for reader, split preview and Live Preview.
- CSP is set both in `index.html` and in `nginx.conf`; keep them identical.

### TUI (`tui-client/`)
- `main.go` dispatches the CLI subcommands; `cmd_*.go` hold the handlers.
- `account/`, `books/`, `feed/`: accounts, books.yaml, SSE, lineage.
- `ui/`: Bubbletea explorer with Miller columns, glamour/chroma prose preview, legend, palette and mouse; goldens in `ui/testdata/golden/`.
- `theme/` has the same 12 themes as the web.

### Apple client (`apple-client/`)
- `Sources/LumiKit/`: platform-agnostic core.
  - `Accounts`, `Books`, `Feed`, `Net`, `Sync`, `CRDT` (yswift), `Offline`: accounts, books, feed, network, sync, CRDT and offline replicas.
  - `Explorer`: tree, Miller nav and shell keymap.
  - `Vim`: full engine with registers, macros, dot-repeat, text objects, ex and the `.vimrc` port.
  - `Preferences`.
- `Sources/LumiUI/`: theme, TextKit 2 editor bridges (mode-shaped cursor overlay), markdown reader matching the web prose system.
- `App/`: the `@main` app — the yazi shell, palette, legend, book settings sheet and split preview.

### UX model (all clients)
The clients are keyboard-first but mouse-friendly, with the same keymap everywhere (contract: `docs/SPEC-PARTE2.md`). Changing a key in one client means changing it in all of them.
- **Navigation is the floating explorer, and only that** (owner decision, 2026-11-24): there is no full-screen explorer or books sidebar.
  - Opening a book restores the saved split layout, else the last opened note, else a calm home with the explorer auto-opened.
  - Open the explorer with `-`, `<leader>e`, `:explore`, the toolbar button or the palette. `gb` opens the books level (accounts → books); `[`/`]` cycle books.
  - Narrow screens get a full-height sheet.
- **Explorer:** `j/k/h/l`, `gg/G`, counts, `/` filter with `n/N`, `a` new (`dir/name` creates folders, `name.canvas` creates a canvas), `r` rename/move, `d` delete with an inline confirm, `y/p` duplicate. `Enter` opens here; `Ctrl-v`/`Ctrl-s` open in a split.
- **Notes:** Reading | Live | Source, plus the split preview beside Source. Kanban notes (Obsidian format) open as a Board and `.canvas` files (JSON Canvas 1.0) as a Canvas; both are registered note views with a Source switch.
- **Splits:** `Ctrl-w v/s/q/o/=`, `:vsplit`/`:split [note]`, and `Ctrl-h/j/k/l` plus `Ctrl-w` to move between panes.
- **Links:** `gf`/`Enter` follow; `Ctrl-w f` and `⌘`/Ctrl-click open in a vsplit, `Ctrl-w F` in an hsplit.
- **Hover card:** after 350 ms over a link, or with `K`.
- **App keys:** `space`/`:`/⌘K palette, `?` help, `f`/`F` hint mode, `,` settings (`:settings` in the editor; ⌘, in the Mac app only). A one-line command legend sits above the status line.
- **Motion "the app feels alive":**
  - spring motion, shared-element transitions, staggered lists;
  - a breathing/pulsing sync dot and a remote-edit flash.
  - It never blocks input, stays ≤220 ms on the typing path, and turns off entirely under Reduce Motion (system or app setting).
- **CLI:** `lumi open [book[/note]]` opens the Mac app via `lumi://open?...` (strictly validated; it never adds accounts or books), or the web deep link `#/b/<vault>/n/<note>` elsewhere.

## Deploy, remotes and releases

- **Remotes:**
  - `origin` (GitHub, ViniZap4/*) is the main remote.
  - `gitea` is the second remote: the private org `lumi` on the owner's Gitea, `ssh://git@192.168.68.54:2222/lumi/<repo>.git`, web at `http://100.66.72.35:3001`, `tea --login gitea`.
  - Push every repo to both remotes. GitHub HTTPS pushes use `git -c credential.helper= -c credential.helper='!gh auth git-credential' push origin main`; don't change the global git config.
- **Production:**
  - Pushing the umbrella `main` to Gitea runs `.gitea/workflows/deploy.yml` on the runner `zima-runner`. It builds, backs up, then runs `docker compose -p lumi-prod up -d` from `deploy/compose.prod.yaml`.
  - Each service sits behind a Tailscale sidecar (TS_USERSPACE, Serve on 443): web `https://lumi.taildfba5.ts.net`, API `https://lumi-api.taildfba5.ts.net`, admin panel `https://lumi-admin.taildfba5.ts.net`.
  - Docs and `**/*.md` changes don't deploy.
  - Gotcha: with TS_USERSPACE the sidecar forwards any tailnet connection to the container's `127.0.0.1`, so binding to loopback does not isolate anything. The owner accepted the tailnet-internal exposure.
- **Releases:**
  - Tag the umbrella repo (`vX.Y.Z`) and create the release on both GitHub (`gh release create`) and Gitea (`tea releases create --login gitea --repo lumi/lumi`) with the same assets: the universal macOS DMG and the TUI binaries (`CGO_ENABLED=0`, `-X main.Version=vX.Y.Z`) for darwin arm64/amd64, linux amd64/arm64 and windows amd64, plus `SHA256SUMS.txt`.
  - Update the README Download section.
  - The DMG is ad-hoc signed and not notarized.

## Local tooling (owner's Mac)

- **Disk space is low:**
  - Always delete DerivedData, `.build`, `.swiftpm`, generated xcodeproj and temp worktrees after use.
  - Don't install toolchains on the Mac itself; use Docker (colima) for anything heavy, then clean up.
- **Server cgo tests:** use `GOTOOLCHAIN=go1.25.5`. Newer local Go fails to link `libyrs.a` (LTO bitcode). Worktrees need `internal/crdt/lib` and `include` copied from the main checkout.
- **Docker:**
  - Run colima with a temp `DOCKER_CONFIG` holding `{}`, plus `DOCKER_HOST=unix://$HOME/.colima/default/docker.sock`. The user's `~/.docker/config.json` has a stale `credsStore`.
  - Cleanup: `docker system prune -af --volumes`, `colima delete -f`, and `rm -rf ~/.colima/_lima/_disks ~/Library/Caches/{colima,lima}`.
- **Agents never launch GUI apps:** no `open`, osascript or input automation. It once stole the owner's keyboard focus. Visual checks use offscreen renders (Apple) or headless Playwright with a mock API (web).

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

Plain markdown files (no frontmatter) are accepted. Saving preserves the on-disk format unless metadata is explicitly added through the UI. Rendered extensions shared by all clients:
- `> [!note]` callouts (11 kinds, `+`/`-` foldable);
- `- [ ]` tasks, toggled with a single-character ranged edit;
- `==mark==`, footnotes and tables.

## Conventions

- **Commit messages:** conventional commits with module scope, e.g. `feat(server, notes): ...`, `fix(web, editor): ...`, `docs(...)`, `chore(monorepo): bump ...`. Push immediately after every commit, to both remotes. After merging submodule work, bump the submodule pointers in the umbrella.
- **Go style:** `gofmt`/`goimports`, meaningful package names, explicit error handling, `domain.ErrValidation` for input errors.
- **Svelte style:** Svelte 5 runes; one component per file; lazy-load heavy chunks (Yjs, CodeMirror).
- **Auth:** `X-Lumi-Token`/Bearer on REST and `?ticket=` on WS. Check per-vault capabilities in every handler.
- **Tests:** required for security-critical and CRDT paths. The CRDT protocol's conformance vectors are shared across clients.
- **Language:** UI copy and ops docs (deploy, admin) are pt-BR; code, comments and specs are English.
- **Pillars** (applied to every decision): Security/LGPD → performance → DX → scale → UX → UI → QA.

## Submodules

| Path | Repo |
|------|------|
| `server/` | `ViniZap4/lumi-server` |
| `web-client/` | `ViniZap4/lumi-web` (includes `admin/`) |
| `tui-client/` | `ViniZap4/lumi-tui` |
| `apple-client/` | `ViniZap4/lumi-apple` |
| `site/` | `ViniZap4/lumi-site` |

All components track `main`. Clone with `git clone --recurse-submodules`. The server has its own nested submodule (`third_party/y-crdt`) that cgo needs.
