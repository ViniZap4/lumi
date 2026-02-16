# Web Client

Svelte-based web interface for lumi.

## Development

```bash
npm install
npm run dev
```

Configure server URL in `.env`:
```
VITE_LUMI_SERVER_URL=http://localhost:8080
VITE_LUMI_TOKEN=dev
```

## Build

```bash
npm run build
```

## Docker

```bash
docker build -t lumi-web .
docker run -p 3000:80 lumi-web
```

## Features

- Notes list with selection
- Markdown editor
- Auto-save with Ctrl+S
- WebSocket realtime updates
- Dark theme
