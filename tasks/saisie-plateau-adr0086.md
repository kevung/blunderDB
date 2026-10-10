# Saisie d'un coup au plateau — plan de tranches (ADR-0086)

Une tranche = un ouvrier, un worktree, une branche. T1 d'abord ; T2 à T5 ensuite, indépendantes
entre elles (chacune ne touche que son mode et sa doc). T6 (onglet d'évaluation) est indépendante de toutes.
Chaque tranche livre sa documentation (`raccourcis.rst`, `manuel.rst` et leurs huit `.po`,
règles de `doc/CLAUDE.md`). Modèle : Opus pour T1 (règle silencieuse : un dé mal choisi reste
un coup légal), Sonnet pour T2-T5.

## T1 — La grammaire, pure, et son câblage commun (faite)

Livrée derrière un interrupteur par mode, pour qu'aucun mode ne change avant sa tranche : un mode
passe à la grammaire en armant son coup par `armBoardMove(play, validate)` (`quizPlayStore.js`),
qui pose `quizPlayValidateStore`. Rappel posé, `boardInteractions` joue le clic de
`boardMove.playClickedChecker` (pression-relâché : clic sur place, glissé ailleurs via `dragStep`),
`diceClick` sur les dés, `boardRightClick` au clic droit du damier ; `Board.svelte` dessine les dés
dans l'ordre `swapped` avec `diceShade` (opacité par dé), n'allume plus de cible et retire
*Recommencer* du menu. Rappel nul : comportement inchangé. Le jet du coup est `play.rolled`,
sinon les dés de la position. En conséquence :
- `quizPlayTargetsStore`, `selectSource` et la branche source/destination de `quizClick` restent
  tant qu'un mode s'en sert ; la dernière tranche les retire.
- La documentation utilisateur (`raccourcis.rst`, `manuel.rst`, `.po`) n'a pas changé en T1 :
  chaque tranche décrit la grammaire pour son mode quand elle le bascule.

Plan d'origine :

- Nouveau `frontend/src/services/boardMove.js` : `orderedDice`, `spentDice`, `usedDice`
  (avec dés gris d'un coup partiel forcé et d'une danse, ADR-0086 §5, et le demi-voile du
  double), `playClickedChecker` (y compris en coup libre : avance du premier dé non joué sans
  `playHop`), `diceClick(play, drawn)` → `validate | swap | null`, `boardRightClick(play)` →
  `reset | null`. Déplacés de `services/duelBoard.js`, qui les réexporte ou les importe.
- `frontend/src/utils/boardInteractions.js` : `quizClick` joue `playClickedChecker` au lieu de
  source/destination ; clic sur les dés (`hitTestSideControls`) quand un coup est armé →
  `diceClick` (la validation passe par un rappel `deps.validatePlay()` fourni par le mode) ;
  `onContextMenu` sur le damier, coup armé avec au moins un pas → reprise ; sans pas joué,
  le menu comme aujourd'hui.
- `frontend/src/stores/quizPlayStore.js` : `swapped` dans l'état ; `quizPlayTargetsStore`
  retiré ; un magasin `quizPlayValidateStore` (rappel du mode armé) ou équivalent.
- `frontend/src/components/Board.svelte` : `diceUsed` dessiné pour tout coup armé ; dés dans
  l'ordre `swapped` hors Duel ; plus de cibles allumées (`playHighlights`) ; *Recommencer*
  retiré du menu.
- `frontend/src/utils/boardScene.js` : `drawDice` accepte un demi-voile (opacité par dé).
- Tests : `boardMove.test.js` (nouveau), `duelBoard.test.js`, `boardInteractions.test.js`,
  `boardScene.test.js`, `quizPlay.test.js`.
- Le clic source/destination bascule ici pour tous les modes : **T1 livre la doc commune**
  (`raccourcis.rst` l. 50-52, menu du plateau ; `manuel.rst` l. ~123-135 ; `.po`).

## T2 — Duel (faite)

Le Duel arme son coup par `armBoardMove(play, validateMove)` ; l'ordre des dés est celui du coup
(`play.swapped`), plus `duelBoardStore.swapped`. `usedDice` et `stepDistance` ne sont plus
exportés (`diceShade` les remplace). Le Duel garde ses propres gestes de plateau (lancer, videau,
invites) : `boardPress`/`boardContext` délèguent le coup à `diceClick`/`boardRightClick`.

Plan d'origine :


- `frontend/src/services/duelBoard.js` : `boardPress`/`boardContext` sur la grammaire de T1 ;
  clic sur les dés pendant le coup intervertit les dés restants ; clic droit sans pas joué :
  plus d'interversion (ADR-0086 §8).
- `frontend/src/services/duelService.js` (`duelBoardContext`, `duelBoardPress`).
- Tests : `duelBoard.test.js`, `duelBoardService.test.js`, `DuelPanel.test.js`.
- Doc : `raccourcis.rst` « Duel au plateau » (l. 150-164), section Duel de `manuel.rst`, `.po`.

## T3 — Entraînement (décision de pions) (faite)

- `frontend/src/services/trainingTabService.js` : fournit `validatePlay` = `answerDecisionBoard` ;
  `playDecisionNotation` inchangé (notation tapée).
- `frontend/src/components/TrainingPanel.svelte` : bouton *Valider*
  (`training-validate-move`) retiré, Entrée gardée ; *Recommencer* reste au panneau.
- Tests : tests de `TrainingPanel`, `frontend/tests/e2e/training-decision.spec.js`.
- Doc : `manuel.rst` (Entraînement, décision de pions), `raccourcis.rst` panneau Entraînement, `.po`.

## T4 — Anki (réponse au plateau)

- `frontend/src/services/ankiBoardAnswer.js` : `validatePlay` = `validateBoardAnswer(card)`.
- `frontend/src/components/AnkiPanel.svelte` : bouton *Valider* (`anki-board-validate`)
  retiré ; Entrée valide le coup achevé, comme le clic sur les dés (décision : à ajouter si le
  panneau ne la lie pas déjà).
- Tests : `ankiBoardAnswer.test.js`, `frontend/tests/e2e/anki-review-session.spec.js`.
- Doc : `manuel.rst` (Anki, réponse au plateau), `raccourcis.rst` « Panneau Anki », `.po`.

## T5 — Transcription (faite)

Le panneau arme le coup du seul jet saisi par `armBoardMove(play, validatePlay)` ; sans jet, rien
n'est armé. `validatePlay` enregistre le coup achevé ou libre (clic sur les dés, Entrée, premier
chiffre ou case du triangle du jet suivant). `ROLLS`, `compatibleRolls`, `choosableRolls`,
`deducedDice`, `freeClick`, `freeSelect` et la branche libre de `quizClick` sont retirés.
`quizPlayTargetsStore`, `selectSource` et la branche source/destination restent tant qu'Anki
(T4) n'a pas basculé : le dernier à fusionner les retire. Plan d'origine :

