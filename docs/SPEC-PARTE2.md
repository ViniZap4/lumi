# Part 2: shared UX and format contract

Status: draft, 2026-11-24. Applies to web, Apple (macOS first) and TUI. The owner asked for all of this in part 2. Every client implements the same keys and formats; a deliberate difference must be listed in §9.

## 1. Look and motion

- **Look:** modern, calm and dense where it helps. Use the theme tokens only. One prose system for rendered markdown (reader, split preview, live preview, hover preview, kanban cards, canvas text nodes).
- **Motion:**
  - Transitions run 120–220 ms on GPU-friendly properties: view changes (explorer ↔ reader ↔ editor), opening and closing a split, the floating explorer scale/fade in, the hover preview fade, kanban card moves and canvas pan/zoom easing.
  - Reduce Motion is the system setting plus an app setting. When it is on, there are no transforms or fades, only instant state changes.
- **"The app should feel alive"** (owner, 2026-11-24): spring-based motion (not linear), with these touches:
  - shared-element transitions: an explorer row grows into the opened note, and a link morphs into its hover card;
  - list items stagger in when the explorer opens;
  - hover lift and press scale on interactive elements;
  - a smooth cursor and caret, plus smooth scrolling;
  - a sync dot that breathes while syncing and pulses on a remote change;
  - a gentle flash on remotely edited ranges;
  - skeleton shimmer while loading;
  - panes and overlays that settle with a slight spring;
  - the mode pill morphing between vim modes;
  - toasts sliding and stacking.
- **Limits:** keep it fast, never blocking input (keys typed during an animation apply immediately), and stay under 220 ms for anything on the typing path.

## 2. Floating explorer (yazi-in-nvim) — the only navigation

**Owner decision (2026-11-24): the floating explorer is the main and only navigation.** The full-screen explorer column layout and the books sidebar are removed from every client.
- **Opening a book** shows its last opened note (per book, per device). With no last note, it shows a calm empty home (book name, recent notes, sync state), and the floating explorer opens automatically.
- **Opening the explorer:** `-` (editor normal mode, reader, empty home), `<leader>e`, `:explore`, a visible explorer button in the top bar (mouse), and the palette.
- **Books:** `gb` / palette "Switch book" opens the same floating overlay at the books level (accounts → books), so switching books is also floating. `[`/`]` still cycle books.
- **Narrow screens** (phone, narrow window, small terminal): the overlay becomes a full-height sheet.
- **Overlay:** a centred overlay (about 80% × 80%) with yazi Miller columns and a preview, starting at the current note's folder.
- Keys inside it:
  - the usual explorer keymap (j/k/h/l, /, a, r, d, y/p, …);
  - `Enter` opens the note in the current pane;
  - `Ctrl-v` opens it in a vertical split and `Ctrl-s` in a horizontal split;
  - `Esc`/`q` closes it.
- Mouse: click selects, double-click opens, click outside closes.
- TUI: rendered as a bordered overlay box with the same keys.

## 3. Splits (multiple notes)

- The note area is a tree of panes, each holding a note (reader or editor) or a preview.
- **Commands:**
  - `:vsplit [note]`, `:split [note]`, `Ctrl-w v` and `Ctrl-w s` split the current note;
  - `Ctrl-w q` / `:q` closes the pane (the last pane closes the note);
  - `Ctrl-w o` keeps only the current pane;
  - `Ctrl-w =` equalises;
  - `Ctrl-h/j/k/l` and `Ctrl-w h/j/k/l/w/p` move between panes (already shipped).
- The source+preview split shipped earlier is one kind of pane pair.
- **Following links:**
  - `gf` or `Enter` on a link opens it in the current pane;
  - `Ctrl-w f` or `Ctrl-w Enter` open it in a vertical split;
  - `Ctrl-w F` opens it in a horizontal split;
  - mouse: Cmd/Ctrl-click opens in a vertical split and Shift-Cmd/Ctrl-click in a horizontal split.
- Each pane has its own live session. Two panes on the same note share one session/doc.
- The layout is remembered per book, per device.
- **Zoom (tmux-style focus/unfocus)** (owner, 2026-11-25):
  - `Ctrl-w z` (and `:zoom`) toggles the focused pane between filling the whole note area and returning to the exact previous layout. The mouse does the same with a zoom button in the pane header, or a double-click on the header.
  - While zoomed, the other panes are hidden (not closed). A "zoomed" badge shows in the pane header and the status line.
  - Moving to another pane (`Ctrl-h/j/k/l`, `Ctrl-w w`) or opening a split unzooms first.
  - The zoom state is not persisted.
  - The zoom animates (shared-element grow/shrink) and is instant under Reduce Motion.
- **Click-to-focus:** the first click or tap anywhere in a pane focuses it and also performs the click (no "first click only focuses" dead clicks). On macOS this means accepting first mouse in pane content views.
- **Opening in a split from the explorer by mouse** (owner, 2026-11-25), besides `Enter` / `Ctrl-v` / `Ctrl-s`:
  - each hovered or selected row shows small inline actions: Open, Open right, Open below;
  - the right-click menu has the same three items;
  - dragging a note row onto the left, right, top or bottom edge of a pane drops it into a new split on that side, with an edge highlight while dragging;
  - the footer hints are clickable.


## 4. Hover link preview

