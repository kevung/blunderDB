# 06 — Split `db.go` into domain-focused files

**Goal:** Decompose `db.go` (16,223 lines, 174 methods, 282 functions) into ~18 focused files. Pure mechanical split — no refactoring, no API changes, no behavior changes.

**Depends on:** Phase 1 complete (CI runs tests to catch regressions).

**Impact:** Critical — single biggest maintainability improvement.

## Rules

1. **Pure file split.** Move functions; do not rename, refactor, or change signatures.
2. **All files stay `package main`** (Wails binding requirement).
3. **One sub-PR per 2–3 files** to keep diffs reviewable.
4. **Run `go test ./...` after each file move** — no step leaves a red suite.
5. **Preserve `//go:build` tags** if any exist.

## File plan

Each row below defines a destination file, the functions/methods to move, and the approximate line range in the current `db.go`.

| # | Destination file | Content | Source lines (approx.) | Est. size |
|---|---|---|---|---|
| A | `db.go` (keep) | `Database` struct, `NewDatabase`, `SetupDatabase`, `applyPragmas`, `OpenDatabase` | L1–40, L763–1205 | ~500 |
| B | `db_schema.go` | `ensureAllTablesExist` | L1702–2027 | ~350 |
| C | `db_migration.go` | `SetMigrationProgress`, `emitMigrationProgress`, all `migrate_X_to_Y` functions | L114–762 | ~650 |
| D | `db_position.go` | Position CRUD (`PositionExists`, `SavePosition`, `UpdatePosition`, `LoadPosition`, `LoadAllPositions`, `DeletePosition`), compact encoding (`encodeBoardCompact`, `decodeBoardCompact`, `isCompactState`, `reconstructPosition`), scan helpers (`positionSelectCols`, `scanPositionRow`, `fullPositionJSON`), `NormalizeForStorage`, `Mirror`, column populators | L2028–2838 + L2114–2181 + L2260–2370 + L5293–5323 | ~1,200 |
| E | `db_analysis.go` | Compression helpers (`compressAnalysisData`, `decompressAnalysisData`, `encodeAnalysisForStorage`, `decodeAnalysisFromStorage`, `recompressAnalysisData`), analysis CRUD (`SaveAnalysis`, `LoadAnalysis`, `DeleteAnalysis`), analysis column populators, rounding helpers (`roundToMillipoint`, `roundToHundredthPercent`, `roundAnalysisForStorage`) | L36–113 + L2188–2257 + L2454–2658 + L3997–4054 | ~800 |
| F | `db_search.go` | `parseIntFilterExpr`, `parseFloatFilterExpr`, `appendIntRangeSQL`, `appendFloatRangeSQL`, `hasBoardFilter`, `analysisMatchesFloatFilter`, `analysisMatchesEquityFilter`, `analysisMatchesMovePattern`, `loadPositionsByFiltersCore`, `LoadPositionsByFilters`, `parseFilterIDList`, `getPositionIDsForMatch`, `getMatchIDsForTournament` | L3053–3907 | ~900 |
| G | `db_filter_match.go` | All `Matches*` methods on Position (`MatchesPipCountFilter`, `MatchesWinRate`, `MatchesDiceRoll`, `MatchesNoContact`, etc.) | L3909–5288 | ~1,400 |
| H | `db_import_common.go` | `importCache`, `savePositionInTxWithCache`, `findOrCreatePositionForCanonicalDuplicate`, `savePositionInTx`, `saveAnalysisInTx`, `mergeCheckerMoves`, `mergePlayedMoves`, `normalizeMove`, `inferMoveMultipliers`, `enginePriority`, `sortCubeAnalysesByEngine`, `determineBestCubeAction`, match hash functions (`ComputeMatchHash`, `computeMatchHashFromStoredData`, `CheckMatchExists`, `CheckMatchExistsLocked`, `checkCanonicalMatchExistsLocked`, `ErrDuplicateMatch`), canonical hash helpers | L8405–9755 + L9758–9893 + L11500–11590 | ~1,000 |
| I | `db_import_xg.go` | `ImportXGMatch`, `RawCubeAction`, `importXGGamesAndMoves`, `importGame`, `importMoveWithCache`, `importMoveWithCacheAndRawCube`, `convertRawCubeAction`, `importMove`, `createPositionFromXG`, `convertXGPlayerToBlunderDB`, `convertBlunderDBPlayerToXG`, `saveMoveAnalysisInTx`, `saveCubeAnalysisInTx`, `saveCheckerAnalysisToPositionInTx`, `saveCubeAnalysisToPositionInTx`, `saveCubeAnalysisForCheckerPositionInTx`, `translateAnalysisDepth`, `convertXGMoveToString`, `convertXGMoveToStringWithHits`, `mergeSlidesWithHits`, `mergeSlides`, `convertCubeAction`, `ComputeCanonicalMatchHashFromXG`, `ImportXGPPosition` | L7385–9188 + L9356–9757 + L11507–11540 + L13229–13323 | ~2,300 |
| J | `db_import_gnubg.go` | `ImportGnuBGMatchFromText`, `ImportGnuBGMatch`, `importGnuBGMatchInternal`, `importGnuBGGame`, `importGnuBGMove`, `importGnuBGCheckerMove`, `importGnuBGCubeMove`, `initStandardGnuBGPosition`, `applyGnuBGCheckerMove`, `createPositionFromGnuBG`, `convertGnuBGMoveToString`, `convertPlayerRelativeMoveToString`, `formatGnuBGMoveItems`, `saveGnuBGMoveAnalysisInTx`, `saveGnuBGCheckerAnalysisToPositionInTx`, `saveGnuBGCubeAnalysisToPositionInTx`, `saveGnuBGCubeAnalysisForCheckerPositionInTx`, `translateGnuBGAnalysisDepth`, `ComputeGnuBGMatchHash`, `ComputeCanonicalMatchHashFromGnuBG` | L9900–11496 | ~1,600 |
| K | `db_import_bgf.go` | `ImportBGFMatch`, `importBGFCheckerMove`, `importBGFCubeMove`, `importBGFCheckerAnalysisOnly`, `importBGFCubeAnalysisOnly`, `createPositionFromBGF`, `bgfInitBoardFromGame`, `bgfApplyCheckerMove`, `bgfConvertMoveToString`, `saveBGFMoveAnalysisInTx`, `saveBGFCheckerAnalysisToPositionInTx`, `saveBGFCubeMoveAnalysisInTx`, `saveBGFCubeAnalysisToPositionInTx`, `saveBGFCubeAnalysisForCheckerPositionInTx`, `bgfConvertAnalysisMoveToString`, `translateBGFAnalysisDepth`, `ComputeBGFMatchHash`, `ComputeCanonicalMatchHashFromBGF`, `ImportBGFPosition`, `ImportBGFPositionFromText`, `saveBGFPositionWithAnalysis`, `convertBGFTextPosition`, BGF helper functions | L11597–13575 | ~2,000 |
| L | `db_export.go` | `ExportDatabase`, `ExportCollections`, `ExportTournaments`, `DeleteFile` | L6559–7384 + L14550–14882 + L15345–15765 | ~1,600 |
| M | `db_import_db.go` | `AnalyzeImportDatabase`, `CommitImportDatabase`, `ImportDatabase`, `CancelImport`, `isImportCancelled`, `resetImportCancellation` | L6040–6563 | ~530 |
| N | `db_match.go` | `GetAllMatches`, `GetMatchByID`, `SaveLastVisitedPosition`, `GetLastVisitedMatch`, `GetGamesByMatch`, `GetMovesByGame`, `DeleteMatch`, `GetMatchMovePositions`, `GetDatabaseStats`, `UpdateMatch`, `SwapMatchPlayers`, `IsPlayer1TakePassCubeAction`, `getPlayer1MovesForPosition`, `MatchesMoveErrorFilter`, `MatchesDateFilter` | L13577–13954 + L4764–4819 + L4823–4928 + L5030–5078 + L15168–15264 | ~1,000 |
| O | `db_collection.go` | All 14 collection methods (`CreateCollection` through `GetPositionCollections`) | L14030–14548 | ~520 |
| P | `db_tournament.go` | All 14 tournament methods (`CreateTournament` through `GetTournamentMatches`) | L14883–15344 | ~470 |
| Q | `db_anki.go` | All 13 Anki methods (`CreateAnkiDeck` through `ResetAnkiDeck`) | L15766–16223 | ~460 |
| R | `db_met.go` | MET tables, `gnuBGInitPreCrawfordMET`, `gnuBGInitPostCrawfordMET`, `gnuBGOverlayKazarossXG2`, `gnuBGGetMETEntry`, `gnuBGGetCubePrimeValue`, `gnuBGGetME`, `convertGnuBGCubeMWCToEMG`, Kazaross-XG2 data arrays | L10846–11189 | ~350 |
| S | `db_comment.go` | `AddComment`, `UpdateCommentEntry`, `DeleteCommentEntry`, `SaveComment`, `LoadComment`, `commentTableHasTimestamps`, `GetCommentsByPosition`, `GetAllComments`, `SearchComments` | L2840–3052 | ~220 |
| T | `db_session.go` | `SaveCommand`, `LoadCommandHistory`, `SearchHistory`, `SaveSearchHistory`, `LoadSearchHistory`, `DeleteSearchHistoryEntry`, `SessionState`, `SaveSessionState`, `LoadSessionState`, `ClearSessionState`, `SaveFilter`, `LoadFilters`, old migrations (1.0→1.3) | L5325–5931 | ~610 |

