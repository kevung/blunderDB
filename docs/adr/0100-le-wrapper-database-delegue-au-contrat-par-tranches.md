# Le wrapper `Database` délègue au contrat `Storage`, par tranches sans changement de comportement

Status: accepted.
See also: ADR-0095 (backend remote), ADR-0001, ADR-0006, ADR-0066 (ce qui retient une position),
`pkg/blunderdb/storage/storage.go`, `tasks/BACKLOG.md`

## Context

Le bureau et la CLI parlent au wrapper `pkg/blunderdb/database.Database` ; le démon parle au
contrat `storage.Storage`. Depuis que le wrapper délègue toutes les familles du contrat (voir
l'historique du backlog), il ne reste qu'une poignée de méthodes qui portent encore leur SQL
propre — mais chacune est une deuxième implémentation d'une règle que le contrat tient déjà,
et c'est par elles que GUI/CLI et serveur divergent.

Inventaire au moment de la décision (fichiers `database/db_*.go`, schéma et migrations exclus) :

| Groupe | Méthodes à SQL propre | Équivalent au contrat | Écart observé |
| --- | --- | --- | --- |
| Écritures de match | `DeleteMatch`, `UpdateMatch`, `SaveLastVisitedPosition`, `GetGamesByMatch` | `DeleteCascade`, `Update`, `SetLastVisitedPosition`, `Games` | aucun (même SQL ; `DeleteMatch` portait la troisième copie de `positionIsHeldSQL`) |
| Lectures de match | `GetAllMatches`, `GetLastVisitedMatch` | `List` (opts nulles), `LastVisited` | le wrapper trie la date sentinelle `0001-01-01T00:00:00Z` et `''` comme une absence de date, le contrat seulement `NULL` ; message « no matches in database » contre `ErrNotFound` |
| | `GetPositionProvenance` | `MatchesByPosition` (ids seuls, triés par id) | pas de méthode qui rende les matchs triés par date |
| | `GetMovesByGame` | `Moves` | le contrat note le coup joué (jointure sur l'analyse), le wrapper non |
| | `GetMatchMovePositions` | `MovePositions` | le wrapper ne lit pas `max_cube` : `MaxCube` vaut 0 côté bureau et CLI |
| | `GetDatabaseStats` (repli table par table) | `Metadata().Counts` | repli propre au wrapper |
| Analyses | `saveAnalysisLocked` (fusion), suite de `LoadAnalysis` (coups joués), `DeleteAnalysis` | `Analyses().Save`/`Load`/`Delete` | fusion écrite ailleurs (backlog : quatre copies des helpers de fusion) |
| Lots gammonNet | `CountPositionsWithoutAnalysis`, `positionIDsWithoutAnalysis[ForMatch]`, `db_gammonnet_compare` | `WithoutAnalysisIDs` (sans filtre de match) | — |
| Import | `CheckMatchExists`, `computeMatchHashFromStoredData`, garde d'`ImportFiles`, `db_import_db.go` | `FindByHash`, … | `db_import_db.go` (import d'une autre base) est le gros morceau |
| Fichier SQLite | `db.go` (ouverture, PRAGMA), `db_vacuum.go`, `db_verify.go`, `libraryDir`, `store.FilePath`/`Vacuum`/`VacuumInto` | aucun : propre au fichier local | à garder, derrière une interface étroite |

Le champ `Database.store` est typé `*sqlite.Storage`, pas `storage.Storage` : seules
`FilePath`, `Vacuum` et `VacuumInto` sortent du contrat.

## Decision

1. **Cible.** `Database` ne garde que le cycle de vie d'un fichier local (ouverture, verrou,
   migration, vacuum, vérification) et l'orchestration (`mu`, annulation, jobs). Tout ce qui lit
   ou écrit le contenu d'une bibliothèque passe par `storage.Storage`. Au terme, `store` est typé
   par le contrat, et les opérations de fichier par une interface étroite à part.
2. **Une tranche = un groupe de l'inventaire, sans changement de comportement.** Une méthode
   dont le contrat diffère (tableau ci-dessus, colonne « écart ») n'est pas déléguée en passant :
   l'écart est d'abord tranché — quel comportement est le bon, pour les trois modes — dans sa
   propre tranche, avec un test qui l'épingle.
3. **Ordre des tranches** :
   1. écritures de match (faite) — le prédicat de rétention n'a plus que ses deux copies
      dialectales, `storage/sqlite` et `storage/postgres` ;
   2. lectures de match, écart par écart (date sentinelle, provenance, notation du coup joué,
      `MaxCube`) ;
   3. analyses, avec l'unification des helpers de fusion ;
   4. lots gammonNet et gardes d'import ;
   5. import d'une autre base (`db_import_db.go`), une fois stabilisés l'export et les
      commentaires signés ;
   6. typage de `store` par le contrat et interface de fichier local.
4. **Contraintes de chaque tranche** : `Database.mu` n'est pas réentrant, une méthode
   déléguée prend le verrou une fois puis appelle le store ; le message d'erreur que voient GUI
   et CLI est conservé quand un appelant le lit (validation de date d'`UpdateMatch`) ; toute la
   suite Go reste verte, `-race` sur `database`.

## Consequences

- La première tranche retire la copie de `positionIsHeldSQL` du wrapper ; l'invariant de
  `CLAUDE.md` et `TestPositionIsHeldPredicateParity` comptent deux copies.
- Une fois la tranche 6 faite, le bureau peut ouvrir un tenant distant (`storage/remote`,
  ADR-0095) et tenir plusieurs bibliothèques ouvertes (#689 à #693) en tenant une
  `storage.Storage` par bibliothèque, sans second chemin de code.
- Tant qu'une méthode n'est pas déléguée, son écart reste une divergence de parité connue et
  inscrite au backlog, pas un comportement voulu.
