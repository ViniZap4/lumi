# lumi — v2 product & architecture spec

Status: locked product spec, pending implementation. Supersedes the data and auth model described in CLAUDE.md (which still describes v1).

## Overview

lumi v2 reframes the product around **vaults** — Obsidian-style portable note collections that may be local-only or bound to a server for collaboration. The server becomes a multi-tenant vault host with per-vault custom roles. Multiple users can co-edit notes inside a vault in real time via Yjs CRDT, while the filesystem (markdown + YAML frontmatter) remains the source of truth for what's on disk; the CRDT is the operational projection used to merge concurrent edits.

## Pillars

Every design and implementation decision is evaluated against these seven pillars, in this priority order when trade-offs arise:

1. **Security first / LGPD compliance** — privacy by design. Brazilian LGPD requirements (data minimisation, consent on register, right of access, right of erasure, right of rectification, audit log, encryption in transit) are hard constraints. See [LGPD compliance](#lgpd-compliance).
2. **Performance** — Fiber + Postgres + Yjs picked for low p95 latency on collab paths. Targets: cold-start <1s, note-open <200ms p95, CRDT update echo <50ms p95 within a region.
3. **DX (developer experience)** — clean module boundaries, sqlc for type-safe SQL, structured logging with request IDs, OpenAPI generation, dev hot-reload, scriptable ops CLI.
4. **Scale** — stateless server (sessions in Postgres), horizontal-ready behind sticky load balancer, CRDT log compaction bounded, no per-process state that prevents replicas.
5. **UX** — multi-vault multi-server clients, invite-link first-signup, presence/awareness, vault portability (export/import), offline-tolerant local-only vaults.
6. **UI** — keyboard-first, vim mode where applicable, 12 shared themes, sensible defaults, no jank on collab updates.
7. **QA** — supersede v1's "no automated tests" rule for security-critical and CRDT paths. See [Testing strategy](#testing-strategy).

When a feature trades against more than one pillar, document the trade-off in the relevant section or PR.

## Locked decisions

| Area | Decision |
|---|---|
| Unit of organisation | Vault (Obsidian-vault metaphor): a self-contained directory of markdown notes |
| Vault storage | Filesystem (markdown + YAML); CRDT-FS sync via watcher diff-merge |
| Vault modes | Local-only (no server) **or** server-bound (collaborative) |
| Server role | Multi-tenant vault host; per-vault membership and roles |
| Conflict resolution | Real-time CRDT — **Yjs** (server engine: `yrs` Rust core via cgo) |
| TUI collab semantics | **Snapshot edit + diff sync** — TUI fetches current text, edits in `$EDITOR`, server diffs into the live CRDT on save |
| Web collab semantics | Live Yjs over WebSocket, awareness/presence (cursors, names) |
| Identity | **Independent per-server accounts**; username + password; sessions in Postgres |
| Signup flow | **Invite-link first-signup** — recipient registers and joins target vault in one step |
| Server-side store | Postgres (users, vaults, members, roles, invites, Yjs state log, metadata) |
| Roles | **Custom per-vault**; v1 ships with seed templates: Admin, Editor, Viewer, Commenter (Commenter equivalent to Viewer until comments ship) |
| Multi-server clients | TUI and web both track multiple vaults; each vault has independent server binding and account |
| HTTP framework | Fiber (server rewrite) |
| Migration of v1 data | Auto: prompt for admin credentials, create `personal` vault on the local server, move existing notes |
| Backwards compatibility | None — v2 is a clean break from v1 single-password model |

## Vault model

A vault is a directory that the user manages on their own filesystem. lumi never owns the vault location; users may keep vaults in iCloud, Syncthing, Git, etc. as they would Obsidian.

### Vault on disk

```
<vault-dir>/
  .lumi/
    vault.yaml          # id, name, slug, server-link, members snapshot (when synced)
    cache/
      yjs/<note-id>.bin # last-known CRDT state (rebuildable)
      search.idx        # search index
    config.yaml         # local prefs scoped to this vault (theme, editor, etc.)
  <note-id>.md          # markdown notes — same format as v1
  <subfolder>/
    <note-id>.md
```

`.lumi/vault.yaml` shape:

```yaml
id: 8c5a1d9f-...                    # uuid; stable across moves
name: Work team
slug: work-team
created_at: 2026-04-28T10:00:00Z

# optional: present iff vault is bound to a server
server:
  url: https://lumi.work.com
  vault_id: 8c5a1d9f-...            # server's id for this vault (matches local)
  last_synced_at: 2026-04-28T10:05:00Z

# cached locally; server is authoritative when bound
members:
  - username: alice
    role: Admin
  - username: bob
    role: Editor

# v1 seed roles, plus any custom additions cached
roles:
  Admin:        { capabilities: [vault.*, note.*, members.*, roles.*] }
  Editor:       { capabilities: [note.read, note.create, note.edit, note.delete] }
  Viewer:       { capabilities: [note.read] }
  Commenter:    { capabilities: [note.read] }     # comments not yet implemented
```

For a local-only vault, the `server`, `members`, and `roles` sections are absent or trivial.

### Vault lifecycle commands (TUI + CLI surface)

- `lumi vault create <dir>` — initialise a local-only vault at `<dir>`
- `lumi vault link <dir> <server-url>` — bind an existing local vault to a server (creates the vault server-side from local content)
- `lumi vault clone <server-url>/<slug> <dir>` — pull a server-hosted vault to a local directory
- `lumi vault unlink <dir>` — drop server binding; keep files
- `lumi vault export <dir> <out.tar.gz>` — bundle vault for portability
- `lumi vault import <archive> <dir>` — restore from bundle
- `lumi vault list` — list known vaults (from `~/.config/lumi/vaults.yaml`)

## Client architecture

### Multi-vault, multi-server clients

Both TUI and web client maintain a list of vaults, each independently bound. The web client runs in a browser served by *one* server but can address vaults hosted on *other* servers via cross-origin requests (server CSP and CORS configured per deployment).

`~/.config/lumi/vaults.yaml`:

```yaml
vaults:
  - path: ~/notes/personal
    server: null
  - path: ~/notes/work
    server: https://lumi.work.com
    account: alice@lumi.work.com    # references servers.yaml
  - path: ~/notes/side-project
    server: https://notes.friend.org
    account: alice2@notes.friend.org
```

`~/.config/lumi/servers.yaml`:

```yaml
servers:
  - url: https://lumi.work.com
    accounts:
      - username: alice
        session_token: <opaque>
        expires_at: 2026-05-28T10:00:00Z
  - url: https://notes.friend.org
    accounts:
      - username: alice2
        session_token: <opaque>
        expires_at: 2026-05-28T10:00:00Z
```

Each server account is independent. There is no central identity.

### Web client running

The web client serves itself from the server it lives on, but treats that server like any other. A user can log in to multiple servers concurrently (separate session tokens stored client-side in `localStorage`, partitioned by server origin).

## Server architecture

### Stack

- Go 1.23+
- Fiber v3 (HTTP) + `github.com/gofiber/contrib/websocket`
- Postgres 16+
- `yrs` (Yjs Rust core) via cgo bindings for CRDT state management
- `fsnotify` for filesystem watching
- `bcrypt` for password hashing; `argon2id` is acceptable if perf permits
- `crypto/rand` for token issuance; `crypto/subtle` for constant-time comparisons

### Layout (`lumi-server` repo)

```
cmd/lumi-server/main.go
internal/
  auth/         # session tokens, login, password hashing
  users/        # user CRUD
  vaults/       # vault CRUD, membership, roles
  invites/      # invite link generation + acceptance
  notes/        # note metadata; thin wrapper around fs + crdt
  crdt/         # yrs cgo binding, doc registry, sync protocol
  storage/
    fs/         # filesystem read/write, atomic rename, safe-join
    pg/         # postgres queries (sqlc-generated or hand)
    watcher/    # fsnotify → diff-merge into CRDT
  ws/           # WS hub: per-vault rooms, awareness fanout
  api/          # Fiber route registration, middleware
  config/       # env parsing, defaults
  migrations/   # SQL files
```

### Data model (Postgres)

```sql
-- Identity
CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username        TEXT NOT NULL UNIQUE,
  password_hash   TEXT NOT NULL,
  display_name    TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
  token           TEXT PRIMARY KEY,           -- 32-byte random hex
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at      TIMESTAMPTZ NOT NULL,
  last_used_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON sessions (user_id);
CREATE INDEX ON sessions (expires_at);

-- Vaults
CREATE TABLE vaults (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug            TEXT NOT NULL UNIQUE,        -- url-safe; matches FS dir name
  name            TEXT NOT NULL,
  created_by      UUID NOT NULL REFERENCES users(id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vault_roles (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id        UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  name            TEXT NOT NULL,               -- 'Admin', 'Editor', custom names...
  capabilities    JSONB NOT NULL,              -- array of capability strings
  is_seed         BOOL NOT NULL DEFAULT FALSE, -- locks the four built-in roles from rename/delete
  UNIQUE (vault_id, name)
);

CREATE TABLE vault_members (
  vault_id        UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id         UUID NOT NULL REFERENCES vault_roles(id),
  joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (vault_id, user_id)
);

CREATE TABLE invites (
  token           TEXT PRIMARY KEY,            -- short opaque
  vault_id        UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  inviter_user_id UUID NOT NULL REFERENCES users(id),
  role_id         UUID NOT NULL REFERENCES vault_roles(id),
  email_hint      TEXT,                        -- optional, for UX
  max_uses        INT NOT NULL DEFAULT 1,
  use_count       INT NOT NULL DEFAULT 0,
  expires_at      TIMESTAMPTZ NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at      TIMESTAMPTZ
);

-- Notes (metadata; full content lives on FS)
CREATE TABLE notes (
  id              TEXT PRIMARY KEY,            -- vault-scoped id from filename
  vault_id        UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  path            TEXT NOT NULL,               -- relative path inside vault
  title           TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL,
  updated_at      TIMESTAMPTZ NOT NULL,
  UNIQUE (vault_id, path)
);
CREATE INDEX ON notes (vault_id);

-- CRDT state
CREATE TABLE note_yjs_snapshots (
  note_id         TEXT PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
  state           BYTEA NOT NULL,              -- compacted yjs state vector + doc
  snapshotted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE note_yjs_updates (
  id              BIGSERIAL PRIMARY KEY,
  note_id         TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  update          BYTEA NOT NULL,              -- yjs incremental update
  origin_user_id  UUID REFERENCES users(id),
  origin_kind     TEXT NOT NULL,               -- 'web', 'tui-diff', 'fs-watcher'
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON note_yjs_updates (note_id, id);

-- Audit
CREATE TABLE audit_log (
  id              BIGSERIAL PRIMARY KEY,
  user_id         UUID REFERENCES users(id) ON DELETE SET NULL, -- nullified on user erasure; row retained
  vault_id        UUID REFERENCES vaults(id) ON DELETE SET NULL,
  action          TEXT NOT NULL,               -- see action vocabulary below
  payload         JSONB,                       -- redacted on user erasure (free-text scrubbed)
  ip              INET,                        -- captured at action time; nullified on user erasure
  user_agent      TEXT,                        -- nullified on user erasure
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON audit_log (user_id, created_at DESC);
CREATE INDEX ON audit_log (vault_id, created_at DESC);
CREATE INDEX ON audit_log (action, created_at DESC);

-- LGPD consent ledger: every consent acceptance is a new row (immutable)
CREATE TABLE user_consents (
  id              BIGSERIAL PRIMARY KEY,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tos_version     TEXT NOT NULL,
  privacy_version TEXT NOT NULL,
  accepted_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ip              INET,
  user_agent      TEXT
);
CREATE INDEX ON user_consents (user_id, accepted_at DESC);

-- LGPD-relevant actions (action vocabulary in audit_log):
--   auth.login, auth.login_failed, auth.logout, auth.register, auth.password_change,
--   user.update, user.delete, user.export_request,
--   vault.create, vault.delete, vault.update,
--   member.invite, member.add, member.remove, member.role_change,
--   role.create, role.update, role.delete,
--   invite.create, invite.accept, invite.revoke,
--   consent.accept, consent.update,
--   note.create, note.delete   (note.edit is high-volume; only edit-session-start logged)
```

### Capability vocabulary

Used in `vault_roles.capabilities` JSONB:

```
note.read             # see and download notes
note.create
note.edit
note.delete
note.move
members.invite        # create invite links
members.manage        # change roles, remove members
roles.manage          # create/edit/delete custom roles
vault.manage          # rename vault, delete vault, manage settings
audit.read            # view vault audit log
```

Wildcards (`note.*`, `members.*`, `*`) are expanded at check time. Seed roles:

| Role | Capabilities |
|---|---|
| Admin | `*` |
| Editor | `note.read, note.create, note.edit, note.delete, note.move` |
| Viewer | `note.read` |
| Commenter | `note.read` (comments capability added in v2.x when feature ships) |

Seed roles are flagged `is_seed = TRUE` and cannot be renamed/deleted from the API; their capability sets are also fixed (administrators can introduce new custom roles for variations).

### CRDT integration

#### Engine

`yrs` (the Rust Yjs reimplementation maintained by the Yjs team) compiled as a static library and exposed via cgo. A thin Go wrapper in `internal/crdt` provides:

```go
type Doc interface {
    ApplyUpdate(update []byte) error
    Encode() []byte                       // full state
    EncodeStateAsUpdate(sv []byte) []byte // delta for sync
    Text() string                         // rendered markdown
    ApplyDiff(old, new string) error      // text-level diff applied as Yjs ops
    Close()
}

type Registry interface {
    Get(noteID string) (Doc, error)       // load from snapshot+log on first access
    Persist(noteID string) error          // compact updates → snapshot
}
```

#### Persistence

Per-note: a single compacted snapshot in `note_yjs_snapshots` plus an append-only log in `note_yjs_updates`. A background compactor folds the log into the snapshot whenever the log exceeds 200 entries or 1 MB. On cold start, the doc is reconstructed from snapshot + replayed updates.

#### FS-CRDT sync

1. **CRDT → FS** (write): every `Doc` mutation enqueues a write to `<vault>/<note-id>.md` via the FS writer. Writer tags the inode with a marker (`xattr` or in-memory map keyed by `(path, mtime, hash)`) so the watcher can identify self-writes.
2. **FS → CRDT** (external edit): the watcher observes a write whose marker doesn't match. It reads the new file content, computes a textual diff against the current `Doc.Text()`, and applies it via `ApplyDiff`. Live collaborators see the change as ordinary CRDT operations. Mid-flight remote edits are preserved.

#### Sync protocol

- **Web client**: standard Yjs sync protocol over WebSocket. Each note gets a "room" `(vault-id, note-id)`. Awareness messages carry user color + cursor offsets. Server is a relay + persistence layer.
- **TUI**: REST-only.
  - `GET /api/vaults/:vault/notes/:id/snapshot` → returns `{ text, vector_clock }`
  - `POST /api/vaults/:vault/notes/:id/diff { base_clock, diff }` → server applies the diff against current state; if the doc has advanced, the diff is replayed against the *current* text (not the base), preserving concurrent edits.

### Auth & sessions

- Login: `POST /api/auth/login { username, password }` → `{ token, user, expires_at }`. `bcrypt.CompareHashAndPassword`; constant-time at every layer.
- Session token: 32 bytes random, hex-encoded, stored in Postgres. TTL: 30 days, sliding (`last_used_at` extends).
- Logout: `POST /api/auth/logout` deletes the session row.
- Server-admin bootstrap: first run prompts for admin username/password (interactive on TTY, or via env `LUMI_ADMIN_USERNAME` + `LUMI_ADMIN_PASSWORD` in non-interactive mode). Creates user, creates `personal` vault, migrates existing v1 notes if present.

### LGPD compliance

Lumi handles personal data (usernames, display names, optional email, note content). Brazilian LGPD requires concrete user-facing controls; v2 ships these as first-class:

- **Consent on registration** — `POST /api/auth/register` requires `consent: { tos_version, privacy_version, accepted_at }`. Server records the acknowledgment in `user_consents` (immutable ledger). Without consent fields, registration fails with `400 consent_required`.
- **Right of access (Art. 18, II)** — `GET /api/users/me/export` streams `application/zip` containing the user's profile, all notes from vaults they own (or have a `vault.export` capability for), audit log entries about themselves, consent history, and a top-level `manifest.json`. Rate-limited to 1/day/user; logged as `user.export_request`.
- **Right of rectification (Art. 18, III)** — `PATCH /api/users/me` updates display name; `POST /api/users/me/password` changes password (with old-password challenge). Username changes are restricted (unique constraint, audited).
- **Right of erasure (Art. 18, VI)** — `DELETE /api/users/me` requires password confirmation. Cascade rules:
  - Sessions: hard-deleted.
  - Vault memberships where user is **not** sole admin: removed; vault retained.
  - Vaults where user is **sole** admin: API returns `409 sole_admin_vaults` listing them; client must transfer ownership or explicitly opt-in to vault deletion via `?delete_sole_admin_vaults=true`.
  - Personal vaults (sole-admin and never shared): hard-deleted including FS dirs and Postgres rows.
  - Notes inside retained vaults: kept (vault content is shared property of remaining members).
  - `audit_log` entries: `user_id` nullified, `ip`/`user_agent` nullified, free-text `payload` fields scrubbed; rows retained for audit integrity.
  - `user_consents`: hard-deleted (cascade via FK).
- **Audit log** — see action vocabulary in the schema. Default retention 90 days; configurable via `LUMI_AUDIT_RETENTION_DAYS`. Background job purges expired rows nightly.
- **Data minimisation** — registration requires only `username`, `password`, `consent`. `display_name` and `email` (recovery) are optional, user-supplied.
- **Encryption in transit** — TLS required in production. `LUMI_REQUIRE_TLS=true` (default) refuses unencrypted listeners except on `127.0.0.1`.
- **Encryption at rest** — out-of-scope for application layer; deployments are expected to use disk-level encryption (LUKS / EBS / etc.). Application logs use a redaction filter for `password`, `token`, `email`, `ip` fields.
- **Bearer credentials** — session tokens are bearer credentials; documented as such in privacy policy template. No tracking cookies, no third-party analytics in shipped clients.
- **Data portability** — export format is JSON manifest + plain markdown files in a zip. No proprietary lock-in.
- **Privacy policy + ToS** — operators publish their own. The web client surfaces consent screens with operator-provided URLs; document version strings are recorded per consent. Endpoints: `GET /api/legal/tos`, `GET /api/legal/privacy` proxy to operator-configured URLs.
- **Data residency** — operator concern; documented in deploy guide. Lumi never phones home.

### Invite + signup flow

1. Vault admin: `POST /api/vaults/:vault/invites { role_id, max_uses, expires_at }` → `{ token, url }`. URL: `https://server/invite/<token>`.
2. Recipient opens URL, web client renders signup-and-join screen showing inviter, vault name, role.
3. Recipient submits `{ token, username, password, display_name }`. Server: validates token → creates user → creates membership with the invite's role → returns session token. One round trip.
4. Existing-user case: same URL surfaces a "log in to join" alternative; `POST /api/invites/:token/accept` with auth header attaches membership.
5. Server-admin policy: a deploy-time env var `LUMI_REGISTRATION = invite-only | open` controls whether non-invited signups are accepted. Default: `invite-only`.

## API surface (REST, summary)

```
POST   /api/auth/login                      → session
POST   /api/auth/logout
POST   /api/auth/register                   → session (only if open registration)
POST   /api/invites/:token/accept           → session (or attaches membership if logged in)

GET    /api/users/me
PATCH  /api/users/me                          → update display_name
POST   /api/users/me/password                 → change password (old-password challenge)
GET    /api/users/me/export                   → zip stream    (LGPD right of access; rate-limited 1/day)
DELETE /api/users/me                          → erasure       (LGPD right of erasure; password confirm)

GET    /api/legal/tos                         → current ToS version + body (operator-served)
GET    /api/legal/privacy                     → current Privacy version + body

GET    /api/vaults                          → vaults the user is a member of
POST   /api/vaults                          → create vault (caller becomes Admin)
GET    /api/vaults/:vault                   → vault detail (members, roles)
PATCH  /api/vaults/:vault                   → rename, etc.  (capability: vault.manage)
DELETE /api/vaults/:vault                   →                (capability: vault.manage)

GET    /api/vaults/:vault/members
PATCH  /api/vaults/:vault/members/:user     → change role     (capability: members.manage)
DELETE /api/vaults/:vault/members/:user     →                  (capability: members.manage)

GET    /api/vaults/:vault/roles
POST   /api/vaults/:vault/roles             → custom role     (capability: roles.manage)
PATCH  /api/vaults/:vault/roles/:role       →                  (capability: roles.manage)
DELETE /api/vaults/:vault/roles/:role       →                  (capability: roles.manage)

POST   /api/vaults/:vault/invites           → invite link    (capability: members.invite)
GET    /api/vaults/:vault/invites           → list           (capability: members.invite)
DELETE /api/vaults/:vault/invites/:token    → revoke

GET    /api/vaults/:vault/notes             → list (paginated)
POST   /api/vaults/:vault/notes             → create
GET    /api/vaults/:vault/notes/:id         → metadata
GET    /api/vaults/:vault/notes/:id/content → rendered markdown text
GET    /api/vaults/:vault/notes/:id/snapshot→ { text, vector_clock }   (TUI)
POST   /api/vaults/:vault/notes/:id/diff    → apply diff               (TUI)
PATCH  /api/vaults/:vault/notes/:id         → rename, move
DELETE /api/vaults/:vault/notes/:id

GET    /api/vaults/:vault/files/:path       → media (images, attachments)
POST   /api/vaults/:vault/files             → upload

GET    /api/vaults/:vault/audit             → audit log     (capability: audit.read)

WSS    /api/vaults/:vault/notes/:id/sync    → Yjs sync + awareness for note
WSS    /api/vaults/:vault/events            → vault-level event stream (member changes, etc.)
```

All authenticated requests use `X-Lumi-Token: <session>` header. WebSocket upgrades require the same header during the handshake (origin allowlist enforced).

## Client surfaces

### Web client

- Login screen scoped to its origin server.
- Vault selector: list of vaults the logged-in user belongs to on this server. (Vaults from *other* servers are not visible to this web client; users access them via that server's web client or via the TUI.)
- Note browser per vault.
- Yjs-backed editor (CodeMirror 6 + `y-codemirror.next` + `@codemirror/lang-markdown`).
- Awareness/presence: live cursors with color + display name.
- Member management UI (gated on capabilities).
- Role editor (gated on `roles.manage`).
- Invite generator (gated on `members.invite`).

### TUI

- `lumi vault list` / `lumi vault create` / `lumi vault link` / `lumi vault clone`.
- Per-vault interactive view (current TUI shape, scoped to one vault at a time).
- Login flow per server (`lumi login <server-url>` interactive prompt).
- Edit flow:
  1. Open note → fetch `snapshot` (synced vault) or read file (local-only).
  2. Launch `$EDITOR`.
  3. On save: post `diff` (synced) or write file (local-only).
- Background WS subscription (synced vaults) updates the note list on remote changes; doesn't disrupt active `$EDITOR` session.

## Migration from v1

On first start of v2 server against a v1 `LUMI_ROOT`:

1. Detect v1 layout (notes at root with no `<workspace-slug>` parent, no `.lumi/vault.yaml`).
2. If non-interactive (`LUMI_ADMIN_USERNAME` + `LUMI_ADMIN_PASSWORD` set): create admin user.
3. If interactive (TTY): prompt for admin username/password.
4. Create `personal` vault with admin as sole member (Admin role).
5. Move all v1 notes into `<LUMI_ROOT>/personal/` and update their FS paths.
6. Initialise CRDT state for each note from current content.
7. Write `<LUMI_ROOT>/personal/.lumi/vault.yaml`.

Migration is idempotent (subsequent starts detect the personal vault and no-op). A `--no-migrate` flag exists to skip and treat as fresh install.

## Phased rollout

| Phase | Scope | Output |
|---|---|---|
| 0 | This spec — review and lock | `SPEC.md` (this file) |
| 1 | `lumi-server`: Postgres + migrations + Fiber + auth + users + vaults + roles + members + invites | One submodule PR |
| 2 | `lumi-server`: CRDT engine (yrs cgo), note persistence, FS watcher, Yjs WS sync, TUI snapshot/diff endpoints | One submodule PR |
| 3 | `lumi-server`: audit fixes from prior security review baked into all handlers (path traversal, atomic writes, CSP/CORS, MaxBytesReader, graceful shutdown, etc.) | One submodule PR |
| 4 | `lumi-web`: auth UI, vault selector, Yjs+CodeMirror editor with awareness, members/roles/invites UI, audit-fix items (DOMPurify, CSP, sanitisation) | One submodule PR |
| 5 | `lumi-tui`: multi-vault model, multi-server account store, login flow, vault link/clone, snapshot+diff sync, audit fixes (frontmatter writer, atomic writes, token perms) | One submodule PR |
| 6 | Docker compose update (Postgres service, env vars, healthcheck), `.env.example` refresh, README rewrite | Submodule + monorepo PRs |
| 7 | Bump submodule pointers in monorepo, update `CLAUDE.md` to describe v2 | Monorepo PR |

Each phase ends with a working build. Phase 1+2+3 must land before Phase 4 starts (web depends on server). Phase 5 (TUI) can start in parallel with Phase 4 once Phase 2's REST endpoints exist.

## Open implementation calls (resolved by default unless flagged)

- **`yrs` cgo binding** — use `github.com/y-crdt/y-crdt`'s C FFI; build via Cargo invoked from `go generate`. Docker base image: `golang:1.23-bookworm` + `rustup`. Acceptable trade-off: Rust toolchain in the build pipeline.
- **Fiber v3** over v2 — current stable, expected long-term.
- **Postgres connection** — `pgx/v5` with the standard pgxpool. No ORM; hand-written SQL with `sqlc` for type-safe codegen.
- **Password hashing** — `bcrypt` cost 12. Future migration to argon2id is straightforward (compare-by-prefix dispatch).
- **Search** — out of scope for v2.0; placeholder is a Postgres trigram index on `notes.title` + `notes.body_cache` (where `body_cache` is the latest CRDT-rendered text). Full-text search is v2.1.
- **Attachments / media** — stored under `<vault>/.lumi/attachments/<sha256>` with content-addressed filenames; metadata in `notes` table reference. Out of scope for v2.0 surface beyond what v1 already does (raw `<vault>/files/<path>` serving).
- **Audit log retention** — default 90 days; configurable via `LUMI_AUDIT_RETENTION_DAYS`.
- **Rate limiting** — token bucket per IP at `/api/auth/*` (10/min) and per-session at write endpoints (60/min). Implementation via `github.com/gofiber/contrib/limiter`.

## Testing strategy

The v1 project convention was "no automated tests" — manual only. v2 supersedes that for security-critical and CRDT paths (per Pillar 7 / QA).

Required automated coverage:

- **Unit tests** (`go test ./...`):
  - `internal/auth` — token issue/validate/expire, constant-time compare, password hashing parameters.
  - `internal/domain/capability` — wildcard matching, role evaluation, denial paths.
  - `internal/storage/fs` — `SafeJoin` rejection of `..`, absolute paths, symlink escapes; atomic write semantics.
- **Integration tests** (`go test -tags=integration` with `testcontainers-go` Postgres):
  - Register → login → access protected endpoint → logout.
  - Invite create → accept (new user signup) → vault membership with correct role.
  - Capability check at every role boundary (Admin / Editor / Viewer / Commenter for the seed roles plus a custom role).
  - CRDT round-trip: web edit → server persist → cold-start → server replay → web edit again. Assert convergence.
  - FS watcher: external file edit → diff-merge into CRDT → broadcast to subscribed clients.
  - LGPD cascades: account deletion empties sessions + sole-admin personal vault; nullifies audit log; preserves shared vault content.
  - LGPD export: zip contains manifest + all owned notes + redacted audit subset.
- **Smoke tests** — `make smoke` brings up `docker-compose` (server + postgres) and exercises register → vault create → note create → invite → accept → edit. Run pre-merge.
- **Web client** — Playwright tests for login, vault selection, edit-with-presence (two browser contexts), and invite-accept signup. Manual UI testing remains primary for visual polish and theme regressions.

Coverage targets: 80%+ for `internal/auth`, `internal/domain/capability`, `internal/storage/fs`. Lower elsewhere; integration tests cover the rest.

CI runs unit + integration on every PR. Smoke + Playwright run nightly and on release branches.

## Out of scope for v2.0

- Comments on notes (Commenter role exists but functions as Viewer until v2.1)
- Full-text search beyond title/path trigram
- Attachments UI beyond v1's raw serving
- Mobile clients
- E2E encryption (server reads plaintext; deployment trust model unchanged)
- Cross-server vault federation / migration tooling beyond manual export/import
- Single-sign-on / OIDC / SAML
- Webhooks / API for third-party integrations
- Public read-only sharing of vaults

## Threat model summary

- Server is trusted by all members of a vault; reads plaintext notes and CRDT state.
- Each server is its own trust boundary; no cross-server identity.
- Session tokens are bearer credentials; treat with same care as passwords. Always over TLS in production.
- Vault content is server-side plaintext; backup/snapshot strategy is the operator's responsibility.
- DOM XSS via user-authored markdown is mitigated client-side via DOMPurify + CSP (defence in depth); server does no markdown sanitisation.

## Glossary

- **Vault** — a directory of notes; the unit of organisation, permissioning, and sharing.
- **Server** — a `lumi-server` instance hosting one or more vaults for one or more users.
- **Account** — a user identity scoped to a single server.
- **Member** — an account with a role on a specific vault.
- **Role** — a named set of capabilities; per-vault.
- **Capability** — a permission string like `note.edit`.
- **CRDT** — Conflict-free Replicated Data Type; here, Yjs.
- **Snapshot** — compacted CRDT state for a single note.
- **Update** — incremental Yjs change record.
- **Local-only vault** — a vault not bound to any server.
- **Server-bound vault** — a vault paired with a server for collaboration.
