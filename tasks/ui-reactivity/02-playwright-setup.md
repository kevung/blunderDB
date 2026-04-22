# 02 — Playwright + harness Wails + specs de reproduction

**Goal :** Piloter le frontend (servi par Vite ou `wails dev`) depuis des tests E2E sans clic manuel. Produire 2 specs qui reproduisent fiablement S2 (transitions Stats cassées) et un variant de S1 (persistance EPC).

**Depends on :** 00, 01.

**Impact :** Les fixes 05.a à 05.f seront validés par ces specs. La Fiche 06 rejouera les mêmes specs avec `logger.perf` actif pour chiffrer l'amélioration.

## Context

- Wails v2 ne propose pas de mode headless natif. Deux stratégies possibles :
  1. **Mode « Vite + window.go mocké »** (recommandé pour CI) — lancer `npm run dev` côté frontend, installer un mock de `window.go.main.Database` avant le bootstrap. Ne teste pas les bindings Wails réels mais suffit pour la réactivité UI.
  2. **Mode « wails dev en tâche de fond »** — plus proche du réel mais lourd et fragile en CI Linux (WebKit). Utile en local.
- Le plan retient la stratégie **1** pour les specs exécutables en CI, stratégie **2** documentée pour debug local.
- Vite dev server par défaut : `http://localhost:5173/`. `wails dev` expose le frontend sur `http://localhost:34115/` avec les bindings live.
- Le mock Wails devra retourner des positions factices, des matches factices, des stats factices. Réutiliser (ou dériver) les fixtures déjà présentes dans `frontend/src/__tests__/` (statsStore.test.js) si possible.

## Files touched

- **Edit:** `frontend/package.json` — devDependency + script `test:e2e`.
- **New:** `frontend/playwright.config.js`.
- **New:** `frontend/tests/e2e/helpers/wailsMock.js` — mock installable via route-level script injection.
- **New:** `frontend/tests/e2e/helpers/fixtures.js` — positions + matches + stats factices.
- **New:** `frontend/tests/e2e/epc-bar-refreshes-on-return.spec.js`.
- **New:** `frontend/tests/e2e/tab-switch-stats.spec.js`.
- **Edit:** `.github/workflows/build.yml` — **n'ajoute pas de job ici**, la Fiche 06 s'en chargera.

## Tasks

### 1. Installation

- [ ] `cd frontend && npm install --save-dev @playwright/test`.
- [ ] `cd frontend && npx playwright install chromium` (premier run local).
- [ ] Ajouter script `"test:e2e": "playwright test"` dans `package.json`.
- [ ] Ajouter script `"test:e2e:ui": "playwright test --ui"` pour debug local.
- [ ] Ajouter `frontend/tests/` et `frontend/playwright-report/`, `frontend/test-results/` à `.gitignore` si absent.

### 2. `playwright.config.js`

