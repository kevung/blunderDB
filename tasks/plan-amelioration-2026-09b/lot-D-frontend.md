<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot D — Frontend Svelte 5

État vérifié le 2026-09-02 : ESLint 0 erreur ; build 35 warnings du
compilateur (26 a11y), non bloquants ; chunk principal 1 759 772 o (gzip
422 kB) dont **1 139 819 o d'i18n des 8 langues non actives** ; vitest 94
fichiers / 987 tests, 52,4 % stmts / 54,2 % lignes ; Playwright 41 tests /
10 specs. L'invariant `.subscribe()` est respecté (5 occurrences justifiées) ;
ADR-0008 est appliquée (0 `font-size` absolue). `Modal.svelte`, `filterModel`,
`boardScene`, `PanelTable`, `positionList`, `EntityAutocomplete` sont bien
livrés.

D.1 à D.6 = **étape 1** (bugs, a11y bloquante) ; D.7 à D.15 = **étape 2**
(perf, design system, dette, tests).

---

## D.1 — Six bugs d'ergonomie à quelques lignes chacun [S] — bug/UX (#201)

- `SearchPanel.svelte:616-626` : `stopPropagation` sur toutes les touches
  quand un champ a le focus → Escape ne remonte jamais au handler global.
- `modeMachine.js:309` : `exitEditMode()` est `async` et appelé sans `await`
  → `enterEPCMode` photographie **toujours** le plateau vide, et `exitEPCMode`
  le restaure (`:366`).
- `panelLayoutStore.js:84-85` : `DEFAULT_PANEL_HEIGHT = 380`, `WIDTH = 520`
  « mirrors config.go » ; `config.go:119,123` dit 250 / 420 → saut de mise en
  page au premier lancement.
- `MatchInfoBar.svelte:120,150-152` : la barre s'insère sans `resize` → plateau
  23 px trop grand pendant toute une revue de match.
- `MatchPanel.svelte:576,982-986` : au premier clic `has-detail` change la
  largeur de la liste, la ligne bouge sous le curseur avant le second clic.
- `modeMachine.js:387` → `loadAllPositions()` → `positionService.js:302-303`
  repositionne sur la dernière position : la position quittée est perdue.
- [ ] Laisser passer `Escape` et `Tab` ; `await exitEditMode()` ; constantes
      alignées + test de synchro `panelDefaults.sync.test.js` (motif
      `fontScale.sync.test.js`) ; `$effect` sur `visible` → `resize` ; largeur
      du volet réservée (ou ouverture au `dblclick` seul) ; résoudre l'index de
      la position quittée avant de repositionner.
- [ ] Une spec Playwright par bug (les 10 specs existantes sont saines :
      0 `waitForTimeout`).

## D.2 — Huit raccourcis « Afficher/cacher » qui ne cachent jamais [S] — bug/doc (#202)

`positionService.js:930-954` : `toggleTab` fait `activeTabStore.set`, sans
bascule ; `raccourcis.rst:131-139` promet « Afficher/cacher » pour Ctrl-L,
Ctrl-P, Ctrl-K, Ctrl-B, Ctrl-Y, Ctrl-D, Ctrl-M, Ctrl-F.
- [ ] Implémenter la bascule (retour à l'onglet précédent, ou fermeture du
      panneau) ; ou renommer en `show*` et corriger la doc. Choix : bascule.
- [ ] `commandProcessor.js:79` : `startsWith('s')` attrape `sve` → exiger `s`
      seul ou `s `.
- [ ] Deux `100` (historique) → une constante ; commentaire `tabHandler.js:20`
      (onglet `log` inexistant).

## D.3 — Parseur de recherche unique côté JS [M] — bug (#203)

