# Saisie d'un coup au plateau — plan de tranches (ADR-0086)

Une tranche = un ouvrier, un worktree, une branche. T1 d'abord ; T2 à T5 ensuite, indépendantes
entre elles (chacune ne touche que son mode et sa doc). T6 attend la réponse de l'utilisateur.
Chaque tranche livre sa documentation (`raccourcis.rst`, `manuel.rst` et leurs huit `.po`,
règles de `doc/CLAUDE.md`). Modèle : Opus pour T1 (règle silencieuse : un dé mal choisi reste
un coup légal), Sonnet pour T2-T5.

## T1 — La grammaire, pure, et son câblage commun

- Nouveau `frontend/src/services/boardMove.js` : `orderedDice`, `spentDice`, `usedDice`
  (avec dés gris d'un coup partiel forcé et d'une danse, ADR-0086 §5, et le demi-voile du
  double), `playClickedChecker` (y compris en coup libre : avance du premier dé non joué sans
  `playHop`), `diceClick(play, drawn)` → `validate | swap | null`, `boardRightClick(play)` →
  `reset | null`. Déplacés de `services/duelBoard.js`, qui les réexporte ou les importe.
- `frontend/src/utils/boardInteractions.js` : `quizClick` joue `playClickedChecker` au lieu de
  source/destination ; clic sur les dés (`hitTestSideControls`) quand un coup est armé →
  `diceClick` (la validation passe par un rappel `deps.validatePlay()` fourni par le mode) ;
  `onContextMenu` sur le damier, coup armé → reprise, menu seulement hors du cadre.
- `frontend/src/stores/quizPlayStore.js` : `swapped` dans l'état ; `quizPlayTargetsStore`
  retiré ; un magasin `quizPlayValidateStore` (rappel du mode armé) ou équivalent.
- `frontend/src/components/Board.svelte` : `diceUsed` dessiné pour tout coup armé ; dés dans
  l'ordre `swapped` hors Duel ; plus de cibles allumées (`playHighlights`) ; *Recommencer*
  retiré du menu.
- `frontend/src/utils/boardScene.js` : `drawDice` accepte un demi-voile (opacité par dé).
- Tests : `boardMove.test.js` (nouveau), `duelBoard.test.js`, `boardInteractions.test.js`,
  `boardScene.test.js`, `quizPlay.test.js`.
- Sans doc utilisateur propre : T1 ne change aucun mode tant qu'aucun ne fournit
  `validatePlay` — sauf le clic source/destination, qui bascule ici pour tous. **Donc T1
  livre aussi la doc commune** (`raccourcis.rst` lignes du menu du plateau, `manuel.rst` §menu
  du plateau, ~l. 123-135).

## T2 — Duel

- `frontend/src/services/duelBoard.js` : `boardPress`/`boardContext` sur la grammaire de T1 ;
  clic sur les dés pendant le coup intervertit les dés restants ; clic droit sans pas joué :
  plus d'interversion (ADR-0086 §8).
- `frontend/src/services/duelService.js` (`duelBoardContext`, `duelBoardPress`).
- Tests : `duelBoard.test.js`, `duelBoardService.test.js`, `DuelPanel.test.js`.
- Doc : `raccourcis.rst` « Duel au plateau » (l. 150-164), section Duel de `manuel.rst`, `.po`.

## T3 — Entraînement (décision de pions)

- `frontend/src/services/trainingTabService.js` : fournit `validatePlay` = `answerDecisionBoard` ;
  `playDecisionNotation` inchangé (notation tapée).
- `frontend/src/components/TrainingPanel.svelte` : bouton *Valider* et Entrée gardés ;
  *Recommencer* reste au panneau.
- Tests : tests de `TrainingPanel`, `frontend/tests/e2e/training-decision.spec.js`.
- Doc : `manuel.rst` (Entraînement, décision de pions), `raccourcis.rst` panneau Entraînement, `.po`.

## T4 — Anki (réponse au plateau)

- `frontend/src/services/ankiBoardAnswer.js` : `validatePlay` = `validateBoardAnswer(card)`.
- `frontend/src/components/AnkiPanel.svelte` : bouton *Valider* gardé.
- Tests : `ankiBoardAnswer.test.js`, `frontend/tests/e2e/anki-review-session.spec.js`.
- Doc : `manuel.rst` (Anki, réponse au plateau), `raccourcis.rst` « Panneau Anki », `.po`.

## T5 — Transcription

**Bloquée tant qu'un autre agent tient `TranscriptionPanel.svelte`.**

- `frontend/src/components/TranscriptionPanel.svelte` : supprimer l'`$effect` de départ
  automatique (`deducedDice` → `sendPlay`) ; fournir `validatePlay` (coup achevé ou libre →
  `sendPlay`/`commitFreePlay`) ; le premier chiffre sur un coup achevé valide (étendre la
  branche « coup libre + chiffre » au coup légal achevé) ; sans jet saisi, coup achevé →
  attendre la case du triangle.
- `frontend/src/services/transcriptionPlay.js` : `freeClick` remplacé par le clic de la
  grammaire en coup libre (`playClickedChecker` libre) ; `freeSelect` retiré.
- `frontend/src/services/transcriptionTheatre.js`, `TheatreMiniBoard.svelte` : cibles retirées
  du mini-plateau si elles y sont dessinées.
- Tests : `TranscriptionPanel.boardPlay.test.js`, `TranscriptionPanel.directPlay.test.js`,
  `transcriptionPlay.test.js`, e2e `transcription-budgets.spec.js` (budgets KLM à refaire).
- Doc : `manuel.rst` l. ~1974-2010, `raccourcis.rst` l. 384-389, `.po`.

## T6 — Analyse (en attente)

Aucun coup ne s'y saisit aujourd'hui (clic sans effet en mode normal). À ouvrir seulement si
l'utilisateur précise ce qu'un coup joué y ferait (sélectionner le candidat correspondant ?).