### Execution order

Split in this order to minimize cross-dependency conflicts:

1. **Self-contained leaf domains:** `db_comment.go`, `db_anki.go`, `db_collection.go`, `db_tournament.go`, `db_met.go`
2. **Session/history:** `db_session.go`
3. **Utilities and helpers:** `db_analysis.go` (compression), `db_filter_match.go`
4. **Position core:** `db_position.go`
5. **Import common:** `db_import_common.go`
6. **Import pipelines:** `db_import_xg.go`, `db_import_gnubg.go`, `db_import_bgf.go`
7. **Search:** `db_search.go`
8. **Export:** `db_export.go`, `db_import_db.go`
9. **Match:** `db_match.go`
10. **Schema/migration:** `db_schema.go`, `db_migration.go`
11. **Final cleanup:** verify `db.go` is ~500 lines

## Tasks per sub-PR

### Sub-PR 1: Leaf CRUD domains
- [x] Create `db_comment.go` — move comment functions
- [x] Create `db_anki.go` — move Anki functions
- [x] Create `db_collection.go` — move collection functions
- [x] Create `db_tournament.go` — move tournament functions
- [x] Create `db_met.go` — move MET functions + data arrays
- [x] `go test ./...` passes

### Sub-PR 2: Session + helpers
- [x] Create `db_session.go` — move session/history/filter-library functions
- [x] Create `db_filter_match.go` — move all `Matches*` methods
- [x] `go test ./...` passes

