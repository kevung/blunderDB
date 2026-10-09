# ADR-0086 — Un coup se saisit au plateau par une grammaire unique

Statut : acceptée. Révise ADR-0052 §3 et §4 (clic), ADR-0072 (clic droit du Duel).
Voir aussi : ADR-0040 (quiz au plateau), ADR-0044, ADR-0052, ADR-0072.

## Contexte

Quatre modes acceptent un coup de pions au plateau, par deux grammaires :

| Mode | Armement | Clic sur un pion | Clic sur les dés | Clic droit sur le damier | Validation |
|---|---|---|---|---|---|
| Duel | `duelBoard.boardPress` | le pion part du dé de gauche, sinon de droite (`playClickedChecker`) | rien joué : intervertit ; coup achevé : valide | pas joué : reprend tout ; sinon intervertit | clic sur les dés, Entrée, Espace |
| Entraînement (décision de pions) | `trainingTabService` → `quizPlayStore` | choisit une source, puis un clic sur la destination (cibles allumées) | rien | menu du plateau, *Recommencer* en tête | bouton *Valider*, Entrée |
| Anki (réponse au plateau) | `ankiBoardAnswer.armBoardAnswer` | idem | rien | idem | bouton *Valider* |
| Transcription, aucun dé saisi | `TranscriptionPanel.loadBoardPlay` (union des 21 jets) | source puis destination ; dés déduits des pas | rien | idem | départ automatique quand un seul jet reste (ADR-0052 §3) |
| Transcription, jet saisi | idem, `rolled` | source puis destination | rien | idem | départ automatique (§3) ; coup libre : Entrée ou chiffre (§4) |

Le glissé existe partout (contraint ; libre en Transcription, jet saisi). Le menu du plateau
porte déjà *Nouvelle vue* (`viewStore.addView`, copie de la vue) et *Évaluer cette position*
(`sendPositionToEval`, la vue courante passe en Eval sur un plateau brouillon). Le reste du
programme — Analyse, Eval, édition — n'accepte aucun coup : en Édition et en Eval le clic droit
pose un pion adverse, ailleurs il ouvre le menu du plateau. Les dés ne sont grisés qu'en Duel
(`usedDice`, dessin `diceUsed`). Le même joueur apprend donc deux gestes pour la même chose,
et un coup achevé part seul en Transcription quand rien ne l'annonce.

## Décision

1. **Une grammaire, dans tout mode qui attend un coup de pions** — Duel, Entraînement, Anki,
   Transcription, et tout mode à venir. Elle est écrite une fois, dans un réducteur pur que les
   modes appellent (`services/boardMove.js`, issu de `duelBoard.js`) ; `quizPlay.js` reste le
   juge de la légalité (`playHop`, alimenté par `App.LegalMoves`).
2. **Clic sur un pion du camp au trait : il bouge aussitôt** du premier dé non joué dans
   l'ordre affiché (gauche, puis droite). Si ce dé ne le déplace pas légalement, le suivant
   est essayé ; si aucun, rien ne se passe, sans message. Pas de sélection, pas de cible
   allumée, pas de flèche, pas de clic sur l'arrivée. Pion sur la barre : il entre. Sortie :
   un pion sort quand le dé l'amène au-delà du plateau et que la règle le permet (`playHop`
   tranche le dé plus fort et le « jouer le plus possible »). Un pion adverse, un point vide,
   le plateau de sortie : rien. Deux dés sur le même pion = deux clics, le second sur le point
   où il est arrivé.
3. **Clic sur les dés** : coup achevé → **valide** ; sinon → **intervertit** l'ordre des dés
   restant à jouer (avant ou pendant le coup). Un pas déjà joué garde son dé. Sur un double,
   ou s'il ne reste qu'un dé, l'interversion est sans effet. L'interversion ne réécrit jamais
   le jet enregistré : un jet est une paire, l'ordre n'est que celui du prochain clic.
4. **Clic droit sur le damier : reprend tous les pas du coup en cours tant qu'au moins un pas
   est joué** ; sans pas joué, il ouvre le menu du plateau, comme hors de tout coup. Les dés
   grisés disent lequel des deux arrivera. Hors du cadre, le menu s'ouvre toujours.
   *Recommencer* quitte le menu du plateau : le clic droit le fait.
