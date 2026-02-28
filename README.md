# lumi

A local-first, Markdown-based note-taking system with beautiful TUI and web clients.

> **Documentation**: [lumi-note.vercel.app/#/docs](https://lumi-note.vercel.app/#/docs)

## Repository Structure

Each component lives in its own repo, linked here as git submodules:

| Component | Repo |
|-----------|------|
| TUI Client | [ViniZap4/lumi-tui](https://github.com/ViniZap4/lumi-tui) |
| Server | [ViniZap4/lumi-server](https://github.com/ViniZap4/lumi-server) |
| Web Client | [ViniZap4/lumi-web](https://github.com/ViniZap4/lumi-web) |
| Site | [ViniZap4/lumi-site](https://github.com/ViniZap4/lumi-site) |

### Clone with All Submodules

```bash
git clone --recurse-submodules git@github.com:ViniZap4/lumi.git
```

If you already cloned without `--recurse-submodules`:

```bash
git submodule update --init --recursive
```

## Quick Start

### Docker Compose (recommended)

```bash
cp .env.example .env    # edit LUMI_PASSWORD before starting
docker compose up -d

# Web UI: http://localhost:3000
# API:    http://localhost:8080
```

### TUI Client (standalone)

```bash
cd tui-client
go build -o lumi
./lumi ../notes
```

### Server (standalone)

```bash
cd server
LUMI_ROOT=../notes LUMI_PASSWORD=secret go run main.go
```

### Web Client (standalone)

```bash
cd web-client
npm install
npm run dev             # http://localhost:5173
```

## Note Format

```markdown
---
id: my-note-id
title: My Note Title
created_at: 2026-02-16T10:00:00Z
updated_at: 2026-02-16T10:00:00Z
tags:
  - example
---

# My Note Title

Your content here with **markdown** formatting.

Link to other notes: [[other-note-id]]
```

## License

MIT

## Credits

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Svelte](https://svelte.dev), and [Go](https://golang.org).
