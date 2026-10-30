# Bancs hors vision

## H1 — WebAssembly dans le front (WebKitGTK sans AVX)

- Cherché : `wasm` / `WebAssembly` / `*.wasm` dans `frontend/dist` (après `npm run build`), `internal/server/webui/dist`, `frontend/src` et les dépendances de production (`chart.js`, `@kurkle/color`, `two.js`).
- Résultat : aucun Wasm (seul faux positif : `wasMatchMode` dans `frontend/src/services/importService.js`). Rien à couper dans `internal/gui/run.go`.
- Suite : le banc V0.1 reprend le sujet quand la vision introduira du Wasm.
