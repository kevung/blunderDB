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

- [x] Coalescer : remplacer les 4 appels directs par `scheduleRedraw()`
      (flag dirty + `requestAnimationFrame` unique). L'ordre synchrone qui
      justifiait `.subscribe` (commentaire l.851-853) est préservé par
      construction : tous les stores ont mis à jour leur valeur quand le rAF
      s'exécute.
- [x] Réduire la surface `.subscribe` : regrouper les abonnements dans une
      unique fonction locale documentée (exception invariant justifiée en
      commentaire + message de commit), convertir en `$effect` ce qui peut
      l'être sans casser l'ordre (cas `positionStore:840` qui porte de la
      logique de reset).
- [x] `resizeBoard` via le même `scheduleRedraw`.
- [x] `drawMoveArrows` : ne redessiner que si nécessaire (au minimum passer
      par `scheduleRedraw` pour coalescer avec le reste).
- [x] `App.svelte:181` : `.subscribe` jamais désabonné → ajouter le cleanup.
- [x] Test de caractérisation : compteur d'appels de `drawBoard` (hook
      injectable ou spy) — une navigation de position = 1 redraw, un resize
      en rafale = 1 redraw par frame max.

## Critères de fin

- [x] Une navigation de position déclenche exactement un redraw (test).
- [x] vitest vert (768 tests, dont les 4 nouveaux de caractérisation) ; e2e
      Playwright non rejoués ici (pas d'environnement graphique dans ce
      worktree) — voir Notes d'exécution pour les vérifications visuelles
      restant à faire à la main.

## Risques & garde-fous

