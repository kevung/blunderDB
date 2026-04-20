# 01 — Backend stats engine (`db_stats.go`)

**Goal:** Implémenter le cœur d'agrégation du panneau Stats : struct de filtre, struct de résultat, et une méthode `ComputeStats` sur `Database` qui exécute ≤ 8 requêtes SQL agrégées et retourne toutes les métriques PR nécessaires aux 3 onglets. Respecte la règle d'agrégation pondérée `sum/sum`.

**Depends on:** rien (first phase)

**Impact:** Critique — fondation utilisée par toutes les fiches 02–11.

## Context

- Schéma v2.3.0, colonnes dénormalisées indexées `analysis.cube_error` / `analysis.best_move_equity_error` (millipoints × 1000), cf. `db.go` CREATE TABLE analysis.
- Pattern `errExpr` existant : `db_search.go:361` → `"CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END"`. À extraire en constante partagée.
- Hiérarchie `tournament → match → game → move → position + analysis`. Jointure player : `match.player{1,2}_name` + `move.player ∈ {0,1}`.
- Convention fichiers : un nouveau fichier `db_stats.go` dans la racine, package `main` (même convention que `db_match.go`, `db_tournament.go`, `db_analysis.go`).
- Tests Go : fixture d'intégration dans `tests/` ou `testdata/` — privilégier `testdata/Quiz.db` s'il contient tournois + matchs + positions analysées, sinon créer une DB dédiée `testdata/stats_fixture.db`.

## Tasks

### 1. Définir les types publics

- [ ] Créer `db_stats.go` avec les structs :
  ```go
  type StatsFilter struct {
      PlayerName    string
      TournamentIDs []int64
      DateFrom      string  // ISO "YYYY-MM-DD"
      DateTo        string
      DecisionType  int     // -1=all, 0=checker, 1=cube
      MatchLength   []int
  }

  type StatsTotals struct {
      NumPositions, NumMatches, NumTournaments, NumDecisions int
  }

  type StatsResult struct {
      Totals              StatsTotals
      PRGlobal            float64
      PRChecker, PRCube   float64
      PRRolling           map[int]float64 // 5,10,50,100,250,500,1000
      PerTournament       []TournamentStats
      PerMatch            []MatchStats
      CubeActionBreakdown []CubeActionStats
      ErrorHistogram      []ErrorBucket
      TopBlunders         []BlunderEntry
  }
  ```
- [ ] Structs auxiliaires `TournamentStats{ID, Name, Date, PR, NumDecisions}`, `MatchStats{ID, Date, PlayerName, PR, NumDecisions}`, `CubeActionStats{Action, PR, NumDecisions, BlunderCount}`, `ErrorBucket{MinMP, MaxMP, Count}`, `BlunderEntry{PositionID, MatchID, TournamentID, ErrorMP, Description}`.
- [ ] MWC fields (`MWCGlobal`, `MWCChecker`, etc.) : ajoutés en fiche 02, **pas ici**.

### 2. Extraire la constante `errExpr`

- [ ] Déclarer dans `db_stats.go` :
  ```go
  const statsErrExpr = "CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END"
  ```
- [ ] Remplacer l'usage inline à `db_search.go:361` par cette constante (refactor pur).

### 3. Helper `buildStatsWhereClause`

- [ ] Fonction `buildStatsWhereClause(filter StatsFilter) (whereSQL string, args []any)` — **privée**, partagée avec la fiche 03.
- [ ] Jointures impliquées : `position p JOIN analysis a ON a.position_id = p.id JOIN move mv ON mv.position_id = p.id JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id LEFT JOIN tournament t ON t.id = m.tournament_id`.
- [ ] Clauses :
  - `PlayerName` → `(m.player1_name = ? AND mv.player = 0) OR (m.player2_name = ? AND mv.player = 1)`
  - `TournamentIDs` → `m.tournament_id IN (?, ?, …)`
  - `DateFrom/DateTo` → `m.match_date BETWEEN ? AND ?`
  - `DecisionType` (≥ 0) → `p.decision_type = ?`
  - `MatchLength` → `m.match_length IN (…)`
- [ ] Tests unitaires : 1 test par clause, 1 test combinaison complète (vérifie `whereSQL` + ordre `args`).

### 4. Implémenter `ComputeStats`

- [ ] Signature :
  ```go
  func (db *Database) ComputeStats(filter StatsFilter) (*StatsResult, error)
  ```
