# CRDT client protocol (lumi, wave 1)

Status: normative for the web, Apple and TUI clients. The server sections are
**requirements** (§10); they are not implemented yet.
Source of truth read for this spec: `server/internal/crdt/{crdt.go,registry.go}`,
`server/internal/wsync/{handler.go,hub.go,protocol.go}`,
`server/internal/notes/notes.go` (GetSnapshot / ApplyDiff / ApplyUpdate /
applyBodyToCRDT), `server/cmd/lumi-server/main.go` (fs-watcher bridge).

Keywords MUST / MUST NOT / SHOULD follow RFC 2119.

---

## 0. TL;DR (the five rules)

1. **Never seed.** A client MUST NOT write a note's text into a Y.Doc it has
   not first synced from the server. The server already holds the text in
   its own Y.Doc. A local insert under another clientID merges in as a
   *second copy* (vector V2).
2. **Join empty, edit after SyncStep2.** Start from an empty Y.Doc, connect,
   and allow editing only after the first SyncStep2 arrives. Until then show
   a read-only preview of the REST body.
3. **Ranged ops only.** Local edits become the smallest possible
   delete/insert ranges, applied to the *current* Y.Text in **one**
   transaction. Never use delete-all + insert-all (vectors V6, V7).
4. **One replica per note, kept for the session.** Reconnecting reuses the same
   Y.Doc. Do not throw it away on refetch or reconnect. If the doc has to be
   recreated, recreate it empty and join again (rule 2).
5. **Opaque editors merge 3-way.** A client that can only produce "new full
   text" (TUI + `$EDITOR`) MUST diff *base text → edited text* and apply those
   ops to a doc restored from the *base state*. It then sends the resulting
   update. It MUST NOT send bare text for the server to diff against current
   state, because that path is last-writer-wins (vector V9a).

---

## 1. Wire facts (as implemented today)

| Item | Value |
|---|---|
| Live endpoint | `GET /api/vaults/:vault/notes/:id/sync` upgraded to WebSocket. Auth is `?token=` (WS) and the server checks capability `note.edit` before the upgrade. Optional `?client_id=<uuid>` sets presence identity. |
| Framing | y-protocols v1, binary frames only (text frames ignored). Max frame 1 MiB. Idle read deadline 60 s, reset by any frame or pong. |
| Message types | `0` Sync (`0` Step1 sv, `1` Step2 update, `2` Update), `1` Awareness (opaque relay, no server register). The server's `3`=Auth and `4`=QueryAwareness differ from y-websocket's (`2`=Auth, `3`=QueryAwareness), see §10 S8. |
| Server handshake | On join the server sends **SyncStep1(server sv)** first. On a client Step1 it replies **Step2(diff since client sv)**. The server treats client Step2 and Update the same way: apply, persist, fan out to the other subscribers (never echoed to the sender), schedule the disk mirror. |
| Doc layout | One root `Y.Text` named **`content`** holds the whole markdown body. Frontmatter is NOT in the CRDT. Updates are lib0 **v1** encoding. |
| Offsets | Y.Text indices are **UTF-16 code units** (JS string semantics). Native clients (Swift `String`, Go `string`) MUST convert. |
| Server init | `Registry.InitFromText` runs at note create, vault copy and federation import. It builds a fresh doc, inserts the body under the server's random clientID, and stores it as the snapshot. |
| Persistence | Every applied update is appended to `note_yjs_updates` and fires the federation persist hook. The log compacts into the snapshot at 200 rows or 1 MiB. |
| Disk mirror | Live updates are mirrored to the `.md` file (frontmatter preserved) after a 500 ms debounce. FS-watcher edits enter the live room as a prefix/suffix `ApplyTextDiff`. |
| REST | `GET …/snapshot` → `{id,path,text,vector_clock}` (vector_clock = base64 state vector). `POST …/diff` takes `{text, base_clock?, origin?}`, which is diffed against the server's current text, so it is LWW; `base_clock` is ignored today. It also takes `{update, origin?}`, a raw lib0-v1 update applied as a CRDT merge. |

---

## 2. Joining a note (live clients: web, Apple)

### 2.1 State machine