Aller = `commandProcessor.parseFilters` (`:199-346`), retour (rejeu depuis
l'historique ou la bibliothèque) = `searchFilterService.parseSearchCommand`
(`:221-300`). Trois filtres traversent l'aller et disparaissent au retour,
sans signalement : `xD` (dés exclus, `:206-211`), `id` (`:325-331`), `co`/`xco`
(`:224`). Exemple : `s D xD65` exclut les 6-5 ; double-clic sur la même ligne
d'historique → les 6-5 reviennent. `xD` et `id` n'ont pas de case à cocher
(`filterGroups`, `SearchPanel:275-287`).
- [ ] Une grammaire `parseSearchTokens(tokens | command) → SearchFilters`,
      appelée par les deux chemins et par `buildSearchFilterPayload` ; les deux
      suites existantes servent de filet croisé ; corpus
      `testdata/search_query_corpus.json` (partagé avec B.18).
- [ ] Cases à cocher `xD` et `id` (ou les documenter comme « ligne de
      commande seulement »).
- [ ] `commandVocabulary.sync.test.js` dans les deux sens.

## D.4 — Tab est confisqué globalement [M] — a11y bloquante (#204)

`keyboardService.js:294-296` : `Tab` sans Ctrl → `preventDefault` +
`activeTabStore.set('search')`, hors modales. La navigation clavier standard
n'existe pas dans l'application. Choix documenté (`raccourcis.rst:118`) mais
il neutralise le focus.
- [ ] Ne confisquer Tab que lorsque le focus est sur le plateau
      (`.scrollable-content`) ou déplacer sur un autre raccourci ; documenter.
- [ ] `PanelTable.svelte:169-170` : `tabindex="-1"` contredit le commentaire
      « reachable from the keyboard » → retirer ; `onclick` sur le bouton.
- [ ] `focusTrap.js` filtre les éléments invisibles (`offsetParent !== null`).
- [ ] `StatusBar.svelte:159` : `aria-live` restreint à `.info-message`.
- [ ] `ContextMenu.svelte:60` `aria-label="Actions"` → i18n (seul texte en dur).

## D.5 — Les 35 warnings du compilateur deviennent un plafond [S puis M] — a11y (#205)

Le job `frontend-lint` (`build.yml:440-444`) ne lance que ESLint et prettier ;
`npm run build` sort 35 warnings et termine en 0. Répartition des 26 a11y :
5 `role="region"` redondants, 5 `autofocus`, 5 `onclick` sans clavier, 2
`<label>` non associés, combobox sans `aria-controls` (`EntityAutocomplete:154`),
dialog sans `tabindex` (`SearchPanel:981`, à migrer sur `Modal.svelte`), 2
`<section>` porteurs d'écouteurs ; 3 graphiques figent leur légende
(`state_referenced_locally`, `BarChart:40`, `LineChart:41`, `ScatterChart:40`) ;
`SearchPanel:193` idem.
- [ ] Job build : échoue si le nombre de warnings dépasse le plafond courant
      (fichier `frontend/.svelte-warnings-budget`), plafond abaissé à chaque
      correction jusqu'à 0.
- [ ] Corriger par lots : graphiques (bug réel : légende masquée quand une 2ᵉ
      série apparaît), puis `region`/`autofocus`, puis clavier.
- [ ] `PanelTable` lignes : `role="row"`, `aria-selected`, tabindex roving ;
      `TabbedPanel` : `aria-controls` / `aria-labelledby` / `tabindex="0"` sur
      le tabpanel.
- [ ] `svelte/require-each-key` → `error` (57/59 déjà keyés) ;
      `infinite-reactive-loop` → `error`.

## D.6 — Un état « planté » du panneau Eval [S] — bug (#206)

`EPCPanel.svelte:149,156,158,206` : `evalSettled` n'est remis à `true` que
dans `applyEvalResult` ; si `EvaluatePositionImmediate` rejette ou si
`gammonnet-eval:error` arrive, seule `logger.error` (invisible en production)
est appelée. Le cas « refusé » est bien traité ; le cas « erreur » reste
bloqué sur « en attente ». C'est le résidu de la dette ADR-0017.
- [ ] Troisième état nommé `evalFailed`, affiché comme `data.error` l'est
      (`:414-417`), avec le message de l'erreur.
- [ ] Les 75 `catch` qui ne parlent qu'à la console (`logger.js:11,16`) :
      `logger.error` alimente aussi un journal consultable (I.31) et un toast
      quand l'action est utilisateur.

---

## D.7 — Charger uniquement la locale active [M] — perf, ×3 sur le bundle (#207)

`i18n/index.js:20-28` et `i18n/help/index.js:4-12` importent statiquement les
9 langues : 511 273 o de JSON + 628 546 o d'aide HTML = 65 % du chunk
principal. `driver.js` (104 kB) est aussi dans le chunk principal
(`tourService.js:1-2`) pour une visite guidée.
- [ ] `import.meta.glob('./locales/*.json')` + `await import()` de la locale
      active dans `initLanguage` ; `en` en statique comme repli ; même chose
      pour l'aide (chargée à l'ouverture de `HelpModal`).
- [ ] `await import('driver.js')` dans `startTour`.
- [ ] `manualChunks` dans `vite.config.js` ; le warning > 1000 kB disparaît.
- [ ] Mesure avant/après dans la PR (objectif : chunk principal < 600 kB brut).

## D.8 — Pagination et virtualisation [M] — perf (#208)

- `MatchPanel.svelte:775` : `indexOf` dans `{#each}` → O(n²), 250 000
  comparaisons par rendu sur un match de 500 coups.
