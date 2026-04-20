# 07 — Replace `fmt.Println` with `slog` structured logging

**Goal:** Replace all 746 `fmt.Println`/`fmt.Printf` error/diagnostic calls with `log/slog` structured logging. Errors become visible in both CLI and GUI modes.

**Depends on:** 06 (db.go split — avoids merge conflicts in a 16k-line file).

**Impact:** High — 383 error prints in db.go are silently lost in GUI mode today.

## Context

| File | `fmt.Println/Printf` count | Nature |
|---|---|---|
| `db.go` (post-split: ~18 files) | 383 | Error reporting + diagnostic |
| `cli.go` | 358 | User-facing output + errors |
| `main.go` | 4 | Startup errors + diagnostics |
| `epc.go` | 1 | Warning |
| **Total** | **746** | |

### Key distinction

- **User-facing output** (CLI `list`, `search`, `export` results): Keep as `fmt.Fprintf(os.Stdout, ...)` — this is intentional program output, not logging.
- **Error/diagnostic prints** (`fmt.Println("Error creating table:", err)`): Replace with `slog.Error(...)`.
- **Double reporting** (`fmt.Println("Error...") + return err`): Remove the print, keep only `return err`.

## Files touched

- **New:** `logging.go` — slog handler setup
- **Edit:** All `db_*.go` files (post-split), `cli.go`, `main.go`, `epc.go`

## Tasks

### 1. Set up `slog` infrastructure

- [x] Create `logging.go` with handler initialization:
  ```go
  package main

  import (
      "log/slog"
      "os"
  )

  func initLogging(mode string) {
      var handler slog.Handler
      switch mode {
      case "cli":
          handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
              Level: slog.LevelInfo,
          })
      case "gui":
          // Log to file at XDG_STATE_HOME/blunderDB/blunderdb.log
          // or fallback to stderr
          handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
              Level: slog.LevelWarn,
          })
      }
      slog.SetDefault(slog.New(handler))
  }
  ```
- [x] Call `initLogging("cli")` from `runCLI()` and `initLogging("gui")` from `runGUI()` in `main.go`
- [x] Support `BLUNDERDB_DEBUG=1` env var to set level to `slog.LevelDebug`

### 2. Categorize all 746 print calls

- [x] Audit each `fmt.Println`/`fmt.Printf` and classify:
  - **Error** (`"Error creating table:", err`) → `slog.Error("creating table", "err", err)`
  - **Warning** (`"Warning: ..."`) → `slog.Warn("...")`
  - **Diagnostic/debug** (`"Processing batch:", n`) → `slog.Debug("processing batch", "n", n)`
  - **User output** (CLI results, stats) → keep as `fmt.Fprintf(os.Stdout, ...)`
  - **Double-report** (print + return err) → remove print, keep return

### 3. Migrate `db_*.go` files (383 calls)

- [x] Replace error prints with `slog.Error`:
  ```go
  // Before:
  fmt.Println("Error creating position table:", err)
  return err

  // After:
  return fmt.Errorf("creating position table: %w", err)
  ```
- [x] For cases where the function continues after the print (non-fatal), use `slog.Warn`
- [x] Remove all `fmt.Println("Error ...")` + `return err` double-reports — the caller logs if needed
- [x] Process one `db_*.go` file at a time, running tests after each

### 4. Migrate `cli.go` (358 calls)

- [x] Separate user-facing output from error reporting:
  - `fmt.Printf("Match: %s vs %s\n", ...)` → keep (intentional CLI output)
  - `fmt.Println("Error importing:", err)` → `slog.Error("importing", "err", err)` or `fmt.Fprintf(os.Stderr, ...)`
- [x] Use `slog.Error` for errors, `fmt.Fprintf(os.Stdout, ...)` for results
- [x] Ensure all CLI error output goes to stderr, results to stdout

### 5. Migrate `main.go` (4 calls)

- [x] Already partially fixed by task 03 — verify remaining prints use `slog` or `log.Fatal`

### 6. Migrate `epc.go` (1 call)

- [x] Replace `fmt.Printf("Warning: ...")` with `slog.Warn("...", "err", err)`

### 7. Remove `fmt` import where no longer needed

- [x] After migration, some files may no longer import `fmt` — remove unused imports
- [x] Keep `fmt` where `fmt.Errorf` is still used for error wrapping

## Acceptance criteria

- [x] Zero `fmt.Println("Error` patterns in any Go source file
- [x] All error/diagnostic output uses `slog.Error`, `slog.Warn`, or `slog.Debug`
- [x] CLI user-facing output still goes to stdout
- [x] CLI error output goes to stderr
- [x] `BLUNDERDB_DEBUG=1` enables debug-level logging
- [x] All tests pass
- [x] No new dependencies (slog is stdlib since Go 1.21)

## Rollback

`git revert` per commit. If partial, the mix of `fmt.Println` and `slog` is harmless.
