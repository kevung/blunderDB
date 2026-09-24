# ADR-0054 — Le trou d'un double trait est une case, et seule une écriture déplace le curseur

- **Statut** : accepté — décidé le 2026-09-24, après usage, le jour de l'ADR-0050.
- **Amende** : ADR-0050 décision 4 (qui ne tenait le curseur qu'après trois gestes) et
  fonctionnel.md §1.4 (« le curseur saute sur la première Incohérence après un Replay »).
- **Applique** : ADR-0044 (rien n'est refusé, tout est marqué) et ADR-0049 (on corrige là
  où le curseur est).

## Le problème

« Quand je supprime une décision dans une transcription, cela laisse un trou, que je ne
peux plus éditer (un joueur joue alors deux fois). » Trois défauts se composaient.

1. **Le double trait était un mur.** La session (`stateOf`) relit après chaque geste
   depuis `Editor.From()`, qui valait le curseur quand le geste n'avait rien écrit, et
   pose le curseur sur la première Incohérence trouvée. Une fois sur l'Action qui suit
   le trou, `h` reculait… et la relecture le ramenait sur le double trait : on ne
   pouvait plus revenir en arrière, ni au clavier ni au clic (un clic est une suite de
   pas).
2. **Un dé tapé devant le trou partait ailleurs.** Après la suppression, le curseur est
   sur la décision précédente (ADR-0050) ; le premier chiffre tapé pour la corriger
   n'écrit rien, la relecture partait du curseur, trouvait le double trait juste après
   et y emportait le curseur — le chiffre était perdu, et `a` insérait alors après la
   mauvaise Action.
3. **Le trou n'était pas une case.** Le Transcript le dessinait comme une cellule vide
   inerte ; `h`/`l` passaient d'une Action du même camp à l'autre sans s'y arrêter.
   La seule façon de le remplir — `a` sur l'Action d'avant — ne se devinait pas.

## La décision

1. **Seul un geste qui écrit déplace le curseur vers une Incohérence.** Se déplacer
   (`h`, `l`, un clic), taper un dé, choisir ou jouer un coup, ouvrir une insertion ou
   une correction TIENNENT le curseur (`Document.HoldCursor`), sauf quand le pas
   enregistre une correction au passage — c'est alors une écriture, et le saut vaut. Le
   saut répond à un Replay ; un geste qui n'a rien écrit n'a rien rejoué.
2. **Le tour qu'un double trait a perdu est un arrêt du curseur.** Entre deux Actions
   qui portent un tour et sont du même camp, `h` et `l` s'arrêtent sur le trou : une
   insertion y est ouverte pour le camp dont le tour manque, et le jet qui s'y tape
   s'insère là. En quitter sans rien taper abandonne l'insertion, comme toute autre.
3. **Le trou se dessine et se clique.** Le Transcript le place dans la colonne du camp
   qui n'a pas joué, encadré de pointillés rouges ; un clic y mène le curseur. Le rang
   des arrêts (`cursorStop`) compte les trous, pour que le chemin d'un clic par-dessus
   un trou ne s'arrête pas un pas trop tôt.

## Conséquences

- `transcript.Apply` enveloppe l'ancien corps (`apply`) et tient le curseur des gestes
  de `movesOnly` qui n'ont rien touché ; `holeBefore`/`onHole`/`openHole` portent la
  règle du trou, qui est celle du `DoubleTurn` du Replay lue sur deux Actions.
- Le front lit le trou dans l'Incohérence `double_turn` que le moteur rend ; il ne
  refait pas la règle.
- Ce qui est ENREGISTRÉ ne change pas : les Actions, le Replay, les Incohérences et le
  Match produit sont ceux qu'ils étaient. Un document cohérent n'a aucun trou et se
  parcourt exactement comme avant.
