# ADR-0050 — Supprimer recule sur la décision précédente, et une partie rouverte se continue sur place

- **Statut** : accepté — décidé le 2026-09-24, le jour même de l'ADR-0049, après usage.
- **Amende** : ADR-0049 (la correction en place) et fonctionnel.md §2, ligne « supprimer »
  (« le Cursor : la suivante ») et scénario 14.
- **Applique** : ADR-0044 (rien n'est refusé, tout est marqué) et ADR-0045 règle 4 (le
  `side` appartient à l'Action).

## Le problème

Deux gestes de correction laissaient le curseur ailleurs que là où l'utilisateur
continue.

1. **Suppr laissait le curseur sur l'Action SUIVANTE**, puis le Replay le posait sur la
   première Incohérence à partir de là — le double trait que la suppression venait de
   créer, c'est-à-dire la même cellule. Or supprimer, dans le geste de l'utilisateur,
   c'est effacer la décision qu'il est en train d'éditer et revenir sur celle d'avant,
   comme Retour arrière dans un texte. Et en bout de document, où la décision éditée
   est la cellule vide ou le jet à moitié tapé, Suppr ne faisait rien du tout.
2. **Une passe corrigée en prise renvoyait le curseur ailleurs.** La prise rouvre la
   partie, mais la cellule qui suit dans le document est déjà l'ouverture de la partie
   suivante : la validation renvoyait le curseur là où la relecture l'avait pris (ou sur
   cette ouverture), et taper la suite de la partie rouverte écrasait la partie d'après.
   L'ADR-0049 décision 3 couvrait l'insertion ouverte par `i`, pas la correction.

## La décision

1. **Suppr (et `x`) retire la décision en cours d'édition et recule sur la précédente**,
   chargée pour correction. Sur une Action écrite, elle disparaît ; sur une insertion
   ouverte, sur le créneau d'une partie continuée ou sur le jet tapé en bout de document,
   c'est cette saisie qui est abandonnée, et rien d'écrit n'est touché. En bout de
   document sans rien de tapé, la cellule vide est la décision : Suppr recule sur la
   dernière Action, et pressée encore, la supprime. Sur la première Action il n'y a rien
   avant : le curseur reste sur ce qui suit. Seul un brouillon vide répond « aucune
   action sous le curseur ».
2. **Une correction qui laisse sa partie ouverte, suivie directement d'une ouverture,
   ouvre un créneau d'insertion juste après elle** — au camp que la suite propose (après
   une prise, le doubleur). C'est la passe devenue prise, et aussi un dernier coup de
   sortie corrigé en coup qui ne sort pas. Le curseur ne revient pas à sa place
   antérieure : l'utilisateur est au milieu d'une partie à finir.
3. **Une insertion qui continue s'arrête quand sa partie se termine.** Passé ce point,
   l'Action suivante est l'ouverture de la partie d'après, et le curseur s'y pose au
   lieu de glisser un jet de cette partie devant elle.
4. **Ces gestes TIENNENT le curseur** (`Document.HoldCursor`, lu par `Editor.From`) : le
   Replay ne le tire pas vers une Incohérence. Le double trait qu'une suppression laisse
   est marqué dans le transcript, devant le curseur ; l'Action qu'une insertion vient
   d'écrire porte ses marques à sa place. C'est la raison qu'on donnait déjà pour ne pas
   déplacer le curseur après un ajout en bout de document : tirer le curseur sur ce qui
   vient d'être tapé ferait corriger au jet suivant au lieu de continuer.

## Conséquences

- `transcript.Apply` : `GestureDelete` recule (`stepBack`) ; `record` appelle
  `continueGame` après un remplacement et consulte `gameEndsAt` avant de rouvrir un
  créneau. `gameEndsAt` paie un Replay du préfixe, et seulement pour une écriture AU
  MILIEU du document — jamais pour l'ajout avec lequel un match se tape.
- Le panneau ne répond plus « aucune action » à Suppr en bout de document.
- Ce qui est ENREGISTRÉ ne change pas : les Actions, le Replay, les Incohérences et le
  Match produit sont ceux qu'ils étaient.
