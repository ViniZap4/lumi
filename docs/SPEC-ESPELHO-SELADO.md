# One-way mirrors and sealed (end-to-end encrypted) books

- Status: proposed design (SPEC-PARTE2 §11). Nothing here is implemented.
- Date: 2026-11-24
- Scope: (A) one-way server-to-server mirrors of a book; (B) sealed books, where
  the server stores and relays only ciphertext; (C) phased plan and the
  decisions the owner must make.
- Grounded in: `SPEC.md` (v2), `SPEC-V3.md` (federation), `docs/SPEC-LIVROS.md`
  (books, feed, `seq`), `docs/CRDT-CLIENT-PROTOCOL.md` (CRDT rules, `crdt_epoch`),
  `docs/SPEC-PARTE2.md`, `server/docs/ADMIN.md`, and the server code under
  `server/internal/{federation,crdt,wsync,notes,feed,fswatch,vaults,auth}` and
  `server/migrations/0001…0009`. Where this document and the CRDT protocol
  disagree about plaintext books, the CRDT protocol wins. Sealed books get their
  own protocol (§B7) and do not change the plaintext one.
- Keywords MUST / MUST NOT / SHOULD / MAY follow RFC 2119.
- Pillars, in priority order: security/LGPD, performance, DX, scale, UX, UI, QA.
- "Book" (UI) = "vault" (code, API, DB), as in SPEC-LIVROS.

---

## 0. Summary

**One-way mirror.** A mirror is an existing federation link with a new
`mode = 'mirror'`. Home authorises it with a federation invite that names the
mode. The operator of the mirror server accepts it with the same signed
handshake used today (version bumped so the mode is signed). Content, metadata,
deletes, epochs and home's signed control state flow home → mirror only. Home
drops every inbound content frame on a mirror link. The mirror clamps every
local role to read-only at the capability resolver (one choke point). It also
turns off fswatch and lazy init for the book and treats notes that home no
longer lists as deleted. Followers today re-announce them instead (gap G2). A
mirror acknowledges home's vault `seq`, so home can show the replication lag
and whether a deletion or erasure has reached the mirror. An optional
**publish** variant serves a filtered subset of the book anonymously from the
mirror. It needs explicit consent from the home owner and is off by default.

**Sealed books.** Clients encrypt everything that has meaning: Yjs updates,
client-made snapshots, awareness, note paths, titles and kinds, and the book
name. Clients get a per-book symmetric **book key** (BK) in numbered
**generations**. The BK is wrapped with HPKE to each member's **user key**.
The user key is wrapped to each of the user's **devices** and to a **recovery
key**. Removing a member rotates the BK. The server becomes an ordered,
deduplicating store-and-relay of opaque envelopes. It assigns a per-note
sequence number and acknowledges each push. Clients do compaction by
uploading encrypted snapshots. The server loses the features that need
plaintext: the FS mirror, fswatch, server-side 3-way `/diff`, `/content`,
title search, server-side copy and plaintext export. It keeps the features that
do not: the feed, `seq`/`changes`, members and roles, audit metadata,
federation and mirrors (as ciphertext replication). Web and Apple get full
support. The TUI shows sealed books as locked in v1 and gets full support later
through a local Yjs engine (yrs compiled to WebAssembly, run by wazero).

Recommended crypto suite **LS1**:

- AES-256-GCM, with per-(book generation, note) keys derived by HKDF-SHA256 and
  96-bit random nonces.
- HPKE (RFC 9180) base mode, DHKEM(X25519, HKDF-SHA256) / HKDF-SHA256 /
  AES-256-GCM, for wrapping.
- Ed25519 for signatures.
- A 256-bit recovery key, so no password KDF is needed.

All of these are native in WebCrypto, CryptoKit and the Go standard library. On
the web every long-term key can stay a non-extractable `CryptoKey`. §B6.1
explains why this suite beats XChaCha20-Poly1305 for lumi.

---

## 1. Federation today (what the design builds on)

Facts read from the code. File references are relative to `server/internal/federation`.

| Aspect | Current behaviour |
|---|---|
| Identity | One Ed25519 keypair per server (`server_keys`, created at first boot). `GET /api/federation/identity` → `{url, public_key}`. `GET /api/info` shows the same key to clients. |
| Link | `vault_federations(vault_id, role ∈ {home, follower}, peer_url, peer_pubkey, jurisdiction, status ∈ {active, revoked, severed}, last_acked_seq)`. Both sides keep a row with the same vault UUID. |
| Authorisation to link | Home: a user with `vault.federate` creates a single-use invite (`federation_invites`, 7-day TTL). Follower: only an operator (`LUMI_FEDERATION_OPERATORS`) may `POST /api/federation/join`. |
| Handshake | The follower signs `lumi-federation-accept-v1|token|follower_url` with its server key. Home verifies it and marks the invite used, atomically (`federation.go` `Accept`). |
| Relay auth | `POST /api/federation/sync-challenge` mints a single-use nonce. Only an active **home**-role row answers; anything else gets a uniform 403. The follower dials `WSS /api/federation/vaults/:vault/sync?peer_url&nonce&sig`, signing `lumi-federation-sync-v1|nonce|vault|follower_url` (`relay_handlers.go`). |
| Outbound network | `NetPolicy` (`netguard.go`): https only, the IP resolved at every dial must be public unicast unless listed in `LUMI_FEDERATION_ALLOWED_CIDRS`, no proxies, no redirects, bounded timeouts. |
| Content plane | One WS per (vault, link). Frames (`protocol.go`): `1` manifest, `2` noteSync (a y-protocols message per note), `3` announce, `4` delete, `5` control state, `6` control ack, `7` move. Epochs ride as optional trailing data. Caps: 32 MiB per message, 200 000 manifest entries. |
| Relay symmetry | The `Session` is symmetric (`relay.go`). The role only decides who sends the opening manifest. Updates, deletes, moves and announces are accepted **from either side** and relayed onward to the vault's other links, except the one they arrived on. |
| Control plane | Home signs **one** full JSON document per vault: name, roles, members keyed `username@server-url`, and a `seq` (`control.go`). The follower verifies the signature against the pinned home key and stores it (`replicated_control_state`). `ControlResolver` answers authorisation for follower vaults from that document. |
| Revocation | A notice signed by home is pushed to `{follower}/api/federation/revoked`, or pulled with a 410 during the challenge when the follower proves its key (`revoke.go`). The follower marks its row `severed` and keeps its copy. |
| Epochs | A follower creating a row adopts the sender's `crdt_epoch`. Existing rows keep theirs (CRDT §12.7). |
| Disk | Followers mirror every note to their own FS through `notes.Service.WriteBodyFromCRDT`. |

### 1.1 Gaps this design must close or work around

- **G1, frontmatter not replicated.** A note announced over the relay gets its
  file created with only `id`, `title` and the follower's local `created_at`
  (`notes.go` `WriteBodyFromCRDT`). Tags and the original `created_at` are lost.
  A mirror used as a backup MUST carry them (§A5).
- **G2, deletes missed while offline are resurrected.** In the manifest
  handler, the follower *announces* every local note that home's manifest does
  not list (`relay.go`, `frameManifest`). Suppose home deleted a note while the
  link was down. On reconnect the follower re-announces it and home
  re-creates it. This is wrong for any link and fatal for erasure on a mirror
  (§A8).
- **G3, home trusts follower writes structurally.** Home applies
  `noteSync`/`delete`/`move`/`announce` from a follower without any
  per-capability check. Only the follower's own capguard stands in the way. A
  compromised follower can delete notes at home. For a mirror link, home MUST
  drop these frames (§A4.3). For replica links this stays a documented
  operator-trust assumption (SPEC-V3 threat model).
- **G4, no metrics endpoint.** Only the counters in `GET /admin/v1/status`
  exist (§A14).

---

# Part A: one-way mirror

## A1. Definition, goals, non-goals

A **mirror** is a server that holds a read-only replica of a book homed
elsewhere, kept current in near real time.

Goals:

1. **Backup.** An off-site copy on another operator or another jurisdiction,
   restorable by promotion (§A12).
2. **Read access close to readers.** Members of the home book can read it on
   the mirror server with their account there.
3. **Publishing (optional).** Anonymous read-only access to a selected subset
   (§A15).

Non-goals (v1):

- Mirrors of mirrors (chains).
- Mirror-local membership. The mirror's own operator cannot grant access; home
  is the only authority.
- Partial mirrors for backup. Filtering exists only for publishing.
- Automatic fail-back after promotion.

## A2. Model

A mirror is a federation link with a mode:

```
vault_federations.mode TEXT NOT NULL DEFAULT 'replica'
    CHECK (mode IN ('replica', 'mirror'))
```

| Side | `role` | `mode` | Meaning |
|---|---|---|---|
| home | `home` | `mirror` | This link points at a mirror. Send only; never accept content. |
| mirror | `follower` | `mirror` | This server holds a read-only mirror of the vault. |

`replica` is today's bidirectional follower, unchanged. Clients see
`federation.role = "mirror"` in `GET /api/vaults` (SPEC-LIVROS §3.5 already
shows `home|follower|null`). A vault on a given server has **at most one**
follower-side row, so it is either a replica or a mirror, never both. Home may
hold any number of mirror links and replica links on the same vault.

## A3. Setup flow

```
Home owner (vault.federate)            Home server                Mirror operator          Mirror server
  │ create mirror invite ─────────────▶│ federation_invites
  │   {mode:"mirror", expires_at,      │   (+ mode, publish policy)
  │    publish:{allowed, scope}?}      │
  │◀── token (shown once) ─────────────│
  │ hand token to mirror operator out of band ─────────────────▶│
  │                                    │               │ POST /api/federation/join
  │                                    │               │   {home_url, token, mode:"mirror"} ─▶│ (operator only)
  │                                    │◀──────── GET identity; POST accept v2 ────────────────│
  │                                    │  verify sig over v2 msg (mode inside)                 │
  │                                    │  mark invite used; insert row (home, mirror)          │
  │                                    │── AcceptResponse{vault, home, mode, publish} ────────▶│
  │                                    │                                         insert row (follower, mirror)
  │                                    │                                         create vault (same UUID), clamp read-only
  │                                    │◀═══════ relay WS (challenge + sig), mode enforced ═══▶│ backfill (§A7)
```

Rules:

- The invite fixes the mode. `POST /api/vaults/:vault/federation-invites`
  gains `{"mode": "replica"|"mirror", "publish": {...}?}`. The default is
  `replica` for backward compatibility. A joiner cannot upgrade a mirror invite
  to a replica, because home reads the mode from its own invite row, not from
  the request.
- Accept message v2: `lumi-federation-accept-v2|<token>|<follower_url>|<mode>`.
  Home accepts v1 signatures only for `replica` invites. A v1 follower that
  presents a mirror invite is refused with `409 {"error":"mirror_unsupported"}`.
  An old follower must not join a mirror invite as a writable replica.
- `AcceptResponse` gains `mode` and `publish`. The mirror stores both. If
  `mode` differs from what the joining operator asked for, the mirror refuses
  to provision.
- **Who authorises, on each side:**
  - Home: a user with `vault.federate` on that vault. That is the owner, or an
    owner-granted role (SPEC-V3 LGPD "owner consent"). Publish consent
    additionally needs `vault.manage` (§A15).
  - Mirror: a server operator (`LUMI_FEDERATION_OPERATORS`), as today.
- **UI copy at invite time** (home, web/Apple):
  - "A full read-only copy of this book, in plain text, will be stored on
    another operator's server (jurisdiction: …). It keeps updating until you
    revoke it. Revoking does not erase what they already hold."
  - The jurisdiction is the one the mirror declares at accept. It is shown to
    the owner in the federations list, as today.
- The mirror vault is provisioned with the same UUID and home's slug, the same
  as a replica. On a slug clash the mirror appends `-m`, `-m2`, … (a local
  slug is not part of the identity).

## A4. Protocol changes

### A4.1 Relay auth

No changes. `sync-challenge` and the upgrade guard still require an active
home-role row. The session object learns `mode` from the row (home side) or
from its own follower row (mirror side).

### A4.2 New frames

```
frameMirrorAck   uint64 = 8  // varUint(home_vault_seq)            mirror → home
frameResumeFrom  uint64 = 9  // varUint(home_vault_seq)            mirror → home (§A7.2)
frameChangesEnd  uint64 = 10 // varUint(home_vault_seq)            home → mirror (§A7.2)
frameNoteMeta2   uint64 = 11 // noteMeta + varBytes(frontmatterJSON) home → mirror/replica (G1)
```

Peers that predate these frames ignore unknown frame types (`handleFrame`
`default: return nil`). That is safe because the new frames are only sent on
links whose mode both sides agreed on at accept time.

`frameNoteMeta2` carries the frontmatter fields that are *not* derived from
the CRDT: `tags`, `created_at`, plus any unknown keys up to 16 KiB of JSON.
`id`, `title` and `updated_at` are excluded because the receiver computes
them. Home sends it after each announce and after each frontmatter change. The
change is detected where the notes service already bumps `change_seq` for
metadata writes and where fswatch merges a file whose frontmatter changed. The
receiver writes those keys into the frontmatter on the next mirror write.
Replica links benefit too (closes G1).

### A4.3 Home side of a mirror link (send only)

| Inbound frame from a mirror | Action |
|---|---|
| `noteSync` Step1 | Answer with Step2 (the diff the mirror lacks). This is the only content input. |
| `noteSync` Step2 / Update | If the update is a no-op (`[0,0]`, or it changes nothing): ignore it. Otherwise: **close the link** and log `federation.mirror_violation` (audit plus an operator alert). A mirror never authors content. |
| `announce`, `delete`, `move`, `noteMeta2` | Close the link, log `federation.mirror_violation`. |
| `controlAck`, `mirrorAck`, `resumeFrom` | Process. |
| anything else | Ignore. |

