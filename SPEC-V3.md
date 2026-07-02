# lumi — v3 product & architecture spec (DRAFT)

Status: **draft, pending review and lock.** Extends and partially supersedes [`SPEC.md`](./SPEC.md) (v2, shipped 2026-05-28). Decisions marked **⚑ OPEN** need an explicit call from the product owner before the phase that implements them starts; everything else follows v2 precedent or is recommended by this draft.

## Overview

lumi v3 reframes the product around three moves:

1. **UI-note first.** lumi is primarily a note-taking *interface*: vim-focused editing and yazi-style navigation in every client, with security, performance, and privacy leading every trade-off. The sync machinery exists to serve that interface, not the other way around.
2. **Vaults become blocks.** The unit of organisation is renamed from *vault* to **block**. A block behaves like an Obsidian vault: a portable directory of markdown notes that is local-only or synced to a server. A server hosts many blocks (multi-tenancy unchanged). A block **belongs to a user** and is shared either by inviting others into its **workspace** (live collaboration) or by **sharing a copy** (a fork with no live link).
3. **Federation.** A block's workspace can be synced not just between clients but **between servers**: a server operator can be invited to federate a block, after which the block converges across all clients of all federated servers. v2 explicitly descoped federation; v3 brings it back CRDT-based, replacing v1's naive last-write-wins peering.

## Pillars

Unchanged from v2, in the same priority order: **Security/LGPD → performance → DX → scale → UX → UI → QA.** Federation is the feature most in tension with pillar 1; this spec resolves that tension with the home-server authority model below. Any federation behavior that cannot satisfy LGPD obligations (consent, erasure, audit) does not ship.

## Terminology

| Term | Meaning |
|---|---|
| **Block** | A portable directory of markdown notes + `.lumi/` metadata. Renamed from v2 *vault*. |
| **Workspace** | The collaborative surface of a block: its members, roles, invites, presence, and live CRDT state. |
| **Owner** | The user a block belongs to. Exactly one per block; transferable. |
| **Copy / fork** | A snapshot duplicate of a block handed to another user or server, with no live link back. |
| **Home server** | The server where a block was created (or migrated to). Authoritative for the block's workspace. |
| **Follower server** | A server invited to federate a block. Replicates content, defers to home for control. |

> **Naming risk (flagged, not blocking):** in the wider note-taking world, "block" usually means a paragraph-level unit (Notion/Logseq blocks). If lumi ever adds block-level references or embeds, the vocabulary will collide. Recommendation: accept the rename as directed, but reserve "note block" / "fragment" now for any future paragraph-level concept.

## Locked-decision candidates

| Area | Decision | Status |
|---|---|---|
| Unit of organisation | **Block** (renamed from vault), Obsidian-vault semantics | directed by owner |
| Rename depth | **Full stack**: UI, docs, CLI, API routes, DB schema, on-disk `.lumi/block.yaml`, code identifiers | **⚑ OPEN** (recommended) |
| Ownership | One owner per block (`owner_user_id`), transferable; distinct from Admin role | recommended |
| Sharing modes | Workspace invite (live, existing) **and** share-a-copy (fork, new) | directed by owner |
| Federation authority | **Home-server model**: home is authoritative for identity, membership, roles, invites, audit; followers replicate content and enforce signed control events | **⚑ OPEN** (recommended over peer-equal multi-master) |
| Federation transport | Server-to-server WebSocket per federated block, Yjs updates + signed control events | recommended |
| Server identity | Ed25519 keypair per server; identity = (URL, public key) | recommended |
| Conflict resolution | Yjs CRDT everywhere (unchanged); federation relays updates, never merges by timestamp | carried from v2 |
| Client sync semantics | Unchanged: web/Apple live Yjs WS, TUI snapshot+diff | carried from v2 |
| Identity | Still server-scoped accounts, **no central SSO**; federated members are recorded as `username@server` | carried from v2, extended |
| Backwards compatibility | None across the rename (v2 precedent); a single migration renames the schema | recommended |

## The rename (vault → block)

Full-stack, executed as one coordinated phase across all five components:

