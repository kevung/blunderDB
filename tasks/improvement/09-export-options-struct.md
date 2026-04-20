# 09 — Extract `ExportOptions` struct

**Goal:** Replace the 12-parameter `ExportDatabase` signature with a single `ExportOptions` struct.

**Depends on:** 06 (db.go split — function lives in `db_export.go` post-split).

**Impact:** Medium — same pattern as task 08 but smaller scope.

## Context

Current signature (db.go L6566, post-split: `db_export.go`):

```go
func (d *Database) ExportDatabase(
    exportPath string,
    positions []Position,
    metadata map[string]string,
    includeAnalysis bool,
    includeComments bool,
    includeFilterLibrary bool,
    includePlayedMoves bool,
    includeMatches bool,
    includeCollections bool,
    collectionIDs []int64,
    matchIDs []int64,
    tournamentIDs []int64,
) error
```

## Files touched

- **Edit:** `model.go` — add `ExportOptions` struct
- **Edit:** `db_export.go` (post-split) — update signature
- **Edit:** `cli.go` — update CLI export call sites
- **Auto-updated:** `frontend/wailsjs/` bindings
- **Edit:** Frontend files calling `ExportDatabase`

## Tasks

### 1. Define `ExportOptions` struct

- [x] Add to `model.go`:
  ```go
  type ExportOptions struct {
      ExportPath           string            `json:"exportPath"`
      Positions            []Position        `json:"positions"`
      Metadata             map[string]string `json:"metadata"`
      IncludeAnalysis      bool              `json:"includeAnalysis"`
      IncludeComments      bool              `json:"includeComments"`
      IncludeFilterLibrary bool              `json:"includeFilterLibrary"`
      IncludePlayedMoves   bool              `json:"includePlayedMoves"`
      IncludeMatches       bool              `json:"includeMatches"`
      IncludeCollections   bool              `json:"includeCollections"`
      CollectionIDs        []int64           `json:"collectionIDs"`
      MatchIDs             []int64           `json:"matchIDs"`
      TournamentIDs        []int64           `json:"tournamentIDs"`
  }
  ```

### 2. Update Go function signature

- [x] Change `ExportDatabase(opts ExportOptions) error`
- [x] Replace all parameter references with `opts.FieldName`

### 3. Update call sites

- [x] `cli.go`: `exportDatabase()`, `exportDatabaseWithOptions()`, `exportMatchesOnly()`
- [x] Frontend: `ExportDatabaseModal.svelte`, `App.svelte`
- [x] Test files: `export_test.go` (20+ tests to update)

### 4. Regenerate Wails bindings

- [x] Run `wails dev` or `wails build` to regenerate `.js`/`.d.ts`
- [x] Verify frontend compiles

## Acceptance criteria

- [x] `ExportDatabase` accepts a single `ExportOptions` struct
- [x] All 20+ export tests pass
- [x] `wails build` succeeds
- [x] Frontend export functionality works

## Rollback

`git revert` — single commit.
