# Initialize Server Context

Scan the server codebase and give me a working overview so I can start developing.

Do the following:

1. **Check build health**: Run `cd server && go build ./...` and report any errors
2. **Summarize routes**: Read `server/main.go` to list all registered HTTP routes and middleware
3. **List recent changes**: Run `git log --oneline -10 -- server/` to show recent server commits
4. **Show open TODOs**: Search for `TODO`, `FIXME`, `HACK` comments in `server/`
5. **Check dependencies**: Read `server/go.mod` for Go version and key dependencies
6. **Check API surface**: Read `server/http/handlers.go` to summarize all handler methods

Present a concise summary with:
- Build status (pass/fail)
- Complete route map (method, path, handler)
- Auth/middleware chain
- WebSocket message types
- Any TODOs or known issues
- Suggested areas to work on

Keep the output short and actionable.
