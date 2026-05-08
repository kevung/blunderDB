# 01 — Setup `@testing-library/svelte` + test canari StatusBar

**Goal :** Pouvoir monter un composant Svelte 5 dans Vitest, déclencher des mutations de store, vérifier le DOM. Produire un test canari sur `StatusBar` qui échoue si la réactivité est cassée.

**Depends on :** 00.

**Impact :** Base de tests unitaires DOM pour les Fiches 05.b (MatchPanel), 05.c (StatsPanel), 05.d (StatusBar). Extension de la suite Vitest existante.

## Context

- Vitest déjà configuré (`frontend/vitest.config.js` avec jsdom, globals activés).
- Tests existants dans `frontend/src/__tests__/` portent sur logique métier (stores, commandProcessor) — aucun ne monte de composant Svelte.
- Svelte 5 : `@testing-library/svelte` ≥ 5.0 requis (support officiel runes).
- Le composant cible `StatusBar.svelte` lit plusieurs stores : `statusBarTextStore`, `currentPositionIndexStore`, `commandTextStore`, `showCommandInputStore`, `commandHistoryStore`, `positionsStore`, `matchContextStore`. Il appelle aussi `LoadCommandHistory`, `SaveCommand` via Wails — à mocker.

## Files touched

- **Edit:** `frontend/package.json` — ajouter 3 devDependencies.
- **New:** `frontend/src/test-setup.js` — import jest-dom.
- **Edit:** `frontend/vitest.config.js` — `setupFiles`, ajuster `include` pour `src/__tests__/**`.
- **New:** `frontend/src/__tests__/StatusBar.reactivity.test.js` — canari.
- **New ou Edit:** `frontend/src/__tests__/README.md` — documenter patterns de test composants + mocks Wails.

## Tasks

### 1. Installer les dépendances

- [x] `cd frontend && npm install --save-dev @testing-library/svelte @testing-library/jest-dom @testing-library/user-event`.
- [x] Vérifier dans `package.json` que les 3 paquets apparaissent en `devDependencies` et que la version de `@testing-library/svelte` est ≥ 5.

### 2. Setup file

- [x] Créer `frontend/src/test-setup.js` avec :
  ```js
  import '@testing-library/jest-dom/vitest';
  ```
- [x] Dans `frontend/vitest.config.js`, ajouter `setupFiles: ['./src/test-setup.js']` dans l'objet `test`.
- [x] Vérifier que `include` capture bien `src/__tests__/**/*.test.js` (actuellement `src/**/*.{test,spec}.{js,ts}` — OK).

> **Note :** Ajout également de `resolve.conditions: ['browser']` dans la config Vite pour que Svelte 5 se résolve sur le build navigateur (sans ça, Vitest+jsdom chargeait `svelte/src/index-server.js` et levait `lifecycle_function_unavailable`).

### 3. Helper de mock Wails pour composants

- [x] Dans `frontend/src/__tests__/helpers/wailsMock.js` (nouveau dossier `helpers/`) :
  - [x] Export `installWailsMock(overrides = {})` qui installe sur `global.window.go.main.Database` un set de méthodes asynchrones par défaut (`LoadCommandHistory → []`, `SaveCommand → undefined`, etc.), extensibles par `overrides`.
  - [x] Export `uninstallWailsMock()` qui nettoie `window.go`.
  - [x] **Alternative retenue :** `vi.mock('../../wailsjs/go/main/Database.js', ...)` directement dans le test (plus fiable avec Vitest).

### 4. Test canari `StatusBar.reactivity.test.js`

- [x] Créer `frontend/src/__tests__/StatusBar.reactivity.test.js`.
- [x] Mock des modules Wails avant l'import du composant :
  ```js
  vi.mock('../../wailsjs/go/main/Database.js', () => ({
      LoadCommandHistory: vi.fn(() => Promise.resolve([])),
      SaveCommand: vi.fn(),
  }));
  ```
- [x] Test 1 — *rendu initial* : monter `StatusBar`, vérifier que le texte initial de `statusBarTextStore` est présent dans le DOM.
- [x] Test 2 — *réactivité du texte* : muter `statusBarTextStore.set('Hello world')`, `await tick()`, vérifier présence.
- [x] Test 3 — *réactivité du compteur position* : muter `positionsStore.set([...])` + `currentPositionIndexStore.set(1)`, vérifier `.position-info` = `2 / 3`.
- [x] Test 4 — *latence* : seuil retenu 150 ms (plus robuste en CI).
- [x] Test 5 — *canari de régression* : trois mutations successives, toutes reflétées.

### 5. Documentation

- [x] `frontend/src/__tests__/README.md` — doc courte :
  - [x] Comment lancer `npm test` et `npm test:watch`.
  - [x] Pattern « mount + mutate store + assert DOM » avec un exemple minimal.
  - [x] Conventions de mock Wails (`vi.mock` versus helper).
  - [x] Règle : tout nouveau composant introduit dans le scope doit avoir au moins un test de réactivité.

### 6. Sanity check

- [x] `cd frontend && npm test` — 290/290 vert.
- [x] Débranchement temporaire de `$statusBarTextStore` dans le template → 4 tests canaris passent rouge. Rebranché → tout vert.

## Acceptance

- [x] 3 devDependencies ajoutées, `package.json` à jour.
- [x] `npm test` vert avec 5 assertions canari sur StatusBar (290 tests total).
- [x] Mutation manuelle du composant fait échouer le canari.
- [x] README de tests mis à jour avec le pattern.

## Status

- [x] Dépendances installées
- [x] Setup file en place
- [x] Helper Wails
- [x] Tests canari écrits (1–5)
- [x] Doc pattern
- [x] Sanity check validé

**DONE** — 2026-04-22
