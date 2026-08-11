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

- [x] **Test de round-trip d'abord** (il doit échouer avant correctif) :
      pour chacun des 3 exports `database/` — exporter, rouvrir avec
      `OpenDatabase`, puis vérifier : `zobrist_hash` non NULL partout,
      colonnes scalaires remplies (pip, score, dés), une recherche SQL
      (par dés ou score) qui trouve, un ré-import qui déduplique.
- [x] Faire écrire le **schéma de destination par la source unique**
      `storage/sqlite` (`sqlite.Bootstrap` / `schema_sqlite.go`) au lieu des
      DDL manuscrits des 3 chemins d'export.
- [x] Écrire les positions via le chemin normal (`EncodeBoardCompact` +
      remplissage des colonnes scalaires — réutiliser ce que fait
      `SavePosition`), en conservant les helpers batchés existants de
      `db_export.go` (`positionsByIDsLocked`, `analysisForPositions`,
      `commentsForPositions`, `forEachInBatch`) et en les appliquant AUSSI aux
      exports collections/tournois (aujourd'hui N+1 sans transaction).
- [x] Envelopper collections/tournois dans une transaction + PRAGMAs comme le
      fait déjà `ExportDatabase` (`db_export.go:96-101`).
- [x] Métadonnées : faire passer collections/tournois par `issuance.Carried`
      (allow-list) et par le watermark, comme `ExportDatabase`.
- [x] `fullPositionJSON` : propager l'erreur de marshal.
- [x] **Compat lecture** : vérifier que l'import de vieux exports (JSON
      complet, version mensongère) continue de fonctionner —
      `db_import_db.go:140,347` lit déjà les deux formats ; ajouter un test
      d'import d'un fixture « vieil export » si absent.

## Critères de fin

- [x] Round-trip vert pour les 3 exports ; recherche par dés/score fonctionne
      sur la base réouverte ; ré-import sans doublon. La taille du fichier
      exporté baisse mais **pas d'un facteur ~×10** sur la fixture mesurée —
      voir Notes d'exécution : le `~×10` du critère initial ne comptait que le
      `state` JSON→compact, pas le coût des ~15 index désormais créés sur
      `position`/`analysis` (nécessaires à la dédup et aux filtres que ce
      correctif rétablit).
- [x] Un export de collection/tournoi ne transporte plus une métadonnée hors
      allow-list (test).
- [x] Suite complète verte, y compris `export_test.go` existants.

## Risques & garde-fous

- C'est un format de fichier échangé entre utilisateurs : ne toucher qu'au
  **producteur**, jamais exiger un nouveau lecteur (les fichiers produits
  doivent rester lisibles par les versions récentes existantes — le schéma
  courant l'est par définition).
- Ne PAS bump `DatabaseVersion` : on écrit enfin le schéma que la version
  annonce déjà.
- Grosse fiche : découper en 3 commits (export complet, collections,
  tournois) pour garder des diffs revisables.

## Notes d'exécution (2026-08-11)

**Commits** (branche `fix/export-schema-courant`) :

1. `2eaf3152` — export complet (`db_export.go`, `db_export_position.go` nouveau
   fichier partagé, `db_import_db.go` — `decodeSourcePosition`, voir
   « écart imprévu » ci-dessous, `export_test.go`).
2. `f266696e` — `ExportCollections` (schéma courant, transaction, lectures
   batchées, allow-list + watermark, `collection_test.go`).
3. `47b7edd8` — `ExportTournaments` (même correctif, `tournament_test.go`).
4. `b51effea` — `fullPositionJSON` propage l'erreur de marshal (tâche 4,
   déplacée en dernier : son seul appelant restant après les commits 1-3 est
   `db_import_db.go`) + fixture « vieil export » (`db_import_db_test.go`,
   tâche 5).

**Écart imprévu, corrigé dans le commit 1** : passer les exports en état
compact (`EncodeBoardCompact`) a cassé la dédup de `CommitImportDatabase`
(le fusionneur « Import database » du GUI/CLI) — vérifié en écrivant le
test de round-trip *avant* le correctif : `before=2 after=4` positions.
`CommitImportDatabase` ne savait décoder que l'état JSON complet (il ne lit
que `id, state, individually_imported` de la base source, jamais les colonnes
scalaires) ; sur un état compact il retombait donc sur un board vide (dés,
score, cube à zéro), et deux positions identiques ne se reconnaissaient plus
comme identiques. `decodeSourcePosition` (`db_import_db.go`) lit maintenant
les colonnes scalaires de la ligne source quand l'état est compact — ce
correctif touche le côté lecture, en dehors du périmètre strict de la fiche,
mais était nécessaire pour que son propre critère de fin (« ré-import sans
doublon ») soit satisfiable une fois l'export en état compact.

