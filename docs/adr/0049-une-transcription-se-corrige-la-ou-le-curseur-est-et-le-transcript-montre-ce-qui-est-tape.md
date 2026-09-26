# ADR-0049 — Une transcription se corrige là où le curseur est, et le transcript montre ce qui est tapé

Statut : acceptée.
Voir aussi : ADR-0044, ADR-0045, ADR-0048 décision 1, ADR-0050, ADR-0054.

## Contexte

Transcrire, c'est se reprendre : une passe lue pour une prise, une ouverture au mauvais camp.
Le transcript ne dessinait que le document (pas l'Entry en cours), les gestes de videau
écrivaient toujours en bout de document, la cellule sous le curseur n'avait pas d'attente
propre, et le plateau lisait `has_position = false` comme « rien à montrer ». Rien ici ne
change ce qui est enregistré.

## Décision

Le geste écrit où le curseur est, et ce qui est tapé se voit à cette place avant d'être écrit.

1. **Le transcript dessine l'Entry**, en pointillés, dans le créneau qu'elle occupera :
   par-dessus la cellule qu'une correction remplace, entre deux voisines pour une insertion,
   au bas de la partie pour une saisie neuve. Dés, puis notation dès qu'un candidat est
   choisi ; le camp se lit à la colonne. Le moteur fournit tout (`transcript.EntryInfo` :
   `Kind`, `Notation`) ; le composant ne dérive rien.
2. **Les quatre gestes de videau écrivent au rang de l'Entry** : sur une cellule relue ils
   remplacent, dans un créneau `i`/`a` ils remplissent, en bout de document ils ajoutent.
   `t` et `p` atteignent donc aussi la cellule tenue par le curseur ; dès qu'un dé est tapé,
   `p` revient au compte de pips global.
3. **Une insertion au milieu du document continue d'insérer** : la validation rouvre un
   créneau vide à la suite ; déplacer le curseur y met fin. Arrêt en fin de partie : ADR-0050
   règle 3.
4. **La cellule sous le curseur a sa propre attente** (`entry.kind`), distincte de
   `next.expects`. Ressaisir une ouverture se comporte en ouverture : dé du joueur 1, dé du
   joueur 2, validation au second ; gros dé d'abord → le joueur 1 commence, petit dé d'abord →
   le joueur 2.
5. **`has_position` n'est pas « rien à montrer »** : le plateau suit le curseur sur toute
   Action et lit `before` ; la fin du document n'est la réponse que passé la dernière.
6. **L'ordre des dés ne nomme le camp que sur l'ouverture.** Ailleurs le trait est proposé par
   le Replay, et `s` le change sur une Action écrite.
7. **Le premier coup de pions suit l'ouverture corrigée** (`followOpening`) — seule entorse à
   ADR-0045 règle 4, parce que ce camp n'a jamais été choisi par l'utilisateur et que rien ne
   marquerait l'erreur (le coup reste légal pour les deux camps). Un premier coup déjà donné à
   l'autre camp par `s` est laissé tel quel ; le reste de la partie garde ses camps.

## Conséquences

- `entryExpects` est écrit une fois, lu par `expectsOpening` et par `Replayer.Replay`.
- Un `t` sur une cellule relue détruit ce qu'elle portait ; recours : `Ctrl+Z`, et le dessin
  qui montre d'avance ce que la validation écrira.

## Garde

`pkg/blunderdb/transcript/inplace_test.go`, `correction_test.go`,
`frontend/src/__tests__/TranscriptView.pending.test.js`, `transcriptionKeys.opening.test.js`.
