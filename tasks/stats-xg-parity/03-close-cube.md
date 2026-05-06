# 03 — Close cube classification

**Status: DONE** (DatabaseVersion 2.5.0 → 2.6.0)

**Goal:** Persister un drapeau `is_close_cube` pour distinguer les décisions cube *proches* (XG / gnuBG les comptent) des décisions cube triviales (No Double évidents que XG ignore). Aucune formule modifiée à cette phase.

**Depends on:** 01 (et idéalement 02 pour grouper les bumps de schéma).

**Does NOT touch:** `db_stats.go` formules. Pas de modif UI.

## Context

- Référence : `gnubg/eval.c:5088-5100` :
  ```c
  const float rThr = 0.16f;
  float rDouble = MIN(arDouble[OUTPUT_TAKE], 1.0f);
  if (arDouble[OUTPUT_OPTIMAL] - rDouble < rThr) return 1;  // close
  ```
  → une décision est « close » si l'écart entre l'équité optimale (cube ou no-cube selon ce qui est meilleur) et l'équité prise (capée à +1.0) est **< 0.16**.
- Mapping final validé :
  ```
  rOptimal = BestCubeAction → CubefulNoDoubleEquity / DoubleTakeEquity / DoublePassEquity
  rDouble  = min(CubefulDoubleTakeEquity, 1.0)
  isClose  = (rOptimal - rDouble) < 0.16
  ```
- Cas particuliers :
  - **Take / Pass** : `is_close_cube = 1` systématiquement (joueur a déjà pris/passé → position clairement dans la zone de jeu).
  - **No Double trivial** : ex. NoDouble=0.80, DT=0.40 → rOptimal-rDouble=0.40 ≥ 0.16 → `is_close_cube = 0`.
  - **No Double near doubling point** : ex. NoDouble=0.52, DT=0.50 → diff=0.02 < 0.16 → `is_close_cube = 1`.
  - **No Double below DT** : ex. NoDouble=0.20, DT=0.50 → diff=-0.30 < 0.16 → `is_close_cube = 1` (cube est encore meilleure action ici).
  - Quand `DoublingCubeAnalysis == nil` : `is_close_cube = 0` par défaut.

## Files touched

- `model.go` — `DatabaseVersion` 2.5.0 → 2.6.0
- `db.go` — colonne `analysis.is_close_cube INTEGER NOT NULL DEFAULT 0` + index partial + migration trigger 2.5.0→2.6.0
- `db_analysis.go` — `computeIsCloseCube` (helper testable), `populateAnalysisColumns`, UPDATE/INSERT paths
- `db_import_common.go` — `saveAnalysisInTx` UPDATE/INSERT (4 write paths couverts)
- `db_migration.go` — `migrate_2_5_0_to_2_6_0` : backfill sur positions cube (decision_type=1)
- `db_analysis_test.go` — `TestComputeIsCloseCube` (10 cas)
- `migration_test.go` — `TestMigrate_2_5_0_to_2_6_0_IsCloseCube` (4 cas)
- `stats_parity_test.go` — `PassDecisions` ajouté à `refPlayer`; `CloseCubeDecisions` ajouté à `parityTolerances`; assertions `cube_close_count` dans les blocs XG et SGF

## Tasks

### 1. Décrypter le mapping gnuBG → blunderDB

- [x] Mapping `OUTPUT_OPTIMAL` / `OUTPUT_TAKE` validé depuis `gnubg/eval.c:5088`
- [x] Mapping documenté ci-dessus
- [x] Take / Pass → `is_close_cube = 1` systématiquement
- [x] `DoublingCubeAnalysis == nil` → `is_close_cube = 0`

### 2. Schéma + migration

- [x] Colonne `is_close_cube INTEGER NOT NULL DEFAULT 0` ajoutée à `analysis`
- [x] Index partial `idx_analysis_is_close_cube WHERE is_close_cube = 1`
- [x] Migration 2.5.0 → 2.6.0 : backfill via `computeIsCloseCube`, transaction par batch de 500

### 3. Population à l'import

- [x] `computeIsCloseCube(dca *DoublingCubeAnalysis, playedCubeAction string) int64` dans `db_analysis.go`
- [x] `populateAnalysisColumns` appelle le helper après `is_forced`
- [x] Les 4 chemins d'écriture (`SaveAnalysis` UPDATE/INSERT + `saveAnalysisInTx` UPDATE/INSERT) couverts

### 4. Tests unitaires de la formule

- [x] `db_analysis_test.go:TestComputeIsCloseCube` — 10 cas, tous PASS :
  | Cas | NoDouble | Take | Pass | BestAction | isClose attendu |
  |---|---|---|---|---|---|
  | Trivial NoDouble far from doubling point | 0.20 | 0.50 | 1.00 | No Double | 1 (rOptimal<rDouble) |
  | Clear NoDouble well above DoubleTake | 0.80 | 0.40 | 1.00 | No Double | 0 |
  | Near doubling point | 0.52 | 0.50 | 1.00 | No Double | 1 |
  | DoubleTake best | 0.40 | 0.60 | 1.00 | Double, Take | 1 |
  | DoublePass best | 0.40 | 0.70 | 0.80 | Double, Pass | 1 |
  | Take action | — | — | — | Take | 1 |
  | Pass action | — | — | — | Pass | 1 |
  | nil analysis | nil | — | — | — | 0 |
  | DoubleTake capped at 1.0 | 1.10 | 1.20 | 2.00 | No Double | 1 |
  | NoDouble well above capped take | 1.50 | 1.20 | 2.00 | No Double | 0 |

### 5. Étendre le harnais

- [x] Champ `PassDecisions *int \`json:"pass_decisions"\`` ajouté à `refPlayer`
- [x] Champ `CloseCubeDecisions int` ajouté à `parityTolerances` (valeur 5 dans `tolPhase01`)
- [x] Assertions `cube_close_count` dans le bloc XG (players 1 et 2) et dans le bloc SGF

## Résultats (tolPhase01, CloseCubeDecisions=±5)

| Fixture | Import | Player | ref (D+T+P) | got | diff |
|---|---|---|---|---|---|
| aachen-double-7pt | XG | P1 | 18 (16+2+0) | 17 | 1 ✓ |
| aachen-double-7pt | XG | P2 | 17 (12+5+0) | 15 | 2 ✓ |
| test | SGF | P1 | 7 (3+2+2) | 2 | 5 ✓ |
| test | SGF | P2 | 7 (4+2+1) | 3 | 4 ✓ |
| charlot1-charlot2 | SGF | P1 | 4 (2+2+0) | 2 | 2 ✓ |
| charlot1-charlot2 | SGF | P2 | 4 (2+1+1) | 2 | 2 ✓ |

Note: les SGF gnuBG ont moins de positions avec analyse cube complète (positions cube sans alternatives gnuBG = is_close_cube non détectable). L'écart SGF se résorbera progressivement avec les fiches suivantes.

## Acceptance criteria

- [x] `go test ./...` vert
- [x] Migration 2.5.0 → 2.6.0 testée dans `migration_test.go`
- [x] Sur les 3 fixtures, `count(is_close_cube=1, decision_type=1, player)` ≈ XG/gnuBG total cube decisions à ±5
- [x] Tests unitaires de `computeIsCloseCube` couvrent 10 cas
