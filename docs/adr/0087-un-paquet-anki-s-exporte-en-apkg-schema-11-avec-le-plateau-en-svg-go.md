# ADR-0087 — Un paquet Anki s'exporte en .apkg de schéma 11, le plateau en SVG dessiné en Go

Statut : acceptée.
Voir aussi : ADR-0007 (rien n'est écrit chez le destinataire), ADR-0019 (une échelle d'équité),
ADR-0025 (la réponse masquée), ADR-0084 (la collection vivante s'évalue dans le moteur).

## Contexte

Réviser un paquet d'étude sur un téléphone demandait `serve` derrière un proxy. Anki, lui, tourne
partout — Anki desktop, AnkiDroid, AnkiMobile — et importe un fichier `.apkg`. Trois formats
de paquet coexistent :

| Contenu du zip | Lu par | Notes |
|---|---|---|
| `collection.anki2` (schéma 11), `media` en JSON, médias numérotés | tout Anki depuis 2.0, AnkiDroid, AnkiMobile | format de genanki ; le plus répandu |
| `collection.anki21` (schéma 11, planificateur v2) + leurre `collection.anki2` | Anki ≥ 2.1.x | n'apporte rien à un paquet sans historique de révision |
| `collection.anki21b` (zstd, médias décrits en protobuf) | Anki ≥ 2.1.50 seulement | refusé par les clients anciens |

Une carte de blunderDB doit montrer le plateau. L'interface le dessine en Svelte, dans la palette
choisie ; le CLI et le daemon n'ont pas de navigateur. `report.Diagram` dessine déjà en Go, pour
le rapport HTML du CLI et du daemon, un SVG autonome du plateau (palette par défaut, joueur au trait
en bas, videau, dés).

## Décision

1. **Le paquet est un `collection.anki2` de schéma 11** (`col.ver = 11`), un index `media` en
   JSON et les médias nommés `0`, `1`… — le format que tous les clients importent. Le fichier
   SQLite est écrit par le pilote du dépôt (`modernc.org/sqlite`, sans CGO), donc le daemon pur
   Go le produit aussi.
2. **Le plateau est le SVG de `report.Diagram`**, un fichier média par position. SVG plutôt que
   PNG : vectoriel, quelques kilo-octets, affiché par la vue web de chaque client, et aucun
   rastériseur à embarquer. Les trois modes produisent ainsi le même fichier, octet pour octet à
   date égale ; la palette choisie à l'écran ne voyage pas.
3. **Une position, une note, une carte** recto-verso. Recto : plateau, score (vu du joueur au
   trait), videau, dés ou « décision de videau ». Verso : meilleur coup ou bonne décision de
   videau, équité (l'échelle unique d'ADR-0019, telle que stockée), erreur du coup joué. La
   réponse au damier (ADR-0025) ne se transpose pas.
4. **Une identité stable** : id et guid de la note dérivés du hash Zobrist, id du type de note
   constant (`apkg.ModelID`), id du paquet dérivé de son nom, `mod` à l'heure de l'export. Un
   réexport importé par-dessus le premier met les notes à jour (« si plus récent ») au lieu de
   les dupliquer ; Anki garde l'historique de révision de son côté.
5. **L'export lit et n'écrit rien** (ADR-0007) : un paquet s'exporte tel qu'il est, sans la
   resynchronisation d'une séance ; la requête d'une collection vivante est évaluée à l'export,
   sous son plafond (ADR-0084). La description du paquet Anki porte les métadonnées de
   `issuance.CarriedMetadataKeys`, rien d'autre.
6. **Une logique, trois modes** : `apkg.Export` sur le contrat de stockage ;
   `Database.ExportAnkiPackage` (GUI), `blunderdb anki export` (CLI), `POST /v1/anki.exportApkg`
   (daemon).

## Conséquences

- Le fichier s'importe partout ; la preuve est l'importeur d'Anki lui-même
  (`scripts/anki-apkg-check.py`, et `TestAnkiImportsThePackage` quand `BLUNDERDB_ANKI_PYTHON`
  nomme un Python muni du paquet `anki`), hors CI.
- Les libellés des cartes suivent la langue demandée (neuf catalogues dans `pkg/blunderdb/apkg`).
- Changer `ModelID`, la dérivation des ids ou la liste des champs casse la mise à jour des notes
  déjà importées : ce sont des constantes de format.