- [ ] Requêtes SQL (≤ 8 roundtrips) :
  1. Totaux globaux : `SELECT COUNT(DISTINCT p.id), COUNT(DISTINCT m.id), COUNT(DISTINCT m.tournament_id), COUNT(*) FROM … WHERE …`
  2. PR global + par `decision_type` : `SELECT decision_type, SUM(err), COUNT(*) FROM … GROUP BY decision_type` → dériver `PRGlobal`, `PRChecker`, `PRCube`.
  3. PR par tournoi : `SELECT m.tournament_id, t.name, t.date, SUM(err), COUNT(*) FROM … GROUP BY m.tournament_id ORDER BY t.date, t.created_at`.
  4. PR par match : `SELECT m.id, m.match_date, SUM(err), COUNT(*) FROM … GROUP BY m.id ORDER BY m.match_date`.
  5. Breakdown cube action : `SELECT a.best_cube_action, SUM(a.cube_error), COUNT(*), SUM(CASE WHEN a.cube_error > BLUNDER_MP THEN 1 ELSE 0 END) FROM … WHERE p.decision_type = 1 GROUP BY a.best_cube_action`. Seuil `BLUNDER_MP = 100` (0.1 EMG).
  6. Histogramme magnitudes : `SELECT CASE … END as bucket, COUNT(*) FROM … GROUP BY bucket`. Buckets : 0-5, 5-10, 10-25, 25-50, 50-100, 100+.
  7. Top blunders : `SELECT p.id, m.id, m.tournament_id, (err) as emg FROM … ORDER BY emg DESC LIMIT 10`.
  8. PR rolling N : `SELECT SUM(err), COUNT(*) FROM (SELECT err FROM … ORDER BY m.match_date DESC, mv.move_number DESC LIMIT N)` pour chaque N ∈ {5,10,50,100,250,500,1000}. En pratique, une seule requête qui charge les N plus récentes, puis sommation en Go pour chaque seuil.
- [ ] **Règle d'agrégation** : chaque PR calculé comme `500 × SUM(err) / 1000 / COUNT(*)` (pondéré). Le PR par tournoi est calculé directement sur les positions du tournoi, **pas dérivé** du PR par match.
- [ ] Helper Go `pr(sumErrMP int64, nDecisions int) float64 { return 500 * float64(sumErrMP) / 1000 / float64(nDecisions) }`.

### 5. Tests unitaires

- [ ] `db_stats_test.go` dans la racine.
- [ ] **Test anti-moyenne-des-moyennes** : fixture avec 2 tournois. Tournoi A = match1 (100 décisions, total err 1000) + match2 (10 décisions, total err 200). PR match1 = 5, PR match2 = 10. Moyenne des PR = 7.5. PR pondéré = 500×1200/1000/110 = 5.45. **Vérifier que `ComputeStats` retourne 5.45, pas 7.5.**
- [ ] Test filtre par joueur : fixture avec 2 matchs, joueur « Alice » en player1 dans un match et player2 dans l'autre. Vérifier que filtrer par « Alice » compte bien les décisions de Alice des deux matchs (jointure `move.player` correcte).
- [ ] Test filtre combiné (player + tournament + date range).
- [ ] Test top blunders ordonnés décroissants.
- [ ] Test histogramme : somme des counts = total décisions.
- [ ] Test performance (benchmark) : `BenchmarkComputeStats` sur une fixture de 10 000+ positions. Budget : < 200 ms.

### 6. Exposer à Wails

- [ ] Ajouter `Bind(db)` dans `main.go` **ne touche rien** — `*Database` est déjà bound, `ComputeStats` sera auto-exposé dans `wailsjs/go/main/Database.d.ts` après `wails dev`.
- [ ] Aucune modif de `main.go` dans cette fiche. La fiche 04 consomme le binding.

## Acceptance criteria

- [ ] `go test -run TestComputeStats ./...` vert (tous les tests listés ci-dessus passent).
- [ ] `go test -run TestBuildStatsWhereClause ./...` vert.
- [ ] `go test -bench BenchmarkComputeStats -benchtime 5s ./...` < 200 ms sur 10 000 positions.
- [ ] `go vet ./...` clean.
- [ ] Pas de régression : `go test ./...` reste vert.
- [ ] Pas de modif de schéma, pas de migration.

## Rollback

Revert = `git revert`. Aucune modif de données.
