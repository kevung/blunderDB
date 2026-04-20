# 04 — CLI round-trip integration tests

**Goal:** The CLI pipeline (`cli.go`, 2,112 lines, 30 methods) has test coverage for its core workflows: import → list → search → export.

**Depends on:** Nothing.

**Impact:** High — `cli.go` is a user-facing entry point with zero test coverage.

## Context

- `CLI` struct (cli.go:42): `{ db *Database; cfg *Config }`
- 30 methods, dispatched by `CLI.Run()` via `os.Args[1]`
- Subcommands: `create`, `import`, `export`, `list`, `delete`, `match`, `verify`, `info`, `edit`, `search`, `help`, `version`
- Tests can instantiate `CLI` directly with an in-memory DB — no need to spawn a subprocess
- Existing fixture files: `testdata/test.xg`, `testdata/test.sgf`, `testdata/*.mat`

## Files touched

- **New:** `cli_test.go`

## Tasks

### 1. Test infrastructure

- [x] Create `cli_test.go` in the repo root (`package main`)
- [x] Helper to create a `CLI` with in-memory DB:
  ```go
  func setupCLI(t *testing.T) *CLI {
      t.Helper()
      db := &Database{}
      require.NoError(t, db.SetupDatabase(":memory:"))
      cfg := &Config{}
      return &CLI{db: db, cfg: cfg}
  }
  ```
- [x] Helper to capture stdout/stderr during CLI method calls (redirect `os.Stdout`/`os.Stderr` to pipes or use `bytes.Buffer`)

### 2. Import round-trip test

- [x] `TestCLI_ImportXG`: Import `testdata/test.xg` via `cli.importMatch()`, verify match count > 0, verify position count > 0
- [x] `TestCLI_ImportSGF`: Import a `.sgf` file, verify match/position creation
- [x] `TestCLI_ImportMAT`: Import a `.mat` file if fixture exists
- [x] `TestCLI_ImportDuplicate`: Import same file twice, verify duplicate detection (second import fails or is skipped)

### 3. List/stats tests

- [x] `TestCLI_ListMatches`: Import a match, call `cli.listMatches()`, verify output contains player names
- [x] `TestCLI_ListPositions`: Import a match, call `cli.listPositions()`, verify output contains position count
- [x] `TestCLI_ShowStats`: Import a match, call `cli.showStats()`, verify output contains match count and position count

### 4. Search test

- [x] `TestCLI_Search`: Import a match, call `cli.runSearch()` with a basic filter (e.g. decision type = checker), verify results returned
- [x] `TestCLI_SearchNoResults`: Search with impossible filter, verify zero results

### 5. Export test

- [x] `TestCLI_Export`: Import a match, export to a temp file via `cli.exportDatabase()`, verify the exported file is a valid SQLite database
- [x] `TestCLI_ExportRoundTrip`: Import → export → re-import into fresh DB → verify same position/match count

### 6. Delete test

- [x] `TestCLI_DeleteMatch`: Import a match, delete it via `cli.deleteMatch()`, verify match count = 0
- [x] Verify associated positions still exist if shared, or are cleaned up if orphaned

### 7. Create/verify tests

- [x] `TestCLI_Create`: Call `cli.runCreate()` with a temp path, verify a valid DB file is created
- [x] `TestCLI_Verify`: Import a match, call `cli.runVerify()`, verify it reports no issues

### 8. Edge cases

- [x] `TestCLI_ImportNonexistentFile`: Import a file that doesn't exist, verify clean error (no panic)
- [x] `TestCLI_ImportCorruptFile`: Import a non-XG file as XG, verify clean error
- [x] `TestCLI_ExportNoData`: Export from empty DB, verify clean behavior

## Acceptance criteria

- [x] ≥15 test functions covering import, list, search, export, delete, create, verify
- [x] All tests pass with `go test -run TestCLI -count=1 -timeout 120s`
- [x] Tests use in-memory DB (no temp files except for export tests)
- [x] Edge cases produce errors, not panics
- [x] `go test -race` passes (no data races in CLI methods)

## Rollback

Delete `cli_test.go`: `git revert`. Tests are additive only.
