# lumi — v3 product & architecture spec

Status: **locked 2026-07-03** (rename dropped by owner decision; federation follows the home-server authority model). Extends [`SPEC.md`](./SPEC.md) (v2, shipped 2026-05-28). The unit of organisation keeps the name **vault**.

## Overview

lumi v3 is three moves on top of the shipped v2 foundation:

1. **UI-note first.** lumi is primarily a note-taking *interface*: vim-focused editing and yazi-style navigation in every client, with security, performance, and privacy leading every trade-off.
2. **Ownership and copies.** A vault **belongs to a user** (transferable). It is shared either by inviting others into its **workspace** (live collaboration, exists today) or by **sharing a copy** (a fork with no live link, new).
3. **Federation.** A vault's workspace can sync not just between clients but **between servers**: an operator can be invited to federate a vault, after which it converges across all clients of all federated servers. CRDT-based, replacing v1's abandoned last-write-wins peering.

## Pillars

Unchanged: **Security/LGPD → performance → DX → scale → UX → UI → QA.** Federation is the feature most in tension with pillar 1; the home-server authority model below resolves it. Any federation behavior that cannot satisfy LGPD obligations (consent, erasure, audit) does not ship.

## Terminology

| Term | Meaning |
|---|---|
| **Vault** | A portable directory of markdown notes + `.lumi/` metadata (unchanged from v2). |
| **Workspace** | The collaborative surface of a vault: members, roles, invites, presence, live CRDT state. |
| **Owner** | The user a vault belongs to. Exactly one per vault; transferable. |
| **Copy / fork** | A snapshot duplicate of a vault handed to another user, with no live link back. |
| **Home server** | The server where a vault was created. Authoritative for the vault's workspace. |
| **Follower server** | A server invited to federate a vault. Replicates content, defers to home for control. |

## Locked decisions

| Area | Decision |
|---|---|
| Naming | **Vault stays** — the block rename was dropped (owner decision, 2026-07-03) |
| Ownership | One owner per vault (`owner_user_id`), transferable; owner holds an irrevocable Admin-equivalent grant |
| Sharing modes | Workspace invite (live, existing) **and** share-a-copy (fork, new) |
| Federation authority | **Home-server model**: home is authoritative for identity, membership, roles, invites, audit; followers replicate content and enforce signed control events |
| Federation transport | Server-to-server WebSocket per federated vault: Yjs updates + signed control events |
| Server identity | Ed25519 keypair per server; identity = (URL, public key) |
| Conflict resolution | Yjs CRDT everywhere (unchanged); federation relays updates, never merges by timestamp |
| Client sync semantics | Unchanged: web/Apple live Yjs WS, TUI snapshot+diff |
| Identity | Server-scoped accounts, no central SSO; federated members recorded as `username@server` |

## Vault model

v2's vault model carries over verbatim, with ownership added:

```sql
ALTER TABLE vaults ADD COLUMN owner_user_id UUID REFERENCES users(id);
-- backfilled from created_by in migration 0002, then NOT NULL
```

- Owner always holds an Admin-equivalent grant that `members.manage` cannot remove.
- `POST /api/vaults/:vault/transfer-ownership { user_id }` — owner only; target must already be a member; audited (`vault.transfer`).
- LGPD erasure re-keys on *owner*: deleting a user hard-deletes their owned never-shared vaults; owned-and-shared vaults require transfer or explicit opt-in (v2 §Right-of-erasure semantics, re-keyed from sole-admin to owner).

## Sharing

**Workspace invite** — unchanged from v2.

**Share a copy** — new:

- `POST /api/vaults/:vault/copies { recipient_username }` (capability `vault.export`): server forks the vault's current FS state + fresh CRDT init into a new vault owned by the recipient, new vault id/slug. No membership, no live link. Provenance recorded (`copied_from` JSONB + `vault.copy` audit event).
- A copy shares nothing after creation: edits diverge permanently.

## Federation

### Model — home-server authority

The home server is authoritative for the **control plane** of a vault: users, membership, roles, invites, revocations, audit, erasure. Follower servers replicate the **content plane** (Yjs state) and enforce control decisions received as signed events. Permissions and LGPD obligations get an unambiguous authority; CRDT content, which merges safely, is the only thing that multi-masters.

### Server identity & handshake

1. Every server generates an Ed25519 keypair at first boot (`server_keys` table); `GET /api/federation/identity` returns `{ url, public_key }`.
2. A vault admin with new capability `vault.federate` creates a **federation invite**: `POST /api/vaults/:vault/federation-invites { server_url, expires_at }` → opaque token, delivered out-of-band to the other operator.
3. The follower operator's server calls home's `POST /api/federation/accept { token, follower_identity, signature }`. Home verifies, records `vault_federations`, returns the vault's current snapshot + control-state bundle. Both sides audit (`federation.invite`, `federation.accept`).
4. Either side can sever: home revokes (signed revocation event) or the follower unlinks. Severed followers keep their last copy (they were trusted with plaintext) but stop syncing and MUST surface "no longer synced" to their users.