- `frontend/src/components/TranscriptionPanel.svelte` : supprimer l'`$effect` de départ
  automatique (`deducedDice` → `sendPlay`) ; fournir `validatePlay` (coup achevé ou libre →
  `sendPlay`/`commitFreePlay`) ; le premier chiffre sur un coup achevé valide (étendre la
  branche « coup libre + chiffre » au coup légal achevé) ; **sans jet saisi, aucun coup armé**
  (`boardPlayOpen` exige `boardRoll`) : plus de `loadBoardPlay` sur les 21 jets, plus de
  `rollsAllowed`, `pickDice` ne choisit plus parmi les jets d'un coup joué.
- `frontend/src/services/transcriptionPlay.js` : `freeClick` remplacé par le clic de la
  grammaire en coup libre (`playClickedChecker` libre) ; `freeSelect` retiré ;
  `ROLLS`, `newBoardPlay` multi-jets, `compatibleRolls`, `choosableRolls`, `deducedDice`
  retirés ou réduits à un jet (vérifier les appelants hors panneau avant de retirer).
- `frontend/src/services/transcriptionTheatre.js`, `TheatreMiniBoard.svelte` : cibles retirées
  du mini-plateau si elles y sont dessinées.
- Tests : `TranscriptionPanel.boardPlay.test.js`, `TranscriptionPanel.directPlay.test.js`,
  `transcriptionPlay.test.js`, e2e `transcription-budgets.spec.js` (budgets KLM à refaire).
- Doc : `manuel.rst` l. ~1974-2010 — le paragraphe « Le coup joué au plateau dispense de
  lire les dés » est réécrit : le jet d'abord (clavier ou triangle), puis le pion cliqué part
  du dé de gauche, les dés se grisent, un clic sur les dés ou le jet suivant enregistre ; le
  glissé hors des règles reste. `raccourcis.rst` l. 384-389 (la ligne « aucun dé saisi » est
  retirée), et leurs huit `.po`.

## T6 — « Évaluer dans un nouvel onglet » (ADR-0086 §10)

Analyse ne saisit aucun coup. Les variantes se jouent dans une vue neuve en Eval.

Mécanisme existant :
- Vues (onglets de position) : `frontend/src/stores/viewStore.js` — `addView()` copie la vue
  courante (position, analyse, liste partagée, onglet, mode) puis `restoreViewState` ;
  `switchTo`, `closeView` ; vues verrouillées pendant un Duel (`viewsLocked`). Le mode EVAL/EDIT
  d'une vue est restauré en NORMAL, l'effet d'onglet d'`App.svelte` le rétablit.
- Eval sur un plateau brouillon : `sendPositionToEval(position)`
  (`frontend/src/services/modeMachine.js` l. ~521, réexporté par `positionService.js`) — id 0,
  `evalSeed`, bascule sur l'onglet `eval`. Son contexte (`savedContext.beforeEval`,
  `lastEvalBoard`, `evalSeed`) est **global au module, pas propre à une vue**.
- Menu du plateau : `openContextMenu` de `frontend/src/components/Board.svelte`, qui porte déjà
  *Évaluer cette position*, *Évaluer la position miroir* et *Nouvelle vue*.

Travail :
- `Board.svelte` : entrée « Évaluer dans un nouvel onglet » (hors Duel) =
  `viewStore.addView()` puis `sendPositionToEval(getDisplayPosition())` dans la vue neuve.
  L'onglet de la vue neuve s'appelle « Variante de #n », n étant le numéro de la vue d'origine
  (clé i18n avec paramètre `n`).
- `viewStore.js` / `modeMachine.js` : la vue neuve part en Eval sans que la vue d'origine
  perde sa position, sa liste ni son analyse ; un aller-retour entre les deux vues rend à
  chacune son plateau (le contexte Eval global ne doit pas fuir d'une vue à l'autre — le
  porter dans la vue, ou le sauver/restaurer avec elle). Voir la mémoire « Plateaux
  brouillons : analysisStore périmé ».
- i18n : la clé dans `frontend/src/i18n/locales/*.json` (toutes les langues).
- Tests : `viewStore` (aller-retour), menu de `Board.svelte`, un e2e court.
- Doc : `manuel.rst` l. ~123-135 (menu du plateau), `raccourcis.rst` si une ligne cite le
  menu, `.po`.
- Modèle : Opus (un contexte global qui fuit d'une vue à l'autre ne se voit pas en rouge).