- Hovering a note link (internal markdown link or `[[...]]`, if supported) for about 350 ms shows a floating card: the rendered note (first ~40 lines, scrollable), title, folder and updated time.
- Actions on the card: open, open in split, close.
- Keyboard: `K` on a link (vim's "hover") opens the same card. `j/k` scroll it, `Enter` opens, `Ctrl-w f` opens in a split, `Esc` closes.
- External links show a URL card only, with no fetch.
- Content always goes through the shared sanitizer.

## 5. Kanban (Obsidian Kanban compatible)

- A note whose frontmatter has `kanban-plugin: board` (or `basic`) is a board. Each `## Heading` is a column, and each list item `- [ ] card text` / `- [x] …` under it is a card. Nested lines belong to the card.
- Keep the plugin's trailing `%% kanban:settings ... %%` block untouched. Unknown content is preserved byte for byte.
- **Views:** a Board view (default for kanban notes) and Source (the markdown). A toggle switches between them.
- **Board keys:**
  - `h/l` column, `j/k` card;
  - `H/L` move the card to the previous/next column, `J/K` move it down/up;
  - `a`/`o` new card, `A` new column, `Enter`/`i` edit card text inline, `x` toggle done, `dd` delete (inline confirm);
  - `/` filter.
- **Mouse:** drag cards between and within columns, and drag columns; double-click edits.
- Every change is a minimal ranged edit of the markdown text (CRDT-safe). A move is a delete plus an insert in one transaction.
- **TUI:** board view with columns side by side and the same keys.

## 6. Canvas (JSON Canvas 1.0)

- Files ending in `.canvas` follow the open JSON Canvas 1.0 spec (jsoncanvas.org), as used by Obsidian. They hold nodes (`text`, `file`, `link`, `group`) with x/y/width/height/color, and edges with from/to node+side, label and color.
- **Server:**
  - Books may hold `.canvas` files next to `.md` notes, with the same id/path rules. The `path` may end in `.md` or `.canvas`.
  - Canvas files have no frontmatter. They sync through the same CRDT (Y.Text holding the JSON), so the protocol rules (ranged ops, epoch) apply.
  - Clients rewrite only the changed JSON span when they can. Otherwise they replace the minimal differing range (common-prefix/suffix diff), never the whole text when only part changed.
- **View (web, macOS):**
  - Infinite board with pan (drag, space+drag, arrows/hjkl) and zoom (pinch, Ctrl/Cmd+wheel, `+`/`-`, `0` fit). Nodes render as cards: text nodes get the shared prose renderer, file nodes embed the note preview, link nodes show a URL card, groups are labelled frames. Edges are curved with arrows and labels.
  - Edit: move/resize nodes, add text node (`a` or double-click empty space), add a file node from the floating explorer, connect edges by dragging from a side handle, delete (`d`/Backspace), edit text inline, colors via a palette.
  - Keys: `Tab`/`Shift-Tab` cycle nodes, `Enter` edit or open, `gf` opens a file node, `Ctrl-w f` opens it in a split.
- **TUI:** read-only canvas outline (groups → nodes, edges listed), with `Enter` on a file node opening the note. Full graphical canvas is out of scope for the terminal.

## 7. Rendered markdown

- One prose system (web `app.css` is the reference), refined further: typography, spacing rhythm, code, tables, callouts, tasks, footnotes, images, embeds.
- Kanban card text and canvas text nodes use the same system.

## 8. Opening the GUI from the terminal

- `lumi open [<book>[/<note-path>]]` (TUI CLI):
  - On macOS it opens the Lumi app through the URL scheme `lumi://open?server=<url>&account=<user>&vault=<uuid>&note=<id>` (or `lumi://open?path=<local book path>&note=<rel path>`). It falls back to the web URL when the app is missing.
  - On Linux and Windows it opens the web client URL for the book/note in the default browser.
  - `lumi open --web` forces the web.
- **The Apple app registers `lumi://`.** It validates every parameter (known account and book only, no path traversal; unknown ⇒ shows an error, never auto-adds accounts or servers) and opens the book/note.
- **Web:** routes `/#/b/<vault>/n/<note-id>` (or the existing router's equivalent) deep-link to a note.

## 9. TUI parity

The TUI gets, terminal-natively:
- the floating explorer;
- splits (vertical/horizontal panes of notes);
- a built-in editor with the same vim engine semantics as far as practical (modes, motions, operators, text objects, registers, `.`, undo), with `$EDITOR` still available via `:edit!` or a setting;
- a reader and a source+preview split;
- the hover-card equivalent via `K`;
- the kanban board and the canvas outline;
- in-TUI account management: login form, account list, sign-out, server add/remove;
- a books screen with the sync status;
- `lumi open`.

The TUI sync remains REST 3-way `/diff` with epoch checks (no live WS) unless a later wave adds it.

## 10. Admin login with a user account

- Server admins are regular users flagged `server_admin` (migration 0009). The flag is granted with the CLI `lumi-server admin grant|revoke <username>`. The first-boot bootstrap admin gets it.
- **Panel login** offers username + password (default tab) and a token tab (`lpa_`).
  - Username + password goes to `POST /admin/v1/session` with `{username, password}`. It is verified with the same bcrypt, lockout and failed-login accounting as the public login, requires `server_admin`, and is audited (`admin.session.create` / `admin.login_failed`).
  - It returns a short-lived admin session token (`lps_…`, stored hashed, TTL ≤ 12 h, revocable, listed with tokens).
- The BFF still holds the token server-side; the browser gets only the `__Host-` cookie.
- Rate limiting and lockout apply per IP and per user. Generic errors never reveal whether the user exists.

## 11. Later in part 2 (not in the first batch)

- One-way mirror between servers (follow a book read-only for backup/publishing).
- Sealed (end-to-end encrypted) books.
- Apple: auto-logout on 4401/expired session; remote cursors (yswift relative positions).
