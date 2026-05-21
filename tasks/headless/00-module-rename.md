# P0 — Rename Go module

**Goal.** Switch the Go module path from `blunderdb` to
`github.com/kevung/blunderdb` so the engine can be imported by external
Go programs after [P1](01b-pkg-library-refactor.md).

**Estimate.** ½ day. **Risk.** Low. **PRs.** 1.

**Prerequisites.** None.

## Why this is its own phase

The rename is mechanically tiny (a single `go.mod` line) but it is a
prerequisite for `pkg/` packages to be go-gettable from outside the repo.
Shipping it on its own keeps the blast radius minimal and gives a clean
baseline commit before the larger refactor.

## Steps

- [x] In `go.mod`, change `module blunderdb` → `module github.com/kevung/blunderdb`.
- [x] Search for internal imports of `blunderdb/...` and fix them:
  ```bash
  grep -rn '"blunderdb/' --include='*.go'
  ```
  Expected: zero hits (current code uses no internal sub-package imports
  — confirmed during exploration). Confirmed: zero hits.
- [x] Confirm `wails.json` still has `"name": "blunderdb"` (binary name,
  unrelated to the module path) and leave it alone.
- [x] Run `go mod tidy`.

## Verification

- [x] `go build ./...` succeeds.
- [x] `go test ./...` green.
- [x] `go test ./tests/...` green.
- [x] `wails build -tags webkit2_41` produces `build/bin/blunderdb`.
- [x] `./blunderdb create --db /tmp/x.db && ./blunderdb info --db /tmp/x.db`
      behaves as before.

## Gotchas

- Tests live in `package main` at the repo root — they reference symbols
  unqualified, so the module rename does not touch them.
- The CI matrix builds on multiple OS/webkit combos (CLAUDE.md). They must
  all stay green; nothing in CI depends on the module path string except
  the `go.mod` content itself.

## PR layout

Single PR titled `chore: qualify Go module path` with the `go.mod` and
`go.sum` diff only.
