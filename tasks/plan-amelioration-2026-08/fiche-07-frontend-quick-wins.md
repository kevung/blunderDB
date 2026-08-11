# Fiche 07 — Frontend : gains rapides

Branche : `fix/frontend-quick-wins`

## Objectif

Solder les défauts frontend visibles à effort S : code mort massif, i18n
troué, actions destructives sans confirmation, recherche sans feedback.

## Tâches

### Code mort et hygiène

- [ ] Supprimer `SearchHistoryPanel.svelte` (721 lignes, jamais monté — aucun
      import ; absorbé par l'onglet history de `SearchPanel:1616-1670`).
      Retirer `PANEL.SEARCH_HISTORY` (`stores/uiStore.js:46`), renommer
      `toggleSearchHistoryPanel` (`keyboardService.js:68-74`) en
      `focusSearchTab`. Vérifier `commandProcessor.js:84` (`history`/`hi`
      reste fonctionnel).
- [ ] Supprimer les 3 commandes de migration mortes
      (`commandProcessor.js:113-142` : `migrate_from_1_0_to_1_1` etc.) et les
      3 méthodes Go exportées correspondantes (`db_session.go:379-491`),
      inatteignables (garde `!= "1.0.0"` + chaîne auto à l'ouverture) ;
      retirer leurs entrées de l'aide (`help/*.js:1172-1180`) et leurs clés
      i18n.
- [ ] Purger les 21 symboles morts préfixés `_` (liste de l'audit :
      `CollectionPanel:192`, `MatchPanel:565`, `AnalysisPanel:143`,
      `StatusBar:159`, `StatsFilterBar:36`, `TournamentPanel:237-238`,
      Board ×13 — lire Board avant : vérifier que `_quadrantN`/`_labels` ne
      sont pas des rétentions volontaires pour two.js). Restreindre
      `varsIgnorePattern` aux destructurations, garder `argsIgnorePattern`.
- [ ] Corriger le warning eslint `MatchInfoBar.svelte:100` (SvelteSet) et les
      warnings `state_referenced_locally` de `stats/charts/Histogram.svelte`.
- [ ] `StatusBar.svelte:229-233` : listener `keydown` global ni démonté ni
      gardé (Ctrl-G en pleine saisie déclenche l'affichage) → le déplacer dans
      `keyboardService.js` avec les gardes standard.

### i18n

- [ ] Solder les 19 clés `KNOWN_GAPS` de `i18nKeys.sync.test.js:28-46`
      (messages de barre d'état affichés en clé brute : `match.matchDeleted`…)
      dans les 9 locales ; vider la liste du test.
- [ ] `MatchPanel.svelte:502` : `confirm()` anglais en dur → clé
      `match.confirmDelete` ×9 (modèle : `TournamentPanel:185`).
- [ ] `MatchPanel.svelte:572-574` : `get(t)` non réactif dans `getPlayerName`
      → `$t`/`$derived`.
- [ ] `App.svelte:469` : « Drop files to import » en dur → clé i18n ×9.
- [ ] Purger les 19 clés orphelines identifiées (bloc `filterLibrary.*` ×14,
      `epc.epcDiff`…, liste complète dans l'audit) et ajouter un test
      « orphelines » avec liste d'exclusion des préfixes dynamiques
      (`search.filters.`, `epc.race.verdicts.`, …).

### Sécurité des actions destructives

- [ ] Confirmation avant : suppression de la base bearoff 1,2 Go
      (`ConfigModal.svelte:226-234`), suppression de position (touche
      `Delete`/commande `d`/bouton — `positionService.js:731-757`), retrait
      de positions d'une collection (`CollectionPanel.svelte:562-573`),
      `deleteDeck`/`resetDeck` Anki (`AnkiPanel.svelte:137,428`).
      Réutiliser `WarningModal` (déjà câblé `App.svelte:542`) en variante
      confirmation ; textes ×9 langues.
- [ ] Documenter ces confirmations dans `doc/source/manuel.rst` si le manuel
      décrit les suppressions concernées.

### Feedback

- [ ] État d'occupation de la recherche : message « Recherche… » dans la barre
      d'état + curseur d'attente posés **avant** l'`await` de
      `loadPositionsByFilters` (`positionService.js:349-534`).
- [ ] `commandProcessor.js` : branche `else` finale → message
      `commands.unknown` (clé ×9) au lieu du no-op muet.
- [ ] Fusionner `handleSearch`/`handleSubSearch` (`commandProcessor.js:155-234`,
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
