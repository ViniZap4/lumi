# Initialize Wiki Context

Scan the wiki and project documentation to give me a working overview.

Do the following:

1. **Read existing docs**: Read `wiki/USER.md` and `wiki/DEV.md` fully
2. **Check CLAUDE.md**: Read the root `CLAUDE.md` for project conventions
3. **Cross-reference accuracy**: Verify that documented keyboard shortcuts, routes, and features match actual code:
   - Check TUI key handlers in `tui-client/ui/keys_home.go` against USER.md shortcuts
   - Check server routes in `server/main.go` against DEV.md API docs
4. **Find undocumented features**: Search for features in code not mentioned in docs (e.g., new themes, new commands)
5. **List recent changes**: Run `git log --oneline -20` to find features added after docs were last updated

Present a concise summary with:
- Current doc coverage (what's documented vs what exists)
- Outdated or inaccurate sections
- Missing documentation
- Suggested documentation tasks

Keep the output short and actionable.
