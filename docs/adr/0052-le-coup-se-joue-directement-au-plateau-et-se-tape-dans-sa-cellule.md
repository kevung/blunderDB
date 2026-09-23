# ADR-0052 — Le coup se joue directement au plateau, et se tape dans sa cellule

- **Statut** : accepté — décidé le 2026-09-24, après usage.
- **Amende** : T2.2 (le filtre par point de départ), T2.4 (le coup illégal : la bascule
  « déplacement libre » et le champ de notation) et ADR-0048 décision 7 (la palette
  « déplie ce qui sauve » : le secours `✎` replié à la place du triangle).
- **Applique** : ADR-0044 (rien n'est refusé, tout est marqué), T2.3 (le réducteur du coup
  joué au plateau, `quizPlay.js`, réutilisé tel quel) et ADR-0049 (la saisie en cours se
  dessine à sa place dans le Transcript).

## Le problème

Le panneau offrait trois façons de désigner un coup, et chacune avait son mode.

1. **Dés non saisis**, le plateau jouait le coup et en déduisait les dés (T2.3).
2. **Dés saisis**, le même clic sur un point du damier ne jouait plus rien : il FILTRAIT la
   liste des candidats (T2.2), avec sa puce `✕` pour le lever. Le geste qui déplaçait un pion
   une seconde plus tôt en cachait maintenant des lignes, selon un état — les dés sont-ils
   tapés ? — que l'œil, occupé par la vidéo, ne relit pas.
3. **Le coup illégal** demandait d'ouvrir le volet `✎`, d'enfoncer « Déplacement libre », de
   déplacer les pions, puis de cliquer « Ce plateau est le coup joué » — ou de taper la
   notation dans un champ du même volet, loin de la cellule qu'elle allait écrire.

À l'usage, le volet et sa bascule ont été jugés peu ergonomiques : trois cibles et un mode à
enfoncer pour un geste qui, à la table, est le même que tous les autres — prendre un pion et
le poser. Et le filtre coûtait un clic qui ne faisait pas avancer le coup.

## La décision

Le coup se joue **directement**, sans changer de vue, avec le moins de gestes possible.

1. **Le plateau joue aussi quand le jet est saisi** — en bout de document comme sur une Action
   relue, dont `settleCursor` charge les dés. Les coups offerts sont ceux de CE jet
   (`LegalMoves` du jet saisi), et l'état du coup porte le jet (`rolled`). Clic source puis
   destination, ou glissé, contraints aux coups légaux comme sans dés.
2. **Chaque pas joué réduit la liste des candidats** aux coups qui contiennent les pas joués,
   multiplicités comprises et dans n'importe quel ordre — la règle d'`alivePlays`, exposée
   par `containsSteps`. Le premier candidat restant est présélectionné. **C'est ce qui
   remplace le filtre par point** : le pas réduit la liste ET avance le coup. Le filtre, son
   magasin, sa puce et `transcriptionFilter.js` sont supprimés.
3. **Un coup légal achevé part aussitôt** : un seul jet est en jeu, donc `deducedDice` le
   rend, et l'Action est envoyée — `enter_die`, `enter_die`, `enter_play`, `validate`, les
   dés dans l'ordre où ils ont été tapés. Sur une Action relue, ces quatre gestes la
   REMPLACENT (`enter_die` sur une Entry pleine recommence le jet).
4. **Glisser = libre, sans bouton.** Le jet saisi, et seulement lui, un glissé qu'aucun coup
   légal n'offre pose le pion là où il est lâché (`dragStep`), y compris depuis un point qui
   n'est pas une source légale mais porte un pion du camp au trait. Dès ce pas, le coup est
   libre (`free`), et la suite l'est aussi, au clic comme au glissé. Un coup libre ne part
   jamais seul : **Entrée** l'enregistre avec les dés saisis, les pas et le plateau obtenu
   (`BoardAfter`). Retour arrière défait le dernier pas ; le rejeu est contraint tant qu'un
   coup légal contient les pas, libre dès le premier qui ne l'était pas — défaire le seul
   pas hors des règles rend donc la liste. La liste d'un coup libre est vide, et une ligne
   la remplace : « Coup hors des règles — Entrée l'enregistre ». Sans jet saisi, le glissé
   reste contraint : un coup illégal ne dit pas quel jet l'a produit, et le deviner écrirait
   dans le match un jet que personne n'a vu.
5. **Le coup se tape dans sa cellule.** Un double-clic sur une cellule du Transcript — coup,
   danse, coup non consigné — la change en champ, pré-rempli de sa notation (vide pour les
   deux autres). On n'y tape QUE le coup : les dés sont ceux de la cellule. Entrée l'écrit
   sur cette Action (le premier clic du double-clic y a mené le Cursor ; sinon le chemin part
   devant, dans la même file) ; un coup illégal est accepté, marqué par le moteur ; un texte
   qui ne dit aucun pas laisse le champ ouvert. Échap le ferme sans rien écrire, par
   `escapeService` — écouté en capture, il ferme le champ et rien d'autre. La cellule en
   pointillés de la saisie en cours (ADR-0049) s'ouvre de même, ses deux dés saisis : c'est
   ce qui remplace le champ de notation du volet pour une Action neuve.
6. **Le volet `✎` est supprimé**, avec la bascule, le bouton « Ce plateau est le coup joué »,
   le champ et leurs clés de traduction. La palette redevient les deux dés, la rangée
   `[D][T][P][R]` et le triangle.

Rien ne change côté moteur Go : `enter_play` avec `BoardAfter` acceptait déjà un coup
illégal, marqué et jamais refusé, et le jette de lui-même quand un coup légal atteint ce
plateau (`transcript.validate`).

## Les options écartées

- **Garder le filtre à côté du coup joué.** Deux sens pour le même clic selon un état
  invisible : c'est ce que la décision retire.
- **Un coup libre qui part seul** quand il « a l'air » fini. Rien ne dit qu'un coup hors des
  règles est achevé ; Entrée est la seule touche qu'il ajoute.
- **Le glissé libre sans jet saisi.** Il faudrait écrire des dés que personne n'a tapés.

## Les conséquences

- Le coup lointain (rang 12 d'ux.md §4.1) se joue en deux touches et deux glissés,
  2 K + 2 H + 2 (P + 2 B) ≈ 4,0 s au modèle KLM, compté par
  `transcription-budgets.spec.js` sur le vrai damier. C'est plus que le filtre annoncé
  (≈ 2,9 s) et que le clavier seul (13 K = 3,6 s) : le gain n'est pas de temps pur mais
  d'interaction — pas de mode, pas de clic qui ne fait rien avancer, et la validation
  disparaît. Le chemin le plus court pour un coup reconnu reste le clavier, et sans dés
  tapés le plateau déduit le jet en 3,4 s.
- Le second clic d'un double-clic tombe souvent avant que le premier soit revenu du moteur :
  `selectAction` ne relance plus un chemin déjà en route vers la même cellule.
- Le panneau ne reprend plus le focus à un champ qui l'a — le document revient du moteur
  pendant qu'on tape dans la cellule.
- `newFreePlay` disparaît de `transcriptionPlay.js` : un coup ne commence plus libre, il le
  devient.
