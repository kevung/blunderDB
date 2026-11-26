# Ark et impression : état de reprise

Chantier : notation ArkID (Animal Register Key), impression dense sans diagramme, archive
papier par codes QR. Issues #984 à #987.

## Fait, dans main

- #984 ArkID : codec `pkg/blunderdb/domain/arkid.go`, commande `arkid`, `--format arkid`,
  routes `positions.arkId` / `positions.fromArkID`, action « Copier l'ArkID », glossaire avec
  le bestiaire dans les neuf langues, ADR-0118.
- #985 archive papier : `pkg/blunderdb/arkqr` (codec, trames, QR, page A4), commande
  `archive export|import`, routes `/v1/exports.archive` et `/v1/imports.archive`, import d'une
  page `.html` par le chemin commun, ADR-0119.

## À faire, dans l'ordre

1. #986 feuille de match imprimable (dépend d'ArkID et du QR, tous deux dans main).
2. #987 listes de positions imprimables, et entrées d'interface de l'archive papier.

Les deux issues citent la maquette par un chemin local : elle est ici,
`tasks/ark-impression/maquette-feuille.html` (données factices). Elle porte l'encodeur de
référence, les trois documents (Match, Positions, Archive) et leurs options.

## Décisions

- Minuscules = joueur au trait, toujours ; la position est écrite de son point de vue.
- Crawford = suffixe `c` ; money = `[$]`.
- Videau proposé (position de prise ou de passe) : `X?`, `X2?`. Ajout fait pendant #984, sans
  lequel ces positions changeaient d'empreinte ; à confirmer par l'utilisateur.
- Un seul coup alternatif par ligne dans les listes de positions.
- Les blunders de l'adversaire ne sont pas exportés par défaut ; l'export le demande.
- Les commentaires s'exportent en option.

## Repères de mesure (base de 298 matchs)

- Tous les matchs tiennent sur une page A4 ; 251 coups imprimés en médiane, 756 au plus.
- Corps d'environ 8 pt pour un match en 7 ou 9 points, 6,5 pt en 11 ou 13 ; plancher 5,5 pt.
- ArkID : 29 caractères en médiane, 34 au plus.
- Listes : environ 160 positions par face avec analyse, 300 sans.
- Codec d'archive (prototype Python) : 201 octets par match en médiane. Le codec Go n'a pas
  été mesuré sur cette base.

## Écarts connus

- L'interface ne copie que la forme compacte d'ArkID ; la développée existe en CLI et serveur.
- Le dossier surveillé relit chaque `.html` à chaque balayage.
- La dernière correction de #985 n'a pas eu de revue indépendante.
- Bestiaire hors français et anglais : à relire par un locuteur.
- Essai de pliage réel (imprimer, plier, scanner) : non fait.