Also on home: `Links.OnPersist`/`NoteCreated`/`NoteDeleted`/`NoteMoved` keep
fanning out to mirror links as they do today. Home does **not** send Step1s on
a mirror link (`sendOpening` skips the per-note Step1). Nothing should flow
back, so there is no reason to ask for it. Home still sends the manifest, or
the incremental resume of §A7.2, and the control state.

### A4.4 Mirror side

| Situation | Action |
|---|---|
| Manifest received | For each entry: `ensureNote` + Step1. Then **delete every local note absent from the manifest** (this replaces the replica's re-announce; fixes G2 for mirrors). |
| Step1 from home | Ignore (home should not send one; defence in depth). |
| Local persist hook fires for this vault | It MUST only fire for `federation:<home>` origins. Any other origin on a mirror vault is a bug: drop the update, log `mirror.local_write_blocked`, raise the `lumi_mirror_local_write_blocked_total` metric. |
| `delete` / `move` / `announce` / `noteMeta2` from home | Apply, then record a tombstone or seq bump in the mirror's own feed (SPEC-LIVROS §3.4), so the mirror's clients update. |
| Control state from home | Verify and store as today. The resolver clamps roles (§A6). |

## A5. What replicates

| Item | Replicated to the mirror | Notes |
|---|---|---|
| Note CRDT state (markdown and canvas) | yes | Yjs state through Step1/Step2 and live updates. Canvas is just a note kind. |
| Note metadata: id, path, title, kind | yes | manifest, announce, move. Kind is derived from the path. |
| Frontmatter (tags, created_at, extra keys) | yes | `frameNoteMeta2` (new; also fixes G1 for replicas) |
| `crdt_epoch` | yes | adopted at row creation (L5). The mirror never seeds (S5), so epochs always name home's history. |
| Deletes / tombstones | yes | `delete` frame plus manifest/resume reconciliation (§A8). The mirror writes its own tombstones with its own `seq`. |
| Vault name | yes | Inside the control state (existing). |
| Roles and members | yes, clamped | Control state (existing). The mirror resolves `username@mirror-url` against it, then clamps (§A6). |
| Federated member grants for other servers | only as control-state entries | Entries for other servers are inert on the mirror. |
| Home's audit log | **no** | Home's audit stays home's (data minimisation). The mirror keeps its own audit for its local events. |
| Invites, sessions, users | **no** | |
| Other federation links of the vault | **no** | A mirror does not learn who else follows the book. |
| Awareness / presence | **no** | Nobody edits on a mirror. |
| Attachments (files other than `.md`/`.canvas` in the vault dir) | **not in v1** | The same limitation as replicas. Listed as a risk for backup completeness (§C3). |
| Sealed book content | ciphertext only | §B11. |

## A6. Read-only enforcement on the mirror (layered)

1. **Capability clamp (primary).** `ControlResolver.RoleForUser` returns the
   control-state role intersected with `{note.read, vault.export}` when the
   vault's follower row has `mode = 'mirror'`. Every REST write route already
   goes through `capguard.RequireCapability`, so every write gets 403. The
   error body changes from the generic `forbidden` to
   `{"error":"mirror_read_only"}` so that clients can explain it. The feed
   `hello`/`vault.access` therefore reports `can_edit = false`, and clients show
   "Somente leitura" with no client changes (SPEC-LIVROS §7 already anticipates
   this).
2. **Live socket.** `ws-ticket` requires `note.edit`, so it is refused. Mirror
   readers use REST and the feed, as read-only viewers already do (CRDT §2.3).
   Phase M2 adds an optional **read-only live view**: `POST …/ws-ticket
   {"mode":"read"}` requires only `note.read`. The hub marks the subscriber
   read-only and closes it with `4403` on any non-empty Step2/Update. Clients
   join empty, receive Step2 and live updates, and never send edits. This also
   helps plain Viewers on any server.
3. **Disk.** The vault directory is not registered with fswatch. External
   edits to the mirror's files are never merged. The next mirror write
   overwrites them, and the nightly integrity pass (§A10) rewrites any file
   whose hash differs from the CRDT text.
4. **No seeding.** Registry lazy init (CRDT §10 S4) is disabled for mirror
   vaults. A note row with no CRDT state stays empty until the relay delivers
   it, and `/content` answers `409 {"error":"mirror_syncing"}` meanwhile.
5. **Internal write paths.** These are refused on a mirror vault:
   `CopyVaultNotes` *into* it, vault rename, transfer of ownership,
   federation invites (`vault.federate` is clamped, so no chains), and the
   `copies` endpoint as a target. A copy *from* a mirror into a new local vault
   is allowed when the reader holds `vault.export`. That is a read.
6. **Admin surfaces.** The admin panel and CLI refuse note-level writes and
   rename on mirror vaults. Delete vault (drop the local copy) and
   resync/promote stay allowed (§A13).

Test obligation: a table-driven test that runs every route registered on the
authed router against a mirror vault, with an Admin-role member. It asserts
that each non-GET route returns 403 `mirror_read_only` (or 404/405 where that
is correct), so a route added later cannot silently bypass the clamp.

## A7. Catch-up and backfill

### A7.1 Initial backfill

The mirror connects. Home sends the manifest, without per-note Step1s on a
mirror link. The mirror runs `ensureNote` for every entry, then sends Step1
with an empty state vector **in a window of at most 32 notes in flight**.
Each Step2 that arrives frees a slot. This bounds memory on both sides. Without
the window, a 50 000-note book would queue 50 000 full states at once, up to
the 1024-frame `sendBuffer` and then a dropped link. The mirror persists a
`backfill_cursor` (the last note id fully applied, in manifest order). After a
reconnect it skips notes that are already complete. The comparison is still
done by state vector, so being wrong is only a cost, never a correctness
problem. Progress (`notes_synced/notes_total`) is shown in the admin panel.

### A7.2 Incremental resume (M2)

A full manifest plus one Step1 per note costs O(notes) on every reconnect.
The mirror stores `home_seq`, the highest home vault `seq` it has fully
applied. It acknowledges this with `frameMirrorAck` (debounced 1 s) and stores
it on home in a new column, `vault_federations.mirror_acked_seq`.

On connect the mirror sends `frameResumeFrom(home_seq)` **before** the opening
completes. Home then:

- if `home_seq >= vault_heads.tombstone_floor`: does not send the manifest.
  Instead it walks `notes WHERE change_seq > home_seq` (announce + noteMeta2 +
  Step1-from-mirror for each) and `note_tombstones WHERE seq > home_seq`
  (delete), then sends `frameChangesEnd(head)`.
- otherwise sends the full manifest (§A7.1). Deletion of absent notes covers the
  tombstones that were pruned.

Content edits bump `change_seq` through the coalescer, so changed content is
covered too. The mirror sends Step1 for those notes and receives only the diff.

### A7.3 Live

After catch-up, home's persist hook fans each update to the mirror link (the
existing path). The latency target is the same as for replicas: under 1 s p95
inside a region.

## A8. Deletes, erasure, lineage

- **Live delete:** the `delete` frame. The mirror deletes the row, the file and
  the CRDT (`DeleteFromFederation`), drops the live room (`4404` to readers),
  writes a tombstone in its own feed, and acknowledges through the next
  `mirrorAck`.
- **Delete missed while offline:** covered by the tombstones in §A7.2 or by
  deleting absent notes in §A4.4. This is the core of **erasure
  propagation**. On home, each link shows `pending_deletes`. It is computed as
  `count(note_tombstones WHERE seq > mirror_acked_seq)`, together with the age
  of the oldest one. When the oldest is older than
  `LUMI_MIRROR_ERASURE_SLA` (default 24 h), the home admin panel flags it and
  writes `federation.erasure_lagging` to the audit log once per link per day.
- **Purge on the mirror:** deletion on a mirror removes the Postgres rows
  (cascade: snapshot, update log, epoch) and the file. Backups are the
  mirror operator's responsibility. The UI copy says so.
- **User erasure at home:** the control state drops the member. The mirror
  revokes that user's access when it applies the next state (`controlAck`
  already lets home see it). Content the erased user authored stays, the same
  as at home (vault content is shared property, SPEC.md LGPD).
- **Lineage:** the mirror adopts epochs at row creation. A delete followed by a
  recreate at home produces a delete frame and then an announce with the new
  epoch. The mirror drops the old room (`4404`) before creating the new row,
  so readers never mix two histories. The CRDT §12.7 known gap, where both
  sides recreate while the link is down, cannot happen on a mirror, because the
  mirror never creates notes.

## A9. Divergence detection

A mirror never authors operations, so for every note **the mirror's state
vector must be dominated by home's**. Two checks:

1. When home answers a mirror Step1 (backfill, resume or the integrity pass),
   it checks whether the mirror's SV holds a client clock that home's SV does
   not cover. If so, the mirror's copy has diverged (a bug, a restored
   foreign backup, or tampering). Home sends the diff as usual and also logs
   `federation.mirror_divergent{note}`.
2. The mirror runs the same check locally against home's SV, which it receives
   in a new optional trailing field of Step2. On divergence it **rebuilds the
   note**. It drops the local doc under the same epoch and requests the full
   state with an empty SV. Then it raises `lumi_mirror_divergent_total`.

## A10. Integrity pass

Every 24 h, with jitter, the mirror sends Step1 for every note, rate-limited
to 50 notes/s. This also re-verifies §A9. It then compares
`sha256(file body)` with `sha256(CRDT text)` and rewrites the files that differ.
For a backup, a full pass that completes is the proof that the copy is good.
The time of the last good pass is shown in the panel.

## A11. Failure modes

| Failure | Behaviour |
|---|---|
| Home unreachable | The mirror keeps serving its last state. The link status in the admin panel is `reconnecting`, using the existing backoff. Readers see the book with sync state "Desatualizado desde <t>" (SPEC-LIVROS §4 states) after `> 5 min`. |
| Mirror unreachable | Home is unaffected. The per-link `sendBuffer` overflow closes the session. `pending_deletes`/lag grow and surface on home. |
| Link revoked by home | Signed notice (push or pull, existing). The mirror marks the link `severed`, then applies the **sever policy** agreed in the invite: `keep` (default for backup mirrors: the copy is frozen read-only; the clamp stays even after severance) or `purge` (default for publish mirrors: the mirror deletes the vault within 1 h and audits `mirror.purged`). The mirror's honesty about this cannot be enforced cryptographically. The UI says so. |
| Mirror operator unlinks | The mirror sets `severed`, stops dialling, and keeps or deletes per the operator's choice. Home sees `revoked` on its next challenge (403 → status `unreachable`, then the operator revokes) or immediately, if the mirror sends the best-effort signed `unlinked` notice (new `POST /api/federation/unlinked`, mirror-signed, symmetric to `revoked`). |
| Mirror sends a content frame | Home closes the link (§A4.3) and logs `federation.mirror_violation`. After 3 violations in 1 h, home auto-revokes the link and notifies the home owner. |
| Manifest larger than the caps | Home pages the manifest (protocol change: `frameManifest` may repeat; the mirror waits for `frameChangesEnd` before deleting absent notes). Required before the cap matters. |
| Disk full on the mirror | `WriteBodyFromCRDT` fails. The CRDT state is in Postgres, so the file is rewritten by the integrity pass once space returns. Alert. |
| Clock skew | Irrelevant to content. Revocation already tolerates 5 min. |
| Postgres restore of an old mirror backup | The mirror's SVs are behind home, so the next resume falls back to the full manifest (its `home_seq` is old) and catches up. Divergence is impossible unless the backup came from another vault. |

## A12. Promotion (disaster recovery)

`lumi-server mirror promote <vault> --confirm` (CLI only; not in the HTTP
admin API, so an admin token alone cannot fork a book):

1. Marks the follower row `severed` (reason `promoted`) and stops the link.
2. Turns the replicated control state into local rows:
   - roles go into `vault_roles`;
   - members whose key is `<username>@<this server>` and who exist locally go
     into `vault_members`;
   - all other members are listed in the output as "not carried over".
   It also lifts the clamp, re-enables fswatch and lazy init, and audits
   `mirror.promote`.
3. The result is an independent home with the same vault UUID and the same
   CRDT histories.

When the old home comes back, both sides consider the link dead (one row is
severed, the other gets 403/410). There is no automatic re-merge. Re-linking is
a new federation, where the promoted server is home and the old home joins as a
replica or a mirror, and the CRDT merges the two histories. Clients move with
"Trocar servidor" (SPEC-LIVROS §2.4), which keeps their Y.Docs. Epochs are the
same, because the mirror adopted them.

## A13. Admin panel, CLI and user API

**User API (home):**

| Route | Change |
|---|---|
| `POST /api/vaults/:vault/federation-invites` | body `{mode, publish?, sever_policy?}` |
| `GET /api/vaults/:vault/federations` | each item: `mode`, `publish`, `sever_policy`, `lag_seq`, `pending_deletes`, `oldest_pending_delete_at`, `last_contact_at` |
| `GET /api/vaults` | `federation.role` may be `"mirror"` (mirror server) |

**Admin API** (`/admin/v1`, `server/docs/ADMIN.md`):

| Route | Purpose |
|---|---|
| `GET /federations?mode=mirror&vault=` | list, with the fields above and the backfill progress |
| `GET /federations/{id}` | detail: status, mode, peer, jurisdiction, `home_seq`/`mirror_acked_seq`, backfill `synced/total`, last integrity pass, divergent count, violations 24 h |
| `POST /federations/{id}/resync` | mirror side: forget `home_seq` and run a full manifest pass. Body `{"note_id"?}` rebuilds a single note. Audited `admin.mirror.resync`. |
| `POST /federations/{id}/pause` / `resume` | stop or restart the dial loop without revoking. Audited. |
| `DELETE /federations/{id}` | existing revoke |

**CLI** (`lumi-server federation …`, mirroring the API):

```
lumi-server federation list [--mode mirror] [--vault v] [--json]
lumi-server federation status <id> [--json]
lumi-server federation resync <id> [--note <id>]
lumi-server federation pause|resume <id>
lumi-server mirror promote <vault> --confirm
```

