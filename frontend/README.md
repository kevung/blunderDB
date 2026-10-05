# blunderDB frontend

Svelte 5 (runes) + Vite. Two entry points share the same toolchain and the
same board renderer:

- `index.html` → `src/main.js` → `src/App.svelte`: the desktop GUI, built into
  `frontend/dist` and embedded by `main.go` through Wails. The Go bindings it
  calls are generated in `wailsjs/`.
- `web.html` → `src/web/main.js` → `src/web/WebApp.svelte`: the web front of
  the `serve` daemon (`vite.web.config.js`, `make web`). Its output is
  committed under `internal/server/webui/dist` and embedded in the binary;
  rebuild it whenever `src/web/` changes.

## Scripts

```bash
npm run dev            # Vite alone (the desktop app runs through `make dev`)
npm run build          # → dist/
npm run build:web      # → ../internal/server/webui/dist
npm test               # vitest (unit/component tests, see src/__tests__/README.md)
npm run test:e2e       # Playwright (see tests/e2e/README.md)
npm run lint           # eslint
npm run format:check   # prettier; CI fails on any diff (`npm run format` fixes)
npm run check          # svelte-check against jsconfig.json
```

`check:svelte-warnings`, `check:types` and `check:e2e-flaky` run the scripts
of the same name under `scripts/`.

Working rules for this directory are in `CLAUDE.md`.
