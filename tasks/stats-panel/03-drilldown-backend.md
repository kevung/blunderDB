# 03 — Drill-down backend

**Goal:** Permettre de résoudre n'importe quelle sélection faite dans le panneau Stats (clic sur un point, barre, ligne) en une liste d'IDs de positions, réutilisable par le frontend pour charger ces positions dans la navigation existante. Invariant : « ce qu'on clique = ce qu'on voit » (les mêmes clauses `WHERE` que `ComputeStats`).

**Depends on:** 02

**Impact:** Indispensable pour l'interactivité du panneau ; sans ça, toutes les vues seraient uniquement informatives.

## Context

- Plan §Interactivité & drill-down.
- Pattern existant : `GetAllMatches`, `GetAllTournaments`, `GetTournamentMatches` (panneau Match/Tournament). Ils retournent des enregistrements complets ; ici on veut uniquement des IDs.
- Le frontend consomme `ids []int64` puis charge les positions via le pipeline existant (cf. fiche 04).

## Tasks

### 1. Types publics

- [ ] Étendre `db_stats.go` :
  ```go
  type SelectionSpec struct {
      Kind         string  // "all", "checker", "cube", "cube_action",
                           // "error_bucket", "tournament", "match",
                           // "last_n", "position", "top_blunders"
      CubeAction   string  // "NoDouble" | "DoubleTake" | "DoublePass" | "TooGood"
      BucketMinMP  int     // inclus
      BucketMaxMP  int     // exclus ; -1 = +∞
      TournamentID int64
      MatchID      int64
      LastN        int
      PositionID   int64
      OnlyWithError bool   // pour "cube_action", "checker", "cube" → error > 0
  }
  ```

### 2. Signatures publiques

- [ ] Dans `db_stats.go` :
  ```go
  func (db *Database) GetPositionIDsByStatsSelection(filter StatsFilter, sel SelectionSpec) ([]int64, error)
  func (db *Database) GetPositionIDsByTournament(tournamentID int64) ([]int64, error)
  func (db *Database) GetPositionIDsByMatch(matchID int64) ([]int64, error)
  ```
- [ ] Les deux derniers sont des raccourcis qui délèguent à `GetPositionIDsByStatsSelection` avec un `SelectionSpec` approprié, **mais** ils ignorent `StatsFilter` — ils retournent toutes les positions du tournoi/match quel que soit le filtre (car l'utilisateur a explicitement cliqué sur *Open tournament*/*Open match*, il veut rouvrir l'élément entier). Les raccourcis n'appliquent que les jointures nécessaires.

### 3. Implémentation

- [ ] Réutiliser `buildStatsWhereClause(filter)` de la fiche 01 pour la base.
- [ ] Ajouter un helper `buildSelectionWhereClause(sel SelectionSpec) (sql string, args []any)` qui produit la clause spécifique à la sélection :
  - `"all"` → `""` (pas de clause sup).
  - `"checker"` → `" AND p.decision_type = 0"` (+ `" AND a.best_move_equity_error > 0"` si `OnlyWithError`).
  - `"cube"` → `" AND p.decision_type = 1"` (+ `OnlyWithError`).
  - `"cube_action"` → `" AND p.decision_type = 1 AND a.best_cube_action = ?"` (+ `OnlyWithError`).
  - `"error_bucket"` → `" AND " + statsErrExpr + " >= ? AND " + statsErrExpr + " < ?"` (ou `>= ?` si `BucketMaxMP = -1`).
  - `"tournament"` → `" AND m.tournament_id = ?"`.
  - `"match"` → `" AND m.id = ?"`.
  - `"last_n"` → pas de clause WHERE ; ajouter `ORDER BY m.match_date DESC, mv.move_number DESC LIMIT ?`.
  - `"position"` → `" AND p.id = ?"` (retourne 1 ID).
  - `"top_blunders"` → `ORDER BY " + statsErrExpr + " DESC LIMIT 10`.
- [ ] Fusion finale : `SELECT p.id FROM position p JOIN … WHERE <filter> <selection> <order/limit>` — deux concaténations de clauses, une seule requête.
- [ ] Retour `[]int64` dédupliqué en SQL via `SELECT DISTINCT p.id`.

### 4. Intégration UI vs panneaux natifs

- [ ] `GetPositionIDsByTournament` et `GetPositionIDsByMatch` sont nécessaires pour les actions *Open positions* depuis les graphes de Progression, **indépendamment** du filtre courant. Mais les clics sur les vues Erreurs doivent **conserver** le filtre courant (le dashboard Stats filtré par joueur → le clic sur DoubleTake ne doit charger que les DoubleTake **du joueur filtré**).
- [ ] Convention explicite : les deux raccourcis (`GetPositionIDsByTournament`/`Match`) n'appliquent pas `StatsFilter`. La méthode générique `GetPositionIDsByStatsSelection` l'applique **toujours**. Le frontend choisit.

### 5. Tests unitaires

- [ ] `db_stats_drilldown_test.go`.
- [ ] **Test invariant "ce qu'on clique = ce qu'on voit"** : pour un filtre + sélection donnée, `COUNT(GetPositionIDsByStatsSelection(filter, sel)) == TotalFromComputeStats(filter, sel)`. Décliné sur :
  - sélection "cube_action" → doit correspondre au `CubeActionStats.NumDecisions` du bon item.
  - sélection "error_bucket" → correspond au `ErrorBucket.Count`.
  - sélection "checker" → correspond à `result.TotalChecker`.
- [ ] **Test `OnlyWithError`** : sélection "cube_action=DoubleTake, OnlyWithError=true" ne retourne que les positions avec `cube_error > 0`. Vérifier avec une fixture contenant quelques DoubleTake corrects (erreur 0) et quelques DoubleTake avec erreur.
- [ ] **Test raccourci tournoi** : `GetPositionIDsByTournament(id)` ignore bien le filtre courant (vérifier avec un filtre bidon qui exclurait tout si appliqué).
- [ ] **Test top blunders** : `GetPositionIDsByStatsSelection(filter, {Kind: "top_blunders"})` retourne ≤ 10 IDs, ordre cohérent avec `result.TopBlunders`.

## Acceptance criteria

- [ ] `go test -run TestDrilldown ./...` vert (tous les tests listés).
- [ ] `go test ./...` reste vert (pas de régression).
- [ ] `go vet ./...` clean.
- [ ] Les helpers `buildStatsWhereClause` / `buildSelectionWhereClause` sont privés et bien testés séparément.

## Rollback

Revert = `git revert`. Fonctions additives, rien de destructif.