**Clients:** a mirror book shows a fixed badge, "Espelho de <home host> ·
somente leitura". The edit affordances are already hidden by `can_edit=false`.
The TUI `lumi vault clone` of a mirror book works and marks the local folder
read-only in `books.yaml` (`read_only: true`, an additive field). The TUI
refuses `$EDITOR` saves on it.

## A14. Metrics

The server has no metrics endpoint today (G4). Proposal: an opt-in
`LUMI_METRICS_ADDR` (default off, bound to loopback) serving Prometheus text
format. It has no labels that carry note ids or user ids, only `vault_id` for
mirrors, and only when `LUMI_METRICS_VAULT_LABELS=true`.

| Metric | Type |
|---|---|
| `lumi_federation_links{mode,role,status}` | gauge |
| `lumi_mirror_lag_seq{vault}` (home side: head − acked) | gauge |
| `lumi_mirror_pending_deletes{vault}` / `lumi_mirror_oldest_pending_delete_seconds{vault}` | gauge |
| `lumi_mirror_backfill_remaining{vault}` | gauge |
| `lumi_mirror_violation_total`, `lumi_mirror_divergent_total`, `lumi_mirror_local_write_blocked_total` | counter |
| `lumi_relay_frames_total{dir,type}`, `lumi_relay_bytes_total{dir}` | counter |
| `lumi_publish_requests_total{status}` (publish mirrors) | counter |

The same numbers appear in `GET /admin/v1/status` under `"mirrors"`.

## A15. Publish variant (anonymous read)

**Proposal:** only a *mirror* can publish. Home never serves anonymous
traffic. This keeps the authoring server off the public internet (a DMZ
pattern) and gives publication a separate kill switch.

### A15.1 Consent and scope

- At invite time, home sets
  `publish: {"allowed": true, "scope": "flagged" | "folder:<prefix>" | "all"}`.
  It requires `vault.manage` as well as `vault.federate`. The UI warning says:
  "These notes become public on the internet. Search engines and other people
  may keep copies after you unpublish."
- **Default scope is `flagged`.** Only notes whose frontmatter has
  `publish: true` are published. Canvas files have no frontmatter, so they need
  `folder:` or `all`.
- **Data minimisation: home filters, not the mirror.** For a publish mirror
  link, home relays only in-scope notes. A note that leaves the scope (the
  frontmatter flag is removed, or it is moved out of the folder) gets a
  `delete` on that link. A note that enters gets `announce` + full state. Out-of-scope
  content never reaches the publishing server. This needs home to evaluate the
  scope on every metadata change. It reuses the `noteMeta2` trigger points (§A4.2).
- The mirror operator also has to opt in: `LUMI_PUBLISH_ENABLED=true` and a
  per-vault `lumi-server publish enable <vault> --base-path /p/<slug>`.

### A15.2 Surface

- `GET /api/public/books/:slug`: `{name, notes:[{id,path,title,kind,updated_at}]}`.
- `GET /api/public/books/:slug/notes/:id`: `{id,path,title,kind,body}` (raw markdown or canvas JSON).
- Web client route `/p/:slug/*`: a static reader with no login, no editor and no feed.
- Responses carry no cookies and `Cache-Control: public, max-age=60`. The CSP
  is at least as strict as the app's, plus `sandbox` on the HTML. The
  `X-Robots-Tag` is set per book (default `noindex`; the owner may allow
  indexing at invite time). Attachment serving is out of v1.
- Rate limit: 60 req/min per IP, 10 000 req/min per book. 429 with `Retry-After`.
- Anonymous reads are **not** audited per request (that would be a reader log).
  Only aggregate counters are kept.
- **Unpublish:**
  - Home revokes the link, so the sever policy `purge` applies (default for
    publish mirrors).
  - Or home flips the scope off, so all deletes propagate.
  - The public endpoints answer `410 Gone` for a severed book until it is
    purged.

### A15.3 Rejected alternative

**Public links on home** (`GET /p/<token>`). This is simpler, but it puts
anonymous traffic on the authoring server, and the whole book is already
there. It is kept as a possible later "share a single note" feature, outside
this spec.

## A16. Security analysis (mirror)

| Threat | Mitigation |
|---|---|
| A rogue mirror tries to write into home | Home drops and closes on any content frame (§A4.3). The mode is signed at accept (v2) and stored on home's own invite and link rows. A replica invite cannot be replayed as a mirror invite, or the other way round. |
| A compromised mirror leaks content | Inherent: a mirror holds plaintext. This is the same trust as a replica (SPEC-V3 threat model). Sealed books (Part B) remove it: a mirror of a sealed book holds only ciphertext. |
| SSRF through the mirror's join | Unchanged: only an operator can join. Every dial goes through `NetPolicy` (public IPs only, https, no redirects). The publish feature adds no outbound requests. |
| Mirror ignores revocation or purge | Cannot be enforced. It is documented to the home owner at invite time. Signed notices give an audit trail. |
| Publishing leaks out-of-scope notes | Home-side filtering. The mirror never receives them. Test: a note without `publish: true` never appears in any frame on a publish link. |
| Path traversal from home's metadata | Existing `validateRelPath` + `fs.SafeJoin`. A mirror applies the same checks to a home it cannot fully trust. |
| Resource exhaustion through backfill | Window of 32 notes in flight, existing frame caps, manifest paging. |
| Mirror readers escalating | The resolver clamp is the single choke point, plus the route-table test (§A6). |

## A17. LGPD (mirror)

- **Controller and processor:** the home book owner decides to mirror (an
  audited consent). The mirror operator processes on their behalf. The
  jurisdiction is recorded at accept and shown to the owner, as for replicas.
- **Notice to members:** `GET /api/vaults/:vault` already lists federations,
  and mirrors appear there with `mode`. The member list UI adds the line "This
  book is mirrored to: …".
- **Right of access:** `GET /api/users/me/export` lists the servers that
  mirror books the user belongs to (SPEC-V3 "Export includes the servers…").
- **Erasure:** note deletion propagates with a measured SLA
  (`pending_deletes`, §A8). User erasure propagates through the control state.
  Neither covers the mirror operator's own backups. That is a contractual
  matter, and the invite UI says so.
- **Publishing:** personal data in published notes becomes public. The consent
  dialog is explicit. The default scope is opt-in per note. Unpublishing has a
  defined propagation (§A15.2).

## A18. Migration `0010_mirror`

```sql
ALTER TABLE federation_invites
  ADD COLUMN mode TEXT NOT NULL DEFAULT 'replica' CHECK (mode IN ('replica','mirror')),
  ADD COLUMN publish JSONB,                         -- null = not allowed
  ADD COLUMN sever_policy TEXT NOT NULL DEFAULT 'keep' CHECK (sever_policy IN ('keep','purge'));

ALTER TABLE vault_federations
  ADD COLUMN mode TEXT NOT NULL DEFAULT 'replica' CHECK (mode IN ('replica','mirror')),
  ADD COLUMN publish JSONB,
  ADD COLUMN sever_policy TEXT NOT NULL DEFAULT 'keep' CHECK (sever_policy IN ('keep','purge')),
  ADD COLUMN mirror_acked_seq BIGINT NOT NULL DEFAULT 0,   -- home side
  ADD COLUMN home_seq BIGINT NOT NULL DEFAULT 0,           -- mirror side: last fully applied home seq
  ADD COLUMN backfill_cursor TEXT,                          -- mirror side
  ADD COLUMN last_contact_at TIMESTAMPTZ,
  ADD COLUMN last_integrity_at TIMESTAMPTZ,
  ADD COLUMN paused_at TIMESTAMPTZ;

-- at most one follower-side row per vault (replica XOR mirror)
CREATE UNIQUE INDEX vault_federations_one_follower
  ON vault_federations (vault_id) WHERE role = 'follower' AND status = 'active';

CREATE TABLE publish_settings (                           -- mirror side, operator opt-in
  vault_id   UUID PRIMARY KEY REFERENCES vaults(id) ON DELETE CASCADE,
  base_path  TEXT NOT NULL UNIQUE,
  robots     TEXT NOT NULL DEFAULT 'noindex',
  enabled_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## A19. Tests (mirror)

The F-phase rule applies (SPEC-V3): two-server integration tests with
testcontainers. Required scenarios:

1. Mirror invite, then join, backfill and live update on the mirror within 1 s.
2. A mirror Admin-role member gets 403 `mirror_read_only` on every write route
   (route-table test).
3. A forged update frame from the mirror makes home close the link and log
   `federation.mirror_violation`; home content is unchanged.
4. Home deletes a note while the mirror is offline; on reconnect the note is
   deleted on the mirror (G2 regression) and `pending_deletes` goes to 0.
5. Resume with `home_seq` below `tombstone_floor` falls back to the manifest
   and still deletes.
6. Delete and recreate at home: mirror readers get `4404`, then the new epoch.
7. Frontmatter tags and `created_at` survive replication (G1).
8. Publish `flagged`: a non-flagged note never crosses the link. Flipping the
   flag off deletes it on the mirror. The public endpoint answers 404.
9. Revoke with `purge`: the mirror deletes the vault. Revoke with `keep`: the
   copy stays read-only.
10. Promote: the mirror becomes writable. Local members carry over. Re-linking
    as a replica merges without duplicated text.
11. v1 follower plus mirror invite gives 409 `mirror_unsupported`.

---

# Part B: sealed (end-to-end encrypted) books

## B1. Goals and non-goals

Goals:

1. The content of a sealed book cannot be read by the server operator, a
   database or backup thief, or any federated replica or mirror: note text,
   canvas JSON, paths, titles, kinds, tags, the book name and presence/cursors.
2. Live multi-user editing with Yjs keeps working, offline included.
3. Membership changes are enforced cryptographically for *future* content
   (key rotation on removal).
4. Losing a device does not mean losing data (recovery).

Non-goals (v1):

- Hiding **who** collaborates, **when** and **how much**. Traffic metadata
  stays visible (§B3).
- Protection against a malicious *member*'s reading. A member can read what
  they were given.
- Forward secrecy for past content against a member who held the key.
- Server-side search over sealed content.
- Sealed books in the TUI in v1 (decision D2).

## B2. Threat model

| Adversary | Capabilities | Guarantee for sealed books |
|---|---|---|
| **Honest-but-curious server** (operator, DBA, backup reader, log reader) | Reads all Postgres, disk and logs. Follows the protocol. | Learns only the metadata in §B3. No content, no names, no paths. |
| **Compromised API server** (an active attacker in the server process, without control of the clients) | Can drop, delay, reorder or replay envelopes, roll back snapshots, substitute public keys, forge membership. | **Confidentiality** holds, except through public-key substitution for members added *without* a link key (TOFU; mitigated by fingerprints, §B9.5). **Integrity:** cannot forge or alter content (AEAD, context-bound). It *can* cause denial of service, withhold updates and roll back to an older state. Clients detect per-note seq regression and keyring regression (§B7.8). Rollback across notes is not fully detectable. |
| **Operator who also serves the web bundle** | Ships modified JavaScript. | **Not protected on the web.** It is the classic web E2E limitation. Mitigations: §B13.1 (separately hosted static client, published bundle hashes). Apple and TUI clients are not affected. |
| **Federated replica or mirror operator** | Holds a replica. | Ciphertext only. The same metadata as the home server, minus the user ids of other servers' users. |
| **Malicious member** | Holds the current BK, edits within their capability. | Can read and leak everything they can access. Can damage content through edits or by uploading a bad snapshot. Every snapshot is signed and attributable, and the server keeps history for recovery (§B7.6). A Viewer cannot write through an honest server. Against a compromised server, a Viewer could forge edits until **book write keys** ship (decision D8, §B6.8). |
| **Removed member** | Held BK generations ≤ g. | Cannot read content written under generation g+1 and later. Can still read what they could before. With an honest server they cannot fetch anything at all after removal. |
| **Lost or stolen device** | Has storage protected by the OS (Keychain `ThisDeviceOnly`, Secure Enclave, browser profile). | Depends on the OS lock. The user revokes the device from another device. Optional "rotate everything" (§B9.7). |
| **Network attacker** | — | TLS (existing), plus E2E encryption. |

## B3. What the server sees and does not see

| Server **can** see | Server **cannot** see |
|---|---|
| that a book is sealed; its UUID and random slug | the book name |
| the member list (users, roles, join times), as today | note paths, titles, kinds, tags, folder structure |
| the number of notes, their random ids and creation and deletion times | note text, canvas JSON, Yjs structure (client ids, item counts) |
| the per-note envelope sizes after padding (§B4.3) | exact plaintext lengths (padding hides them up to a bucket) |
| the timing and author (user id, device id) of every push | awareness: cursors, selections, display names inside awareness |
| which notes a user opens (WS joins, REST fetches) | search queries (local only) |
| the key generation, rotation times and who rotated | key material |
| users' and devices' public keys | private keys, recovery key, book keys |

The note-open pattern and edit timing are *traffic metadata*. Hiding them
would need a separate design (dummy traffic, one socket per book with
constant-rate framing). It is out of scope and stated plainly in the UI help
text.

## B4. Identifiers and metadata encryption

### B4.1 Identifiers

- **Book slug:** `s` followed by 20 random base32 characters (`[a-z2-7]`). The
  name is not derived from it. It fits the existing slug regex
  `^[a-z0-9][a-z0-9-]{0,62}$`.
- **Note id:** 26 lowercase base32 characters of 128 random bits, **generated
  by the client**. Sealed ids never carry `.canvas` (the kind is secret) and
  never derive from the title. The plaintext id rule (CRDT §13) does not apply
  to sealed books.
- **`crdt_epoch`:** generated by the client at note creation (32 hex
  characters, 128 random bits) and stored opaquely by the server. Lineage
  semantics are unchanged (CRDT §12). The epoch is bound into every envelope's
  AAD (§B6.4), so updates from an old lineage cannot be replayed into a new
  one.

### B4.2 Encrypted note metadata

Each note has a **meta envelope** (kind `0x04`) whose plaintext is canonical
JSON:

```json
{"v":1,"path":"projects/plan.md","title":"Plan","kind":"markdown","tags":["x"],"created_at":"2026-11-24T10:00:00Z","fm":{...}}
```

`fm` holds the other frontmatter keys. The server stores the envelope with an
integer `meta_version` and replaces it by compare-and-swap. A move or rename
is a new envelope. **Path uniqueness** cannot be checked by the server. If two
notes decrypt to the same path, every client applies the same deterministic
rule: the note with the lexicographically smaller id keeps the path, and the
other is displayed as `<name> (2).<ext>`. The first client with `note.move`
that notices the clash writes a corrected meta for the second note.
Alternative considered: an HMAC "path tag" the server could enforce as unique.
It was rejected because tags must survive key rotation, which means keying
them with a long-lived key, and they leak path equality over time.

The **book name** and book settings are a book-meta envelope (kind `0x05`) on
the vault row. `vaults.name` holds the fixed placeholder `"🔒"` so that admin
tools keep working. `PATCH /api/vaults/:vault {name}` is refused for sealed
books. The client writes the book-meta envelope instead.

### B4.3 Padding

Plaintext framing before encryption: `varuint(len(payload)) || payload || 0x00…`
up to the padded length `L`.

| Kind | Padded length `L` |
|---|---|
| update (`0x01`), awareness (`0x03`) | next multiple of 64 bytes (keystroke-sized updates all look alike) |
| snapshot (`0x02`), note-meta (`0x04`), book-meta (`0x05`) | **Padmé**(n) with n = framed length, minimum 256: `E = ⌊log2 n⌋, S = ⌊log2 E⌋ + 1, z = E − S, mask = 2^z − 1, L = (n + mask) & ~mask`. Overhead ≤ 12 %, and only O(log log n) bits of length leak. |

Decision D3 covers whether to encrypt paths and titles at all.
Recommendation: yes.

## B5. Key hierarchy

```
                     Recovery Key RK (256-bit, shown once)         Passkey PRF (optional, web)
                              │ HKDF → AES-GCM                           │ HKDF → AES-GCM
                              ▼                                          ▼
 Device keys DK_d ──HPKE──▶  User Key UK_u = { X25519 uk_enc, Ed25519 uk_sig }   (one per account per server)
 (X25519 + Ed25519,              │  self-signed; signs device certificates, wraps, rotations
  private never leaves           │
  the device)                    │ HPKE (one wrap per member per generation)
                                 ▼
                        Book Key BK[v, g]  (32 bytes, generation g = 1, 2, …)
                                 │  prev-chain: AES-GCM(BK[g], BK[g−1]) lets the holder of the latest key read history
                                 │ HKDF-SHA256 (per purpose, per note)
             ┌──────────────┬────┴─────────┬──────────────┐
             ▼              ▼              ▼              ▼
      K_note[g,n]    K_aware[g,n]     K_meta[g]      K_book[g]
      (updates,      (awareness)      (note meta)    (book name/settings)
       snapshots)
