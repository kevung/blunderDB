# ADR-0052 — Le coup se joue directement au plateau, et se tape dans sa cellule

Statut : acceptée.
Voir aussi : ADR-0044, ADR-0048 décision 7, ADR-0049 règle 1.

## Contexte

Le même clic sur le damier jouait un pion sans dés et filtrait la liste avec dés — deux sens
selon un état que l'œil, sur la vidéo, ne relit pas. Le coup illégal demandait un volet, une
bascule et un bouton de confirmation loin de la cellule écrite.

## Décision

1. **Le plateau joue aussi quand le jet est saisi**, en bout de document comme sur une Action
   relue : coups offerts = `LegalMoves` du jet saisi (`rolled`) ; clic source/destination ou
   glissé, contraints aux coups légaux.
2. **Chaque pas joué réduit la liste des candidats** aux coups qui le contiennent (multiplicités
   comprises, ordre indifférent : `containsSteps`) ; le premier restant est présélectionné. Il
   n'y a pas de filtre par point.
3. **Un coup légal achevé part aussitôt** (`enter_die` ×2, `enter_play`, `validate`, dés dans
   l'ordre tapé) ; sur une Action relue, il la remplace.
4. **Glisser = libre, sans bouton, et seulement le jet saisi.** Un glissé qu'aucun coup légal
   n'offre pose le pion où il est lâché (`dragStep`) ; le coup devient libre (`free`), la suite
   aussi. Un coup libre ne part jamais seul : Entrée (ou le chiffre du jet suivant) l'enregistre
   avec `BoardAfter`. Retour arrière défait le dernier pas ; défaire le seul pas illégal rend la
   liste. La liste d'un coup libre est remplacée par « Coup hors des règles — Entrée
   l'enregistre ». Sans jet saisi, le glissé reste contraint : on n'écrit pas un jet que
   personne n'a vu.
5. **Le coup se tape dans sa cellule** : double-clic sur une cellule coup / danse / coup non
   consigné du Transcript → champ pré-rempli de la notation. On n'y tape que le coup ; Entrée
   l'écrit sur cette Action (un coup illégal est accepté et marqué), un texte sans pas laisse
   le champ ouvert, Échap ferme sans écrire (`escapeService`, capture). La cellule en
   pointillés de l'Entry s'ouvre de même, ses deux dés saisis.
6. **Pas de volet `✎`**, ni bascule « déplacement libre », ni bouton « Ce plateau est le coup
   joué », ni champ de notation dans la palette.

## Conséquences

- Aucun changement moteur : `enter_play` avec `BoardAfter` accepte un coup illégal et le
  jette quand un coup légal atteint ce plateau (`transcript.validate`).
- Un coup lointain au plateau coûte ≈ 4,0 s (KLM) contre 3,6 s au clavier seul : le gain est
  l'absence de mode, pas le temps.
- `selectAction` ne relance pas un chemin déjà en route vers la même cellule (double-clic) ;
  le panneau ne reprend pas le focus à un champ qui l'a.
- Écartés : garder le filtre (deux sens pour un clic) ; un coup libre qui part seul (rien ne
  dit qu'il est fini) ; le glissé libre sans jet (des dés inventés).

## Garde

`frontend/src/__tests__/TranscriptionPanel.directPlay.test.js`,
`TranscriptionPanel.boardPlay.test.js`, `transcriptionPlay.test.js`,
`frontend/tests/e2e/transcription-budgets.spec.js`.