- Transcript non virtualisé (`MatchPanel:748-800`) ; `GetAllMatches()` sans
  `MatchListOpts` (`:169`) ; `GetCollectionPositions` renvoie des positions
  complètes (`CollectionPanel:208,238`) ; résultats de recherche complets via
  IPC (`positionService.js:443,504`) alors que `positionList.js` ne transporte
  que des ids.
- `positionService.js:189,235` : `LoadAnalysis` puis `LoadComment` sérialisés
  + deep copy JSON ; molette sans debounce (`App.svelte:215-224`).
- `StatusBar.svelte:195-200` : trois parcours O(n) dans le template.
- [ ] Index global pré-calculé dans le `$derived` ; `Promise.all` + debounce
      60 ms ; trois `$derived` dans StatusBar. (S, immédiat.)
- [ ] La recherche renvoie des ids (B.10) et réutilise `positionList` ;
      `MatchListOpts` câblé ; collections paginées ; transcript par partie
      (`<details>`) ou virtualisé. (M, après B.10.)

## D.9 — Tokens de couleur, puis thème sombre [M puis M] — design system (#209)

`style.css` n'a que `--ui-scale` et 6 `--font-size-*` ; 108 couleurs hex
distinctes, top 10 = 342 occurrences ; trois bleus primaires concurrents
(`#1976d2` ×23, `#1a56c4` ×10, `#1a73e8` ×8) ; `#888` (3,54:1) et `#999`
(2,85:1) sur le texte le plus petit (WCAG AA = 4,5:1) ; trois composants hors
Nunito (`StatusBar:217`, `MatchInfoBar:156`, `EPCPanel:533`) et six piles
monospace ; 34 corps de règles CSS dupliqués ; 0 token d'espacement/rayon.
- [ ] ADR « une seule palette » sur le modèle d'ADR-0008 : `--color-text`,
      `--color-text-muted` (≥ 4,5:1), `--color-border`, `--color-surface`,
      `--color-surface-alt`, `--color-primary`, `--color-danger`,
      `--font-family-ui`, `--font-family-mono`, `--space-1..4`, `--radius`.
- [ ] Test garde (motif `fontScale.sync.test.js`) qui échoue sur tout hex
      dans un `<style>` hors `style.css`.
- [ ] Migration progressive ; `PickList.svelte` partagé et dialogue « Sauver
      le filtre » sur `Modal.svelte` au passage.
- [ ] Thème sombre = second jeu de tokens + `prefers-color-scheme` + réglage
      dans Configuration ; le plateau two.js lit ses couleurs des tokens
      (I.30).

## D.10 — Découpage des modules-dieux [M] — dette (#210)

`positionService.js` 1 004 l., 33 exports, 6 responsabilités ;
`importService.js` 1 003 l., 4 flux ; `clipboardService.js:441-495` embarque
un rasteriseur canvas ; 8 composants > 700 l. (`SearchPanel` 1 481,
`MatchPanel` 1 386, `TournamentPanel` 913, `AnkiPanel` 909, `CollectionPanel`
878, `ConfigModal` 771, `EPCPanel` 744, `ExportDatabaseModal` 706).
- [ ] `positionNavigation.js`, `xgid.js`, `tabToggles.js` ; `importSources.js`
      / `importIngest.js` ; `utils/canvasTable.js`.
- [ ] `MatchDetailPane.svelte` (transcript + métadonnées + stats,
      `MatchPanel:708-920`) ; `SearchHistoryTab.svelte`.
- [ ] `openPanels`/`tabHandler.js` supprimés : `PANEL.ANALYSIS`/`COMMENT` ne
      sont jamais ouverts, fermés 2× (`modeMachine.js:187,324`), trois fichiers
      commentent leur mort ; `importService.js:461,482,504,543,806` ouvre un
      panneau sans poser l'onglet (état incohérent).
- [ ] Prédicats « coup joué » : supprimer la copie d'`analysisRows.js:374,384`
      (signature différente de `playedMarks.js:33,58`) ; le test de parité
      compare enfin les deux côtés.
- [ ] Helper `onChange(getter, fn)` pour les quatre verrous `_prev*` ;
      `StatusBar:28` en `$derived` ; `AnalysisPanel:34` (mute `analysisStore`
      dans un `$effect`) en `$derived`.

## D.11 — `generateXGID` avec perte [S] — bug de partage (#211)

`positionService.js:97-109` : longueur de match = plus grand score away,
Crawford déduit de `score === 1`, Jacoby/cube max émis à 0. Un XGID copié
depuis blunderDB peut décrire une autre position que celle affichée.
- [ ] Encoder depuis `match_length`, le drapeau Crawford réel, Jacoby/beaver/
      cube max ; corpus `testdata/xgid_corpus.json` mis à jour consciemment
      (`xgidCanonical` ajouté, le test JS fantôme cité par
      `domain/xgid_contract_test.go` retrouve un vrai homologue).