```

- **User Key (UK).** One per (server, account), because identity is
  server-scoped. Created on the user's first sealed-capable device. Book keys
  are wrapped to `uk_enc`, so a rotation costs O(members), not O(devices).
  `uk_id` = the first 16 bytes of `SHA-256(uk_sig_pub || uk_enc_pub)`.
- **Device Key (DK).** Per installation. The private parts never leave the
  device: a WebCrypto non-extractable key, or Keychain plus Secure Enclave
  wrapping. The UK secret is HPKE-wrapped to each enrolled device. Each device
  is certified by `uk_sig`.
- **Recovery Key (RK).** 256 random bits. It wraps the UK secret. Shown once
  and never stored by lumi (§B9.8).
- **Book Key (BK).** Symmetric, per book and generation. A new generation is
  created on member removal, on a device-loss rotation, or on demand. The
  `prev` chain means a new member, or a member returning on a new device, needs
  only the wrap of the **current** generation to read the whole history.

## B6. Cryptographic construction (suite LS1)

### B6.1 Primitive choice

| Need | LS1 choice | Why |
|---|---|---|
| Content AEAD | **AES-256-GCM**, 96-bit random nonce, per-(generation, note) key from HKDF | Native in WebCrypto, CryptoKit and Go stdlib. On the web, BK can be imported as a **non-extractable HKDF base key** and per-note AES keys derived non-extractable, so XSS can use keys during the session but cannot steal them for later. XChaCha20-Poly1305 needs a JS or WASM implementation on the web (keys as raw bytes in the JS heap) and libsodium on Apple (CryptoKit has only 96-bit-nonce ChaChaPoly). The usual reason to prefer XChaCha, random-nonce safety at high volume, is neutralised by per-note keys plus a usage cap (§B6.3). |
| KDF | HKDF-SHA256 (RFC 5869) | Native everywhere |
| Key wrapping (public key) | **HPKE RFC 9180 base mode**, suite `KEM 0x0020 DHKEM(X25519, HKDF-SHA256)`, `KDF 0x0001 HKDF-SHA256`, `AEAD 0x0002 AES-256-GCM` | Standardised with published test vectors. CryptoKit `HPKE` (macOS 14 / iOS 17), Go (`crypto/hpke` where available, else `cloudflare/circl/hpke`), web `@hpke/core` over WebCrypto X25519. |
| Signatures | Ed25519 (RFC 8032) | WebCrypto (current Chrome, Firefox and Safari), CryptoKit, Go stdlib |
| Hash | SHA-256 | |
| Password KDF | **none in the default design**. Argon2id (`m = 64 MiB, t = 3, p = 1`) only if the owner picks the passphrase recovery option (D1-b) | A 256-bit recovery key needs no stretching, so there is no Argon2 in the browser. |

The alternative suite, **LS1-X** (XChaCha20-Poly1305 via libsodium/noble), is
specified only as a fallback in decision D4. The envelope `version` byte allows
it.

Encoding conventions in this section:

- `||` is concatenation.
- `u8`, `u32be`, `u64be` are fixed-width integers.
- `LP(x)` = `u16be(len(x)) || x`.
- Strings are UTF-8.
- `vault_id` is the 16 raw UUID bytes.
- Labels are ASCII, written in quotes.

### B6.2 Key derivation

```
salt        = vault_id                                   (16 bytes)
K_note[g,n] = HKDF-SHA256(IKM = BK[g], salt, info = "lumi/ls1/k/note"  || 0x00 || LP(note_id), L = 32)
K_aware[g,n]= HKDF-SHA256(IKM = BK[g], salt, info = "lumi/ls1/k/aware" || 0x00 || LP(note_id), L = 32)
K_meta[g]   = HKDF-SHA256(IKM = BK[g], salt, info = "lumi/ls1/k/meta"  || 0x00, L = 32)
K_book[g]   = HKDF-SHA256(IKM = BK[g], salt, info = "lumi/ls1/k/book"  || 0x00, L = 32)
K_prev[g]   = HKDF-SHA256(IKM = BK[g], salt, info = "lumi/ls1/k/prev"  || 0x00 || u32be(g), L = 32)
```

On the web: `importKey("raw", BK, "HKDF", false, ["deriveKey"])`, then
`deriveKey({name:"HKDF", hash:"SHA-256", salt, info}, base, {name:"AES-GCM", length:256}, false, ["encrypt","decrypt"])`.

### B6.3 Envelope format

```
offset  size  field
0       2     magic   0x4C 0x53  ("LS")
2       1     version 0x01       (LS1)
3       1     kind    0x01 update | 0x02 snapshot | 0x03 awareness | 0x04 note-meta | 0x05 book-meta
4       4     key_gen u32be      (≥ 1)
8       16    msg_id             (random; dedupe key; unique per (vault, note))
24      12    nonce              (random)
36      n+16  ciphertext || GCM tag
```

`header` = bytes 0..35 (36 bytes, sent in clear, parsed by the server).

The server parses and validates `magic`, `version`, `kind`, `key_gen` and
`msg_id`, and the size limits (§B7.9). It never touches the rest.

**Usage cap.** A client counts the envelopes it encrypts per (g, note). The
server counts the envelopes it stores per (note, key_gen) and answers
`409 {"error":"key_exhausted"}` at 2^30. That forces a rotation long before the
2^32 random-nonce bound. In practice the cap is never reached.

### B6.4 Associated data

```
AAD = "lumi/ls1/aad" || 0x00
   || vault_id                           (16)
   || LP(note_id)                        ("" for book-meta)
   || LP(crdt_epoch)                     ("" for book-meta and note-meta)
   || extra(kind)
   || header                             (36)

extra(update)     = ""                    (seq is assigned by the server after encryption; msg_id is in the header)
extra(awareness)  = ""
extra(snapshot)   = u64be(base_seq_home) || SHA-256(canonical(coverage))   (§B7.6)
extra(note-meta)  = u64be(meta_version)
extra(book-meta)  = u64be(book_meta_version)
```

The binding prevents:

- moving an envelope to another book, note, lineage or kind (cut-and-paste by
  the server);
- downgrading `key_gen` or `version` (both are in the header, which is in the
  AAD);
- replaying an old meta as a newer version.

Replaying an *update* is harmless, because Yjs is idempotent (V13). The server
also deduplicates by `msg_id`.

### B6.5 HPKE wraps

All wraps use `SetupBaseS(pkR, info)` and `Seal(aad = "", pt)`. Stored form:
`enc (32) || ct`.

| Wrap | Recipient key | `info` | Plaintext |
|---|---|---|---|
| book key | member `uk_enc` | `"lumi/ls1/wrap/book" || 0x00 || vault_id || u32be(g) || LP(member_key) || uk_enc_pub` | `BK[g]` (32) |
| user key → device | device `dk_enc` | `"lumi/ls1/wrap/device" || 0x00 || LP(user_id) || uk_id || device_id(16) || dk_enc_pub` | `uk_enc_sk (32) || uk_sig_seed (32)` |

`member_key` = `username@server-base-url` (the existing federation key
format). It is used even for local members, so that wraps survive federation.

Every wrap that one party makes for another is **signed** by the sender:

```
wrap_sig = Ed25519(uk_sig_sender, "lumi/ls1/sig/wrap" || 0x00 || SHA-256(info) || enc || ct)
```

A recipient accepts a book-key wrap only if `wrap_sig` verifies under the
`uk_sig` of a **current member** of the book, according to the member key list
(§B9.5 pinning applies).

### B6.6 Previous-generation chain

```
prev_wrap[g] = nonce(12) || AES-256-GCM(K_prev[g], nonce, BK[g−1],
                aad = "lumi/ls1/prev" || 0x00 || vault_id || u32be(g))           for g ≥ 2
gen_sig[g]   = Ed25519(uk_sig_rotator, "lumi/ls1/sig/gen" || 0x00 || vault_id || u32be(g)
                || SHA-256(prev_wrap[g]) || u64be(created_at_unix) || u8(reason))
```

`reason`: `1` created, `2` member removed, `3` device loss, `4` manual,
`5` key exhausted.

### B6.7 Identity signatures and fingerprints

```
uk_self_sig = Ed25519(uk_sig, "lumi/ls1/sig/uk" || 0x00 || LP(user_id) || LP(member_key)
                       || uk_enc_pub || uk_sig_pub || u64be(created_at_unix))
device_cert = Ed25519(uk_sig, "lumi/ls1/sig/device" || 0x00 || LP(user_id) || uk_id || device_id
                       || dk_enc_pub || dk_sig_pub || u64be(created_at_unix))
