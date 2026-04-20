# 11 — Extract shared import helpers

**Status: Done**

**Goal:** Reduce duplication across the XG, GnuBG, and BGF import pipelines by extracting common patterns into shared helpers.

**Depends on:** 06 (db.go split — import files are separate).

**Impact:** Medium — eliminates structural duplication across import pipelines.

## What was done

### Previously completed (before this task)

Tasks 1–3 from the original plan were already implemented:

- **`parseMatchDate`** — shared date parser in `db_import_common.go`, used by all 3 importers
- **`checkDuplicateMatchLocked`** — shared duplicate check, used by all 3 importers
- **`moveAnalysisRow` + `insertMoveAnalysisRow`** — shared INSERT helper, used by all 3 importers

### Completed in this pass

1. **`autoLinkTournament(tx, matchID, eventName)`** — extracted 3 identical 15-line tournament find-or-create + match-link blocks into a single shared helper in `db_import_common.go`. Updated all 3 importers.

2. **`cubeAnalysisParams` + `buildDoublingCubeAnalysis(params)`** — extracted the repeated `DoublingCubeAnalysis` struct construction (including `computeBestCubeAction` call and error delta calculation) into a shared builder. Replaced 5 near-identical 20+ line struct literals across:
   - `saveCubeAnalysisToPositionInTx` (XG)
   - `saveCubeAnalysisForCheckerPositionInTx` (XG)
   - `saveGnuBGCubeAnalysisToPositionInTx` (GnuBG)
   - `saveGnuBGCubeAnalysisForCheckerPositionInTx` (GnuBG)
   - `saveBGFCubeAnalysisToPositionInTx` (BGF) — with post-build `BestCubeAction` override for BGF-specific `stateOnMove`/`stateOther` logic
   - `saveBGFCubeAnalysisForCheckerPositionInTx` (BGF)

3. **Simplified XG & BGF `*CubeAnalysisForCheckerPositionInTx`** — removed redundant `SELECT id, data FROM analysis` + `decodeAnalysisFromStorage` that duplicated logic already in `saveAnalysisInTx`. Now matches GnuBG's cleaner approach of building a `PositionAnalysis` and delegating merge to `saveAnalysisInTx`.

### Not extracted (by design)

- **`save*CheckerAnalysisToPositionInTx`** — format-specific move conversion dominates each function (XG: `inferMoveMultipliers` + `convertXGMoveToStringWithHits`; GnuBG: SGF vs MAT coordinate conversion; BGF: `map[string]interface{}` extraction). The shared part is just the `saveAnalysisInTx` call, which is already shared.

## Files touched

- **`db_import_common.go`** — added `autoLinkTournament`, `cubeAnalysisParams`, `buildDoublingCubeAnalysis`; added `log/slog` import
- **`db_import_xg.go`** — used `autoLinkTournament`, `buildDoublingCubeAnalysis`; simplified `saveCubeAnalysisForCheckerPositionInTx`
- **`db_import_gnubg.go`** — used `autoLinkTournament`, `buildDoublingCubeAnalysis`
- **`db_import_bgf.go`** — used `autoLinkTournament`, `buildDoublingCubeAnalysis`; simplified `saveBGFCubeAnalysisForCheckerPositionInTx`

### 5. Extract import progress helper

- [ ] If the three imports report progress in the same way, extract the progress-reporting loop

### 6. Verify no behavioral changes

- [ ] Run all import-related tests:
  ```bash
  go test -run 'TestImport|TestXG|TestGnuBG|TestBGF|TestCanonical' -count=1 -timeout 120s
  ```
- [ ] Run cross-format canonical hash tests
- [ ] Verify import benchmarks are not regressed

## Acceptance criteria

- [ ] `parseMatchDate` used by all 3 import pipelines (single source of truth for date formats)
- [ ] `checkDuplicateMatchLocked` used by all 3 import pipelines
- [ ] `saveMoveAnalysisInTx` has a single implementation (or documented reason for divergence)
- [ ] Total line count of the 4 import files is reduced by ≥200 lines
- [ ] All import tests pass
- [ ] No behavioral changes (same matches import the same way)

## Rollback

`git revert` — shared helpers are additive, and the call sites are straightforward to revert.
