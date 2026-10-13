# ADR-0091 — La transcription a trois modes : Saisie, Revue, Édition

Statut : remplacée par ADR-0092 (les blocages de la Revue, vécus à l'usage).
Remplace : ADR-0048 décision 1 (le chiffre « commence un jet là où le Cursor est ») ; ADR-0051
décision 1 ; la règle « le Cursor revient à sa place antérieure après
validation » d'une correction en place (fonctionnel.md §2).
Voir aussi : ADR-0049, ADR-0052, ADR-0086.

## Contexte

Le sens de la touche chiffrée tenait à un état visible mais implicite : la cellule encadrée.
En bout de document le chiffre validait, sur une Action relue il recommençait son jet, sur la
dernière Action il faisait l'un puis l'autre (ADR-0051). Une correction validée renvoyait le
curseur au `Return`, en pratique la fin du match. À l'usage, on sautait à la dernière décision
après une correction, on hésitait entre retaper les dés et valider par Entrée, et en voulant
saisir un jet neuf on modifiait la décision relue.

## Décision

1. **Trois modes, affichés en permanence** sur la barre de saisie avec l'effet d'Entrée :
   - **Saisie** — le curseur est au bout du document ou sur une insertion ;
   - **Revue** — le curseur est sur une décision existante ;
   - **Édition coup N** — la décision au curseur a été rouverte.
   Le panneau dérive le mode (`transcriptionMode(annotated, editingAt)`) ; la machine à
   touches le reçoit en contexte et ne connaît toujours pas le document.
2. **En Revue, la saisie de dés ne modifie jamais une décision** : chiffres, `f`, `j`/`k`,
   Entrée, Retour arrière et Échap sont pris sans effet, et le triangle des jets est retiré.
   Le plateau ne joue pas en Revue. Les gestes de correction (`i`, `a`, `x`, `s`), de videau
   et de déplacement restent actifs.
3. **On entre en Édition explicitement** : `e` (qui ouvre aussi le champ du coup de la
   cellule, ADR-0052), un double-clic sur la cellule, un clic sur une ligne de l'analyse.
   Un pas du curseur, une validation, une insertion, une suppression, une annulation la
   referment.
4. **La validation automatique n'agit qu'en Saisie.** En Édition, le chiffre recommence le
   jet sur place, même après un coup achevé au plateau, et seule une validation explicite
   écrit : Entrée, ou le clic sur les dés d'un coup achevé (grammaire d'ADR-0086), la
   dernière Action comprise.
5. **Valider une correction place le curseur sur la décision suivante**, chargée en Revue,
   jamais en fin de match ; après la dernière Action, c'est le bout du document. Entrée sur
   une décision rouverte et inchangée n'écrit rien mais avance de même. Le moteur ne garde
   plus de place de retour (`Return`). `Fin` ramène à la Saisie.

## Conséquences

Corriger une décision relue coûte une touche de plus (`e`, ou `e` puis Échap pour garder le
clavier hors du champ du coup) ; les budgets d'ux.md §4.3 suivent
(`transcription-budgets.spec.js`). En contrepartie, une frappe de dés en Revue ne peut plus
écraser une décision.

## Garde

`frontend/src/__tests__/transcriptionKeys.correction.test.js` (ADR-0091),
`frontend/src/__tests__/TranscriptionPanel.correction.test.js` (Saisie, Revue, Édition),
`frontend/tests/e2e/transcription-budgets.spec.js`, `pkg/blunderdb/transcript/inplace_test.go`.
