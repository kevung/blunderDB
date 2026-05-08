# 02 — Forced moves classification

**Goal:** Persister un drapeau `is_forced` pour chaque décision checker afin de pouvoir exclure les coups forcés des compteurs et du PR (alignement XG / gnuBG). Aucune formule de stats n'est encore modifiée — fiche 04 fera le câblage.

**Depends on:** 01 (le harnais doit déjà tourner).

**Does NOT touch:** `db_stats.go`, formules PR/MWC. Pas de modif UI.

**Status: DONE** ✓

## Context

- Référence gnuBG : `gnubg/analysis.c:458-462` accumule l'erreur checker uniquement quand `pmr->ml.cMoves > 1`.
- Côté blunderDB, l'analyse XG/gnuBG/MAT importée stocke `CheckerAnalysis.Moves[]` (`model.go:235`). `len(Moves) == 1` ⇒ forcé (un seul candidat légal). `len(Moves) == 0` ⇒ aucune analyse, ne compte pas comme forcé.
- Le parser gnuBG expose des compteurs agrégés `Unforced[2]` / `Forced[2]` (`gnubgparser/types.go:171-172`) — utilisés pour la validation croisée dans le harnais.
- **Gotcha gnuBG** : le gnuBG importer appelle `saveGnuBGCubeAnalysisForCheckerPositionInTx` qui ajoute `DoublingCubeAnalysis` à tous les checker positions (pour afficher les infos cube en appuyant sur 'd'). On ne peut donc pas utiliser `DoublingCubeAnalysis == nil` comme proxy du type de décision. Le prédicat retenu : `CheckerAnalysis != nil && len(Moves) == 1` (sans guard sur DoublingCubeAnalysis).
- **Gap structurel SGF** : gnuBG n'enregistre pas d'alternatives pour les positions forcées dans le SGF (≈20 positions P2 dans test.sgf n'ont pas de ligne `analysis`). Ces positions sont déjà exclues du harnais par la jointure interne. Le compte `is_forced=1` est donc inférieur au vrai compte gnuBG pour P2 test.sgf — toléré à ±45 (même que `CheckerDecisions`).

## Files touched

- `model.go` — bump `DatabaseVersion`: `"2.4.0"` → `"2.5.0"`.
- `db.go` — ajout colonne `is_forced INTEGER NOT NULL DEFAULT 0` au CREATE TABLE `analysis` ; index partiel `idx_analysis_is_forced WHERE is_forced = 1` ; migration trigger `2.4.0 → 2.5.0`.
- `db_analysis.go` — struct `analysisColumns.IsForced int64` ; calcul dans `populateAnalysisColumns` ; `is_forced` dans INSERT et UPDATE de `SaveAnalysis`.
- `db_import_common.go` — `is_forced` dans INSERT et UPDATE de `saveAnalysisInTx` (path import).
- `db_migration.go` — `migrate_2_4_0_to_2_5_0` : ALTER TABLE + index + backfill par batch dans transaction.
- `migration_test.go` — `TestMigrate_2_4_0_to_2_5_0_IsForced` : 3 cas (forcé, non-forcé, cube).
- `stats_parity_test.go` — `importAndStats` retourne maintenant `(matchID, db)` ; `countForcedChecker` helper SQL ; assertions `checker_forced_count` sur les 3 fixtures.

## Tasks

### 1. Schéma + migration

- [x] Dans `db.go`, ajouter à la définition `CREATE TABLE IF NOT EXISTS analysis` :
  ```sql
  is_forced INTEGER NOT NULL DEFAULT 0
  ```
- [x] Index : `CREATE INDEX IF NOT EXISTS idx_analysis_is_forced ON analysis(is_forced) WHERE is_forced = 1`.
- [x] Étape de migration `2.4.0 → 2.5.0` dans `db_migration.go:migrate_2_4_0_to_2_5_0` :
  1. `ALTER TABLE analysis ADD COLUMN is_forced INTEGER NOT NULL DEFAULT 0` (idempotent).
  2. `CREATE INDEX IF NOT EXISTS idx_analysis_is_forced …`.
  3. Backfill par batch (1000 rows) dans une transaction : join `analysis JOIN position WHERE decision_type = 0`, décompresser JSON, `is_forced=1` si `len(CheckerAnalysis.Moves) == 1`.
- [x] Bump `DatabaseVersion` dans `model.go` : `"2.4.0"` → `"2.5.0"`.

### 2. Population à l'import

- [x] `populateAnalysisColumns` (`db_analysis.go`) :
  ```go
  if a.CheckerAnalysis != nil && len(a.CheckerAnalysis.Moves) == 1 {
      c.IsForced = 1
  }
  ```
  (sans guard `DoublingCubeAnalysis == nil` — cf. gotcha gnuBG ci-dessus).
- [x] `is_forced` inclus dans l'INSERT et UPDATE de `SaveAnalysis` (`db_analysis.go`).
- [x] `is_forced` inclus dans l'INSERT et UPDATE de `saveAnalysisInTx` (`db_import_common.go`).

### 3. Cohérence parsers

- [x] **gnuBG (XG import)** : `count(is_forced=1, dec_type=0, P1)` = 19 / `count(P2)` = 38 → exact match avec gnuBG ref (tol=45).
- [x] **gnuBG (SGF import)** : P1=19 exact, P2=18 (20 positions sans ligne analysis dans le SGF). Gap documenté, tolérance = CheckerDecisions (45).
- [x] **XG** : contrôle indirect via harnais (checker_unforced devra correspondre après fiche 04).

### 4. Test migration

- [x] `TestMigrate_2_4_0_to_2_5_0_IsForced` dans `migration_test.go` :
  - forced checker (1 move) → `is_forced=1` ✓
  - unforced checker (2 moves) → `is_forced=0` ✓
  - cube decision → `is_forced=0` ✓

### 5. Harnais étendu

- [x] `importAndStats` retourne `(player1Name, player2Name string, matchID int64, db *Database, s *MatchDetailStats)`.
- [x] `countForcedChecker(db, matchID, player)` : COUNT via SQL join (analysis + position + move + game).
- [x] Assertions `checker_forced_count` sur XG (tol=CheckerDecisions) et SGF (tol=CheckerDecisions).

## Résultats observés (phase 02)

| Fixture | Source | Player | ref forced | bDB forced | diff |
|---|---|---|---|---|---|
| test | XG→gnuBGref | P1 | 19 | 19 | 0 ✓ |
| test | XG→gnuBGref | P2 | 38 | 38 | 0 ✓ |
| test | SGF | P1 | 19 | 19 | 0 ✓ |
| test | SGF | P2 | 38 | 18 | 20 (gap structurel SGF) |
| charlot | SGF | P1 | 8 | 8 | 0 ✓ |
| charlot | SGF | P2 | 29 | 20 | 9 (gap structurel SGF) |

## Acceptance criteria

- [x] `go test ./...` vert.
- [x] Migration testée : DB v2.4.0 → v2.5.0 sans perte de données, colonne backfillée.
- [x] Sur les 3 fixtures, `count(is_forced=1, decision_type=0, player)` dans la tolérance.
- [x] Sur la fixture `test.xg` (import XG), forced count P1=19, P2=38 (exact, via gnuBG ref).
- [x] Aucune régression sur les tests existants.

## Notes importantes

- `len(Moves) == 1` (non `<= 1`) : si `len == 0`, pas d'analyse donc pas de décision comptée → ne marquer pas comme forcé.
- Les 4 chemins d'écriture analysis ont été mis à jour : `SaveAnalysis` UPDATE, `SaveAnalysis` INSERT, `saveAnalysisInTx` UPDATE, `saveAnalysisInTx` INSERT.
- Le gap structurel SGF (positions forcées sans ligne analysis) sera comblé si on découple le comptage des positions de l'analyse — hors scope immédiat, documenté dans fiche 07.
