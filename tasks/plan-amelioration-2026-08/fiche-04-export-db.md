# Fiche 04 — Bug d'export .db : schéma courant partout

Branche : `fix/export-schema-courant`

## Objectif

Qu'un fichier `.db` produit par **n'importe quel** chemin d'export soit une
base au schéma courant : positions compactes, `zobrist_hash` rempli, colonnes
scalaires remplies, index présents, métadonnées par allow-list. C'est le bug
le plus grave de l'audit : il livre à un tiers des données inutilisables.

## Constats (vérifiés sur artefacts réels)

- `db_export.go:108` : la table `position` de l'export complet n'a que
  `id, state, individually_imported` ; `:446` écrit `fullPositionJSON`
  (JSON complet ~864 o au lieu de la forme compacte ~60 o) ; `:322` estampille
  `database_version = 2.14.0`. Résultat : `runMigrationChain` ne joue rien,
  `ensureAllTablesExist` ajoute des colonnes qui restent NULL à jamais.
  Mesuré : `a.db` → 463 positions, 463 `zobrist_hash` NULL, table `position`
  = 60 % du fichier, ~14× la taille attendue. Dédup et filtres SQL morts chez
  le destinataire.
- Même défaut dans `ExportCollections` (`db_collection.go:550-890`, INSERT
  ligne 790) et `ExportTournaments` (`db_tournament.go:385-804`, ligne 604),
  avec en plus des schémas déjà divergents entre eux (table `analysis`).
- Ces deux exports copient `metadata` **par inclusion brute**
  (`db_collection.go:876-879`, `db_tournament.go:798-801`) au lieu de
  l'allow-list `issuance.Carried`, et n'appliquent pas le watermark — double
  violation des invariants CLAUDE.md (« Exports carry an allow-list »).
- `ingest/sqlite_export.go` (serveur) n'exporte aucun match (assumé dans son
  commentaire) et aucun watermark — constat noté, traité en backlog (unifier
  les 4 exports est un chantier séparé).
- `fullPositionJSON` (`db_position.go:138-141`) avale l'erreur de
  `json.Marshal` et renvoie `""`.
- Les tests actuels n'assertent que « le fichier existe et n'est pas vide ».

## Tâches

- [ ] **Test de round-trip d'abord** (il doit échouer avant correctif) :
      pour chacun des 3 exports `database/` — exporter, rouvrir avec
      `OpenDatabase`, puis vérifier : `zobrist_hash` non NULL partout,
      colonnes scalaires remplies (pip, score, dés), une recherche SQL
      (par dés ou score) qui trouve, un ré-import qui déduplique.
- [ ] Faire écrire le **schéma de destination par la source unique**
      `storage/sqlite` (`sqlite.Bootstrap` / `schema_sqlite.go`) au lieu des
      DDL manuscrits des 3 chemins d'export.
- [ ] Écrire les positions via le chemin normal (`EncodeBoardCompact` +
      remplissage des colonnes scalaires — réutiliser ce que fait
      `SavePosition`), en conservant les helpers batchés existants de
      `db_export.go` (`positionsByIDsLocked`, `analysisForPositions`,
      `commentsForPositions`, `forEachInBatch`) et en les appliquant AUSSI aux
      exports collections/tournois (aujourd'hui N+1 sans transaction).
- [ ] Envelopper collections/tournois dans une transaction + PRAGMAs comme le
      fait déjà `ExportDatabase` (`db_export.go:96-101`).
- [ ] Métadonnées : faire passer collections/tournois par `issuance.Carried`
      (allow-list) et par le watermark, comme `ExportDatabase`.
- [ ] `fullPositionJSON` : propager l'erreur de marshal.
- [ ] **Compat lecture** : vérifier que l'import de vieux exports (JSON
      complet, version mensongère) continue de fonctionner —
      `db_import_db.go:140,347` lit déjà les deux formats ; ajouter un test
      d'import d'un fixture « vieil export » si absent.

## Critères de fin

- Round-trip vert pour les 3 exports ; taille du fichier exporté divisée
  (~×10 sur la fixture) ; recherche par dés/score fonctionne sur la base
  réouverte ; ré-import sans doublon.
- Un export de collection/tournoi ne transporte plus une métadonnée hors
  allow-list (test).
- Suite complète verte, y compris `export_test.go` existants.

## Risques & garde-fous

- C'est un format de fichier échangé entre utilisateurs : ne toucher qu'au
  **producteur**, jamais exiger un nouveau lecteur (les fichiers produits
  doivent rester lisibles par les versions récentes existantes — le schéma
  courant l'est par définition).
- Ne PAS bump `DatabaseVersion` : on écrit enfin le schéma que la version
  annonce déjà.
- Grosse fiche : découper en 3 commits (export complet, collections,
  tournois) pour garder des diffs revisables.