5. **Les dés se grisent à mesure qu'ils sont joués.** Jet simple : chaque dé se grise quand
   son pas est joué. Double : les deux dés dessinés valent deux pas chacun ; le dé de gauche
   se voile à moitié au 1er pas et se grise au 2e, celui de droite de même aux 3e et 4e.
   **Coup partiel forcé** (la règle n'autorise pas tous les dés) : quand le coup est achevé,
   les dés non joués se grisent aussi — tous les dés gris disent toujours « un clic valide ».
   **Aucun coup légal** (danse) : les dés sont gris d'emblée.
6. **Validation** : clic sur les dés quand le coup est achevé (`completedPlay`). Les
   équivalents clavier restent : Entrée (Entraînement, Anki, Transcription, Duel), Espace
   (Duel). En Anki, Entrée valide le coup achevé comme le clic sur les dés. Les boutons
   *Valider* d'Entraînement et d'Anki sont retirés : les dés en tiennent lieu. **Plus aucun
   coup ne part seul.**
7. **Transcription.** Le coup est validé par un clic sur les dés, ou par la saisie du jet
   suivant (premier chiffre) quand il est achevé. Un chiffre tapé sur un coup inachevé garde
   son sens clavier actuel : il valide le candidat présélectionné (ADR-0052 §2), qui contient
   les pas joués. De ADR-0052 :
   - **§3 (départ automatique) est abrogé.** Un coup légal achevé attend son clic ou son chiffre.
   - **§4 (glissé libre) reste** : jet saisi, un glissé qu'aucun coup légal n'offre pose le
     pion où il est lâché et le coup devient libre. En coup libre, le **clic** suit la même
     grammaire (le pion avance du premier dé non joué), sans contrainte ; il ne choisit plus de
     source. Le coup libre se valide comme un autre (clic sur les dés, chiffre, Entrée).
   - **§5 (cellule) reste** tel quel : le double-clic sur une cellule n'est pas un geste du plateau.
   - **§1–§2 restent** : le jet saisi borne les coups offerts ; chaque pas réduit la liste.
   - **Les dés sont obligatoires.** Sans jet saisi, le plateau n'arme aucun coup : un clic sur
     un pion ne fait rien, un glissé non plus. La déduction des dés par les pas (union des 21
     jets, `deducedDice`, `choosableRolls`, cases du triangle restreintes) disparaît, glissé
     compris : garder le glissé sans jet garderait la déduction, donc une seconde grammaire.
     Le triangle redevient la seule saisie souris du jet.
8. **Duel.** Le clic droit sur le damier sans pas joué n'intervertit plus les dés (§4 : le
   clic sur les dés le fait). Le reste de ADR-0072 est déjà cette grammaire.
9. **Hors de la grammaire** : Édition et Eval (le clic droit y pose un pion, les dés s'y
   règlent au clic), le videau (Duel : propose un double ; Transcription : demande `d`),
   le double-clic hors du cadre (Pile).
10. **Analyse : aucune saisie de coup.** On y lit une position et son analyse ; jouer une
    variante se fait ailleurs. Le menu du plateau gagne **« Évaluer dans un nouvel onglet »** :
    une vue neuve (`viewStore.addView`) reçoit la position affichée en Eval (plateau
    brouillon, `sendPositionToEval`), où l'on pose librement des variantes. Son onglet
    s'appelle **« Variante de #n »**, n étant le numéro de la vue d'origine ; la vue d'origine
    garde sa position, sa liste et son analyse. *Évaluer cette position*, qui remplace la vue
    courante, reste à côté. Absent pendant un Duel (les vues y sont verrouillées).

## Conséquences

- Les modes basculent un par un : un mode passe à la grammaire en armant son coup avec son
  rappel de validation (`armBoardMove`, `quizPlayValidateStore`). Sans rappel, le plateau garde
  la saisie source puis destination ; un coup désarmé efface le rappel.

- `quizPlay.selectSource` et `quizPlayTargetsStore` (cibles allumées) n'ont plus d'appelant au
  plateau ; `playHop` reste l'unique point d'entrée d'un pas contraint.
- L'état du coup porte l'ordre des dés (`swapped`) hors du Duel aussi ; `diceUsed` se dessine
  pour tout coup armé, pas seulement en Duel.
- Transcription : un coup lointain coûte un clic de plus (la validation) ; c'est le prix d'un
  coup qui ne part jamais sans qu'on le dise. La note KLM d'ADR-0052 est à refaire.
- Pendant un coup, le menu du plateau ne s'ouvre sur le damier qu'avant le premier pas : la
  documentation (`raccourcis.rst`, `manuel.rst`) change avec chaque tranche.
- La Transcription perd « le coup joué au plateau dispense de lire les dés » : tout coup au
  plateau commence par le jet, au clavier ou au triangle.
- Le contexte Eval de `modeMachine` (`beforeEval`, `lastEvalBoard`, `evalSeed`) est global,
  pas propre à une vue : la vue ouverte en Eval doit, en revenant à la vue d'origine puis à
  elle, retrouver sa variante et non le plateau d'une autre vue.
- Écartés : garder source/destination pour le coup libre (deux grammaires) ; le clic droit qui
  n'ouvre jamais le menu sur le damier (le menu perdrait sa place habituelle hors des coups) ;
  le glissé sans jet qui déduit les dés ; quatre dés pour un double
  (le dessin des dés change pour un seul cas) ; un clic sans jet qui devine un dé.

## Garde

À écrire avec les tranches : `frontend/src/__tests__/boardMove.test.js` (grammaire pure),
`duelBoard.test.js`, `boardInteractions*.test.js`, les tests de panneau d'Entraînement,
d'Anki et de Transcription, un test de `viewStore` pour l'onglet d'évaluation, `frontend/tests/e2e/transcription-budgets.spec.js`.