```
            new session(vault, note)            first SyncStep2
  IDLE ───────────────▶ CONNECTING ──▶ SYNCING ───────────────▶ LIVE
                            ▲  │ fatal close / auth dead           │  socket drop
                            │  ▼                                   ▼
                            │ CLOSED (terminal)               OFFLINE (editable)
                            └───────────── backoff reconnect ◀─────┘
```

### 2.2 Algorithm

1. Create a **new, empty** `Y.Doc` (fresh random clientID) and get
   `ytext = doc.getText('content')`. A session is bound to exactly one
   `(vaultID, noteID)`. Switching notes MUST destroy the session (provider
   first, then the doc) and build a new one. Retargeting a doc in place is
   forbidden: the previous note's items would leak into the new room.
2. Connect the provider. y-websocket does the following on its own: it sends
   Step1(local sv). It answers the server's Step1 with Step2, which for an
   empty doc is the 2-byte empty update `[0,0]`. It raises `sync(true)` when it
   receives Step2.
3. While in CONNECTING or SYNCING:
   - the editor MUST be read-only or not mounted. Show the REST `/content`
     body as a rendered preview with a "connecting" indicator.
   - the client MUST NOT insert anything into `ytext`.
4. On the first `sync(true)`, enter LIVE. Record `emptyAtSync =
   ytext.length === 0`. Mount the editor **with its initial document set to
   `ytext.toString()`**. Delta-based bindings such as y-codemirror.next only
   forward deltas, so an editor that starts empty over a non-empty Y.Text
   applies every later delta at the wrong offsets.
5. **Legacy and uninitialised notes.** Suppose `emptyAtSync` is true and the
   REST body is non-empty. Then the server never initialised this note's CRDT
   (no snapshot, no log). The client MUST NOT seed automatically, because
   editing an empty doc would make the next mirror overwrite the file with
   only the new keystrokes. The client MAY offer an explicit user action,
   "Load file contents", which inserts the body in one transaction with
   origin `lumi:init-from-file`. The action is allowed only when LIVE and
   `ytext.length === 0`. The proper fix is server-side lazy init (§10 S4).
   Once that ships, this branch becomes unreachable.
6. `sync(false)` after a disconnect does **not** return to read-only. The
   local doc is a valid replica, and offline edits are merged on reconnect
   (§5).

### 2.3 Read-only viewers

Clients without `note.edit` cannot upgrade. They render `/content` or
`/snapshot` text only and poll or refresh. They MUST NOT build a Y.Doc from
that text.

---

## 3. Local edits → minimal ranged ops

### 3.1 Editor bindings that emit changes (CodeMirror, ProseMirror, NSTextView delegate)

Forward each editor change set to the Y.Text as ranged `delete(index, len)`
and `insert(index, text)` calls inside one `doc.transact(fn, ORIGIN_EDITOR)`.
y-codemirror.next already does this. For native text views, use the delegate
callback (`textView(_:shouldChangeTextIn:replacementString:)` or
`textDidChange` with the edited range). Convert the NSRange, which is in
UTF-16 already, directly to a Y index.

### 3.2 Whole-text editors (SwiftUI `TextEditor`, "save" buttons, `$EDITOR`)

When all the client has is `newText`:

```
cur  = ytext.toString()            // CURRENT replica text, read now, not "last saved"
if cur == newText: return
p = common prefix length (UTF-16 units)
s = common suffix length, limited to min(len(cur), len(newText)) - p
// never split a surrogate pair:
if p > 0 and isHighSurrogate(cur[p-1]):  p -= 1
if s > 0 and isLowSurrogate(cur[len(cur)-s]): s -= 1
doc.transact(ORIGIN_EDITOR):
    if len(cur)-p-s > 0: ytext.delete(p, len(cur)-p-s)
    if len(newText)-p-s > 0: ytext.insert(p, newText[p : len(newText)-s])
```

- The diff MUST be computed against the **current** Y.Text and applied
  synchronously. Remote updates must not interleave between the read and the
  write. On Apple, do both on the doc's serial queue or actor.