snap_sig    = Ed25519(dk_sig, "lumi/ls1/sig/env" || 0x00 || SHA-256(AAD || ciphertext))   (snapshot, note-meta, book-meta)
```

**Fingerprint (safety number):**
`fp = SHA-256("lumi/ls1/fp" || 0x00 || LP(member_key) || uk_sig_pub || uk_enc_pub)`.
It is displayed as 6 groups of 5 digits: each group is a 5-byte big-endian
chunk mod 100000, taken from bytes 0..29. For a pair of users, both 30-digit
strings are shown, sorted (Signal style). A QR code encodes
`lumi-fp:1:<base32(fp)>`.

### B6.8 Book write keys (deferred, decision D8)

Optionally, each generation also has an Ed25519 **book write key**. Its private
part is wrapped only to members whose role includes `note.edit`. Every
envelope then carries `Ed25519(bwk, SHA-256(AAD || ct))`. Readers verify it,
so a Viewer cannot forge edits even with a compromised server. The cost is
64 bytes per envelope plus a second wrap set, and role changes trigger
rotations. Recommendation: a later phase (S7).

## B7. Live collaboration over encrypted relay

The server cannot merge Yjs, so for sealed books it is an **ordered,
deduplicating envelope log with fan-out**. The Y.Doc exists only on clients.
CRDT rules 1–4 still hold for clients. Rule 5 (opaque editors, 3-way) moves
entirely to the client (§B13.3).

### B7.1 Server state per note

- `sealed_note_log(vault_id, note_id, seq, msg_id, key_gen, envelope, origin, origin_seq, author_user_id, author_device_id, created_at)`.
  `seq` is a per-note, gap-free, server-local sequence. `origin`/`origin_seq`
  record where the envelope was first accepted, for federation (§B11).
- `sealed_note_snapshots(vault_id, note_id, base_seq, coverage, key_gen, envelope, device_id, sig, created_at)`.
  The last `LUMI_SEALED_SNAPSHOTS_KEEP` (default 3) are kept.
- `notes` row: `crdt_epoch`, `meta_version`, `meta_envelope`, `meta_sig`,
  `meta_device_id`, `change_seq` (the feed works unchanged).

### B7.2 WebSocket endpoint and frames

`POST /api/vaults/:vault/notes/:id/ws-ticket` (existing) returns
`{ticket, expires_at, crdt_epoch, sealed: true, key_gen}` for sealed notes. It
requires **`note.read`**. Readers may join live. A push needs `note.edit`.

`GET /api/vaults/:vault/notes/:id/sealed-sync?ticket=…` is upgraded to a
WebSocket. Binary frames only. The 1 MiB frame cap applies, except Snapshot
frames, which are capped at 8 MiB.

All frames are `varUint(type) body`, with lib0 varints and `varBytes`:

| Type | Name | Direction | Body |
|---|---|---|---|
| `0x20` | Hello | S→C | `varUint(head_seq) varUint(snapshot_base_seq) varUint(current_key_gen) varString(crdt_epoch) varUint(conn_id)` |
| `0x21` | Pull | C→S | `varUint(since_seq)` |
| `0x22` | Snapshot | S→C | `varUint(base_seq) varBytes(coverageJSON) varBytes(envelope) varBytes(device_id) varBytes(sig)` |
| `0x23` | Updates | S→C | `varUint(n)` then n× `varUint(seq) varBytes(envelope) varBytes(author_device_id)` |
| `0x24` | Push | C→S | `varBytes(envelope)` |
| `0x25` | Ack | S→C | `varBytes(msg_id) varUint(seq)` |
| `0x26` | Awareness | C→S / S→C | C→S: `varBytes(envelope)`; S→C: `varUint(conn_id) varBytes(envelope)` |
| `0x27` | PeerLeft | S→C | `varUint(conn_id)` |
| `0x28` | CompactionWanted | S→C | `varUint(head_seq) varUint(lease_ms)` |
| `0x29` | KeyGen | S→C | `varUint(current_key_gen)` |
| `0x2A` | Caught | S→C | `varUint(head_seq)` (end of the catch-up batch) |
| `0x2F` | Error | S→C | `varString(code)` (followed by close) |

Close codes:

- Unchanged: `4401` (auth / `ticket_stale`), `4403` (capability, including a
  reader pushing), `4404` (note deleted), `1008` (socket cap), and the
  client-side `4409` (lineage).
- New, `4412 key_gen_stale`: a push under `key_gen < current`. The client
  refreshes the keyring, re-encrypts pending updates and reconnects.
- New, `4423 sealing`: the book is being converted (§B15).

### B7.3 Join

1. The client loads its persisted replica, if any (the doc plus `last_seq`,
   `ackedSV` and `epoch`), or an empty doc. A plaintext REST preview does not
   exist for sealed books. The UI shows "Decrypting…" until the first `Caught`.
2. It mints a ticket and checks the lineage with the ticket's `crdt_epoch`
   (CRDT §12.5, unchanged). On a mismatch it saves a conflict copy, drops the
   replica and starts empty.
3. It connects. The server sends `Hello`.
4. The client sends `Pull(since = last_seq)`. The server answers:
   - if `since < snapshot_base_seq` (the client is behind the compaction
     horizon), or the client has no replica: the `Snapshot`, then `Updates`
     with `seq > snapshot_base_seq`;
   - otherwise: `Updates` with `seq > since`.
   Batches are at most 256 envelopes or 1 MiB each, then `Caught(head)`.
5. The client decrypts and applies each envelope in seq order, with
   `origin = provider`. It sets `last_seq` and merges each update's SV into
   `ackedSV` (the server has it).
6. On `Caught`, the doc enters LIVE. The editor mounts with `ytext.toString()`
   (CRDT §2.2 step 4).
7. If the replica holds local changes not covered by `ackedSV`, the client
   pushes `encodeStateAsUpdate(doc, ackedSV)` as one update. This is the doc
   acting as the queue, which is CRDT §5 adapted.

### B7.4 Steady state

- Local transaction → Yjs update `u` → pad → encrypt under the current
  generation → `Push`. The client keeps `pending[msg_id] = u` until the `Ack`
  arrives. On `Ack` it merges `u`'s SV into `ackedSV` and records `last_seq`
  if it is contiguous.
- Server on `Push`:
  1. check `note.edit`;
  2. validate the header (key_gen = current, or 4412; a known kind; size);
  3. insert into the log with the next `seq`, idempotent on
     `(vault, note, msg_id)` (a duplicate returns the original `seq`);
  4. fan out `Updates{seq, envelope}` to the other subscribers (never echoed);
  5. `Ack` to the sender;
  6. bump the content coalescer (feed `vault.head`, SPEC-LIVROS §3.4);
  7. call the federation persist hook (§B11).
  Per socket, frames are processed in order and persisted before the next one
  is read. This is the same guarantee CRDT §12.6 relies on.
- Remote `Updates` → decrypt → apply. A seq gap, meaning the received `seq`
  is not `last_seq + 1`, triggers `Pull(last_seq)`.
- Decryption failure (bad tag, unknown key_gen, wrong AAD): the client drops
  the envelope, counts it, and shows "Uma alteração não pôde ser verificada"
  once per session. It then fetches the keyring, in case of a new generation.
  It MUST NOT apply unauthenticated bytes.

### B7.5 Offline and acknowledgement

The rules are the same as CRDT §12.6, with one simplification. The server acks
every push with a seq, so a replica is "acknowledged up to" the SV made of the
acked updates. No probe is needed. Persisted replicas store `{doc_state,
last_seq, ackedSV, crdt_epoch, key_gen_seen}`, **encrypted at rest** (§B13).

### B7.6 Compaction by clients

The log grows without limit unless a client folds it into a snapshot.

1. When the log since the last snapshot exceeds 200 rows or 1 MiB, the server
   picks **one** connected, caught-up subscriber with `note.edit` and sends it
   `CompactionWanted(head, lease_ms = 60000)`. The lease means it asks only
   one client per note at a time.
2. The client builds a **fresh temporary Y.Doc** from what it downloaded: the
   previous snapshot plus the decrypted log up to `N = head`. It never uses
   the live doc, which may hold unacked local edits that other replicas have
   not seen. It then takes `S = encodeStateAsUpdate(tmp)` and checks that
   `tmp.getText('content').toString()` equals the text the live doc had at
   `N`. If the live doc has pending edits, it skips the check.
3. The plaintext of the snapshot is
   `varBytes(S) || varBytes(encodeStateVector(tmp))`. The client encrypts it
   (kind `0x02`, current key_gen, `extra` with `base_seq = N` and the coverage
   digest), signs it with `snap_sig`, and sends
   `PUT …/sealed/notes/:id/snapshot {base_seq, coverage, envelope, device_id, sig}`.
4. The server accepts it iff:
   - `base_seq > current snapshot base_seq` and `base_seq ≤ head`;
   - the coverage equals what the server computes from its log up to
     `base_seq` (§B11.2);
   - the key_gen is current;
   - the device belongs to a member with `note.edit`;
   - the signature verifies against the registered `dk_sig`.
   Otherwise `409`.
5. Log rows with `seq ≤ base_seq` become **eligible for deletion** after
   `LUMI_SEALED_HISTORY_DAYS` (default 30). Until then they stay as the
   sabotage-recovery history. Readers behind the horizon get the snapshot.
6. **Verification by other clients.** A client that receives a snapshot while
   it holds a replica at or beyond `base_seq` checks that the snapshot's
   embedded SV covers every SV it applied from envelopes with `seq ≤ base_seq`.
   On failure (the snapshot drops operations), it reports
   `POST …/sealed/notes/:id/snapshot-dispute {base_seq, device_id}`. The
   server audits it, marks the snapshot disputed and stops serving it. Clients
   then fall back to the previous snapshot plus the log, which is still
   retained. Admins see disputes in the book's audit.

Compaction re-encrypts old-generation content under the current generation.
Over time this retires old generations.

### B7.7 Awareness

The awareness update (y-protocols encoding) is encrypted with `K_aware[g,n]`
(kind `0x03`). It is relayed to the note's other subscribers with the
sender's `conn_id` and **never persisted**. The server cannot build an
awareness "left" frame (CRDT §10 S7), so it sends `PeerLeft(conn_id)`. Each
client keeps a `conn_id → {awareness clientIDs}` map from the frames it saw,
and removes those states locally. Display names inside awareness are encrypted,
so the server learns nothing new.

### B7.8 Rollback and freshness checks (client)

- Per note: persist `last_seq`. A `Hello.head_seq < last_seq` (the server
  forgot or rolled back) shows "O servidor voltou no tempo para esta nota" and
  does not drop local state. The client pushes its state as in §B7.3 step 7.
- Per book: persist the highest `key_gen` seen. A keyring response with a
  lower current generation is rejected.
- A snapshot whose `base_seq` is lower than one already accepted for the note
  is ignored.

### B7.9 Limits

| Item | Limit |
|---|---|
| Update / awareness envelope | 1 MiB |
| Snapshot envelope | 8 MiB (a 4 MiB body plus history, padded) |
| Meta envelope | 64 KiB |
| Pushes per socket | 50/s burst 200 (then `1008`) |
| Notes per sealed book | the same as plain |

## B8. REST API (sealed)

All routes are under the authed `/api` group. They return 404 on non-sealed
vaults, and plaintext note routes return
`409 {"error":"sealed"}` on sealed vaults (§B10).

**User and device keys**

| Route | Cap | Purpose |
|---|---|---|
| `GET /api/keys/users/:username` | authed | `{user_id, member_key, uk_id, uk_enc_pub, uk_sig_pub, self_sig, created_at, devices:[{device_id, dk_enc_pub, dk_sig_pub, cert, name, created_at, revoked_at}]}` |
| `PUT /api/users/me/keys` | authed + fresh password (≤ 5 min) | Create the UK (first time) or **reset** it (§B9.9). Body: the public parts, `self_sig`, the first device and its cert and wrap, and the recovery blob. |
| `GET /api/users/me/key-unlocks` | authed | this device's wrap (by `X-Lumi-Device`), plus the recovery blob and passkey blobs (all ciphertext) |
| `POST /api/users/me/devices` | authed | enrol a device directly (from an existing device: cert + wrap) |
| `POST /api/users/me/device-requests` | authed | a new device asks: `{device_id, dk_enc_pub, dk_sig_pub, name}`. The feed sends `device.request` to the user's other sessions. |
| `GET /api/users/me/device-requests` / `POST …/:id/approve {cert, wrap}` / `DELETE …/:id` | authed | approval from an existing device after comparing the SAS (§B9.3) |
| `DELETE /api/users/me/devices/:id` | authed + fresh password | revoke a device: its wrap and sessions are deleted (§B9.7) |
| `PUT /api/users/me/key-unlocks/passkey` | authed | add or replace a PRF-wrapped UK blob `{credential_id, blob}` |

**Books**

| Route | Cap | Purpose |
|---|---|---|
| `POST /api/vaults {sealed:true, book_meta_envelope, key_gen:1, gen_sig, wraps:[self]}` | authed | create a sealed book (random slug) |
| `GET /api/vaults/:vault/sealed/keyring` | `note.read` | `{current_key_gen, generations:[{key_gen, prev_wrap, gen_sig, created_by_uk, reason, created_at}], my_wraps:[{key_gen, wrap, sender_uk_id, wrap_sig}], rotation_due}` |
| `GET /api/vaults/:vault/sealed/member-keys` | `note.read` | member list with UK publics, `key_status ∈ {ok, pending, stale}` |
| `POST /api/vaults/:vault/sealed/keys/wraps` | `members.invite` or `members.manage` | add wraps of the current generation for pending members |
| `POST /api/vaults/:vault/sealed/keys/rotate` | `members.manage` | `{expected_key_gen, key_gen, prev_wrap, gen_sig, reason, wraps:[…], remove_members?:[member_key]}`. Compare-and-swap on the generation. The wrap set MUST equal the current members minus removed ones, else `409 {"error":"wrap_set_mismatch","expected":[…]}`. Member removal and rotation commit in **one transaction**. |
| `PUT /api/vaults/:vault/sealed/book-meta` | `vault.manage` | CAS on `book_meta_version` |

**Notes**

| Route | Cap | Purpose |
|---|---|---|
| `GET /api/vaults/:vault/sealed/notes?after=N&limit=` | `note.read` | the `changes` equivalent: `{notes:[{id, crdt_epoch, meta_version, meta_envelope, meta_sig, meta_device_id, change_seq, head_seq}], deleted:[{id,seq}], head, more}`. The same 410 `resync_required` rule as SPEC-LIVROS §3.5. |
| `POST /api/vaults/:vault/sealed/notes` | `note.create` | `{id, crdt_epoch, meta_envelope, meta_sig, device_id, snapshot?}`. The id is client-generated. 409 if taken. |
| `PUT /api/vaults/:vault/sealed/notes/:id/meta` | `note.move` (or `note.edit` for tags) | `{expected_meta_version, meta_envelope, meta_sig, device_id}`. CAS. |
| `DELETE /api/vaults/:vault/sealed/notes/:id` | `note.delete` | as plain: room dropped (`4404`), tombstone |
| `GET /api/vaults/:vault/sealed/notes/:id/log?since=&limit=` | `note.read` | REST pull: `{snapshot?, updates:[{seq, envelope, author_device_id}], head, crdt_epoch}` |
| `POST /api/vaults/:vault/sealed/notes/:id/log` | `note.edit` | REST push `{envelope}` → `{seq}`; the same validation as WS |
| `PUT /api/vaults/:vault/sealed/notes/:id/snapshot` | `note.edit` | §B7.6 |
| `POST /api/vaults/:vault/sealed/notes/:id/snapshot-dispute` | `note.read` | §B7.6 |

**Invites** (existing route, extended): `POST /api/vaults/:vault/invites`
accepts `{…, sealed_wrap: {key_gen, nonce_ct, inviter_uk_id, wrap_sig}}`
(§B9.4).

**Feed** (SPEC-LIVROS §3.3), additive:

- `hello`/`vault.access` items gain `sealed: true`, `key_gen`, and
  `key_status ∈ {ok, pending}` (pending = a member without a wrap yet).
- New event `vault.keys {vault_id, key_gen, rotation_due}`.
- New event `device.request {request_id}` (user-scoped).
- As before: no titles, paths or note ids.

## B9. Key lifecycle flows

### B9.1 First sealed-capable sign-in (UK creation)

1. The client generates DK (non-extractable where possible) and UK.
2. It generates RK and shows it as `LR1-XXXXXX-…` (§B9.8). The user must
   confirm by re-typing 2 random groups, or by "Salvei o arquivo" for a
   downloaded `.txt`.
3. The client computes the `self_sig`, the device `cert`, the device wrap and
   the recovery blob. It then calls `PUT /api/users/me/keys`. The server
   stores the public parts and ciphertexts, then audits `user.keys.create`.

UK creation is lazy. It happens the first time the user creates or joins a
sealed book, so users who never use sealed books are unaffected.

### B9.2 Unlocking on an enrolled device

The client calls `GET /api/users/me/key-unlocks` with `X-Lumi-Device`, then
does an HPKE open with DK. That yields the UK secret, which it imports as
non-extractable keys (§B13). For each sealed book it opens, it fetches the
keyring, opens `my_wraps[current]` with `uk_enc`, verifies `wrap_sig`, and
walks `prev_wrap` down to the oldest generation it needs (lazily). Then it
imports the BKs as HKDF base keys.

### B9.3 Adding a device

- **(a) Approve from another device (default).** The new device posts
  `device-requests`. The old device gets `device.request` and shows the
  device name and a **6-digit SAS**. Both devices show
  `SAS = u32be(SHA-256("lumi/ls1/sas" || 0x00 || dk_sig_pub_new || dk_enc_pub_new || request_id)[0..3]) mod 10^6`.
  If they match, the old device computes the cert and the wrap and approves.
  The server stores them and the new device unlocks. The SAS prevents the
  server from substituting the new device's key.
- **(b) Recovery key.** The user types RK. The client opens the recovery blob,
  self-enrols, and the server audits `user.device.add{via:"recovery"}`.
- **(c) Passkey PRF** (web, optional). §B13.1.

### B9.4 Inviting a new member (link key, default)

1. The inviter's client generates a link secret `s` (32 bytes) and computes:
   ```
   K_inv = HKDF-SHA256(IKM = s, salt = vault_id, info = "lumi/ls1/k/invite" || 0x00 || LP(invite_token_id))
   sealed_wrap = nonce || AES-256-GCM(K_inv, nonce, BK[g], aad = "lumi/ls1/invite" || 0x00 || vault_id || u32be(g))
   ```
   It signs `sealed_wrap` with `wrap_sig`, then posts the invite with
   `sealed_wrap`.
2. The link is `https://server/invite/<token>#lk=<base64url(s)>`. The
   **fragment never reaches the server.** Clients MUST strip it from history
   (`history.replaceState`) and never log it.