- [ ] Cibler un `webServer` qui lance `npm run dev` sur port 5173 avant les tests.
- [ ] `testDir: 'tests/e2e'`.
- [ ] `timeout: 10000`, `expect.timeout: 2000` (les transitions UI devraient être rapides ; si le test attend > 2 s c'est un bug).
- [ ] `use.browserName: 'chromium'`, `use.viewport: { width: 1280, height: 800 }`.
- [ ] `reporter: [['list'], ['html', { open: 'never' }]]`.

### 3. Helper `wailsMock.js` injectable côté page

- [ ] Exporter `await installWailsMock(page, overrides)` : `page.addInitScript(script)` qui installe sur `window.go.main.Database` les méthodes requises. Signature identique au helper Fiche 01 (≠ implémentation : côté Playwright on injecte dans la page, côté Vitest on installe dans jsdom).
- [ ] Méthodes par défaut à fournir : `OpenDatabase`, `LoadPositions`, `GetStats`, `GetMatches`, `LoadCommandHistory`, `SaveCommand`, `SaveSessionState`, `LoadSessionState`, etc. Remplir la liste au fil des besoins des specs.
- [ ] Aussi stubber `window.runtime` de Wails (événements, dialogs) à l'identique.

### 4. Fixtures

- [ ] `frontend/tests/e2e/helpers/fixtures.js` exporte :
  - `positionA`, `positionB` (structures `Position` minimales, avec `board.points`, `cube`, `dice`, `score`).
  - `matchesSample` (≥ 2 matches).
  - `statsResult` (résultat `ComputeStats` factice, structure observée dans `statsStore.js`).

### 5. Spec `tab-switch-stats.spec.js` (S2)

- [ ] Before each : `installWailsMock(page, { LoadPositions: () => [positionA], GetStats: () => statsResult, GetMatches: () => matchesSample })`.
- [ ] Naviguer vers `/`, ouvrir une DB factice, attendre que l'app soit prête.
- [ ] Bascule Match → Stats : cliquer sur l'onglet Stats. Vérifier que **le contenu du panel Stats** est rendu (`data-testid="stats-panel"` ou similaire ; à ajouter au composant si absent — minimal, un seul attribut). Et vérifier que le panel Match n'est plus visible (`display: none` ou absent du DOM).
- [ ] Bascule Stats → Match : inverse. Le contenu Match doit apparaître.
- [ ] Répéter 5 bascules rapides : toutes doivent passer.
- [ ] Variante Match ↔ EPC ↔ Stats ↔ Anki pour reproduire le bug S2 dans toutes ses variantes.
- [ ] **Attendu initial** : la spec est **rouge** sur `ui-reactivity` (bug présent), deviendra verte après Fiche 05.a.

### 6. Spec `epc-bar-refreshes-on-return.spec.js` (S1 étendu)

- [ ] Naviguer, charger position A (via mock `LoadPositions` returning [A]).
- [ ] Cliquer onglet EPC. Attendre que la barre d'état affiche un texte EPC (vérifier via regex `/EPC[:\s]+\d/`). Mémoriser la valeur.
- [ ] Cliquer onglet Stats.
- [ ] Remplacer le mock `LoadPositions` par `[positionB]` et déclencher le re-load (soit via un helper côté app, soit via `page.evaluate(() => positionsStore.set([...]))`).
- [ ] Retour onglet EPC. Vérifier dans la barre d'état que la valeur EPC **a changé** (différente de la mémorisée). Timeout 200 ms.

### 7. Attributs de test à ajouter aux composants

- [ ] Ajouter **uniquement si absent** : `data-testid="tab-<name>"` sur les boutons d'onglets de `TabbedPanel.svelte` (ou sur le conteneur parent). Un seul `data-testid` par composant majeur suffit — ne pas en mettre partout.
- [ ] Ajouter `data-testid="status-bar"` sur le `.status-bar` de `StatusBar.svelte`.
- [ ] Ces ajouts vont dans la Fiche 05.e (TabbedPanel doc) et peuvent être commités là. Pour la Fiche 02, les tests peuvent utiliser des sélecteurs CSS directs (`.status-bar`, `.info-message`, etc.) s'ils sont suffisamment stables.

### 8. Sanity check

- [ ] `npm run test:e2e` sur `ui-reactivity` → au moins une spec rouge qui reproduit le bug S2. Si toutes les specs sont vertes, soit le mock n'expose pas le bug, soit le bug n'est pas reproductible dans ce setup → revoir.
- [ ] Documenter comment débugger (`npm run test:e2e:ui`).

### 9. Documentation

- [ ] `frontend/tests/e2e/README.md` (≤ 60 lignes) :
  - Comment lancer en local (Vite + Playwright, ou `wails dev` + Playwright).
  - Convention des helpers et fixtures.
  - Règle : chaque bug UI signalé devrait d'abord être reproduit par une spec rouge avant d'être fixé.

## Acceptance

- [ ] `npm run test:e2e` exécutable, specs lisibles.
- [ ] Au moins 1 spec **rouge** sur `ui-reactivity` reproduisant S2 (10/10 runs).
- [ ] Specs passent en **vert** sur une branche de test où l'on stub manuellement le handler Stats (prouve que la spec détecte bien le bug).
- [ ] Doc d'usage.

## Status

- [ ] Playwright installé
- [ ] Config
- [ ] Helper mock + fixtures
- [ ] Spec S2 (tab-switch-stats)
- [ ] Spec S1 étendu (epc-bar-refreshes)
- [ ] Data-testid stratégiques
- [ ] Sanity check
- [ ] Doc