- [ ] Idéalement : encodeur XGID **en Go** exposé au front (le Go a déjà
      `DecodeXGID`), un seul encodeur pour GUI/CLI/serveur.

## D.12 — Formats de date et de nombre liés à la langue [S/M] — i18n (#213)

`CommentPanel.svelte:177` code `'fr-FR'` en dur ; `SearchPanel:568` locale du
navigateur ; `MatchInfoBar:83` / `metadataStatus.js:34-35` `'sv-SE'` ;
`CollectionPanel:175` / `matchTable.js` formateurs maison ; 0
`Intl.NumberFormat` (`toFixed` partout) ; pas de pluriels
(`i18n/index.js:57-60`).
- [ ] `utils/format.js` (`formatDate`, `formatNumber`, `formatPercent`) dérivé
      du store `language` ; `Intl.PluralRules` minimal ; test par langue.
- [ ] Chaînes `config.bearoffIntro` (« panneau EPC ») et
      `config.gammonnetIntro` (« (ADR-0011) ») corrigées en 9 langues (H.6).

## D.13 — Tests frontend [M] — fiabilité (#214)

`databaseService.js` 7,4 % et 0 fichier de test ; `positionService` 31 %,
`importService` 40,9 %, `clipboardService` 40,2 %, `keyboardService` 42,9 % ;
11 modules à 0 % (`tourService`, `MatchInfoBar`, `MetadataPanel`, `ViewTabs`,
`StatsErrorsTab` 360 l., `StatsPlayersTab`, `StatsProgressionTab` 297 l., 4
graphiques) ; `CollectionPanel` (878 l.) et `TournamentPanel` (913 l.) sans
fichier de test ; 52/94 fichiers mockent `wailsjs` localement sans mock partagé.
- [ ] `src/__mocks__/wails.js` unique + test de synchro contre
      `frontend/wailsjs/go/**` (motif `commandVocabulary.sync.test.js`).
- [ ] Seuil non régressif `thresholds: { lines: 54 }` dans `vitest.config.js`.
- [ ] Tests par priorité : `databaseService`, `modeMachine` (les 6 globales),
      `CollectionPanel`, `TournamentPanel`, `StatsErrorsTab`.
- [ ] Playwright : `workers: process.env.CI ? 2 : undefined` ; échouer si
      `flaky > 0` (reporter JSON).
- [ ] `svelte-check` : `checkJs: true` est activé sans vérificateur → script
      `check` + job CI.

## D.14 — Petites ergonomies [S chacune] — UX (#215)

- Pas de `MinWidth`/`MinHeight` (`internal/gui/run.go:22-31`) → 900 × 600.
- Trois graphies de raccourcis (`Ctrl+Tab` / `Ctrl-C` / `CTRL-N`) → une.
- Réordonnancement des onglets non persisté (`TabbedPanel:140-144`) ; onglets
  masquables.
- `ViewTabs` occupe 26 px avec une seule vue (`:62,79-89`).
- Menu contextuel du plateau à un seul item (`Board.svelte:364-379`) : y
  mettre « copier image + analyse » (Ctrl-X ×2 indécouvrable), « miroir »,
  « nouvelle vue », « ajouter au paquet Anki ».
- Corbeille des positions (I.29) plutôt que 12 `confirmAction` irréversibles.

## D.15 — Dépendances et outillage front [S] — dette (#216)

`vite` 7 → 8, `@sveltejs/vite-plugin-svelte` 6 → 7, `jest-dom` 6 → 7,
`prettier-plugin-svelte` 3 → 4, `jsdom` 29 → 30, `driver.js` 1.4 → 1.8 ;
1 `high` transitif (`nanoid < 3.3.18`, `npm audit fix`).
- [ ] Montées une par une, Dependabot npm groupé en mensuel.
- [ ] `{@html}` dans `HelpModal.svelte:21,124` : échapper les valeurs
      interpolées ; ne désactiver la règle que sur les onglets statiques ;
      corpus `help/*.js` sorti du JS (H.7 le rend un artefact de build).

---

## Résumé du lot

| Fiche | Effort | Étape |
|---|---|---|
| D.1, D.2, D.6, D.11 | S | 1 |
| D.3, D.4, D.5 | M | 1 |
| D.7, D.8, D.9, D.10, D.13 | M | 2 |
| D.12, D.14, D.15 | S | 2 |

Ordre conseillé en étape 2 : D.7 (gain immédiat, isolé) → D.9 (tokens ; les
lots D.10 et I.30 en dépendent) → D.8 (après B.10) → D.10 → D.13 en continu.
