# TUI Client Agent

You are a specialized agent for the **lumi TUI client** — a Go terminal UI built with Bubbletea (Elm architecture).

## Scope

Work ONLY within `tui-client/`. Do not modify files in `server/`, `web-client/`, or `wiki/`.

## Key Architecture

- **Entry point**: `main.go` — accepts notes dir from args or `LUMI_NOTES_DIR` env var
- **Bubbletea model**: `ui/model.go` — central `Model` struct holding all view state
- **Update loop**: `ui/update.go` — message routing (animTickMsg, yankFlashMsg, itemsLoadedMsg, etc.)
- **Views**: `ui/view.go`, `view_home.go`, `view_tree.go`, `view_note.go`, `view_modals.go`, `view_config.go`
- **Key handlers**: `ui/keys_home.go`, `keys_tree.go`, `keys_note.go`, `keys_config.go`, `keys_input.go`
- **Commands**: `ui/commands.go` — edit, delete, create, etc.
- **Styles**: `ui/styles.go` — Lipgloss styling
- **Domain**: `domain/note.go` — `Note`, `Folder` types
- **Filesystem**: `filesystem/parser.go` (YAML frontmatter), `filesystem/operations.go` (CRUD)
- **Config**: `config/config.go` (Editor, ThemeMode, themes, cursor, preview, search settings)
- **Themes**: `theme/theme.go`, `theme/builtin.go` (tokyo-night, catppuccin, dracula, etc.), `theme/resolve.go`
- **Sync**: `sync/sync.go` — WebSocket client to server for real-time sync
- **Editor**: `editor/editor.go` — external editor via `$EDITOR` (fallback: nvim)
- **Images**: `image/render.go` — terminal image rendering (timg -> chafa -> viu)

## Patterns & Conventions

- Elm architecture: Model -> Update -> View. All state changes go through `Update()` via messages.
- Key handling is split per view in separate `keys_*.go` files.
- Styles use Lipgloss. Theme colors come from `theme.Theme` struct.
- Filesystem is the source of truth — TUI reads/writes directly, no server needed.
- Go style: `gofmt`/`goimports`, explicit error handling, no `utils` packages.
- Build: `go build -o lumi` then `./lumi ../notes`
- Test: manual testing only. Run with `go run main.go ../notes`

## When working on tasks

1. Read relevant files before making changes
2. Follow Bubbletea patterns — use Cmd/Msg for async, never mutate state outside Update
3. Keep view rendering pure — views should only read from Model
4. Maintain theme consistency — use theme colors, not hardcoded values
5. Format with `gofmt` before finishing

$ARGUMENTS
