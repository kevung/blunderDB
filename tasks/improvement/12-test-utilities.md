# 12 — Extract shared test utilities

**Goal:** Replace 4 independent `setup*DB` helpers with a single shared `test_helpers_test.go` providing common fixtures and setup functions.

**Depends on:** Nothing (can be done at any time).

**Impact:** Low — reduces test boilerplate, makes writing new tests easier.

## Context

Four separate setup helpers exist:

| File | Helper | Line |
|---|---|---|
| `benchmark_test.go` | `setupBenchDB` | L41 |
| `export_test.go` | `setupExportTestDB` | L21 |
| `gnubg_import_test.go` | `setupTestDB` | L14 |
| `search_rewrite_test.go` | `setupSearchTestDB` | L216 |

Each creates an in-memory DB independently. Some also import test fixtures.

## Files touched

- **New:** `test_helpers_test.go`
- **Edit:** `benchmark_test.go`, `export_test.go`, `gnubg_import_test.go`, `search_rewrite_test.go` — replace local helpers with shared ones

## Tasks

### 1. Create `test_helpers_test.go`

- [ ] Create shared helpers:
  ```go
  package main

  import "testing"

  // newTestDB creates a fresh in-memory database with the current schema.
  func newTestDB(t *testing.T) *Database {
      t.Helper()
      db := &Database{}
      if err := db.SetupDatabase(":memory:"); err != nil {
          t.Fatalf("SetupDatabase: %v", err)
      }
      return db
  }

  // newTestDBWithXG creates an in-memory DB and imports testdata/test.xg.
  func newTestDBWithXG(t *testing.T) *Database {
      t.Helper()
      db := newTestDB(t)
      if err := db.ImportXGMatch("testdata/test.xg"); err != nil {
          t.Fatalf("ImportXGMatch: %v", err)
      }
      return db
  }

  // newTestDBWithSGF creates an in-memory DB and imports the test SGF fixture.
  func newTestDBWithSGF(t *testing.T) *Database {
      t.Helper()
      db := newTestDB(t)
      if err := db.ImportGnuBGMatch("testdata/test.sgf"); err != nil {
          t.Fatalf("ImportGnuBGMatch: %v", err)
      }
      return db
  }
  ```
- [ ] Add `t.Helper()` in every helper for correct line reporting
- [ ] Use `t.Fatalf` (not `require`) to avoid introducing a test dependency if not already present

### 2. Migrate existing test files

- [ ] Replace `setupTestDB` in `gnubg_import_test.go` with `newTestDB`
- [ ] Replace `setupExportTestDB` in `export_test.go` with `newTestDBWithXG` (or appropriate variant)
- [ ] Replace `setupBenchDB` in `benchmark_test.go` with `newTestDBWithXG`
- [ ] Replace `setupSearchTestDB` in `search_rewrite_test.go` with appropriate shared helper
- [ ] If any setup helper does extra work beyond DB creation + import, keep that extra work in the test file and call the shared helper for the common part

### 3. Add helper for common assertions

- [ ] Optional: add helpers for common test patterns:
  ```go
  // positionCount returns the number of positions in the database.
  func positionCount(t *testing.T, db *Database) int {
      t.Helper()
      var count int
      err := db.db.QueryRow("SELECT COUNT(*) FROM position").Scan(&count)
      if err != nil {
          t.Fatalf("positionCount: %v", err)
      }
      return count
  }

  // matchCount returns the number of matches in the database.
  func matchCount(t *testing.T, db *Database) int {
      t.Helper()
      var count int
      err := db.db.QueryRow("SELECT COUNT(*) FROM match").Scan(&count)
      if err != nil {
          t.Fatalf("matchCount: %v", err)
      }
      return count
  }
  ```

## Acceptance criteria

- [ ] `test_helpers_test.go` exists with shared DB setup functions
- [ ] At least 3 of the 4 existing setup helpers replaced
- [ ] All tests pass: `go test -count=1 -timeout 300s ./...`
- [ ] New task-05 tests (collections/tournaments/Anki) can use shared helpers

## Rollback

`git revert` — additive file + mechanical call-site changes.