- Prefix/suffix produces one hunk. That is correct but coarse: if the user
  changed two distant places, everything between them is deleted and
  re-inserted, and concurrent remote edits inside that span get displaced. A
  **Myers / diff-match-patch** diff is RECOMMENDED. Apply its hunks in one
  transaction, **right to left** (descending index), so earlier indices stay
  valid.
- Forbidden: `ytext.delete(0, ytext.length); ytext.insert(0, newText)`.
  It resurrects text that was concurrently deleted (V7) and misplaces
  concurrent inserts (V6).
- Debounce whole-text diffs (≈150–300 ms idle). Do not run one per keystroke
  on large notes.

### 3.3 What gets sent

Live clients send nothing manually. The provider's `doc.on('update')`
handler ships each transaction's update as Sync/Update. A REST-only CRDT
client (Apple fallback) sends `POST /diff {update: base64(encodeStateAsUpdate(doc,
lastServerSV)), origin}`. After a 2xx it sets `lastServerSV` to the returned
`vector_clock`.

---

## 4. Remote updates → editors

1. Observe `ytext.observe((event, tr) => …)` and **skip** transactions whose
   `tr.origin` is the local editor origin. Those are already in the editor.
2. Convert `event.delta` (`retain` / `insert` / `delete`) into editor changes
   at running offsets, applied as one editor transaction annotated as remote.
   The annotation keeps them out of the local undo history and stops them
   being re-sent.
3. **Cursor and selection preservation.**
   - Editors with change mapping (CodeMirror, ProseMirror, NSTextStorage edits)
     map the selection through the change set themselves. Nothing else is
     needed.
   - Whole-text views (you can only `setText`): *before* applying the remote
     update, capture `anchorRel = Y.createRelativePositionFromTypeIndex(ytext,
     anchor, assoc)`, and the same for the head. After applying it, set the
     text and restore the selection with
     `Y.createAbsolutePositionFromRelativePosition(rel, doc).index`.
     Vector V14: a cursor at 6 becomes 9 after a remote `">> "` insert at 0.
     Store relative positions, not integers, for anything long-lived:
     bookmarks, comment anchors, remote cursors via awareness.
4. Never call the whole-text diff of §3.2 in reaction to a remote update.
   It would re-encode the remote change as a local one and double-apply it
   under a new clientID.

---

## 5. Reconnection and resync

- Keep the **same** Y.Doc across disconnects. On reconnect the provider sends
  Step1(local sv), so the server returns everything we missed. It answers the
  server's Step1 with Step2, so the server receives every offline edit. No
  separate pending queue is needed, and none is allowed, because the doc is
  the queue. Vector V11.
- Backoff: exponential from 100 ms, capped at ~2.5 s (y-websocket default),
  with jitter recommended for native clients.
- **Fatal closes**: stop reconnecting (provider `disconnect()`) on:
  - `4401`: auth invalid or expired. Clear the session and route to login.
  - `4403`: capability missing. Show a read-only state.
  - `4404`: note deleted.
  - `1008`: policy, the per-user concurrent-socket cap (10).

  Codes 4401, 4403 and 4404 are proposed (§10 S6). Today the upgrade is
  rejected with HTTP 401/403, which a browser sees as close `1006`. So after
  **2 consecutive attempts that never opened**, the client MUST probe
  `GET /api/users/me`. A 401 or `token_expired` is treated as `4401`. A 200
  means retry with a visible "cannot reach live editing" notice.
- A client that has to drop its doc (memory pressure, app relaunch without
  local persistence) re-joins **empty** (§2). If it persisted its doc locally
  (y-indexeddb, a file), it MAY restore that doc and reconnect. The handshake
  merges both ways. It MUST NOT rebuild the doc from text.
- REST refetch (Apple "pull to refresh") of a note that has a live doc MUST
  merge the server state into the existing doc. Two ways to do that:
  - `Y.applyUpdate(doc, serverStateDiff)` once §10 S11 exposes state.
  - simply rely on the socket.

  It MUST NOT replace the doc with a fresh one and then `setText` (V8).

---

## 6. Origin tagging

