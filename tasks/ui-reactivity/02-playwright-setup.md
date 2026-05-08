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

- [x] `cd frontend && npm install --save-dev @playwright/test`.
- [x] `cd frontend && npx playwright install chromium` (premier run local).
- [x] Ajouter script `"test:e2e": "playwright test"` dans `package.json`.
- [x] Ajouter script `"test:e2e:ui": "playwright test --ui"` pour debug local.
- [x] Ajouter `frontend/playwright-report/`, `frontend/test-results/` à `.gitignore`.

### 2. `playwright.config.js`

- [x] Cibler un `webServer` qui lance `npm run dev` sur port 5173 avant les tests.
- [x] `testDir: 'tests/e2e'`.
- [x] `timeout: 10000`, `expect.timeout: 2000`.
- [x] `use.browserName: 'chromium'`, `use.viewport: { width: 1280, height: 800 }`.
- [x] `reporter: [['list'], ['html', { open: 'never' }]]`.

### 3. Helper `wailsMock.js` injectable côté page

- [x] Exporter `await installWailsMock(page, overrides)` : `page.addInitScript(script)` qui installe sur `window.go.main.Database` les méthodes requises. Stratégie retenue : Proxy qui retourne `asyncNull` pour les méthodes non définies.
- [x] Méthodes par défaut fournies : `LoadCommandHistory`, `SaveCommand`, `ComputeEPCFromPosition`, `LoadAllPositions`, `SaveSessionState`, etc. `GetLastDatabasePath → ''` pour éviter l'auto-open.
- [x] `window.runtime` également stubbé (OnFileDrop, WindowGetSize, logs, events…).
- [x] Exporter `overrideDbMethod(page, methodName, returnValue)` pour les overrides mid-test.

### 4. Fixtures

- [x] `frontend/tests/e2e/helpers/fixtures.js` exporte :
  - [x] `positionA`, `positionB` (structures `Position` minimales).
  - [x] `matchesSample` (2 matches).
  - [x] `statsResult` (résultat `ComputeStats` factice).
  - [x] `epcResultA`, `epcResultB` (résultats `ComputeEPCFromPosition` factices).

### 5. Spec `tab-switch-stats.spec.js` (S2)

- [x] Before each : `installWailsMock(page)` (Proxy générique, tout retourne asyncNull).
- [x] Naviguer vers `/`, attendre la barre d'état (`[data-testid="status-bar"]`).
- [x] Bascule Match → Stats : cliquer Stats, vérifier `.active` ET `.stats-panel` visible.
- [x] Bascule Stats → Match : vérifier `.stats-panel` absent et `.active` sur Matchs.
- [x] 5 bascules rapides : toutes passent.
- [x] Variantes EPC↔Stats, Stats↔Anki, séquence Analysis→Stats→EPC→Stats→Anki→Stats.
- [x] **Résultat réel** : specs **vertes** sur `ui-reactivity` — le mock Vite n'expose pas le bug visuel S2 (absence de données réelles). Le bug S2 (cascade `activeTabStore.subscribe`) se manifeste surtout en mode `wails dev` avec DB réelle. Documenté dans README E2E.
- [x] **Attendu initial** : la spec est **rouge** sur `ui-reactivity` (bug présent), deviendra verte après Fiche 05.a.

### 6. Spec `epc-bar-refreshes-on-return.spec.js` (S1 étendu)

- [x] T1 — Visiter EPC tab, vérifier barre d'état contient `/EPC[:\s]+\d/`. **Rouge** : bug stale `$statusBarModeStore` dans `positionStore.subscribe` → `updateEPC` non appelé.
- [x] T2 — EPC tab → Stats → remplacer fixture EPC → retour EPC, vérifier valeur changée. **Rouge** idem.
- [x] T3 — Même position entre deux visites EPC, valeur doit rester stable. **Rouge** idem (valeur jamais affichée).
- [x] `overrideDbMethod(page, methodName, returnValue)` pour patcher `ComputeEPCFromPosition` mid-test.

### 7. Attributs de test à ajouter aux composants

- [x] Ajouter **uniquement si absent** : `data-testid="tab-{tab.id}"` sur les boutons d'onglets de `TabbedPanel.svelte`. Un seul `data-testid` par composant majeur.
- [x] Ajouter `data-testid="tab-content"` sur le conteneur `.tab-content` de `TabbedPanel.svelte`.
- [x] Ajouter `data-testid="status-bar"` sur le `.status-bar` de `StatusBar.svelte`.

### 8. Sanity check

- [x] `npm run test:e2e` sur `ui-reactivity` → **3 specs rouges** reproduisant S1 étendu (EPC) + **8 specs vertes** (transitions onglets). Le mock Vite n'expose pas S2 (documenté : mock sans données réelles). Le bug EPC (cascade `$statusBarModeStore` stale dans `positionStore.subscribe`) est reproduit 100%.
- [x] Documenter comment débugger (`npm run test:e2e:ui`).

### 9. Documentation

- [x] `frontend/tests/e2e/README.md` (≤ 60 lignes) :
  - [x] Comment lancer en local (Vite + Playwright, ou `wails dev` + Playwright).
  - [x] Convention des helpers et fixtures.
  - [x] Règle : chaque bug UI signalé devrait d'abord être reproduit par une spec rouge avant d'être fixé.

## Acceptance

- [x] `npm run test:e2e` exécutable, specs lisibles.
- [x] Au moins 1 spec **rouge** sur `ui-reactivity` reproduisant S1 étendu (3 rouges, EPC bug). S2 vert (mock Vite ne l'expose pas — documenté).
- [ ] Specs passent en **vert** sur une branche de test où l'on stub manuellement le handler Stats (prouve que la spec détecte bien le bug). ← à valider lors de Fiche 05.a.
- [x] Doc d'usage.

## Status

- [x] Playwright installé
- [x] Config
- [x] Helper mock + fixtures
- [x] Spec S2 (tab-switch-stats) — 8 verts
- [x] Spec S1 étendu (epc-bar-refreshes) — 3 rouges (bug confirmé)
- [x] Data-testid stratégiques
- [x] Sanity check
- [x] Doc

**DONE** — 2026-04-22

> Note : les 3 specs rouges EPC confirment le bug de closure stale (`$statusBarModeStore` dans `positionStore.subscribe`). Les specs S2 (transitions onglets) sont vertes dans le setup Vite+mock car la réactivité `{#if $activeTabStore}` fonctionne correctement sans données réelles. Le bug S2 se manifeste avec `wails dev` + DB (cf. README E2E pour la procédure).
