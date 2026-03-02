# Initialize TUI Client Context

Scan the TUI client codebase and give me a working overview so I can start developing.

Do the following:

1. **Check build health**: Run `cd tui-client && go build ./...` and report any errors
2. **Summarize current state**: Read `tui-client/ui/model.go` to understand the current Model fields and view modes
3. **List recent changes**: Run `git log --oneline -10 -- tui-client/` to show recent TUI commits
4. **Show open TODOs**: Search for `TODO`, `FIXME`, `HACK` comments in `tui-client/`
5. **Check dependencies**: Read `tui-client/go.mod` for Go version and key dependencies

Present a concise summary with:
- Build status (pass/fail)
- Current views and modes available
- Any TODOs or known issues
- Key dependencies and their versions
- Suggested areas to work on

Keep the output short and actionable.