| Where | Origin value | Meaning / rule |
|---|---|---|
| client txn | editor binding config (e.g. y-codemirror `YSyncConfig`) or `'lumi:editor'` | Local keystrokes. The **only** origin tracked by the `Y.UndoManager`. |
| client txn | `'lumi:init-from-file'` | Explicit legacy-note recovery (§2.2.5). Not undoable by default. |
| client txn | provider instance | Remote updates applied by the provider. Never re-sent (y-websocket filters `origin === provider`). Never undoable. |
| client txn | `'lumi:3way'` | Ops built in a temporary base doc (§7). Never applied to a live editor doc directly. |
| server `originKind` (update log) | `web` (live WS, `wsync.OriginLive`), `tui-diff`/`tui`, `apple-diff`, `fs-watcher`, `snapshot-init`, `vault-copy`, `web-patch`, federation link kinds | Audit, plus echo suppression. The federation relay never sends an update back to the link whose kind matches. `fs-watcher` updates do not schedule a disk mirror, to avoid a loop. Clients SHOULD send a specific `origin` on `/diff`: `tui`, `apple`. |

Rules: a client MUST NOT echo an update it received. Awareness state is
ephemeral and MUST NOT be persisted.

---

## 7. Offline and opaque-editor clients (TUI and `$EDITOR`; Apple offline)

### 7.1 Flow

```
1. GET /snapshot            → { text: baseText, vector_clock: baseSV, state: baseState* }
   persist {baseText, baseSV, baseState} next to the edit (e.g. <notesDir>/.lumi/cache/<vault>/<note>.base)
2. write baseText to a temp file, run $EDITOR       (any amount of time, offline OK)
3. editedText = read temp file;  if editedText == baseText → done
4. tmp = new Y.Doc (fresh clientID); Y.applyUpdate(tmp, baseState)
   assert tmp.getText('content').toString() == baseText
5. hunks = diff(baseText, editedText)               (Myers; prefix/suffix acceptable)
   tmp.transact('lumi:3way'): apply hunks right-to-left as ranged delete/insert
6. update = Y.encodeStateAsUpdate(tmp, baseSV)      (only the TUI's new ops)
7. POST /diff { update: base64(update), origin: "tui" }
   → { text: mergedText, vector_clock }            (server merges: concurrent edits survive)
8. write mergedText to the local .md file (frontmatter untouched), drop the .base cache
```

`*` `state` does not exist yet (§10 S11). Until it does, the TUI can rebuild
`baseState` only if it keeps its own replica.

Why the base and not current: the ops have to say "delete *these* items, insert
after *this* item" in terms of the version the user actually saw. Applying
them as a CRDT update leaves concurrent inserts by other users intact. Inserts
inside a range the TUI deleted also survive (V10), and inserts elsewhere keep
their place (V9b). Diffing edited text against *current* text (what
`ApplyDiff` does today) treats everything the user did not see as something
they deleted (V9a).

### 7.2 Clients that cannot run a CRDT locally

The TUI is pure Go today. It MAY delegate steps 4 to 6 to the server by
sending `POST /diff {text: editedText, base_state: base64(baseState) |
base_clock, origin: "tui"}`. The server MUST then do steps 4 to 6 itself
(§10 S1). It MUST NOT fall back to diff-against-current, except when
`base_clock` equals the current state vector, because then nothing
concurrent happened and the two are identical.

### 7.3 Direct filesystem writes

The TUI also edits `.md` files directly. The server's fs-watcher picks those
writes up as a prefix/suffix diff against the live text, which is the same
LWW weakness as V9a. A TUI connected to a server SHOULD prefer the §7.1 flow
over raw FS writes for notes that are open elsewhere. The plain FS path stays
for offline-only vaults.

---

## 8. Undo

Use `Y.UndoManager(ytext, { trackedOrigins: new Set([ORIGIN_EDITOR]) })`. It
undoes only this user's edits, as CRDT ops, so it never reverts a peer's
change. Undo history is per session and dropped with the doc.

---

## 9. Test vectors

The results below were computed with yjs 13.6 by
`wave1/vectors.mjs` (next to this file). Server seed: `S = "hello world"` via InitFromText.
`A` and `B` are forks of S with clientIDs 1 and 2. "Sync" means a full
state exchange. Every client implementation (yrs, Swift, Go) MUST reproduce
these results.

