# ADR-0092 — La transcription ne bloque aucune frappe ; un indicateur dit son effet

Statut : acceptée (décision de l'utilisateur).
Remplace : ADR-0091 (les trois modes Saisie, Revue, Édition).
Voir aussi : ADR-0044, ADR-0052, ADR-0082, ADR-0086.

## Contexte

ADR-0091 rendait la Revue inerte : sur une décision existante, chiffres, `j`/`k`, flèches,
Entrée et plateau étaient avalés tant que `e` ne l'avait pas rouverte. À l'usage, le curseur
se retrouvait sur une décision existante sans que l'utilisateur l'ait voulu (vidéo suivie,
pas en arrière), et la transcription semblait cassée : plus de choix des coups du moteur,
plus de coup au plateau, flèches figées. La pastille de mode, absente du théâtre, ne levait
pas l'ambiguïté. En Saisie, la ligne présélectionnée n'était plus dessinée.

## Décision

1. **Aucune frappe n'est avalée.** Partout, `j`/`k` et ↑/↓ choisissent une ligne de
   l'analyse, le clic la choisit, le double-clic l'écrit, le plateau joue, les chiffres tapent
   les dés. Toute modification d'une décision rejoue les suivantes.
2. **Le chiffre dépend de la place du curseur.** En bout de document (ou sur une insertion),
   sur un jet saisi, il valide la ligne présélectionnée (ligne 0 par défaut, ou celle choisie,
   ou le coup joué au plateau) puis ouvre le jet suivant. Sur une décision existante, il retape
   son jet sur place ; Entrée valide.
3. **Un indicateur permanent dit l'effet de la frappe**, sur la barre du panneau et sur celle
   du théâtre : « Saisie » ou « Coup N », puis « Entrée : valider <coup> », puis l'effet des
   chiffres (« valider et jet suivant », « jet », « retaper les dés du N »).
4. **En Saisie, la ligne présélectionnée est dessinée** : c'est le coup qu'Entrée ou le jet
   suivant enregistrera. Sur une décision existante, seule la ligne désignée (survol, `j`/`k`,
   clic) se dessine, à la place du coup écrit.
5. **`e` ouvre le champ « dés: coup »** de la cellule (« 62: 24/18 13/11 ») : une seule
   validation corrige jet et coup, puis le curseur passe à la décision suivante. Sans
   « dés: », les dés de la cellule restent ; le même jet dans l'autre ordre n'est pas une
   correction. Des dés illisibles ou un coup sans pas laissent le champ ouvert et sont
   signalés ; un coup illégal pour ces dés est enregistré et marqué (ADR-0044), jamais
   remplacé par un coup deviné.
6. **Valider une correction place le curseur sur la décision suivante**, jamais en fin de
   match (repris d'ADR-0091 décision 5). `Fin` ramène au bout du document.

## Conséquences

Une frappe sur une décision existante peut la modifier sans geste préalable : l'indicateur
le dit et `Ctrl+Z` la défait. Il n'y a plus d'état « rouvert » à tenir dans le panneau.

## Garde

`frontend/src/__tests__/transcriptionKeys.correction.test.js`,
`frontend/src/__tests__/TranscriptionPanel.correction.test.js`,
`frontend/src/__tests__/TranscriptionPanel.arrows.test.js`,
`frontend/src/__tests__/TranscriptionPanel.directPlay.test.js`,
`frontend/tests/e2e/transcription-saisie.spec.js`.