3. The invitee signs up or logs in, accepts (as today), then creates their UK
   if needed. The client decrypts `BK[g]` with `s`, verifies the inviter's
   `wrap_sig` against the member keys, and **re-wraps BK to its own `uk_enc`**
   (`POST …/sealed/keys/wraps`, a self-wrap). It then discards the invite
   wrap. The server deletes `sealed_wrap` on accept, and on expiry or
   revocation.
4. If the invite is accepted without the fragment (the link was retyped
   without it), membership is created with `key_status = pending` and B9.5
   applies.

Effect: even a server that substitutes public keys cannot obtain BK through
this flow. The link is as sensitive as the book for its lifetime (single use,
short TTL, default 48 h for sealed books), and the UI says so.

### B9.5 Adding an existing user without a link key (pending members)

The admin adds `username` (or a federated `user@server`), and the server
creates the membership with `key_status = pending`. Any member with
`members.invite`/`members.manage` whose client is open sees "N pessoas
aguardando a chave". The client fetches the pending member's UK, shows its
**fingerprint**, then wraps and posts. Verification is optional but
prominent. It can be done by comparing the safety number in person, or by
scanning a QR code.

Pinning: clients pin `(member_key → uk_id)` per book on first use (TOFU). If a
member's UK changes, every client warns ("A chave de <user> mudou") and
**refuses to wrap new generations** for that member until someone confirms.

### B9.6 Removing a member (rotation)

The remover's client:

1. generates `BK[g+1]`;
2. computes `prev_wrap[g+1]`, `gen_sig`, and wraps for every remaining member
   (current UK publics, pinned);
3. calls `POST …/sealed/keys/rotate {remove_members:[x], …}`. The server
   checks the CAS and the wrap set, then commits removal and rotation
   atomically. Sockets of the removed user close `4403`, and other sockets get
   `KeyGen`;
4. other clients fetch the keyring. Pending pushes under `g` get `4412` and
   are re-encrypted under `g+1`.

If the removal happens **without a keyed client**, the server only removes the
member and sets `rotation_due = true`:

- admin panel or CLI removal;
- LGPD user erasure;
- federation control-state changes;
- an admin who removes from the web without unlocked keys.

The first client of a member with `members.manage` that opens the book rotates
automatically (reason `2`). Writes under `g` continue meanwhile (decision D10).
An honest server already denies the removed user all access, so the exposure
exists only if the server is compromised *and* colludes with the removed
member during the gap.

Old content stays under old generations until compaction re-encrypts it. The
removed member could already read it, so there is no loss.

### B9.7 Device revocation and device loss

`DELETE /api/users/me/devices/:id` (from another device, with a fresh
password): the server deletes that device's UK wrap, its sessions and its
pending requests, and audits it. The lost device still holds the UK secret and
cached BKs, protected by its OS. The user may choose **"Girar tudo"**:

1. reset the UK (§B9.9) with new publics, then re-wrap to the remaining
   devices and the RK;
2. self-wrap the current BK of every book under the new UK;
3. for each book where the user has `members.manage`, rotate (reason `3`);
4. elsewhere, post a rotation request (`rotation_due` + `vault.keys` to
   admins).

### B9.8 Recovery key

- 32 random bytes `rk`.
- Display: `"LR1-" + crockford_base32(rk || SHA-256("lumi/ls1/rk" || rk)[0..1])`,
  in groups of 6. That is 34 bytes, so 55 characters, plus the hyphens.
  Decoding is case-insensitive, `O→0` and `I/L→1`, and the checksum catches
  typos.
- Blob:
  ```
  K_rec = HKDF-SHA256(IKM = rk, salt = user_id(16), info = "lumi/ls1/k/recovery" || 0x00 || uk_id)
  recovery_blob = nonce || AES-256-GCM(K_rec, nonce, uk_enc_sk || uk_sig_seed, aad = "lumi/ls1/recovery" || 0x00 || user_id || uk_id)
  ```
- The server stores the blob. It cannot attack it: 256-bit entropy, no
  password involved.
- "Gerar nova chave de recuperação" (from an unlocked device) replaces the
  blob and audits `user.recovery.rotate`. The old RK stops working because its
  blob is gone. The server is trusted to delete it, which is the honest
  limitation of any stored recovery blob.

### B9.9 Key reset (everything lost)

If the user has lost every device and the RK, they can, with a fresh password,
run `PUT /api/users/me/keys` with `reset: true`. That gives them a new UK and
new devices. The server then:

- marks the user's memberships `key_status = pending` in every sealed book;
- audits `user.keys.reset`;
- notifies members through `vault.keys`.

Other members' clients warn about the key change (§B9.5) and re-wrap after
confirmation. **Books where the user was the only member are unrecoverable.**
The UI says this when the book is created and again at reset time.

## B10. Server features lost or changed for sealed books