- **Server**: API routes `/api/vaults/*` → `/api/blocks/*` (no aliases — clients ship in lockstep, matching the v2 no-compat precedent); migration `0002_vaults_to_blocks` renames tables (`vaults`→`blocks`, `vault_roles`→`block_roles`, `vault_members`→`block_members`, FK columns `vault_id`→`block_id`); Go packages `internal/vaults` → `internal/blocks`; capability strings `vault.manage`/`vault.export` → `block.manage`/`block.export` (migration rewrites stored role JSONB); env `LUMI_ROOT` semantics unchanged.
- **On disk**: `.lumi/vault.yaml` → `.lumi/block.yaml`. Clients read `block.yaml`, fall back to `vault.yaml`, and rewrite to the new name on first write (blocks are user-owned directories that lumi doesn't control; the fallback never expires).
- **TUI**: `lumi vault ...` → `lumi block ...` subcommands; `~/.config/lumi/vaults.yaml` → `blocks.yaml` with the same read-fallback + rewrite. The TUI/Apple byte-matched writer contract moves with it.
- **Web / Apple / site / docs / SPEC / CLAUDE.md / memory**: mechanical rename.

## Block model

v2's vault model carries over verbatim (portable dir, local-only or server-bound, `.lumi/` metadata, FS as source of truth, FS↔CRDT bridge) with one addition:

```sql
ALTER TABLE blocks ADD COLUMN owner_user_id UUID NOT NULL REFERENCES users(id);
-- backfilled from created_by in the rename migration
```

- Owner always holds an irrevocable Admin-equivalent grant; `members.manage` cannot remove the owner.
- `POST /api/blocks/:block/transfer-ownership { user_id }` — owner or server admin only; audited (`block.transfer`).
- LGPD erasure: v2's sole-admin cascade rules re-key on *owner* — deleting a user with owned, never-shared blocks hard-deletes them; owned-and-shared blocks require transfer or explicit opt-in, exactly as v2 §Right-of-erasure specified for sole-admin vaults.

## Sharing

**Workspace invite** — unchanged from v2 (invite links, per-block roles, capability gating).

**Share a copy** — new, deliberately simple:

- `POST /api/blocks/:block/copies { recipient_username }` (capability `block.export`): server forks the block's current FS state + fresh CRDT init into a new block owned by the recipient, with a new block id. No membership, no live link, provenance recorded (`copied_from` metadata + `block.copy` audit event).
- Client-side equivalent already exists (`lumi block clone` + export/import); the endpoint adds server-side user-to-user handoff.
- A copy shares nothing after creation: edits diverge permanently.

## Federation

### Model (recommended: home-server authority)

The home server is authoritative for the **control plane** of a block: users, membership, roles, invites, revocations, audit, erasure. Follower servers replicate the **content plane** (Yjs state) and enforce control decisions they receive as signed events. Rationale: permissions and LGPD obligations have an unambiguous authority; CRDT content, which merges safely, is the only thing that multi-masters.

**⚑ OPEN — the alternative** is peer-equal multi-master (membership/roles as CRDTs too). Maximum availability, no privileged server, but conflicting permission changes and erasure across independent operators have no clean resolution — this draft recommends against it as a pillar-1 violation.

### Server identity & handshake

1. Every server generates an Ed25519 keypair at first boot (`server_keys` table); `GET /api/federation/identity` returns `{ url, public_key }`.
2. A block admin with new capability `block.federate` creates a **federation invite**: `POST /api/blocks/:block/federation-invites { server_url, expires_at }` → opaque token, delivered out-of-band to the other operator.
3. The follower operator accepts: their server calls home's `POST /api/federation/accept { token, follower_identity, signature }`. Home verifies, records a `block_federations` row `(block_id, follower_url, follower_pubkey, status, created_at)`, and returns the block's current snapshot + control-state bundle. Both sides audit (`federation.invite`, `federation.accept`).
4. Either side can sever: home revokes (`DELETE .../federations/:id`, signed revocation event) or the follower unlinks. Severed followers keep their last copy (they were trusted with plaintext; pretending otherwise is theater) but stop receiving updates and MUST surface "no longer synced" state to their users.

### Content plane

- Per federated block, follower maintains a server-to-server WS to home (`WSS /api/federation/blocks/:block/sync`), authenticated by signature over a challenge, reconnect with backoff.
- Yjs updates flow both directions, tagged `origin_kind = 'federation:<server-url>'`. Loop prevention by origin tag; CRDT idempotency makes residual echoes harmless.
- Followers persist their own snapshot + update log and mirror to their own FS — a follower is a full replica of content, so local clients (web/Apple live, TUI diff) work against their own server unchanged.
- Home offline: follower clients keep editing locally-converging state; it reconciles by state-vector diff on reconnect. Follower offline: same, other direction.

### Membership across servers

- Federated users are members recorded on **home's** member list as `username@follower-url` (member row gains `origin_server`). Role checks on a follower use its replicated control-state cache; grants/revocations only originate at home.
- Presence/awareness relays across the federation link (clientID namespaced by server) — cursors work cross-server.

### LGPD across the federation

Federation moves personal data to another operator. Non-negotiable behaviors:

- **Owner consent**: creating a federation invite requires the block *owner's* action (or explicit owner-granted `block.federate`), is audited, and the UI states plainly that content will be stored plaintext on the other operator's infrastructure.
- **Member notice**: all members see the federated-server list on the block (`GET /api/blocks/:block` includes `federations`).
- **Erasure propagation**: user-erasure and note-deletion at home emit signed control events; followers MUST apply the same cascades and audit compliance. Follower non-compliance (no ack within a bounded window) is surfaced to the home operator.
- **Export (right of access)** includes the list of servers a user's data was federated to.
- **Residency**: each federation accept records operator-declared jurisdiction (free text), shown to the owner before confirming.

## Yazi navigation (UX pillar work, no architecture)

- **TUI**: bring the 3-column browser to yazi key parity — `h`/`l` walk columns, `H`/`L` history, `gg`/`G`, `z`-prefix jumps, preview pane toggling. The structure already matches; this is keybinding + polish.
- **Web / Apple**: adopt the same miller-column browsing model for block trees so the navigation grammar is identical across clients.

## Data model deltas (beyond the rename)

```sql
CREATE TABLE server_keys (           -- exactly one row
  public_key   BYTEA NOT NULL,
  private_key  BYTEA NOT NULL,       -- encrypted at rest by deployment
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE block_federations (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  block_id      UUID NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
  role          TEXT NOT NULL,       -- 'home' | 'follower'
  peer_url      TEXT NOT NULL,
  peer_pubkey   BYTEA NOT NULL,
  jurisdiction  TEXT,
  status        TEXT NOT NULL,       -- 'active' | 'revoked' | 'severed'
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at    TIMESTAMPTZ
);

CREATE TABLE federation_events (     -- signed control-plane log, home-authored
  id            BIGSERIAL PRIMARY KEY,
  block_id      UUID NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
  kind          TEXT NOT NULL,       -- member.add/remove/role_change, role.*, erasure.*, federation.revoke, ...
  payload       JSONB NOT NULL,
  signature     BYTEA NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE blocks        ADD COLUMN owner_user_id UUID NOT NULL REFERENCES users(id);
ALTER TABLE blocks        ADD COLUMN copied_from JSONB;              -- provenance for copies
ALTER TABLE block_members ADD COLUMN origin_server TEXT;             -- null = local member
```

New capabilities: `block.federate`; renamed: `vault.manage`→`block.manage`, `vault.export`→`block.export`.

## Phased rollout

| Phase | Scope | Depends on |
|---|---|---|
| R | Full-stack rename across server, web, TUI, Apple, site, monorepo docs; migration `0002`; disk-format fallback readers | ⚑ rename-depth lock |
| O | Ownership column + transfer endpoint + erasure re-key; share-a-copy endpoint + UI in all clients | R |
| U | Yazi navigation parity (TUI keys; web/Apple miller columns) — parallelizable with O | R |
| F1 | Server identity (keypair, identity endpoint), federation invite/accept handshake, `block_federations` | ⚑ federation-model lock |
| F2 | Content-plane relay: server-to-server WS, origin tagging, replica FS mirror, reconnect/state-vector reconciliation | F1 |
| F3 | Control plane: signed `federation_events`, membership-across-servers, erasure propagation, LGPD surfaces (consent, notice, residency, export) | F2 |

Each phase ends with a working build; the v2 testing bar carries over, plus F-phases require **two-server integration tests** (testcontainers with two composed lumi-servers: invite → accept → concurrent edits converge → revoke → erasure propagates). This also finally forces the `-tags=integration` suite the v2 Makefile promised.

## Out of scope for v3.0

- Paragraph-level blocks / transclusion (see naming risk).
- E2E encryption — federation raises its value considerably (followers hold plaintext); flagged as the top v3.x candidate, not in v3.0.
- Central identity / SSO across federated servers.
- Public read-only sharing; comments; full-text search (unchanged from v2's out-of-scope list).

## Threat model deltas

- A follower server is a **full-trust replica** for the blocks it federates: it sees plaintext and holds a copy after severance. Federation is therefore an *operator-to-operator* trust decision made per block by the owner — the UI must never let it look like a casual toggle.
- Signed control events prevent a follower from forging membership/role changes; they do not prevent a malicious follower from ignoring revocations locally. Severance semantics (keep-copy, stop-sync, surface state) are honest about this.
- Server private keys are new crown jewels: compromise allows impersonating the server in federation handshakes. Stored encrypted at rest; rotation procedure documented before F1 ships.
