# Tests E2E — Playwright

Tests de bout en bout pour vérifier la réactivité de l'UI blunderDB dans un
navigateur Chromium headless. Stratégie : Vite dev server + mock de `window.go`
et `window.runtime` (pas de binaire Wails requis).

## Lancer les tests

```bash
# Depuis frontend/
npm run test:e2e           # headless, reporter « list »
npm run test:e2e:ui        # mode interactif (Playwright UI) — debug local
```

Un serveur Vite est démarré automatiquement sur `http://localhost:5173/`.
Si un serveur tourne déjà, Playwright le réutilise (`reuseExistingServer: true`
hors CI).

## Structure

```
tests/e2e/
├── helpers/
│   ├── wailsMock.js   – injecte window.go + window.runtime avant le boot
│   └── fixtures.js    – positions, matches, stats, résultats EPC factices
├── tab-switch-stats.spec.js          – S2 : transitions d'onglets Stats
├── epc-bar-refreshes-on-return.spec.js – S1 étendu : mise à jour EPC
└── README.md (ce fichier)
```

## Helpers

### `installWailsMock(page, opts?)`

À appeler dans `test.beforeEach` **avant** `page.goto()`. Installe via
`page.addInitScript` un Proxy qui intercepte tous les appels à
`window.go.main.{Database,Config,App}` et à `window.runtime`.

Les méthodes non surchargées retournent `Promise.resolve(null)`.

```js
import { installWailsMock } from './helpers/wailsMock.js';

test.beforeEach(async ({ page }) => {
  await installWailsMock(page);
  await page.goto('http://localhost:5173/');
});
```

### `overrideDbMethod(page, methodName, returnValue)`

Modifie dynamiquement une méthode Database **après** le chargement de la page.
`returnValue` doit être JSON-sérialisable.

```js
import { overrideDbMethod } from './helpers/wailsMock.js';
await overrideDbMethod(page, 'ComputeEPCFromPosition', epcResultB);
```

## Conventions

- **Un bug = une spec rouge.** Chaque comportement cassé doit d'abord être
  reproduit par une spec rouge avant d'être corrigé.
- **data-testid stratégiques.** On n'en met que sur les composants majeurs :
  `[data-testid="tab-<id>"]`, `[data-testid="tab-content"]`,
  `[data-testid="status-bar"]`. Ne pas proliférer.
- **Timeout court.** `expect.timeout: 2000`. Si une transition prend > 2 s
  c'est un bug UI, pas un seuil à relever.

## Mode wails dev (debug local avancé)

Pour tester avec le vrai binaire Wails (bindings réels, pas de mock) :
```bash
# Terminal 1 — depuis la racine du projet
wails dev
# Terminal 2 — depuis frontend/
BASE_URL=http://localhost:34115 npx playwright test
```
Adapter `playwright.config.js` en remplaçant `webServer` par l'URL fixe et en
désactivant le mock Wails.
