# 03 — Fix silent exit-zero on startup failures

**Goal:** Fatal startup errors in GUI mode produce non-zero exit codes and visible error messages instead of silently succeeding.

**Depends on:** Nothing.

**Impact:** Medium — users currently get no feedback when startup fails.

## Context

`main.go` (106 lines) has three failure paths in `runGUI()` that print to stdout and `return` (exit 0):

| Line | Failure | Current behavior |
|------|---------|-----------------|
| ~64  | `cfg.LoadConfig()` error | `fmt.Println("Error loading...")` → `return` |
| ~70  | `db.SetupDatabase(":memory:")` error | `fmt.Println("Error setting up...")` → `return` |
| ~106 | `wails.Run()` error | `println("Error:", err.Error())` — no `os.Exit` |

Additionally, lines ~71-72 print diagnostic output on every normal startup:
```go
fmt.Println("Initial dimensions:", initialWidth, "x", initialHeight)
fmt.Println("Aspect ratio:", float64(initialHeight)/float64(initialWidth))
```

## Files touched

- **Edit:** `main.go`

## Tasks

### 1. Fix fatal error exits in `runGUI()`

- [ ] Replace `fmt.Println(...) + return` with `log.Fatal(err)` (or `fmt.Fprintln(os.Stderr, ...) + os.Exit(1)`) for:
  - Config load failure
  - Database setup failure
- [ ] For `wails.Run()` error at end of function: add `os.Exit(1)` after the error print
- [ ] Use `os.Stderr` for error output, not `os.Stdout`

### 2. Remove diagnostic prints

- [ ] Remove or gate the `fmt.Println("Initial dimensions:", ...)` and `fmt.Println("Aspect ratio:", ...)` lines
- [ ] Option A: delete them entirely (preferred — they serve no runtime purpose)
- [ ] Option B: gate behind an environment variable: `if os.Getenv("BLUNDERDB_DEBUG") != "" { ... }`

### 3. Verify `runCLI()` is already correct

- [ ] Confirm `runCLI()` already uses `fmt.Fprintln(os.Stderr, ...)` and `os.Exit(1)` — no changes needed there

## Acceptance criteria

- [ ] Config load failure → exit code 1, error on stderr
- [ ] Database setup failure → exit code 1, error on stderr
- [ ] `wails.Run()` failure → exit code 1, error on stderr
- [ ] Normal startup produces no stdout output (no dimension/ratio prints)
- [ ] `go build && ./blunderdb` still starts normally in GUI mode
- [ ] `go test ./...` still passes

## Rollback

`git revert` — trivial, only `main.go` touched.
