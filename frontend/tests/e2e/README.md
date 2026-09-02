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

Un serveur Vite est démarré automatiquement sur `http://localhost:5173/`, qui
est aussi le `baseURL` de la configuration : les specs naviguent avec
`page.goto('/')`, jamais vers une URL absolue. C'est ce qui permet de pointer
toute la suite ailleurs quand le port 5173 est déjà pris — sinon Playwright
`reuseExistingServer` s'y attache et teste silencieusement une autre
application.
Si un serveur tourne déjà, Playwright le réutilise (`reuseExistingServer: true`
hors CI).

## Structure

```
tests/e2e/
├── helpers/
│   ├── wailsMock.js   – injecte window.go + window.runtime avant le boot
│   ├── fixtures.js    – positions, matches, stats, résultats EPC factices
│   └── showcase.js    – jeu « vitrine » de la capture d'écran (30 positions, match, analyse)
├── tab-switch-stats.spec.js          – S2 : transitions d'onglets Stats
├── epc-bar-refreshes-on-return.spec.js – S1 étendu : mise à jour EPC
├── search-flow.spec.js               – Recherche : filtres + structure, résultats, navigation
├── match-navigation.spec.js          – Match : ouverture, parcours des coups, sortie
├── import-position.spec.js           – Import : XGID collé, fichier via dialogue
├── screenshot.spec.js                – Capture documentaire (hors suite, voir plus bas)
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
    await page.goto('/');
});
```

Un second argument passe des constantes par méthode, une par espace de noms
(`database`, `app`, `config`, `runtime`). `openLibraryMock()` (fixtures.js)
fournit le jeu qui fait démarrer l'app sur une base ouverte de trois positions :

```js
await installWailsMock(page, openLibraryMock({ database: { GetAllMatches: [matchSample] } }));
```

Chaque appel de binding est journalisé ; `getWailsCalls(page, 'LoadPositionsByFilters')`
renvoie `{ ns, method, args }` pour vérifier ce que le frontend a demandé au backend.

### `overrideDbMethod(page, methodName, returnValue)`

Modifie dynamiquement une méthode Database **après** le chargement de la page.
`returnValue` doit être JSON-sérialisable.

```js
import { overrideDbMethod } from './helpers/wailsMock.js';
await overrideDbMethod(page, 'ComputeEPCFromPosition', epcResultB);
```

### `overrideDbMethodByArg(page, methodName, table, fallback?)`

Fait répondre une méthode Database selon son premier argument : `table`
associe à `String(arg)` la valeur à renvoyer, `fallback` (null par défaut)
couvre le reste. C'est ce qui donne à chaque position sa propre analyse.

```js
await overrideDbMethodByArg(page, 'LoadAnalysis', { 4033: analysisA, 4030: analysisB });
```

### `overrideDbMethodThen(page, methodName, returnValue, afterCall)`

Simule une mutation du backend : `methodName` répond `returnValue` et, dès son
premier appel, les méthodes de `afterCall` sont remplacées (une position
enregistrée apparaît ensuite dans `LoadAllPositions`).

## Panneaux ancrés et touches nues

Le panneau Match (onglet ouvert au chargement) et le panneau Recherche (tant
qu'un champ a le focus) gardent les touches nues pour eux : Tab, flèches, j/k
n'atteignent pas le dispatcher global. Dans une spec, ouvrir la recherche par
`Ctrl+F`, sortir d'un champ en cliquant un bouton, et parcourir la bibliothèque
depuis l'onglet Analyse.

## Conventions

- **Un bug = une spec rouge.** Chaque comportement cassé doit d'abord être
  reproduit par une spec rouge avant d'être corrigé.
- **data-testid stratégiques.** On n'en met que sur les composants majeurs :
  `[data-testid="tab-<id>"]`, `[data-testid="tab-content"]`,
  `[data-testid="status-bar"]`. Ne pas proliférer.
- **Timeout court.** `expect.timeout: 2000`. Si une transition prend > 2 s
  c'est un bug UI, pas un seuil à relever.

## Capture d'écran documentaire

`screenshot.spec.js` n'est pas un test : il produit
`doc/source/_static/screenshot.png` (l'image du README) depuis l'interface
réelle, sur le jeu de données de `helpers/showcase.js` — bibliothèque de 30
positions, match en revue sur une position de milieu de partie, onglet Analyse
avec la table des coups et le coup joué surligné. Anglais, 1280×960, thème et
disposition par défaut. Il est ignoré par la suite normale (`test.skip` sans
la variable) :

```bash
# Depuis frontend/
SCREENSHOT=1 npx playwright test screenshot
```

Le PNG est ensuite passé au premier optimiseur trouvé sur le PATH — `pngquant`,
`oxipng` ou `optipng` — et le script échoue au-delà de 400 Ko. Pour refaire la
capture après un changement d'interface : lancer la commande, regarder le PNG,
committer. Les deux derniers paragraphes de la spec disent pourquoi 960 px de
haut et pourquoi un `resize` est dispatché avant la capture.

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
