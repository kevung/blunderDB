# ADR-0054 — Le trou d'un double trait est une case, et seule une écriture déplace le curseur

Statut : acceptée.
Voir aussi : ADR-0044, ADR-0049, ADR-0050 règle 4.

## Contexte

Après une suppression, le double trait laissé était un mur : la relecture qui suit chaque
geste reposait le curseur sur la première Incohérence, si bien que `h` y revenait sans fin et
qu'un dé tapé devant le trou partait ailleurs. Le trou lui-même était une cellule inerte que
`h`/`l` sautaient.

## Décision

1. **Seul un geste qui écrit déplace le curseur vers une Incohérence.** Se déplacer (`h`,
   `l`, clic), taper un dé, choisir ou jouer un coup, ouvrir une insertion ou une correction
   tiennent le curseur (`Document.HoldCursor`), sauf si le pas enregistre une correction au
   passage — c'est alors une écriture.
2. **Le tour qu'un double trait a perdu est un arrêt du curseur** : entre deux Actions à tour
   du même camp, `h` et `l` s'y arrêtent ; une insertion y est ouverte pour le camp manquant,
   et en sortir sans rien taper l'abandonne.
3. **Le trou se dessine et se clique** : dans la colonne du camp qui n'a pas joué, encadré de
   pointillés rouges ; `cursorStop` compte les trous dans le rang des arrêts.

## Conséquences

- `transcript.Apply` enveloppe `apply` et tient le curseur des gestes `movesOnly` sans effet ;
  `holeBefore`/`onHole`/`openHole` lisent la règle du `DoubleTurn` du Replay.
- Le front lit le trou dans l'Incohérence `double_turn` ; il ne refait pas la règle.
- Un document cohérent n'a aucun trou et se parcourt comme avant.

## Garde

`pkg/blunderdb/transcript/hole_test.go`, `frontend/src/__tests__/TranscriptView.hole.test.js`.