- Board n'a aucun test de rendu : rester chirurgical, ne pas réorganiser le
  dessin lui-même (c'est le backlog `boardRenderer`).
- Vérifier à la main la sélection de coup + flèches + cube offert après
  coalescing (les 3 chemins qui redessinaient « pour être sûrs »).

## Notes d'exécution

Exécutée dans `/home/unger/src/blunderDB-fiche12` (branche `perf/board-redraw`,
déjà créée). État de départ : fiches 07/08/09/11 mergées ; les numéros de
ligne du constat initial avaient dérivé (bloc des 4 `.subscribe()` en
`Board.svelte:826-855` avant la fiche, pas `:840-869`).

**Mécanique retenue** : un `scheduleRedraw()` unique (flag `redrawScheduled` +
`requestAnimationFrame`) remplace tous les appels directs à `drawBoard()`
déclenchés par un store. `resizeBoard()` garde la mesure du conteneur
(`clientWidth`/`clientHeight`, `two.width`/`two.renderer.setSize`)
synchrone — seule la repeinte lourde (`drawBoard()`, qui appelle déjà
`two.update()` en interne) passe par `scheduleRedraw()`, ce qui a permis de
supprimer l'appel `two.update()` redondant qui suivait `drawBoard()` dans
l'ancien `resizeBoard()`. `drawMoveArrows` n'est jamais appelée hors de
`drawBoard()` (elle dessine les flèches du coup sélectionné en fin de
fonction) ; la coalescer revenait donc à faire passer le déclencheur
`selectedMoveStore` (mis à jour par `AnalysisPanel.svelte` au survol/sélection
d'un coup) par `scheduleRedraw()`, ce qui est fait dans le point d'abonnement
unique.

**Vérification « aucun dessin synchrone après un store.set() »** (demandée
par la mission) : recherché tous les usages de `backgammon-board` /
`getContext` / lecture de `two`/canvas en dehors de `Board.svelte` —
seul `frontend/src/services/clipboardService.js` (`copyBoardImage`,
`copyBoardWithAnalysisImage`) lit le SVG du plateau, mais depuis une action
utilisateur indépendante (bouton « copier l'image »), jamais juste après un
`positionStore.set()`/`.update()` dans le même appel synchrone — le délai
d'un `requestAnimationFrame` (~16 ms) ne peut pas être observé par ce chemin.
Aucun autre appelant externe de `drawBoard()` (le composant l'exporte mais
rien ne l'invoque via `bind:this` ailleurs dans le code actuel). Les specs
Playwright existantes (`frontend/tests/e2e/*.spec.js`) ne touchent pas au
rendu du plateau.

**`.subscribe()` regroupés, pas éliminés** : les 4 déclencheurs de redraw
(`positionStore`, `selectedMoveStore`, `analysisStore`,
`searchOfferedCubeStore`) sont maintenant un seul point d'abonnement
documenté, `subscribeBoardRedrawTriggers()`, au lieu de 4 blocs séparés avec
4 variables `unsubscribe*`. La logique métier portée par le `.subscribe` de
`positionStore` (reset de `selectedMoveStore` uniquement sur un changement
réel d'`id`, pas à chaque tick du store) est restée dans ce subscriber plutôt
que d'être convertie en `$effect` — analyse : avec le redraw désormais
différé au rAF, l'ordre `$effect` (microtâche, avant paint) vs
`scheduleRedraw()` (rAF, après) serait en théorie sûr, mais Board.svelte n'a
aucun test de rendu pour attraper une régression d'ordre subtile ; gardé
l'ordre `.subscribe()` synchrone, sans ambiguïté, et documenté ce choix en
commentaire (exception à l'invariant Svelte 5 justifiée, comme demandé par
CLAUDE.md et par la fiche elle-même qui autorise explicitness ce
compromis : « converti-la en `$effect` si possible sans casser l'ordre, sinon
garde-la dans le subscriber unique »).

**Nettoyage à la destruction** : un redraw peut être en attente (rAF déjà
demandé) au moment du démontage du composant ; `onDestroy` annule maintenant
ce rAF (`cancelAnimationFrame(redrawFrameId)`) en plus de désabonner les 4
stores, pour ne jamais dessiner sur un `two`/`canvas` détaché. C'est un
ajout par rapport à la fiche mais découle directement du mécanisme de
coalescing (le risque n'existait pas avant, `drawBoard()` étant toujours
appelée de façon synchrone).

**`App.svelte:170`** (`positionsStore.subscribe`, ligne 181 dans l'énoncé de
la fiche avant dérive des numéros) : l'abonnement était licite (le
commentaire d'exception existant tient toujours — `positions` n'est jamais lu
dans le template) mais la fonction de désabonnement retournée n'était jamais
capturée. Capturée dans `unsubscribePositions` et appelée dans le `onDestroy`
déjà présent en bas du fichier.

**Test de caractérisation** (`frontend/src/__tests__/Board.redraw.test.js`,
4 tests) : Board.svelte n'ayant aucun test de rendu (two.js a besoin d'un
vrai canvas/SVG), `two.js` est mocké entièrement (`vi.mock('two.js', …)`)
avec une classe factice qui trace les appels à `clear()` — chaque
`drawBoard()` commence par `two.clear()`, donc compter les appels à `clear()`
compte les redraws sans instrumenter Board.svelte. `requestAnimationFrame` /
`cancelAnimationFrame` sont stubbés par une file `Map` gérée à la main
(`flushFrame()`), plutôt que de dépendre du rAF réel de jsdom, pour un test
déterministe. Les 4 scénarios :
1. navigation de position touchant `positionStore` + `selectedMoveStore` +
   `analysisStore` dans le même tick → exactement 1 frame demandée, 1 seul
   `clear()` après le flush ;
2. survol/sélection successifs de plusieurs coups (le chemin AnalysisPanel →
   `selectedMoveStore`) → coalescés en 1 seul redraw ;
3. rafale de 3 événements `resize` → 1 seule frame demandée, 1 seul redraw ;
4. démontage avec une frame en attente → `cancelAnimationFrame` empêche tout
   redraw après coup (ce test aurait échoué avec un stub `cancelAnimationFrame`
   no-op — il exerce réellement le nettoyage `onDestroy` ajouté ci-dessus).

**Tâche annexe (constat fiche 09)** : `AnalysisPanel.svelte:419` portait
`role="dialog" aria-modal="true"` alors que c'est un panneau docké (comme
`MatchPanel`/`TournamentPanel`/`CollectionPanel`, corrigés dans la fiche 09,
commit `2986517a`). Remplacé par `role="region" aria-label={$t('analysis.panelLabel')}`
— la clé de traduction existait déjà dans les 9 langues (utilisée pour le
même `aria-label`, seul le rôle changeait). N'a pas porté le reste de ce
commit (le `panelKeyGuard` partagé pour Match/Tournament/Collection) : hors
périmètre de la mission, qui ne demandait que le changement de rôle ARIA.

**Validations** : `npm run lint`, `npm run format:check`,
`npm test -- --run` (49 fichiers, 768 tests, dont les 4 nouveaux) verts avant
chaque commit. Go non touché (mission purement frontend), donc pas de
`go test`/`go vet` relancés.

**À vérifier à l'écran** (impossible dans ce worktree sans affichage) :
- Navigation entre positions : le plateau doit toujours se redessiner
  immédiatement à l'œil (le délai d'une frame, ~16 ms, doit être
  imperceptible) ; pas de scintillement.
- Survol/sélection d'un coup dans le panneau d'Analyse : les flèches
  apparaissent/disparaissent sans lag ni redessin dupliqué.
- Cube offert (décision take/pass) : bascule immédiate entre position
  centrée et position du propriétaire au bord du plateau, y compris en
  changeant rapidement de position pendant qu'une décision de cube est
  affichée.
- Redimensionnement de la fenêtre (drag continu du bord, ou bascule
  panneau latéral/dock qui déclenche un `resize` synthétique dans
  `App.svelte`) : le plateau doit suivre sans à-coups perceptibles ni retard
  visible malgré le throttling à 1 redraw/frame.
- Changement de palette de couleurs (`ConfigModal` → onglet Couleurs) :
  toujours appliqué immédiatement (passe aussi par `scheduleRedraw()`
  désormais).
- `AnalysisPanel` : vérifier au lecteur d'écran ou dans les devtools
  d'accessibilité que le panneau n'est plus annoncé comme une boîte de
  dialogue modale (le reste de l'app doit rester atteignable pendant qu'il
  est ouvert, comme les autres panneaux dockés).
