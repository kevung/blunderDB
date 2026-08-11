# Fiche 12 — Performance frontend : redraws du plateau

Branche : `perf/board-redraw`

## Objectif

Supprimer les reconstructions multiples et non coalescées du plateau
(two.js) sans refonte du renderer (la refonte est au backlog).

## Constats

- `Board.svelte:840-869` : 4 souscriptions (`positionStore`,
  `selectedMoveStore`, `analysisStore`, `searchOfferedCubeStore`) appellent
  chacune `drawBoard()` — naviguer d'une position déclenche 3-4
  reconstructions complètes (~100 nœuds SVG détruits/recréés à chaque fois,
  `two.clear()` ligne 945).
- `resizeBoard()` (`:469-487`) appelle `drawBoard()` à chaque événement
  `resize`, sans rAF ni debounce.
- `drawMoveArrows` (`:301`, fin `:486-487`) reconstruit tout le plateau au
  survol d'un coup dans le panneau d'analyse.
- Les `.subscribe()` non justifiés (violation invariant Svelte 5) sont
  traités ici puisqu'on touche ce bloc : un seul point d'abonnement documenté.

## Tâches

- [ ] Coalescer : remplacer les 4 appels directs par `scheduleRedraw()`
      (flag dirty + `requestAnimationFrame` unique). L'ordre synchrone qui
      justifiait `.subscribe` (commentaire l.851-853) est préservé par
      construction : tous les stores ont mis à jour leur valeur quand le rAF
      s'exécute.
- [ ] Réduire la surface `.subscribe` : regrouper les abonnements dans une
      unique fonction locale documentée (exception invariant justifiée en
      commentaire + message de commit), convertir en `$effect` ce qui peut
      l'être sans casser l'ordre (cas `positionStore:840` qui porte de la
      logique de reset).
- [ ] `resizeBoard` via le même `scheduleRedraw`.
- [ ] `drawMoveArrows` : ne redessiner que si nécessaire (au minimum passer
      par `scheduleRedraw` pour coalescer avec le reste).
- [ ] `App.svelte:181` : `.subscribe` jamais désabonné → ajouter le cleanup.
- [ ] Test de caractérisation : compteur d'appels de `drawBoard` (hook
      injectable ou spy) — une navigation de position = 1 redraw, un resize
      en rafale = 1 redraw par frame max.

## Critères de fin

- Une navigation de position déclenche exactement un redraw (test).
- Aucun changement visuel ; vitest + e2e verts.

## Risques & garde-fous

- Board n'a aucun test de rendu : rester chirurgical, ne pas réorganiser le
  dessin lui-même (c'est le backlog `boardRenderer`).
- Vérifier à la main la sélection de coup + flèches + cube offert après
  coalescing (les 3 chemins qui redessinaient « pour être sûrs »).