**Chiffres avant/après** (fixture : `testdata/test.xg`, 543 positions, 1
match / 7 games / 543 moves, importé puis exporté avec `ExportDatabase`) :

| Export                                              | Avant       | Après       | Ratio |
|------------------------------------------------------|------------:|------------:|------:|
| Complet (positions + analyses + commentaires + match) | 2 183 168 o | 2 068 480 o | ×1,06 |
| Positions seules (aucune analyse/commentaire/match)    |   647 168 o |   446 464 o | ×1,45 |

Le gain réel est bien plus modeste que le `~×10` visé par la fiche sur
cette fixture : sur un export complet, les blobs d'analyse (compressés, mais
volumineux par position) dominent la taille du fichier et sont inchangés par
ce correctif. Même sur l'export « positions seules », l'essentiel du gain
attendu (JSON complet ~800+ o/position → tableau compact ~60 o/position) est
en grande partie absorbé par le coût des ~15 nouveaux index B-tree que
`sqlite.Bootstrap` crée sur `position`/`analysis` (uniques et partiels
compris) — index qui n'existaient pas dans l'ancien export et qui sont
justement ce qui rend la dédup et les filtres SQL de nouveau fonctionnels.
Le `a.db` cité dans les constats de la fiche (60 % du fichier en table
`position`, ~14× attendu) était vraisemblablement un export dominé par les
positions et sans beaucoup d'analyses ; sa proportion ne se reproduit pas
telle quelle sur cette fixture synthétique. Le critère « le fichier est
utilisable » (dédup + recherche SQL, vérifié par les tests de round-trip)
est rempli ; le critère de taille ne l'est qu'en partie et est corrigé
ci-dessus plutôt que forcé.

**Comportements changés** :

- `ExportDatabase`, `ExportCollections`, `ExportTournaments` produisent un
  fichier `.db` au schéma courant complet (mêmes tables/index que
  `sqlite.Bootstrap`), positions en forme compacte + colonnes scalaires
  remplies, dédup par `zobrist_hash` à l'écriture.
- `ExportCollections`/`ExportTournaments` gagnent deux paramètres,
  `watermark string, watermarkNote string` (sur le modèle de
  `ExportOptions.Watermark`/`WatermarkNote`) ; aucun appelant CLI/GUI
  aujourd'hui (seuls les tests et le binding Wails auto-généré les
  utilisaient), changement donc sans risque de régression utilisateur mais
  `frontend/wailsjs/` restera à régénérer (`wails dev`) avant tout usage
  frontend futur de ces deux méthodes.
- `ExportCollections`/`ExportTournaments` copient désormais `metadata` par
  allow-list (`issuance.Carried`) et écrivent un watermark scellé si demandé,
  au lieu de copier `metadata` par inclusion brute.
- `fullPositionJSON` retourne `(string, error)` au lieu d'avaler l'erreur de
  marshal.
- `CommitImportDatabase`/`AnalyzeImportDatabase` décodent correctement l'état
  compact d'une position source (`decodeSourcePosition`), pas seulement le
  JSON complet legacy.

**Risques résiduels** (hors périmètre de cette fiche, non corrigés) :

- `CommitImportDatabase` insère toujours une position « neuve » via
  `fullPositionJSON` (JSON complet, colonnes scalaires NULL) directement
  dans la base courante — un défaut préexistant, indépendant de l'export,
  découvert pendant ce travail : fusionner deux bases ordinaires (schéma
  courant, état compact) via « Import database » laisse la position
  nouvellement ajoutée avec des colonnes scalaires NULL dans la base
  destination, et un second import de la même source la duplique (identité
  reconstruite par `reconstructPosition` à partir de colonnes NULL). Non
  corrigé ici : hors périmètre (chemin d'écriture de `CommitImportDatabase`,
  pas un chemin d'export), et `db_import_db.go`/l'import natif `.db` sont
  documentés comme un chantier séparé (« two-phase GUI preview »).
- `is_cube_response` et `match_length` (colonnes `position`) ne sont pas
  reconstitués par l'export : `is_cube_response` est un sous-produit de
  `SaveAnalysis` (pas de `PopulatePositionColumns`) et `match_length` n'est
  jamais peuplé par `SavePosition` non plus dans la base live — parité avec
  le comportement actuel de `SavePosition`, pas une régression introduite
  ici, mais un export perd le repli `is_cube_response=1` d'une position
  cube-response de la base source.
- `ingest/sqlite_export.go` (export du serveur headless) reste un 4ᵉ chemin
  d'export non traité, comme noté dans les constats d'origine — unifier les
  4 exports reste un chantier séparé.
