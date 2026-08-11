# Fiche 07 — Frontend : gains rapides

Branche : `fix/frontend-quick-wins`

## Objectif

Solder les défauts frontend visibles à effort S : code mort massif, i18n
troué, actions destructives sans confirmation, recherche sans feedback.

## Tâches

### Code mort et hygiène

- [x] Supprimer `SearchHistoryPanel.svelte` (721 lignes, jamais monté — aucun
      import ; absorbé par l'onglet history de `SearchPanel:1616-1670`).
      Retirer `PANEL.SEARCH_HISTORY` (`stores/uiStore.js:46`), renommer
      `toggleSearchHistoryPanel` (`keyboardService.js:68-74`) en
      `focusSearchTab`. Vérifier `commandProcessor.js:84` (`history`/`hi`
      reste fonctionnel).
- [x] Supprimer les 3 commandes de migration mortes
      (`commandProcessor.js:113-142` : `migrate_from_1_0_to_1_1` etc.) et les
      3 méthodes Go exportées correspondantes (`db_session.go:379-491`),
      inatteignables (garde `!= "1.0.0"` + chaîne auto à l'ouverture) ;
      retirer leurs entrées de l'aide (`help/*.js:1172-1180`) et leurs clés
      i18n.
- [x] Purger les 21 symboles morts préfixés `_` (liste de l'audit :
      `CollectionPanel:192`, `MatchPanel:565`, `AnalysisPanel:143`,
      `StatusBar:159`, `StatsFilterBar:36`, `TournamentPanel:237-238`,
      Board ×13 — lire Board avant : vérifier que `_quadrantN`/`_labels` ne
      sont pas des rétentions volontaires pour two.js). Restreindre
      `varsIgnorePattern` aux destructurations, garder `argsIgnorePattern`.
      Voir Notes d'exécution : `StatsFilterBar`/`TournamentPanel` n'étaient
      pas morts, et la vraie liste des morts diffère du compte de l'audit.
- [x] Corriger le warning eslint `MatchInfoBar.svelte:100` (SvelteSet) et les
      warnings `state_referenced_locally` de `stats/charts/Histogram.svelte`.
- [x] `StatusBar.svelte:229-233` : listener `keydown` global ni démonté ni
      gardé (Ctrl-G en pleine saisie déclenche l'affichage) → le déplacer dans
      `keyboardService.js` avec les gardes standard.

### i18n

- [x] Solder les 19 clés `KNOWN_GAPS` de `i18nKeys.sync.test.js:28-46`
      (messages de barre d'état affichés en clé brute : `match.matchDeleted`…)
      dans les 9 locales ; vider la liste du test.
- [x] `MatchPanel.svelte:502` : `confirm()` anglais en dur → clé
      `match.confirmDelete` ×9 (modèle : `TournamentPanel:185`).
- [x] `MatchPanel.svelte:572-574` : `get(t)` non réactif dans `getPlayerName`
      → `$t`/`$derived`.
- [x] `App.svelte:469` : « Drop files to import » en dur → clé i18n ×9.
- [x] Purger les 19 clés orphelines identifiées (bloc `filterLibrary.*` ×14,
      `epc.epcDiff`…, liste complète dans l'audit) et ajouter un test
      « orphelines » avec liste d'exclusion des préfixes dynamiques
      (`search.filters.`, `epc.race.verdicts.`, …). Voir Notes d'exécution :
      40 clés retirées au total, pas 19.

### Sécurité des actions destructives

- [x] Confirmation avant : suppression de la base bearoff 1,2 Go
      (`ConfigModal.svelte:226-234`), suppression de position (touche
      `Delete`/commande `d`/bouton — `positionService.js:731-757`), retrait
      de positions d'une collection (`CollectionPanel.svelte:562-573`),
      `deleteDeck`/`resetDeck` Anki (`AnkiPanel.svelte:137,428`).
      Réutiliser `WarningModal` (déjà câblé `App.svelte:542`) en variante
      confirmation ; textes ×9 langues.
- [x] Documenter ces confirmations dans `doc/source/manuel.rst` si le manuel
      décrit les suppressions concernées.

### Feedback

- [x] État d'occupation de la recherche : message « Recherche… » dans la barre
      d'état + curseur d'attente posés **avant** l'`await` de
      `loadPositionsByFilters` (`positionService.js:349-534`).
- [x] `commandProcessor.js` : branche `else` finale → message
      `commands.unknown` (clé ×9) au lieu du no-op muet.
- [x] Fusionner `handleSearch`/`handleSubSearch` (`commandProcessor.js:155-234`,
      même corps à `slice`/`restrictToPositionIDs` près).

## Critères de fin

- `npm run lint` sans warning ; vitest vert ; `KNOWN_GAPS` vide ; nouveau
  test « clés orphelines » vert.
- Chaque action destructive listée passe par une confirmation traduite.

## Risques & garde-fous

- La suppression des méthodes Go bound exige de régénérer `frontend/wailsjs`
  (jamais à la main — relancer `wails dev`/build) ; coordonner avec un test
  de compilation.
- Les confirmations ne doivent pas casser les parcours clavier existants
  (Enter valide, Esc annule ; réutiliser le focus trap existant).

## Notes d'exécution

Réalisé dans `/home/unger/src/blunderDB-fiche07` (branche
`fix/frontend-quick-wins`), un commit par sous-tâche. `go build ./pkg/...
./internal/... ./cmd/...`, `go test ./pkg/blunderdb/database/...`,
`npm run lint` (0 warning), `npm run format:check` et `npm test -- --run`
(683 tests) verts avant chaque commit.

**Symboles `_` — le compte de l'audit ne correspond pas exactement au
réel.** `StatsFilterBar:36` (`_metric`) et `TournamentPanel:237-238`
(`_m`/`_s`) ne sont **pas** morts : ce sont des lectures de store à
l'intérieur d'un `$effect` dont le seul but est d'enregistrer une
dépendance réactive (le `$effect` doit re-tourner quand le store change,
même s'il n'utilise pas la valeur lue). Supprimer ces lignes aurait cassé
la sauvegarde de la métrique de stats et le refiltrage des matchs de
tournoi. Gardées, mais réécrites en `void expr;` (expression, pas
déclaration de variable) pour documenter l'intention et rester compatibles
avec le resserrement de `varsIgnorePattern`. Dans Board.svelte,
`createLabels()`/`createQuadrant()` dessinent directement sur la scène
two.js partagée (`two.makeGroup`/`two.makeText`) : les appels sont
retenus, seule l'affectation à une variable inutilisée est retirée.

Le resserrement de `varsIgnorePattern` (destructurations seulement) a fait
remonter des morts que l'audit initial n'avait pas listés :
`App.svelte._isResizing`, `SearchPanel._savedFilterName`/
`_savedFilterCommand`, `positionService._savedPositionBeforeCollection`/
`_savedPositionIndexBeforeCollection`/`_savedPositionsBeforeCollection`
(écrits, jamais lus), la prop `onToggleCommandMode`/`onToggleEPCMode` de
`Toolbar.svelte` (jamais consommée côté Toolbar — retirée aussi du call
site `App.svelte`), et `CommentPanel`'s prop `onClose` (jamais utilisée).
Tout a été vérifié individuellement avant suppression.

**Clés i18n orphelines — 40 retirées, pas 19.** Les 14 `filterLibrary.*` +
15 clés explicites de la fiche (29) sont confirmées mortes par grep. En
écrivant `i18nOrphanKeys.sync.test.js`, le test a aussi mis au jour 11
clés `searchHistory.*` (title, empty, colDate, colCommand, colActions,
addToLibraryTooltip, deleteTooltip, commandLabel, filterNameLabel,
filterNamePlaceholder, saveDialogTitle) propres à `SearchHistoryPanel.svelte`
supprimé plus tôt dans cette fiche — retirées dans le même commit. Un
premier jet du test, basé uniquement sur un scan d'appels littéraux
`$t('...')` (comme `i18nKeys.sync.test.js`), produisait des dizaines de
faux positifs : plusieurs composants (`TabbedPanel`, `tours.js`,
`matchTable.js`) référencent leurs clés indirectement via un objet de
config (`labelKey: 'tabbedPanel.search'`) puis un `$t(variable)` ailleurs.
Le test final cherche donc la clé comme chaîne littérale n'importe où
dans les sources, complété par la liste de préfixes dynamiques demandée.

**Bonus découvert en implémentant `match.confirmDelete` sur le modèle
`TournamentPanel:185`.** `tournament.confirmDelete` — le modèle cité par
la fiche — n'avait en réalité **jamais** eu de traduction dans aucune
locale : `i18nKeys.sync.test.js` ne reconnaissait que les appels
`$t`/`tr`/`tMsg`/`translate`, pas `get(t)('...')`. Corrigé (le test
reconnaît maintenant aussi `get(t)(...)`), et au passage 3 clés
`merge.errorLoad`/`errorSelect`/`errorCanonical` (`MergePlayersModal`,
même angle mort) comblées.

**Confirmations destructives.** `WarningModal.svelte` étendu d'un mode
`confirm` (deux boutons, Entrée=confirmer/Échap=annuler comme
`window.confirm()` natif, indépendamment du bouton qui a le focus).
Nouveau `services/confirmService.js` (`confirmAction()`/`resolveConfirm()`,
promesse-based) **décorrélé** de la pile `activeModal` exclusive : la
suppression de la base bearoff se déclenche depuis `ConfigModal`, et
réutiliser `activeModal` aurait remplacé ce modal par la confirmation au
lieu de se superposer — z-index dédié (1100) pour rester au-dessus quel
que soit l'ordre DOM.

**e2e non exécutés.** `npx playwright test` échoue dans cet environnement
(`chromium_headless_shell-1217` absent du cache Playwright local,
`npx playwright install` n'a pas pu télécharger — pas d'accès réseau
sortant apparent dans ce sandbox). Limitation d'environnement, sans
rapport avec les changements de cette fiche ; à relancer sur une machine
avec accès réseau ou cache Playwright à jour.

**Points à vérifier visuellement** (rendu non contrôlable depuis cet
environnement, seulement testé via vitest/eslint) :
- Le dialogue de confirmation (`WarningModal` mode `confirm`) : taille,
  alignement des boutons, contraste du bouton rouge destructif sur les
  9 langues (textes de longueurs très différentes, notamment allemand et
  finnois).
- Le focus trap et l'enchaînement clavier `Delete` → `Entrée` en conditions
  réelles (testé unitairement via `confirmService.test.js`, pas en
  intégration DOM).
- Les guillemets par langue (`« »` fr/el/ru, `„ "` de, `「」` ja) rendus
  correctement dans les messages avec `{name}`/`{player1}` etc.