### Sub-PR 3: Core data
- [x] Create `db_analysis.go` — move compression + analysis CRUD + rounding
- [x] Create `db_position.go` — move position CRUD + compact encoding
- [x] `go test ./...` passes

### Sub-PR 4: Import pipeline
- [x] Create `db_import_common.go` — move shared import infrastructure
- [x] Create `db_import_xg.go` — move XG import pipeline
- [x] Create `db_import_gnubg.go` — move GnuBG import pipeline
- [x] Create `db_import_bgf.go` — move BGF import pipeline
- [x] `go test ./...` passes

### Sub-PR 5: Search + export + schema
- [x] Create `db_search.go` — move search/filter functions
- [x] Create `db_export.go` — move export functions
- [x] Create `db_import_db.go` — move DB-to-DB import
- [x] Create `db_match.go` — move match CRUD
- [x] `go test ./...` passes

### Sub-PR 6: Schema + migration + final
- [x] Create `db_schema.go` — move `ensureAllTablesExist`
- [x] Create `db_migration.go` — move all migration functions
- [x] Verify `db.go` is ~951 lines (includes core setup/open logic)
- [x] `go test ./...` passes
- [x] `go build` succeeds

## Acceptance criteria

- [x] `db.go` ≤ 951 lines (core setup/open/pragmas logic)
- [x] No file exceeds ~2,500 lines (largest: db_import_bgf.go at 1,998)
- [x] All tests pass
- [x] `go build` succeeds
- [x] Pure file split, no logic changes
- [x] Every new file has the same `package main` declaration and necessary imports

## Rollback

`git revert` per sub-PR. Since this is a pure split with no behavior changes, reverting any sub-PR recombines the functions back into `db.go`.