| Feature | Plain book | Sealed book |
|---|---|---|
| Disk mirror `<vault>/<path>.md` | yes | **disabled.** The vault dir holds only `.lumi/vault.yaml` (`sealed: true`, no name). Ciphertext lives in Postgres only, so backups are pg dumps. |
| fswatch FS → CRDT | yes | **disabled** (the dir is not watched) |
| Server Y.Doc, lazy init, single seeder | yes | **none.** Clients seed at create (the creator's first snapshot or update). |
| `GET …/content`, `/snapshot`, `/state-vector`, `POST /diff`, `PATCH …/notes/:id` body/title/path | yes | `409 {"error":"sealed"}`. Use the `sealed/*` routes. |
| Title/path search on the server | yes | **none.** Clients index locally (web: an in-memory index, persisted encrypted; Apple: an in-memory or local index). |
| Hover preview, backlinks, kanban, canvas | client-side | client-side, over decrypted data (unchanged UX once unlocked) |
| `POST /vaults/:vault/copies` (server-side copy) | yes | `409 sealed`. Use client-side "Copiar livro…" (decrypt, then create a new sealed or plain book; explicit). |
| `GET /users/me/export` (LGPD) | plaintext zip | sealed books exported as **ciphertext** (envelopes + wraps + recovery blob) plus a `README` explaining this. Clients offer "Exportar decifrado" locally. |
| Feed, `seq`, `changes`, tombstones | yes | yes (ids only, as before) |
| Roles, members, invites, audit | yes | yes, plus key-status and key events |
| Rename the book | `PATCH name` | client book-meta envelope |
| Admin panel vault view | name, counts, bytes | "🔒 Livro selado", counts, bytes, key_gen, rotation_due. No names. |
| Admin `vault delete` | yes | yes (crypto-shredding of server state) |
| Federation replica | plaintext CRDT relay | ciphertext log relay (§B11) |
| Mirror, publish | yes | mirror yes (ciphertext backup), **publish no** in v1 |
| TUI | full | locked in v1 (D2) |

## B11. Federation and mirrors of sealed books

### B11.1 What crosses the link

New relay frames, used only for sealed vaults. `sealed` is part of the
control state, and both sides refuse to mix modes. Peers without support
reject the invite: `accept` returns `409 sealed_unsupported`.

```
frameSealedManifest  = 12  // n × { varBytes(note_id) varBytes(epoch) varUint(meta_version) varBytes(meta_envelope)
                           //       varBytes(meta_sig) varBytes(meta_device) varUint(snapshot_coverage_digest_present) … }
frameSealedEnvelope  = 13  // varBytes(note_id) varBytes(origin) varUint(origin_seq) varBytes(envelope) varBytes(author_device_id)
frameSealedSnapshot  = 14  // varBytes(note_id) varBytes(coverageJSON) varBytes(envelope) varBytes(device_id) varBytes(sig)
frameSealedMeta      = 15  // note meta change (create/move) — same fields as a manifest entry
frameSealedKeyring   = 16  // full keyring JSON: generations + all wraps (ciphertext) + book_meta envelope
frameSealedUserKeys  = 17  // follower → home: UK publics + device certs of its local members (member_key scoped)
frameSealedHave      = 18  // per note: coverage vector — "I have everything up to these origin_seqs"
```

- **Key material:** the keyring and wraps are ciphertext and replicate in
  full, so a follower can serve its local members. The **control plane stays
  home-authoritative.** Home is the only place rotations and wraps are
  accepted. A follower forwards a member's `POST …/keys/wraps|rotate` to home
  over the link and relays the result. (On a mirror these routes are clamped
  anyway.)
- **Member keys across servers:** a federated member publishes their UK on
  their own server. The follower forwards it with `frameSealedUserKeys`
  (signed by the user's `uk_sig`; the follower cannot forge it). Home stores
  it in `sealed_user_keys` by `member_key`. Pinning is as in §B9.5.

### B11.2 Ordering, dedupe and coverage

- Each server appends relayed envelopes to its own log with its own `seq`.
  `(origin, origin_seq)` is kept. `msg_id` is unique per note, so duplicates
  arriving from two paths are dropped.
- **Coverage** of a snapshot is a map `{origin_url: origin_seq}`. A server
  computes it from its log as, per origin, the highest `origin_seq` among rows
  with local `seq ≤ base_seq`. The links are FIFO per origin, so this is a
  contiguous prefix.
- Reconnect: both sides exchange `frameSealedHave` per note. Each sends the
  envelopes the other lacks, in `(origin, origin_seq)` order. This is the
  state-vector exchange of the plaintext relay, done on server-visible
  counters.
- **Snapshots are accepted only at home.** A follower's client that gets
  `CompactionWanted` is never chosen: followers do not send it. Home relays
  accepted snapshots down. Each follower maps the coverage to its own `seq`
  and truncates after the history window.
- **Mirror of a sealed book:** the same frames, one-way (Part A rules). This is
  the **recommended backup design for sealed books**: an untrusted off-site
  operator holding only ciphertext. Restoring after promotion works, because
  members keep their keys.

## B12. Offline replicas

The CRDT §12 rules hold unchanged: an epoch check before any merge, conflict
copies, and replicas deleted only after an ack. Differences:

- The ack is the server's `Ack{msg_id, seq}` (§B7.5). No state-vector probe is
  needed.
- **Persisted replicas MUST be encrypted at rest**:
  - Web: IndexedDB store `sealed-replicas`, AES-GCM under a per-device
    *storage key*, a non-extractable `CryptoKey` in IndexedDB.
  - Apple: files under Application Support with
    `NSFileProtectionCompleteUntilFirstUserAuthentication`, encrypted with a
    storage key held in the Keychain.
  - TUI: §B13.3.
  - Plaintext on a user's own disk is allowed only where the user explicitly
    exports, or keeps a local folder copy (TUI/Apple folder books, a deliberate
    opt-in labelled "cópia local em texto puro").
- A replica records `key_gen_seen`. After a rotation, unsent local changes are
  re-encrypted under the new generation at push time. Plaintext Yjs updates
  never touch the server.

## B13. Clients

### B13.1 Web

- **Baseline:**
  - WebCrypto X25519, Ed25519, HKDF, AES-GCM (current Chrome, Edge, Firefox
    and Safari). On a browser without X25519/Ed25519 in WebCrypto, sealed
    books show "Navegador sem suporte a livros selados". There is no JS
    fallback, which would mean extractable keys (decision D11 may relax this).
  - HPKE via `@hpke/core` (WebCrypto-backed; no WASM, so the CSP stays
    `script-src 'self'`). Pinned version, vendored into the bundle.
- **Key storage** (IndexedDB `lumi`, store `sealed-keys`):
  - `device`: `{device_id, dk_enc: CryptoKey(X25519, non-extractable), dk_sig: CryptoKey(Ed25519, non-extractable), publics}`.
  - `uk`: after unwrap, the 64 bytes are imported as non-extractable
    X25519/Ed25519 `CryptoKey`s, and the byte buffer is zeroed (best effort).
  - `bk/<vault>/<g>`: a non-extractable HKDF base `CryptoKey`, cached only if
    "Lembrar neste aparelho" is on. Otherwise it is held in memory per tab.
  - `storage-key`: AES-GCM, non-extractable, used for encrypted replicas and
    the search index.
  - Raw BK bytes exist transiently: on unwrap, and when the client wraps BK
    for others (invites, rotations, pending members). For a rewrap the client
    re-opens the wrap with `uk_enc`. It never exports a `CryptoKey`.
- **Without "Lembrar neste aparelho"**, the device keys stay in memory only,
  and the tab is a **session device**:
  - it is not enrolled;
  - it unlocks the UK from the RK or a passkey each session;
  - nothing persists.
- **Passkey PRF (optional, recommended on web).**
  - At setup: a WebAuthn credential with `extensions.prf.eval.first = SHA-256("lumi/ls1/prf" || user_id)`.
  - Then `K_prf = HKDF(prf_output, salt = user_id, info = "lumi/ls1/k/prf" || 0x00 || credential_id)`,
    which wraps the UK bundle like the recovery blob.
  - Synced passkeys (iCloud Keychain, Google Password Manager) make this a
    cross-device unlock. It is not counted as *the* recovery mechanism (it
    depends on the platform account).
- **Honest threat note** (shown in the help): "No navegador, a cifra protege
  contra o servidor, backups e outros servidores. Ela não protege contra quem
  controla o código da página. Para o máximo de proteção, use o app ou o web
  publicado por lumi (`app.lumi…`) apontando para o seu servidor." Mitigations:
  - Publish a SHA-256 manifest of each web release, and add an `About` panel
    that shows the running bundle's hash.
  - Support the web client hosted on a separate static origin. SPEC-LIVROS
    multi-server already needs CORS per server.
  - Strict CSP (existing), no runtime third-party code.
- **Editor session:** a `SealedProvider` replaces `y-websocket` for sealed
  books. It has the same surface (`on('sync')`, `awareness`) and implements
  §B7. `y-codemirror.next` is unchanged.

### B13.2 Apple (macOS, iOS)

- Device keys: CryptoKit `Curve25519.KeyAgreement` and `Curve25519.Signing`
  private keys, serialized and stored in the Keychain
  (`kSecAttrAccessibleWhenUnlockedThisDeviceOnly`, `service = com.vinizap.lumi.sealed`).
  On devices with a **Secure Enclave**, the Keychain item is additionally
  encrypted under a key agreed with an SE-resident
  `SecureEnclave.P256.KeyAgreement` key. The SE supports only P-256, so the
  X25519/Ed25519 keys themselves cannot live there. Optional access control:
  `.userPresence`, for "Exigir Face ID/Touch ID para abrir livros selados".
- UK and BKs: in memory while unlocked. BK cache in the Keychain (same
  protection), per "Lembrar".
- HPKE: CryptoKit `HPKE.Sender/Recipient` with
  `.Curve25519_SHA256_AES_GCM_256`. AES-GCM: CryptoKit. HKDF: CryptoKit.
- yswift drives the doc. A `SealedSyncClient` actor implements §B7 over
  `URLSessionWebSocketTask`. It applies the epoch, deferred-Step1 and ack
  rules, adapted (no probe needed).
- Folder-backed Apple books: a sealed book MAY have a local plaintext folder
  only by explicit opt-in.

### B13.3 TUI

REST 3-way `/diff` (CRDT §7.2) cannot work: the server cannot see text.
Options (decision D2):

- **(a) v1: locked (recommended for v1).**
  - `lumi vaults` and the picker list sealed books with `🔒` and the hint
    "abra no web ou no app (suporte no TUI previsto)".
  - `lumi vault clone` refuses with exit code 3.
  - No keys are stored by the TUI.
- **(b) Read-only through a text rendition.** Clients would upload an
  encrypted plaintext rendition with each snapshot. It is cheap, but stale
  (snapshot-only), and every snapshot doubles its server-visible size.
  Not recommended.
- **(c) Full (phase S6).**
  - yrs compiled to `wasm32-wasip1` (a tiny C-ABI shim over yffi) and run with
    **wazero**, which is pure Go. The TUI stays `CGO_ENABLED=0` and
    cross-compiles as today.
  - Keys through `go-keyring` (macOS Keychain, Secret Service, Windows
    Credential Manager). It falls back to a 0600 file encrypted with a key
    from RK entry. The TUI enrols as a device by SAS approval or RK.
  - Sync loop per book: pull the log (`GET …/log`) into a local replica
    `.lumi/cache/<vault>/<note>.ydoc`, encrypted with a storage key; render
    the text to the folder file; on save, do the §7.1 3-way locally (base
    replica → diff → ops) and push the encrypted update.
  - Decrypted files in the folder are the user's deliberate local copy (TUI
    books are folders). The book is marked `sealed: true` in `books.yaml` and
    the TUI warns once.

## B14. Audit

New actions. Payloads hold ids and numbers only.

| Action | Payload |
|---|---|
| `user.keys.create` / `user.keys.reset` | `{uk_id}` |
| `user.device.add` / `user.device.revoke` | `{device_id, via: "approve"\|"recovery"\|"passkey"}` |
| `user.recovery.rotate` | `{}` |
| `sealed.vault.create` | `{}` |
| `sealed.keys.rotate` | `{key_gen, reason, members}` (count) |
| `sealed.keys.wrap` | `{key_gen, member_key}` |
| `sealed.member.pending` | `{member_key}` |
| `sealed.snapshot.replace` | `{note_id, base_seq, device_id}` |
| `sealed.snapshot.dispute` | `{note_id, base_seq, device_id}` |
| `sealed.convert.begin` / `.commit` / `.abort` | `{notes}` |

Note edits are not audited per push. The existing rule (only the start of an
edit session) applies.

## B15. Migration: converting a book to sealed

**Only plain → sealed is supported. There is no unseal endpoint.** To go
back, a member uses "Exportar decifrado" or "Copiar livro…" into a new plain
book (new vault id), explicitly.

Flow (an Admin with `vault.manage`, on web or Apple):

1. **Preconditions:**
   - every member has a UK (members without one become `pending`);
   - no active **replica** federation links, unless every peer advertises
     sealed support;
   - the UI shows the members, followers and mirrors that will remain.
2. `POST /api/vaults/:vault/seal/begin` freezes the book:
   - writes are refused with `423 {"error":"sealing"}`;
   - live sockets close `4423`;
   - the feed sends `vault.updated{fields:["sealing"]}`;
   - it returns a `sealing_id`;
   - the freeze lasts at most 30 min, then aborts automatically.
3. The client reads every note through the plaintext API (`/snapshot` with
   `state`, plus meta and frontmatter). It generates `BK[1]`, a **new random
   note id and a new epoch per note**, the meta envelopes, and one snapshot
   per note, encrypting the existing Yjs state (history preserved). It also
   computes the wraps for all keyed members.
4. `POST /api/vaults/:vault/seal/commit {sealing_id, book_meta, gen_sig, wraps, notes:[{old_id, id, crdt_epoch, meta…, snapshot…}]}`.
   The upload is chunked (`/seal/notes` batches of 100, then the final commit).
   The commit, in one transaction:
   - inserts the sealed rows;
   - deletes the plaintext rows (notes and CRDT tables) and their tombstones;
   - inserts tombstones for the old ids;
   - sets `vaults.sealed = true`, a random slug and the name placeholder.
   After the commit it renames the vault directory and deletes the `.md`
   files (best-effort overwrite, then unlink), and audits
   `sealed.convert.commit`.
5. Clients:
   - old note ids are tombstoned, so replicas of them hit the §12.5 "lineage
     changed" path, and pending offline edits become conflict copies the user
     can re-apply;
   - the UI warns before step 2: "Sincronize todos os aparelhos antes de
     selar".

Why new ids: plaintext ids are title slugs (`plano-2027`) and would leak
titles forever (D9).

**What conversion cannot undo:**

- Plaintext already in server backups, in followers' and mirrors' copies and
  their backups, and in published mirrors stays there.
- Replica and mirror links of a plain book must be re-established as sealed,
  or they are revoked at conversion: replica and mirror links are revoked at
  `begin`, with the sever policy applied.

The dialog states all of this.

## B16. Database migrations (sealed)

`0011_user_keys`:

```sql
CREATE TABLE user_keys (
  user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  uk_id       BYTEA NOT NULL CHECK (octet_length(uk_id) = 16),
  enc_pub     BYTEA NOT NULL CHECK (octet_length(enc_pub) = 32),
  sig_pub     BYTEA NOT NULL CHECK (octet_length(sig_pub) = 32),
  self_sig    BYTEA NOT NULL CHECK (octet_length(self_sig) = 64),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reset_count INT NOT NULL DEFAULT 0
);
CREATE TABLE user_devices (
  id          UUID PRIMARY KEY,                     -- client-generated device_id
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name        TEXT NOT NULL CHECK (char_length(name) <= 64),
  enc_pub     BYTEA NOT NULL, sig_pub BYTEA NOT NULL,
  cert        BYTEA NOT NULL,                       -- signed by uk_sig
  uk_wrap     BYTEA NOT NULL,                       -- HPKE(enc || ct) of the UK secret
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ,
  revoked_at  TIMESTAMPTZ
);
CREATE INDEX user_devices_user_idx ON user_devices (user_id);
CREATE TABLE user_key_unlocks (                     -- recovery + passkey blobs
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind        TEXT NOT NULL CHECK (kind IN ('recovery','passkey')),
  ref         TEXT NOT NULL DEFAULT '',             -- credential id for passkey
  blob        BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, kind, ref)
);
CREATE TABLE device_requests (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  device_id   UUID NOT NULL, name TEXT NOT NULL,
  enc_pub     BYTEA NOT NULL, sig_pub BYTEA NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE sealed_user_keys (                     -- federated members' UK publics (home side)
  member_key  TEXT PRIMARY KEY,
  uk_id BYTEA NOT NULL, enc_pub BYTEA NOT NULL, sig_pub BYTEA NOT NULL, self_sig BYTEA NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

`0012_sealed_books`:

```sql
ALTER TABLE vaults
  ADD COLUMN sealed BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN key_gen INT NOT NULL DEFAULT 0,
  ADD COLUMN rotation_due BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN book_meta_version BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN book_meta_envelope BYTEA,
  ADD COLUMN sealing_id UUID, ADD COLUMN sealing_until TIMESTAMPTZ;

ALTER TABLE vault_members
  ADD COLUMN key_status TEXT NOT NULL DEFAULT 'ok' CHECK (key_status IN ('ok','pending'));
ALTER TABLE federated_vault_members
  ADD COLUMN key_status TEXT NOT NULL DEFAULT 'ok' CHECK (key_status IN ('ok','pending'));

CREATE TABLE sealed_key_generations (
  vault_id    UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  key_gen     INT NOT NULL CHECK (key_gen >= 1),
  prev_wrap   BYTEA,                                -- NULL for gen 1
  gen_sig     BYTEA NOT NULL,
  created_by_uk BYTEA NOT NULL, created_by_user UUID REFERENCES users(id) ON DELETE SET NULL,
  reason      SMALLINT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (vault_id, key_gen)
);
CREATE TABLE sealed_key_wraps (
  vault_id    UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  key_gen     INT NOT NULL,
  member_key  TEXT NOT NULL,
  recipient_uk_id BYTEA NOT NULL,
  wrap        BYTEA NOT NULL, sender_uk_id BYTEA NOT NULL, wrap_sig BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (vault_id, key_gen, member_key),
  FOREIGN KEY (vault_id, key_gen) REFERENCES sealed_key_generations (vault_id, key_gen) ON DELETE CASCADE
);
ALTER TABLE invites ADD COLUMN sealed_wrap BYTEA, ADD COLUMN sealed_wrap_gen INT,
                    ADD COLUMN sealed_wrap_sender BYTEA, ADD COLUMN sealed_wrap_sig BYTEA;

ALTER TABLE notes
  ADD COLUMN meta_version BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN meta_envelope BYTEA, ADD COLUMN meta_sig BYTEA, ADD COLUMN meta_device_id UUID;
-- sealed notes: path = the opaque note id and title = '' (placeholders), so the existing
-- PRIMARY KEY (vault_id, id) and UNIQUE (vault_id, path) hold without schema changes.

CREATE TABLE sealed_note_log (
  vault_id   UUID NOT NULL,
  note_id    TEXT NOT NULL,
  seq        BIGINT NOT NULL,
  msg_id     BYTEA NOT NULL CHECK (octet_length(msg_id) = 16),
  key_gen    INT NOT NULL,
  envelope   BYTEA NOT NULL,
  origin     TEXT NOT NULL, origin_seq BIGINT NOT NULL,
  author_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  author_device_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (vault_id, note_id, seq),
  UNIQUE (vault_id, note_id, msg_id),
  FOREIGN KEY (vault_id, note_id) REFERENCES notes (vault_id, id) ON DELETE CASCADE
);
CREATE TABLE sealed_note_heads (                    -- per-note seq allocator (row-locked)
  vault_id UUID NOT NULL, note_id TEXT NOT NULL,
  head_seq BIGINT NOT NULL DEFAULT 0, log_bytes BIGINT NOT NULL DEFAULT 0, rows_since_snapshot INT NOT NULL DEFAULT 0,
  PRIMARY KEY (vault_id, note_id),
  FOREIGN KEY (vault_id, note_id) REFERENCES notes (vault_id, id) ON DELETE CASCADE
);
CREATE TABLE sealed_note_snapshots (
  vault_id   UUID NOT NULL, note_id TEXT NOT NULL,
  base_seq   BIGINT NOT NULL,
  coverage   JSONB NOT NULL,
  key_gen    INT NOT NULL,
  envelope   BYTEA NOT NULL,
  device_id  UUID NOT NULL, sig BYTEA NOT NULL,
  disputed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (vault_id, note_id, base_seq),
  FOREIGN KEY (vault_id, note_id) REFERENCES notes (vault_id, id) ON DELETE CASCADE
);
```

The FKs follow the `notes` key `PRIMARY KEY (vault_id, id)` from `0001_init`. Erasure: `ON DELETE CASCADE` from
users removes keys, devices, unlocks and requests. Authored log rows keep
`author_user_id = NULL`, matching the audit rule.

## B17. Test vectors to define

Put them in the monorepo at `fixtures/sealed/ls1-vectors.json` and copy them
into each submodule (the same pattern as `fixtures/books.v1.yaml`). They are
generated by a Go reference implementation (`server/internal/sealed/ls1`, which
the server also uses to validate headers, signatures and coverage). Each
vector has fixed inputs (keys, nonces, ids) and the exact expected bytes.

| # | Vector |
|---|---|
| T1 | HKDF derivations: `K_note/K_aware/K_meta/K_book/K_prev` for a fixed BK, vault_id and note_id (including a non-ASCII note_id, which the id rules reject: a negative test) |
| T2 | Padding: framed lengths 1, 63, 64, 65, 255, 256, 257, 10 000 and 4 MiB for both rules (64-byte blocks, Padmé min 256) |
| T3 | Envelope encode/decode for each kind, AAD bytes and ciphertext with a fixed nonce |
| T4 | Negative AAD cases: changed vault, note, epoch, kind, key_gen, meta_version, base_seq or coverage digest, each failing authentication |
| T5 | HPKE book-key wrap: the RFC 9180 suite with fixed `skE` (the deterministic test API), the expected `enc`/`ct`, and `wrap_sig` |
| T6 | HPKE device wrap, device cert, UK self-signature |
| T7 | prev-chain: 3 generations; unwrapping BK[1] from BK[3] |
| T8 | Recovery key: encoding, checksum, typo detection (1 flipped character), `O/0 I/1` normalisation, blob decryption |
| T9 | Invite link secret: `K_inv`, `sealed_wrap` |
| T10 | Fingerprint digits and SAS digits |
| T11 | Coverage canonical JSON + digest (keys sorted, no spaces, integers) |
| T12 | **CRDT over the encrypted log:** CRDT vectors V3, V5, V11 and V13 replayed through envelopes (encrypt → server log → decrypt → apply) give the same texts. A snapshot at N plus the tail equals the full log replay. |
| T13 | Compaction safety: a snapshot built from a live doc with unacked edits (the forbidden path) shows the divergence; a snapshot from a temp doc does not |
| T14 | Rotation race: two rotations with the same `expected_key_gen`; exactly one wins and the other gets 409 |
| T15 | Lineage: an update encrypted under epoch E1 fails to decrypt with E2 |
| T16 | Awareness envelope roundtrip and the PeerLeft mapping |

Cross-implementation requirement: Go (reference), TypeScript (web) and Swift
(Apple), and later Go+wazero (TUI), MUST all pass T1–T11 byte for byte and
T12–T16 behaviourally.

## B18. LGPD (sealed)

- **Minimisation:** the operator processes ciphertext plus the metadata in
  §B3. Privacy-policy template text: "Para livros selados, o operador não tem
  acesso ao conteúdo; guarda metadados de uso (quem, quando, quanto)."
- **Access (Art. 18 II):** the server export contains ciphertext and wraps.
  The client export decrypts locally. Both are documented. The user holds
  the keys, so access is satisfied by the client.
- **Erasure:** deleting a note or book deletes the ciphertext rows. Deleting
  a user deletes their keys, devices and unlock blobs. Copies held by other
  members' devices are outside the operator's control, the same as for plain
  books.
- **Breach:** a database leak exposes metadata and ciphertext. The risk
  assessment for ANPD notification can say so.
- **Data portability:** `Exportar decifrado` gives markdown + canvas in the
  same layout as the plain export.

---

# Part C: plan, risks, decisions

## C1. Phased plan

Mirrors and sealed books are independent. Mirrors ship first: they are smaller
and they give sealed books their backup story.

### Mirrors

| Phase | Server | Web | Apple | TUI |
|---|---|---|---|---|
| **M0: prerequisite fixes** | `frameNoteMeta2` (G1). Manifest semantics for followers: a replica stops resurrecting notes home deleted while the link was down (it compares against home's tombstones once §A7.2 lands; until then it logs). Route-table test harness. | — | — | — |
| **M1: core mirror** | migration `0010`; invite `mode`; accept v2; home-side send-only filter; mirror resolver clamp + `mirror_read_only`; fswatch/lazy-init off; delete of absent notes; `federation.role="mirror"`; CLI `federation list/status`; two-server tests A19 #1–4, 6, 7, 11 | mirror badge; invite dialog with a mode selector and consent text; read-only copy | mirror badge; read-only | `books.yaml` `read_only`; clone of a mirror |
| **M2: operations** | windowed backfill; `mirrorAck`/`resumeFrom`/`changesEnd`; `pending_deletes` + SLA alert; divergence detection; integrity pass; admin API + CLI resync/pause/resume; `mirror promote`; read-only live WS (`ws-ticket mode=read`); opt-in Prometheus | federations panel: lag, pending deletes; Viewer live read | same | — |
| **M3: publish** | publish scope filter at home; `publish_settings`; public endpoints; rate limits; sever `purge` | `/p/:slug` reader; publish consent dialog | consent dialog | — |

### Sealed books

| Phase | Server | Web | Apple | TUI |
|---|---|---|---|---|
| **S0: crypto core + spikes** | `internal/sealed/ls1` reference implementation; vectors T1–T11 | `lib/sealed/ls1.ts` passes the vectors; spike: `@hpke/core` + WebCrypto X25519/Ed25519 non-extractable in the 3 browsers | `LumiKit/Sealed` passes the vectors; spike: CryptoKit HPKE + SE wrapping | spike: yrs → wasm32-wasip1 under wazero (size, speed) |
| **S1: keys & storage** | `0011`, `0012`; user/device/unlock routes; device requests + feed `device.request`; sealed vault create; keyring routes; audit actions | UK setup + recovery key UX; device approval (SAS); unlock | same, Keychain/SE | locked listing (D2-a) |
| **S2: single-user live sealed** | sealed WS endpoint (log, fan-out, ack, awareness relay, PeerLeft); REST log/snapshot; compaction leases; history retention job; `409 sealed` on plaintext routes; fswatch/mirror off | `SealedProvider`; encrypted replicas; local search index; compaction; meta envelopes (create/move/rename) | `SealedSyncClient` with yswift; encrypted replicas | — |
| **S3: multi-user** | invites with `sealed_wrap`; pending members; rotate endpoint (CAS + wrap-set check, atomic removal); `rotation_due`; snapshot disputes | link-key invites; pending-member wrapping with fingerprints; removal → rotation; key-change warnings | same | — |
| **S4: lifecycle & migration** | conversion `seal/begin|commit|abort`; device revoke; key reset; LGPD export of ciphertext | conversion wizard; "Exportar decifrado"; "Copiar livro…"; passkey PRF (optional); "Girar tudo" | conversion; export; Face ID gate | — |
| **S5: federation & mirror of sealed books** | sealed relay frames (§B11); coverage; home-only compaction; `sealed_user_keys` | — | — | — |
| **S6: TUI** | — | — | — | full support via wazero-yrs (D2-c) |
| **S7: hardening (optional)** | book write keys (§B6.8); optional per-update signatures | verify | verify | verify |

Gate between phases: every phase ends with its vectors and two-client
integration tests green. External cryptographic review before S3 ships to
users (§C2).

## C2. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Web bundle served by the operator | E2E on the web is only as strong as the operator's honesty about JS | Honest copy in the UI; published bundle hashes; support a separately hosted web client; recommend the apps for high-threat use |
| Key loss | Permanent data loss for sole-member books | Mandatory recovery key at setup; re-confirmation; admin re-share for shared books; local plaintext exports encouraged |
| Bugs in a custom protocol | Confidentiality or integrity failures | Standard primitives only (HPKE, AES-GCM, HKDF, Ed25519); byte-exact cross-language vectors; external review before S3; fuzzing of the envelope and frame decoders |
| Compaction sabotage or bugs | Content loss | Temp-doc rule; signed snapshots; client verification and disputes; 30-day history; 3 snapshots kept |
| Rotation races, partial wrap sets | Members locked out | CAS on generation; server-side wrap-set equality check; `wrap_set_mismatch` with the expected list |
| Browser support (WebCrypto X25519/Ed25519) | Some users cannot open sealed books | Feature detection with a clear message; D11 |
| yrs-in-WASM maturity for the TUI | S6 slips | S0 spike; TUI stays locked meanwhile |
| Large books: first open decrypts N snapshots | Slow cold start | Lazy decryption per opened note; metadata-only list (meta envelopes are small); parallel WebCrypto calls |
| Loss of server-side conveniences (search, copy, FS files) | UX regressions vs plain books | Clear "selado" affordances; client-side equivalents; sealed is opt-in per book |
| Mirror attachments not replicated | Incomplete backups | Documented; attachment replication is a follow-up for replicas and mirrors alike |
| Mirror operators keeping data after revoke | LGPD exposure | Consent copy; sever policy; sealed + mirror for untrusted operators |
| Followers resurrecting notes (G2) | Erasure failure | Fixed for mirrors in M1; for replicas in M0/M2 |

## C3. Decisions for the owner

Each question lists the **recommended option first**.

1. **D1: Recovery mechanism for sealed books?**
   - (a) **Mandatory recovery key (256-bit, shown once, re-confirmed), plus key
     reset with re-sharing by book admins.**
   - (b) A recovery passphrase stretched with Argon2id. The server could attack
     weak passphrases offline.
   - (c) No recovery (losing all devices means losing all sealed books).
2. **D2: TUI support level in v1?**
   - (a) **Locked: sealed books are listed with 🔒 and cannot be opened; full
     support comes later via yrs-in-WASM (wazero).**
   - (b) Read-only through an encrypted text rendition uploaded by other
     clients.
   - (c) Full support in v1 (delays sealed books by the S6 work).
3. **D3: Encrypt note paths, titles and the book name?**
   - (a) **Yes: random note ids, encrypted meta envelopes, encrypted book
     name, client-side path-collision rule.**
   - (b) No: only bodies encrypted. Simpler, but titles and folder structure
     leak.
4. **D4: Content cipher?**
   - (a) **AES-256-GCM with per-(generation, note) HKDF keys (WebCrypto
     non-extractable, CryptoKit and Go stdlib native).**
   - (b) XChaCha20-Poly1305 (libsodium/noble; keys as raw bytes in JS; libsodium
     on Apple).
5. **D5: Passkeys?**
   - (a) **Optional on the web (PRF unlock), strongly suggested; never
     required.**
   - (b) Required for sealed books on the web.
   - (c) Not supported.
6. **D6: Can mirrors be public?**
   - (a) **Yes, only as publish mirrors: home owner consent (`vault.manage`),
     default scope "notes flagged `publish: true`", home-side filtering, off
     unless the mirror operator enables it.**
   - (b) Yes, the whole book.
   - (c) No public mirrors.
7. **D7: Default behaviour of a mirror when home revokes it?**
   - (a) **`keep` for backup mirrors (frozen read-only copy), `purge` for
     publish mirrors.**
   - (b) Always purge.
   - (c) Always keep.
8. **D8: Cryptographically stop Viewers from forging edits (book write keys)?**
   - (a) **Later (S7); v1 relies on the server ACL for write authorisation.**
   - (b) In v1.
   - (c) Never.
9. **D9: When converting a plain book to sealed, re-key note ids?**
   - (a) **Yes: new random ids and epochs, book frozen during conversion,
     offline edits become conflict copies.**
   - (b) Keep the old ids (old slugs keep leaking titles; replicas keep
     working).
10. **D10: After a member is removed without a keyed client (admin panel,
    erasure), allow writes until a rotation happens?**
    - (a) **Allow, with `rotation_due` and an automatic rotation on the next
      admin open.**
    - (b) Block writes to the book until rotated.
11. **D11: Browsers without WebCrypto X25519/Ed25519?**
    - (a) **Refuse sealed books there with a clear message.**
    - (b) A pure-JS fallback with extractable keys and a warning.
12. **D12: Who may read a mirror?**
    - (a) **Only members granted by home's control state (`user@mirror`),
      clamped to read.**
    - (b) Also mirror-local readers granted by the mirror operator.
13. **D13: Mirror promotion?**
    - (a) **CLI-only (`lumi-server mirror promote`), audited; no automatic
      fail-back.**
    - (b) Also from the admin HTTP API.
    - (c) Not supported (a mirror is backup-only; restore means export and
      import).
14. **D14: History retention for sealed books (sabotage recovery)?**
    - (a) **30 days of log after compaction, plus the last 3 snapshots
      (configurable).**
    - (b) 7 days.
    - (c) Keep forever.
15. **D15: Sealed books on federated replicas in the first sealed release?**
    - (a) **No: sealed books are single-server until S5 (mirrors of sealed
      books also land in S5).**
    - (b) Ship federation together with S3.

## Owner decisions (2026-11-24)

- **D1 Recovery:** (a) mandatory recovery key (256-bit, shown once, re-confirmed) + key reset with re-sharing by book admins.
- **D2 TUI in v1:** (a) locked — sealed books listed with 🔒, not openable; full support later (S6).
- **D3 Metadata:** (a) encrypt note paths, titles and the book name (random note ids, encrypted meta envelopes).
- **D6 Public mirrors:** (a) only as publish mirrors, with owner consent, default scope `publish: true` notes, filtered at home, off unless the mirror operator enables it.
- D4, D5, D7–D15: pending owner confirmation; the recommended options (a) are the working assumption.
