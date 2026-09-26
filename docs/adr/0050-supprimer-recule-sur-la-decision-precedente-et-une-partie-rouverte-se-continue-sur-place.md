# ADR-0050 — Supprimer recule sur la décision précédente, et une partie rouverte se continue sur place

Statut : acceptée.
Voir aussi : ADR-0045 règle 4, ADR-0049, ADR-0054.

## Contexte

Suppr laissait le curseur sur l'Action suivante (puis le Replay le posait sur le double trait
créé), et ne faisait rien en bout de document. Une passe corrigée en prise rouvre la partie,
mais la cellule suivante est l'ouverture de la partie d'après : la suite tapée l'écrasait.

## Décision

1. **Suppr (et `x`) retire la décision en cours d'édition et recule sur la précédente**,
   chargée pour correction. Sur une Action écrite, elle disparaît ; sur une insertion ouverte,
   un créneau de continuation ou un jet tapé en bout de document, seule la saisie est
   abandonnée. En bout de document sans rien de tapé, Suppr recule sur la dernière Action, et
   pressée encore, la supprime. Sur la première Action, le curseur reste sur ce qui suit. Seul
   un brouillon vide répond « aucune action sous le curseur ».
2. **Une correction qui laisse sa partie ouverte, suivie directement d'une ouverture, ouvre un
   créneau d'insertion juste après elle**, au camp que la suite propose (après une prise, le
   doubleur).
3. **Une insertion qui continue s'arrête quand sa partie se termine** : le curseur se pose sur
   l'ouverture suivante.
4. **Ces gestes tiennent le curseur** (`Document.HoldCursor`, lu par `Editor.From`) : le Replay
   ne le tire pas vers une Incohérence ; le double trait laissé est marqué devant lui. Les gestes
   qui n'écrivent rien : ADR-0054 règle 1.

## Conséquences

- `transcript.Apply` : `GestureDelete` recule (`stepBack`) ; `record` appelle `continueGame`
  après un remplacement et consulte `gameEndsAt`, qui rejoue le préfixe — seulement pour une
  écriture au milieu du document, jamais pour l'ajout en bout.
- Rien de ce qui est enregistré ne change.

## Garde

`pkg/blunderdb/transcript/continuation_test.go`, `correction_test.go`.
