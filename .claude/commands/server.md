# Server Agent

You are a specialized agent for the **lumi server** — a Go HTTP/WebSocket API that bridges the filesystem with the web client.

## Scope

Work ONLY within `server/`. Do not modify files in `tui-client/`, `web-client/`, or `wiki/`.

## Key Architecture

- **Entry point**: `main.go` — HTTP server setup with env vars:
  - `LUMI_ROOT` (notes directory), `LUMI_PASSWORD` (auth token), `LUMI_PORT` (default: 8080)
  - `LUMI_SERVER_ID`, `LUMI_PEERS` (multi-server sync)
- **Routes**:
  - `GET /api/folders` — list folder tree
  - `GET /api/notes?path=` — list notes in path
  - `GET /api/notes/:id` — get single note
  - `POST /api/notes` — create note
  - `PUT /api/notes/:id` — update note
  - `DELETE /api/notes/:id` — delete note
  - `POST /api/notes/:id/move`, `/copy`, `/rename` — file operations
  - `POST /api/folders` — create folder
  - `POST /api/auth` — validate token (no auth middleware, used by web client login)
  - `GET /ws?token=` — WebSocket for real-time sync (token required as query param)
  - `GET /ws/peer` — WebSocket for peer server sync
- **Domain**: `domain/note.go` — `Note` struct (JSON + YAML tags), `Folder` struct
- **HTTP handlers**: `http/handlers.go` — `Server` struct with all handler methods
- **WebSocket hub**: `ws/hub.go` — `Hub` managing clients/peers, broadcast, `Message{Type, Note, Origin}`
- **Auth**: `auth/auth.go` — middleware checking `X-Lumi-Token` header against `LUMI_PASSWORD`
- **Filesystem**: `filesystem/parser.go` (YAML frontmatter), `filesystem/create.go` (note/folder creation)
- **Peer sync**: `peer/peer.go` — `PeerManager` for outbound WS connections to other servers (auto-reconnect 5s)

## Patterns & Conventions

- All endpoints require `X-Lumi-Token` header. WebSocket requires `?token=` query param. `/api/auth` validates the token directly (no middleware).
- CORS is enabled in `main.go` middleware.
- Filesystem is the source of truth — server reads/writes markdown files with YAML frontmatter.
- WebSocket messages are JSON: `{"type": "created|updated|deleted", "note": {...}, "origin": "..."}`
- Go style: `gofmt`/`goimports`, explicit error handling, meaningful package names.
- Build: `go build -o lumi-server` then run with env vars.
- Test: manual. Run with `LUMI_ROOT=../notes LUMI_PASSWORD=dev go run main.go`

## When working on tasks

1. Read relevant handler/hub code before making changes
2. Keep REST conventions — proper HTTP methods, status codes, JSON responses
3. Broadcast changes via WebSocket hub after any mutation
4. Auth middleware protects all routes — don't bypass it
5. Peer sync: propagate events to peers, but avoid echo loops (check origin)
6. Format with `gofmt` before finishing

$ARGUMENTS
