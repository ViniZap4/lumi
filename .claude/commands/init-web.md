# Initialize Web Client Context

Scan the web client codebase and give me a working overview so I can start developing.

Do the following:

1. **Check dependencies**: Read `web-client/package.json` for dependencies and scripts
2. **Check build health**: Run `cd web-client && npm run build 2>&1 | tail -20` and report status
3. **Summarize app state**: Read the top ~100 lines of `web-client/src/AppFinal.svelte` to understand state shape and view modes
4. **List API surface**: Read `web-client/src/lib/api.js` to list all API functions
5. **List recent changes**: Run `git log --oneline -10 -- web-client/` to show recent web commits
6. **Show open TODOs**: Search for `TODO`, `FIXME`, `HACK` comments in `web-client/src/`
7. **Check themes**: Read `web-client/src/lib/themes.js` to list available themes

Present a concise summary with:
- Build status (pass/fail)
- Current view modes and state shape
- Complete API function list
- Available themes
- Any TODOs or known issues
- Suggested areas to work on

Keep the output short and actionable.
