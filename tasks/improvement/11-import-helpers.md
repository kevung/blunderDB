# 11 — Extract shared import helpers

**Goal:** Reduce duplication across the XG, GnuBG, and BGF import pipelines by extracting common patterns into shared helpers.

**Depends on:** 06 (db.go split — import files are separate).

**Impact:** Medium — eliminates ~21 parallel functions worth of structural duplication.

## Context

The three import pipelines share these patterns (each duplicated 3–4 times):

| Pattern | Current duplication |
|---|---|
| Date parsing with format fallback | Copy-pasted 4 times |
| Match hash check + canonical hash check | 3 identical sequences |
| Transaction setup + commit/rollback | 3 identical blocks |
| `saveMoveAnalysisInTx` (move_analysis INSERT) | 3 near-identical functions |
| `saveCubeAnalysisForCheckerPositionInTx` | 3 parallel functions |
| Import progress reporting | 3 identical patterns |

## Files touched

- **Edit:** `db_import_common.go` (post-split) — add new shared helpers
- **Edit:** `db_import_xg.go` — use shared helpers
- **Edit:** `db_import_gnubg.go` — use shared helpers
- **Edit:** `db_import_bgf.go` — use shared helpers

## Tasks

### 1. Extract `parseMatchDate` helper

- [ ] Create shared date parser in `db_import_common.go`:
  ```go
  // parseMatchDate tries multiple date formats and returns the parsed time.
  // Returns zero time if no format matches.
  func parseMatchDate(dateStr string) time.Time {
      formats := []string{
          "2006-01-02",
          "2006-01-02 15:04:05",
          "01/02/2006",
          "1/2/2006",
          "2006/01/02",
          "January 2, 2006",
          "Jan 2, 2006",
          "02-Jan-2006",
          "2006-01-02T15:04:05Z07:00",
          time.RFC3339,
      }
      for _, f := range formats {
          if t, err := time.Parse(f, dateStr); err == nil {
              return t
          }
      }
      return time.Time{}
  }
  ```
- [ ] Replace the 4 copy-pasted date-parsing loops in XG, GnuBG, and BGF imports

### 2. Extract `checkDuplicateMatchLocked` helper

- [ ] Create shared duplicate check in `db_import_common.go`:
  ```go
  // checkDuplicateMatchLocked checks both format-specific and canonical hash
  // for duplicate matches. Must be called with d.mu held.
  // Returns (existingMatchID, isCanonicalDuplicate, error).
  func (d *Database) checkDuplicateMatchLocked(
      matchHash, canonicalHash string,
  ) (int64, bool, error)
  ```
- [ ] Replace the 3 identical check sequences in `ImportXGMatch`, `ImportGnuBGMatch`, `ImportBGFMatch`

### 3. Unify `saveMoveAnalysisInTx` functions

- [ ] The three functions (`saveMoveAnalysisInTx`, `saveGnuBGMoveAnalysisInTx`, `saveBGFMoveAnalysisInTx`) all INSERT into `move_analysis` with the same columns
- [ ] Create a shared `saveMoveAnalysisInTx`:
  ```go
  type moveAnalysisRow struct {
      MoveID               int64
      AnalysisType         string
      Depth                string
      Equity               int64
      EquityError          int64
      WinRate              int64
      GammonRate           int64
      BackgammonRate       int64
      OpponentWinRate      int64
      OpponentGammonRate   int64
      OpponentBackgammonRate int64
  }

  func saveMoveAnalysisInTx(tx *sql.Tx, row moveAnalysisRow) error
  ```
- [ ] Keep format-specific conversion logic in each import file, but the actual INSERT is shared
- [ ] Each import file converts its format-specific data into `moveAnalysisRow` then calls the shared function

### 4. Unify `saveCubeAnalysisForCheckerPositionInTx` pattern

- [ ] The three versions share the same logic: look up existing analysis, merge cube info, save back
- [ ] If the merge logic is identical, extract; if format-specific, document why they differ

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
