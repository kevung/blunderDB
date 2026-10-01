# ADR-0049 — Une transcription se corrige là où le curseur est, et le transcript montre ce qui est tapé

Statut : acceptée.
Voir aussi : ADR-0044, ADR-0045, ADR-0048 décision 1, ADR-0050, ADR-0054.

## Contexte

Transcrire, c'est se reprendre : une passe lue pour une prise, un premier coup au mauvais camp.
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
   `Notation`, `GameStart`) ; le composant ne dérive rien.
2. **Les quatre gestes de videau écrivent au rang de l'Entry** : sur une cellule relue ils
   remplacent, dans un créneau `i`/`a` ils remplissent, en bout de document ils ajoutent.
   `t` et `p` atteignent donc aussi la cellule tenue par le curseur ; dès qu'un dé est tapé,
   `p` revient au compte de pips global.
3. **Une insertion au milieu du document continue d'insérer** : la validation rouvre un
   créneau vide à la suite ; déplacer le curseur y met fin. Arrêt en fin de partie : ADR-0050
   règle 3.
4. **La cellule sous le curseur a sa propre attente** (`entry`), distincte de `next` : elle
   dit si elle est le premier coup d'une partie (`game_start`), celui de la cellule ou du
   créneau visé, non celui du bout du document.
5. **`has_position` n'est pas « rien à montrer »** : le plateau suit le curseur sur toute
   Action et lit `before` ; la fin du document n'est la réponse que passé la dernière.
6. **L'ordre des dés ne nomme le camp que sur le premier coup d'une partie.** Il n'y a pas
   d'Action d'ouverture : la partie commence par son premier coup, joué par le gagnant du jet
   d'ouverture avec ce jet. Ses deux dés se tapent comme ce jet — dé du joueur 1, puis dé du
   joueur 2 ; gros dé d'abord → le joueur 1 joue, petit dé d'abord → le joueur 2 — puis le
   coup se choisit comme tout coup. Un double ne nomme personne : il est saisi tel quel et
   marqué `inconsistent_dice`, aucun jet d'ouverture n'étant un double. Les égalités, rejouées
   à la table, ne se transcrivent pas. Ailleurs le trait est proposé par le Replay, et `s` le
   change sur une Action écrite, premier coup compris.
7. **Retaper les dés d'un premier coup redécide son camp**, et le coup suit en miroir s'il
   reste légal (la position de départ est symétrique) ; sinon le premier candidat est
   présélectionné et la cellule marquée à revoir. Le reste de la partie garde ses camps
   (ADR-0045 règle 4).

## Conséquences

- `slotOpensGame` est écrit une fois, lu par `GestureEnterDie` et par `Replayer.Replay`.
- Un `t` sur une cellule relue détruit ce qu'elle portait ; recours : `Ctrl+Z`, et le dessin
  qui montre d'avance ce que la validation écrira.

## Garde

`pkg/blunderdb/transcript/inplace_test.go`, `correction_test.go`,
`frontend/src/__tests__/TranscriptView.pending.test.js`.