### Content plane

- Per federated vault, follower maintains a server-to-server WS to home (`WSS /api/federation/vaults/:vault/sync`), authenticated by Ed25519 signature over a challenge; reconnect with backoff.
- Yjs updates flow both directions, tagged `origin_kind = 'federation:<server-url>'`. Loop prevention by origin tag; CRDT idempotency makes residual echoes harmless.
- Followers persist their own snapshot + update log and mirror to their own FS — a follower is a full content replica, so its local clients work against it unchanged.
- Home offline: follower clients keep editing; state reconciles by state-vector diff on reconnect. Follower offline: same, other direction.

### Membership across servers

- Federated users are members on **home's** member list as `username@follower-url` (`origin_server` column on members). Role checks on a follower use its replicated control-state cache; grants/revocations only originate at home.
- Presence/awareness relays across the federation link (clientID namespaced by server).

### LGPD across the federation

- **Owner consent**: creating a federation invite requires the vault owner (or owner-granted `vault.federate`), is audited, and the UI states plainly that content will be stored plaintext on the other operator's infrastructure.
- **Member notice**: `GET /api/vaults/:vault` includes the `federations` list.
- **Erasure propagation**: user-erasure and note-deletion at home emit signed control events; followers MUST apply the same cascades and ack; missing acks within a bounded window are surfaced to the home operator.
- **Export** includes the servers a user's data was federated to.
- **Residency**: each federation accept records operator-declared jurisdiction, shown to the owner before confirming.

## Yazi navigation (UX work, no architecture)

- **TUI**: 3-column browser to yazi key parity — `h`/`l` walk columns, `H`/`L` history, `z`-prefix jumps, preview toggling.
- **Web / Apple**: adopt the same miller-column browsing grammar.

## Data model deltas

> **Implementation deviation (F3, 2026-07-03):** the `federation_events`
> incremental log below was replaced by ONE signed full control-state
> document per vault (`federation_control_state` on home,
> `replicated_control_state` on followers), versioned by `seq` and pushed
> whole on every change. Full-state replication is idempotent and immune to
> event-ordering/replay bugs at vault-membership scale; change history stays
> in `audit_log`. Cross-server members live in `federated_vault_members`
> (member_key = `username@server-url`) because `vault_members.user_id`
> FK-references local users. Follower replication lag is tracked as
> `vault_federations.last_acked_seq`.

```sql
CREATE TABLE server_keys (           -- exactly one row
  public_key   BYTEA NOT NULL,
  private_key  BYTEA NOT NULL,       -- encrypted at rest by deployment
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vault_federations (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id      UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
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
  vault_id      UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  kind          TEXT NOT NULL,       -- member.*, role.*, erasure.*, federation.revoke, ...
  payload       JSONB NOT NULL,
  signature     BYTEA NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE vaults        ADD COLUMN owner_user_id UUID REFERENCES users(id);  -- NOT NULL after backfill
ALTER TABLE vaults        ADD COLUMN copied_from JSONB;
ALTER TABLE vault_members ADD COLUMN origin_server TEXT;                       -- null = local member
```

New capability: `vault.federate` (granted to Admin seed role via wildcard).

## Phased rollout

| Phase | Scope |
|---|---|
| O | Ownership (migration 0002, transfer endpoint, erasure re-key) + share-a-copy (endpoint + audit) on the server; surface in web/TUI/Apple |
| U | Yazi navigation parity (TUI keys; web/Apple miller columns) — parallel with O |
| F1 | Server identity (keypair, identity endpoint), federation invite/accept handshake, `vault_federations` |
| F2 | Content-plane relay: server-to-server WS, origin tagging, replica FS mirror, state-vector reconciliation |
| F3 | Control plane: signed `federation_events`, cross-server membership, erasure propagation, LGPD surfaces |

Each phase ends with a working build. F-phases require **two-server integration tests** (testcontainers, two composed lumi-servers: invite → accept → concurrent edits converge → revoke → erasure propagates) — this finally lands the `-tags=integration` suite the v2 Makefile promised.

## Out of scope for v3.0

- Paragraph-level blocks / transclusion.
- E2E encryption — federation raises its value (followers hold plaintext); top v3.x candidate.
- Central identity / SSO across federated servers.
- Public read-only sharing; comments; full-text search (unchanged from v2).

## Threat model deltas

- A follower server is a **full-trust replica** for the vaults it federates: it sees plaintext and keeps its copy after severance. Federation is an operator-to-operator trust decision made per vault by the owner — the UI must never make it look like a casual toggle.
- Signed control events prevent followers forging membership/role changes; they cannot force a malicious follower to honor revocations locally. Severance semantics are honest about this.
- Server private keys are crown jewels: compromise allows impersonation in federation handshakes. Encrypted at rest; rotation documented before F1 ships.
