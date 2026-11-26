# ArkID est l'écriture textuelle lisible et canonique d'une position

Statut : acceptée.
Voir aussi : ADR-0001 (identité par l'empreinte Zobrist), ADR-0028 (règles de session hors de l'identité).

## Contexte

Une position s'échange sous trois écritures qui ne sont pas faites pour être lues : l'XGID
(eXtreme Gammon), le GNUbg ID (deux chaînes base64) et l'OGID (une lettre par pion, plateau
absolu, points marqués au lieu de scores « à combien »). Aucune ne se prononce ni ne se
relit à l'œil. blunderDB veut une écriture qu'un joueur lit à voix haute et vérifie, sans
perdre l'identité d'une position (ADR-0001).

## Décision

1. **Une grammaire, `[score]dés:videau(portes)reste`**, écrite du côté du joueur au trait : les
   minuscules sont ses pions, les majuscules ceux de l'adversaire, tous numérotés comme le
   joueur au trait numérote son plateau. `A`…`W` sont les points 1 à 23, `Y` le point 24, `Z`
   la barre, `X` est le videau. Une lettre est l'initiale d'un animal (le bestiaire du
   glossaire) : le moyen mnémotechnique est la raison d'être de la notation.
2. **Le point de vue est celui du stockage.** Une position est stockée du côté du joueur au
   trait (`NormalizeForStorage`), l'ArkID s'écrit depuis cette forme et se relit en elle : la
   position collée retombe sur la même empreinte Zobrist, quelle que soit la couleur du joueur
   au trait dans la source.
3. **Lecture tolérante, écriture stricte.** Le lecteur prend la forme compacte et la forme
   développée, un ordre libre et plusieurs groupes de parenthèses ; l'écrivain produit
   toujours la forme compacte canonique (la développée sur demande). Deux écritures de la
   même position par blunderDB sont identiques au caractère près ; c'est ce qui permet de
   comparer deux ArkID comme deux chaînes.
4. **Le score est celui de la position**, un *Away score* (CONTEXT.md) : deux chiffres accolés
   si les deux sont de 9 au plus, `a-b` sinon, `[$]` à l'argent, un `c` pour la partie
   Crawford, rien après la Crawford. Les sentinelles se décodent dans les deux sens ; aucune
   longueur de match n'est inventée.
5. **Ce que la notation ne porte pas, elle ne le suppose pas** : les règles de session (Jacoby,
   castor) et le plafond du videau restent à zéro à la lecture, comme pour l'OGID
   (ADR-0028) ; un videau au centre se lit à 1 sans exposant.
   Un videau **proposé** (position de réponse, `IsResponsePosition` : au centre, de valeur
   non nulle, ADR-0083) s'écrit `X`, l'exposant s'il dépasse 1, puis `?` (`X?` : proposé à 2) et
   se relit au centre à cette valeur, de sorte que l'empreinte est conservée ; `x?` est refusé.
   Ce que la notation ne sait pas écrire sans le perdre — dés d'une décision de videau, videau
   central au-dessus de 1 sur une décision de pions — est refusé par l'encodeur.
6. **Une seule place dans le code, `domain.EncodeArkID` / `DecodeArkID`**, à côté des autres
   codecs ; l'ArkID est proposé partout où l'XGID l'est (commande `arkid`, `--format arkid`
   de `search` et de `capture`, route `positions.arkId` / `positions.fromArkID`, copie du
   menu d'une position, collage et `import`, arguments de position des commandes d'analyse).
   `parser.ParsePosition` le lit comme un OGID : une position, aucune analyse.

## Conséquences

- Aucun changement de schéma, aucun changement de l'empreinte (`engine.init` intact).
- Le bestiaire est de la documentation traduite par langue, pas du code : les lettres sont
  les mêmes partout.
- Une position qu'aucun ArkID ne sait écrire (argent pour un seul joueur, plus de quinze pions
  par camp) est refusée avec une erreur ; elle n'est jamais tronquée.