| # | Scenario | Ops | Expected |
|---|---|---|---|
| V1 | Correct join | empty client syncs with S | client `hello world`, server unchanged |
| V2 | **Seeding bug** (old web) | client inserts `hello world` into its empty doc **before** syncing | `hello worldhello world` on both sides (and on disk) |
| V3 | Concurrent inserts, different places | A `insert(5," brave")`, B `insert(11,"!")` | `hello brave world!` |
| V4 | Concurrent inserts, same index | A `insert(5,"A")`, B `insert(5,"B")` | `helloAB world` (lower clientID left; deterministic on all peers) |
| V5 | Delete range vs insert inside it | A `delete(5,6)`, B `insert(8,"X")` | `helloX` (concurrent insert survives the deletion) |
| V6 | **Delete-all/insert-all** vs concurrent insert (old Apple) | A replaces all with `hello brave world`, B `insert(5,"X")` | `Xhello brave world`: B's insert is displaced |
| V6b | Same with prefix/suffix ops | A `insert(5," brave")` (computed), B `insert(5,"X")` | `helloX brave world` |
| V7 | **Delete-all/insert-all** resurrects a concurrent delete | A replaces all with `hello world!`, B `delete(0,6)` | `hello world!`: B's deletion is lost |
| V7b | Same with prefix/suffix ops | A `insert(11,"!")`, B `delete(0,6)` | `world!` |
| V8 | **Fresh-doc refetch + setText** (old Apple) | new empty doc, `insert(0,"hello world!")`, sync with S | `hello world!hello world` |
| V9a | **TUI today** (text diffed vs current) | base S; web inserts `# ` at 0 → server `# hello world`; TUI edited base → `hello brave world`; POST text | `hello brave world`: web's `# ` is lost |
| V9b | **TUI 3-way** (§7) | same, but ops built on a base-state doc and sent as an update | `# hello brave world` |
| V10 | 3-way, concurrent insert inside a TUI deletion | web `insert(8,"X")` → `hello woXrld`; TUI edited base → `hello` | `helloX` |
| V11 | Offline edit + reconnect | client offline `insert(11,"?")`; server meanwhile `insert(0,"> ")`; reconnect | both sides `> hello world?` |
| V12 | Surrogate safety | text `a😀b` → whole-text editor sets `a😁b` (shared high surrogate D83D) | prefix backs off from 2 to 1 so the op replaces the whole pair (`delete(1,2)`, `insert(1,"😁")`); result `a😁b`, length 4; no lone surrogate is ever inserted |
| V13 | Idempotent replay | apply the same update twice | applied once (`xhello world`) |
| V14 | Relative-position cursor | cursor relpos at index 6, remote `insert(0,">> ")` | restored index 9 |
| V15 | User types before Step2 | client in SYNCING; user types | rejected or no-op (editor read-only); nothing is sent before Step2 |
| V16 | Switch note mid-sync | open n1, switch to n2 before Step2 | n1 provider and doc destroyed; n2 doc empty, then n2 text only; n1 server text unchanged |
| V17 | Up-to-date client reconnect | synced client reconnects, no edits | server receives Step2 `[0,0]`, which MUST NOT create a log row (§10 S3) |

The web client has executable versions of V1, V2, the concurrent-join case,
V11, V16 and the legacy-empty case in `src/lib/editor-session.test.ts` and
`src/views/VaultHomeView.test.ts` (branch `fix/wave1-crdt-web`).

---

## 10. Required server changes (not implemented; tracked separately)

- **S1. `ApplyDiff` must honour the base (3-way).** Accept `base_state`, the
  opaque full update from `/snapshot`. Load a temporary doc from it. Diff
  `baseText → text` (Myers, UTF-16 offsets) and apply the hunks there.
  Encode `state_diff_v1(baseSV)` and apply that update to the live doc
  through the S2 path. A state vector alone (`base_clock`) is not enough to
  rebuild the base text after GC. Use `base_clock` only to detect "no
  concurrent change" (sv equal → the plain diff is safe). If neither
  `base_state` is present nor sv-equal holds, reply
  `409 {error:"base_required"}` instead of overwriting.
