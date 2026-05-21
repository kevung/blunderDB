# Phase A — Extract `cli.go` into per-subcommand files

**Status: DONE** in commit `4743e21d refactor: split cli.go (2241 lines) by
subcommand`. CLAUDE.md updated in `3cf02237`. The remainder of this sheet
is kept for historical reference and to document the rationale.

**Goal.** Split the 2241-line `cli.go` into one file per subcommand
(`cli_<cmd>.go` at the repo root, still in `package main`), without any
behavioural change. This is a prerequisite to
[P7 (CLI 100 % coverage)](07-cli-full-coverage.md), where the dispatcher
becomes table-driven.

**Estimate.** 1-2 days. **Risk.** Low. **PRs.** 1.

**Prerequisites.** [P0](00-module-rename.md).

## Why split now

- A single 2 200-line file is hard to extend without merge conflicts.
- The dispatcher in [P7](07-cli-full-coverage.md) reads from a routing
  table; that's easier to layer on top of separated handlers.
- The split is purely mechanical (function moves, no logic changes) — a
  good "warm-up" PR before the structural P1 refactor.

## Target structure

```
internal/cli/
  cli.go              ← CLI struct, NewCLI, Run() dispatch
  cmd_create.go       ← runCreate
  cmd_import.go       ← runImport
  cmd_export.go       ← runExport
  cmd_list.go         ← runList
  cmd_match.go        ← runMatch
  cmd_verify.go       ← runVerify
  cmd_delete.go       ← runDelete
  cmd_info.go         ← runInfo
  cmd_edit.go         ← runEdit
  cmd_search.go       ← runSearch
  cmd_help.go         ← runHelp, usage strings
  cmd_version.go      ← runVersion
  flags.go            ← flag parsing helpers if any
```

Each `cmd_*.go` file holds the subcommand handler **and** its
flag/argument parsing helpers (if any).

## Steps

- [ ] Create `internal/cli/` directory.
- [ ] Move the file `cli.go` to `internal/cli/cli.go`, change `package main`
      → `package cli`. Adjust capitalisation of exported symbols
      (`NewCLI`, `Run`) and re-import in `main.go`.
- [ ] In a second commit, split out each `run<X>` method to its own
      `cmd_<x>.go`. Keep them as methods on `*CLI` so they share the
      `*Database` handle.
- [ ] Update `main.go` to import `"github.com/kevung/blunderdb/internal/cli"`
      and call `cli.NewCLI(...).Run(os.Args)`.
- [ ] Preserve the `cliCommands` constant (at `main.go:25`) verbatim — the
      GUI vs CLI dispatch depends on it.

## Verification

- [ ] `go build ./...` succeeds.
- [ ] `go test ./...` green; in particular `cli_test.go` (which exercises
      handlers directly) still works after the move.
- [ ] All historical subcommands still respond identically:
  ```bash
  ./blunderdb help
  ./blunderdb create --db /tmp/x.db
  ./blunderdb info --db /tmp/x.db
  ./blunderdb list --db /tmp/x.db --type stats --format json
  ./blunderdb import --db /tmp/x.db --type match --file testdata/...xg
  ./blunderdb search --db /tmp/x.db --filter '...'
  ```
- [ ] `wails dev` starts the GUI (the CLI extract must not regress the GUI
      dispatch in `main.go`).
- [ ] No diff in the JSON output of `info`/`list`/`search` vs before the
      refactor (byte-for-byte where feasible — use a regression fixture).

## Gotchas

- `cli.go` references `Database` directly. After this phase `Database` is
  still in `package main` at the repo root. Either:
  - Make `internal/cli` import `package main` types (impossible — Go does
    not allow importing `main`), **or**
  - Move enough of `model.go` / `db.go` shared types to a neutral package
    first. Easiest path: defer this and do the move as part of
    [P1](01b-pkg-library-refactor.md), keeping this phase strictly as a
    file-split inside the root `main` package.
- **Decision for this phase**: keep `internal/cli/` files in `package
  main` for now (one of the few exceptions where multiple directories share
  a package — actually Go forbids this). Alternative: leave the file split
  inside the repo root itself (`cmd_create.go`, `cmd_import.go`, … all in
  `package main`). Pick **the latter** to stay zero-package-rename until P1.
- Therefore: the "directory move to `internal/cli/`" is **deferred to P1**.
  Phase A only splits the file in-place at the repo root.

## Revised steps (in-place split)

- [ ] Keep all files in repo root in `package main`.
- [ ] Move each subcommand handler from `cli.go` to a new
      `cli_<subcommand>.go` at the repo root.
- [ ] Keep flag-parsing helpers in `cli.go` (or a `cli_flags.go`).
- [ ] No exported-symbol renaming.

This is purely a code-organisation refactor; the package boundary moves
later in P1.

## PR layout

One PR: `refactor(cli): split cli.go into per-subcommand files`.
