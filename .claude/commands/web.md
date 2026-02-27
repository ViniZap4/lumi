# Web Client Agent

You are a specialized agent for the **lumi web client** — a Svelte 5 single-page app using Vite 7.

## Scope

Work ONLY within `web-client/`. Do not modify files in `tui-client/`, `server/`, or `wiki/`.

## Key Architecture

- **Entry point**: `src/main.js` — mounts root App component
- **Root component**: `src/App.svelte` — auth gate + view routing + keyboard handlers:
  - Shows `LoginView` when not authenticated, main views otherwise
  - Calls `store.checkAuth()` on mount to restore saved sessions
  - Initializes data loading and WebSocket only after successful auth
- **Store**: `src/lib/store.svelte.js` — reactive state with Svelte 5 runes:
  - Auth: `authenticated`, `login(password)`, `logout()`, `checkAuth()`
  - Data: viewMode (home/tree/note/config), editMode, selectedNote, allNotes, allFolders
  - Search modal, theme state, editor state, folder preview cache
  - Command modal (input/confirm/select kinds)
- **Views**: `src/views/` — LoginView, HomeView, TreeView, NoteView, ConfigView
- **API client**: `src/lib/api.js` — fetch wrapper with dynamic token auth:
  - `setToken(t)`, `getToken()`, `login(password)` — token management
  - `getFolders()`, `getNotes(path)`, `getNote(id)`, `createNote(note)`, `updateNote(id, note)`
  - `deleteNote(id)`, `moveNote(id, dest)`, `copyNote(id, dest)`, `renameNote(id, newName)`, `createFolder(name)`
  - Env vars: `VITE_LUMI_SERVER_URL` (default http://localhost:8080)
  - Token is set at runtime via login, persisted in localStorage
- **WebSocket**: `src/lib/ws.js` — auto-reconnect (3s delay), token auth via query param
  - `connectWebSocket(onMessage)`, `disconnect()`
- **Themes**: `src/lib/themes.js` — 12+ built-in themes, Svelte store, CSS variable application
  - `applyTheme(name)`, `resolveTheme(mode)`, `loadThemeSettings()`, `saveThemeSettings()`, `watchSystemTheme()`
- **Markdown**: `src/lib/markdown.js` — custom renderer (line-by-line, supports wikilinks, inline formatting)
- **Editor**: `src/lib/editor.js` — CodeMirror v6 with vim mode (`@replit/codemirror-vim`)
  - `createEditor(container, content, theme)`, `destroyEditor()`, `updateTheme()`, `getVimMode()`
- **Components**: `src/components/Editor.svelte` (CodeMirror wrapper), `NotesList.svelte`
- **Styles**: `src/app.css` — CSS variables tied to active theme

## Patterns & Conventions

- Svelte 5 runes (`$state`, `$derived`, `$effect`) — NOT Svelte 4 stores for new code.
- One component per file. Reactive declarations over manual updates.
- `AppFinal.svelte` is the monolith — all view logic lives there currently.
- Theme colors are applied as CSS variables on `:root` via `applyTheme()`.
- All server communication goes through `api.js` (REST) and `ws.js` (real-time).
- Vim keybindings in editor via CodeMirror plugin.
- Dev: `npm run dev` (Vite on :5173). Build: `npm run build` (to `dist/`).

## When working on tasks

1. Read `AppFinal.svelte` and relevant lib files before changes
2. Use Svelte 5 runes for reactivity in new code
3. Keep API calls in `api.js` — don't scatter fetch calls
4. Theme changes must update CSS variables — don't hardcode colors
5. Test with `npm run dev` against a running server

$ARGUMENTS
