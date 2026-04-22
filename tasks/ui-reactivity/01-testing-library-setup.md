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

- [ ] `cd frontend && npm install --save-dev @testing-library/svelte @testing-library/jest-dom @testing-library/user-event`.
- [ ] Vérifier dans `package.json` que les 3 paquets apparaissent en `devDependencies` et que la version de `@testing-library/svelte` est ≥ 5.

### 2. Setup file

- [ ] Créer `frontend/src/test-setup.js` avec :
  ```js
  import '@testing-library/jest-dom/vitest';
  ```
- [ ] Dans `frontend/vitest.config.js`, ajouter `setupFiles: ['./src/test-setup.js']` dans l'objet `test`.
- [ ] Vérifier que `include` capture bien `src/__tests__/**/*.test.js` (actuellement `src/**/*.{test,spec}.{js,ts}` — OK).

### 3. Helper de mock Wails pour composants

- [ ] Dans `frontend/src/__tests__/helpers/wailsMock.js` (nouveau dossier `helpers/`) :
  - [ ] Export `installWailsMock(overrides = {})` qui installe sur `global.window.go.main.Database` un set de méthodes asynchrones par défaut (`LoadCommandHistory → []`, `SaveCommand → undefined`, etc.), extensibles par `overrides`.
  - [ ] Export `uninstallWailsMock()` qui nettoie `window.go`.
  - [ ] **Alternative plus simple** si Vitest gère mal les window-level mocks : utiliser `vi.mock('../../wailsjs/go/main/Database.js', () => ({ ... }))` directement dans chaque test. Choisir selon ce qui passe.

### 4. Test canari `StatusBar.reactivity.test.js`

- [ ] Créer `frontend/src/__tests__/StatusBar.reactivity.test.js`.
- [ ] Mock des modules Wails avant l'import du composant :
  ```js
  vi.mock('../../wailsjs/go/main/Database.js', () => ({
      LoadCommandHistory: vi.fn(() => Promise.resolve([])),
      SaveCommand: vi.fn(),
  }));
  ```
- [ ] Test 1 — *rendu initial* : monter `StatusBar`, vérifier que le texte initial de `statusBarTextStore` est présent dans le DOM (`screen.getByText(...)` ou `.info-message` query).
- [ ] Test 2 — *réactivité du texte* : muter `statusBarTextStore.set('Hello world')`, `await tick()` (ou `await screen.findByText('Hello world')`), vérifier présence.
- [ ] Test 3 — *réactivité du compteur position* : muter `positionsStore.set([{...}, {...}, {...}])` et `currentPositionIndexStore.set(1)`, vérifier que `.position-info` affiche `2 / 3`.
- [ ] Test 4 — *latence* : mesurer `performance.now()` avant/après mutation+flush, échouer si > 50 ms (seuil généreux pour CI lent ; ajuster si flaky — envisager 150 ms).
- [ ] Test 5 — *canari de régression* : après la mutation, changer **une seconde fois** le store ; le DOM doit à nouveau refléter. Ce test attrape les patterns où la réactivité ne fonctionne que pour le premier changement (typique de closures stales).

### 5. Documentation

- [ ] `frontend/src/__tests__/README.md` — doc courte (≤ 80 lignes) :
  - Comment lancer `npm test` et `npm test:watch`.
  - Pattern « mount + mutate store + assert DOM » avec un exemple minimal.
  - Conventions de mock Wails (`vi.mock` versus helper).
  - Règle : tout nouveau composant introduit dans le scope doit avoir au moins un test de réactivité (store → DOM).

### 6. Sanity check

- [ ] `cd frontend && npm test` doit être vert.
- [ ] Débrancher temporairement une souscription dans `StatusBar.svelte` (par exemple, ne plus lire `$statusBarTextStore` dans le template) → le test canari doit passer rouge. Rebrancher, vérifier vert. Cette étape vaut plus que mille docs.

## Acceptance

- [ ] 3 devDependencies ajoutées, `package.json` à jour.
- [ ] `npm test` vert avec 5 assertions canari sur StatusBar.
- [ ] Mutation manuelle du composant fait échouer le canari (mutation testing manuel).
- [ ] README de tests mis à jour avec le pattern.

## Status

- [ ] Dépendances installées
- [ ] Setup file en place
- [ ] Helper Wails
- [ ] Tests canari écrits (1–5)
- [ ] Doc pattern
- [ ] Sanity check validé
