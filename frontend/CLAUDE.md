# frontend — rules

- **Svelte 5 stores**: inside components, `$store` or `$effect(() => { const v = $store; … })`,
  never `.subscribe()` (stale closures invisible to the compiler). A rare exception is justified
  in the commit message.
- **One type scale**: `--font-size-base/-small/-title` from `src/style.css`, never an absolute
  `font-size`; form controls carry `font: inherit`. Hierarchy comes from weight and colour;
  exceptions are named in ADR-0008.
- **Keyboard**: letter shortcuts match `event.key` (`isLetter()` in `src/utils/keys.js`), digit shortcuts
  match `event.code` — layout-independent on AZERTY/QWERTZ.
- **Generated, never hand-edited**: `wailsjs/` (restart `wails dev` after changing a bound Go
  method); `src/i18n/help/*.js` (`make help`, ADR-0034).
- **The web front** (`src/web/`) ships as `internal/server/webui/dist/`, committed and embedded:
  run `make web` after touching it — nothing detects a stale bundle (ADR-0039).
- `App.svelte` stays thin; logic lives in `components/` panels and one store per area under
  `stores/`. `commandVocabulary.js` is locked to `commandProcessor.js` by
  `commandVocabulary.sync.test.js`.
- A fresh worktree fails the eslint pre-commit hook: symlink `frontend/node_modules` from the
  main checkout rather than `--no-verify`.
- Playwright locally: port 5173 may be held by another app (gammonGo) — start Vite on a free
  port, or the e2e suite silently tests the wrong app.
