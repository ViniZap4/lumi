# Working with Submodules

lumi is organized as a monorepo where each component is a separate Git repository linked via [git submodules](https://git-scm.com/book/en/v2/Git-Tools-Submodules).

## Repository Map

```
lumi/                          # ViniZap4/lumi (monorepo)
├── tui-client/                # ViniZap4/lumi-tui
├── server/                    # ViniZap4/lumi-server
├── web-client/                # ViniZap4/lumi-web
└── site/                      # ViniZap4/lumi-site
```

Each subfolder is its own Git repo with its own history, branches, and remote.

## Cloning

Always clone with `--recurse-submodules` to pull all components:

```bash
git clone --recurse-submodules git@github.com:ViniZap4/lumi.git
```

If you already cloned without it:

```bash
git submodule update --init --recursive
```

## Day-to-Day Workflow

### Making changes in a submodule

Each submodule is a full Git repo. You `cd` into it, work normally, commit, and push:

```bash
cd tui-client
# make changes...
git add -A
git commit -m "feat: add new keybinding"
git push
```

After pushing the submodule, go back to the root and update the submodule reference:

```bash
cd ..
git add tui-client
git commit -m "feat(tui): add new keybinding"
git push
```

This is important — the root repo tracks which commit each submodule points to. If you skip this step, other people cloning the repo will get the old version.

### Pulling updates

To pull changes from all repos at once:

```bash
git pull
git submodule update --init --recursive
```

Or in one command:

```bash
git pull --recurse-submodules
```

To also fetch the latest from each submodule's remote:

```bash
git submodule update --remote --merge
```

### Checking status across all submodules

```bash
git submodule status
```

This shows the commit SHA each submodule is pinned to. A `+` prefix means the submodule has local changes not yet recorded in the root.

To see a summary of changes in each submodule:

```bash
git submodule foreach 'git status'
```

Or check what's ahead/behind:

```bash
git submodule foreach 'git log --oneline origin/main..HEAD'
```

## Common Scenarios

### I changed code in a submodule but forgot to commit the root

The root repo will show the submodule as modified:

```bash
$ git status
modified:   tui-client (new commits)
```

Just stage and commit it:

```bash
git add tui-client
git commit -m "chore: update tui-client submodule ref"
```

### I want to work on a single component

You can clone any submodule independently:

```bash
git clone git@github.com:ViniZap4/lumi-tui.git
```

This is useful if you only care about one component and don't need the full monorepo.

### Submodule is in detached HEAD state

This happens after `git submodule update`. The submodule checkout is pinned to a specific commit, not a branch. To fix it:

```bash
cd tui-client
git checkout main
```

### I need to push changes to multiple submodules

Push each one, then update the root:

```bash
# Push all submodules that have changes
git submodule foreach 'git push || true'

# Update root references
git add tui-client server web-client site
git commit -m "chore: update submodule refs"
git push
```

### Someone else updated a submodule and I need to get their changes

```bash
git pull                                    # pull root (gets new submodule refs)
git submodule update --init --recursive     # checkout the pinned commits
```

## The Mental Model

Think of it like this:

- **Submodule repos** (`lumi-tui`, `lumi-server`, etc.) hold the actual code and history for each component.
- **Root repo** (`lumi`) is a "manifest" that says: "tui-client should be at commit X, server at commit Y, etc."

When you commit in a submodule and push, the code is on GitHub in that submodule's repo. But the root repo still points to the old commit until you `git add <submodule> && git commit` in the root.

```
Root commit history:
  abc1234  "update tui-client"  → tui-client pinned to commit aaa111
  def5678  "update server"      → server pinned to commit bbb222

tui-client commit history (independent):
  aaa111  "feat: new feature"
  aaa000  "fix: bug"
```

## Quick Reference

| Task | Command |
|------|---------|
| Clone everything | `git clone --recurse-submodules <url>` |
| Pull everything | `git pull --recurse-submodules` |
| Init submodules after clone | `git submodule update --init --recursive` |
| Check submodule status | `git submodule status` |
| Run command in all submodules | `git submodule foreach '<cmd>'` |
| Fetch latest from submodule remotes | `git submodule update --remote --merge` |
| Fix detached HEAD | `cd <submodule> && git checkout main` |