- **S2. REST writes must go through the live Room.** `ApplyDiff`,
  `ApplyUpdate` and `Update(body)` → `applyBodyToCRDT` load a **cold** doc
  from Postgres even when a Room is active. Three things go wrong:
  - live clients never receive the change.
  - the Room's in-memory doc diverges from the log until it is evicted.
  - the Room's debounced mirror later overwrites the file with the stale
    room text.

  Mirror the fs-watcher pattern: `if room := hub.RoomIfActive(...)` then
  apply to `room.Doc()` and `room.ApplyAndBroadcast…`, otherwise cold path.
- **S3. Drop no-op updates.** y-websocket answers the server's Step1 with
  Step2, and from an up-to-date or empty client that is `[0,0]`. Every
  connect currently appends a log row, fires the federation hook and
  schedules a disk mirror. Compare the state vector (and delete set) before
  and after `ApplyUpdate`, and skip persistence, broadcast and mirror when
  nothing changed.
- **S4. Lazy server-side init.** When `Registry.LoadDoc` finds no snapshot
  and no updates for an existing note, run `InitFromText(body-from-disk)`
  under a per-note singleflight lock and persist it. This removes the
  legacy-empty branch of §2.2.5. Also, `Create` currently ignores
  `InitFromText` errors (`_ =`). Surface them or retry.
- **S5. Single seeder invariant.** Exactly one site may call `InitFromText`
  for a given note. Federation already ships the state from home, which is
  correct. Document that replicas MUST NOT seed from their own disk copy, or
  every replica adds its own duplicate body.
- **S6. Auth close codes.** Upgrade failures are HTTP 401/403 today, so
  browsers see an opaque 1006. Prefer to complete the upgrade and close with
  `4401` (token invalid/expired), `4403` (`note.edit` missing) or `4404`
  (unknown note). Keep `1008` for the concurrency cap. Also re-validate the
  session periodically on long-lived sockets: a revoked token keeps its socket
  today.
- **S7. Presence "left" frame is malformed.** `broadcastPresenceLeave` sends a
  JSON object inside an Awareness frame. y-protocols clients decode that
  payload as a lib0 awareness update, so it throws or corrupts awareness.
  Fix it in one of two ways:
  - emit a proper awareness update (`clientID → null` state, clock + 1),
    learning the numeric awareness clientID from the subscriber's own frames.
  - move presence to a separate message type.
- **S8. Message-type numbering.** Align with y-protocols / y-websocket:
  `2`=Auth, `3`=QueryAwareness. The server's `3`/`4` mean a client
  query-awareness frame is misparsed as Auth. That is harmless only because
  both are dropped.
- **S9. `GET /snapshot` should return `state`.** This is the base64 full
  update (`EncodeStateAsUpdate`). Opaque clients need it for §7, and
  REST-only Apple refetch needs it to merge instead of replacing.
- **S10. `ApplyTextDiff` is prefix/suffix only.** It is fine for fs-watcher
  input. For S1, add a Myers-based multi-hunk variant so concurrent edits
  between two distant changes are not displaced.

---

## 11. Client follow-ups

**Web** (done on `fix/wave1-crdt-web`):
- no seeding, sync-gated editor, single-note session.
- CodeMirror initial doc = ytext.
- the Cmd+S `/diff` text push is removed (LWW).
- auth-aware reconnect stop.

**Apple:**
- Keep one `LumiCRDT` doc per open note for the whole session and persist it
  per note if offline editing is wanted.
- Replace `setText` (delete-all + insert-all) with the §3.2 diff, Myers
  preferred, against the current Y.Text inside one transaction.
- Refetch merges into the existing doc and never creates a fresh one (V8).
- Join empty (§2): wait for Step2 before enabling the editor.
- Restore the selection through relative positions (§4.3).
- Use origin `apple`.

**TUI:**
- Implement §7.1 with the base persisted in `.lumi/cache`, and send
  `update`. If no Go CRDT is available, use §7.2 once S1 ships.
- Stop POSTing bare `text` with an empty `base_clock`.
- Keep FS-direct writes only for offline vaults.
