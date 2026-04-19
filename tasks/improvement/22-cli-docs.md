# 22 — Document missing CLI commands

**Goal:** Document the 5 undocumented CLI subcommands (`create`, `match`, `info`, `edit`, `verify`) and the `batch` import subtype in `CLI_USAGE.md`.

**Depends on:** Nothing.

**Impact:** Medium — users can't discover these commands without reading source code.

## Context

### Current documentation coverage

| Command | In CLI_USAGE.md? | In CLI_README.md? |
|---------|-------------------|-------------------|
| `import` (match/position) | YES | YES |
| `import --type batch` | **NO** | **NO** |
| `export` | YES | YES |
| `list` | YES | YES |
| `search` | YES | **NO** |
| `delete` | YES | YES |
| `help` | YES | YES |
| `version` | YES | YES |
| `create` | **NO** | **NO** |
| `match` | **NO** | **NO** |
| `info` | **NO** | **NO** |
| `edit` | **NO** | **NO** |
| `verify` | **NO** | **NO** |

### Source of truth
- `cli.go` — `Run()` method switch statement, `printUsage()` method
- Each subcommand has a handler method on the `CLI` struct

## Files touched

- **Edit:** `CLI_USAGE.md` — add sections for missing commands
- **Edit:** `CLI_README.md` — add missing commands to overview list

## Tasks

### 1. Read `cli.go` for undocumented commands

- [ ] Read the `create` handler — document flags (`--db`, `--user`, `--description`, `--force`)
- [ ] Read the `match` handler — document flags (`--db`, `--match`, `--format` json/text/summary)
- [ ] Read the `info` handler — document flags (`--db`, `--format` text/json)
- [ ] Read the `edit` handler — document flags (`--db`, `--user`, `--description`)
- [ ] Read the `verify` handler — document flags (`--db`, `--match`, `--file`)
- [ ] Read the `batch` import handler — document batch import from directory

### 2. Document `create` command

- [ ] Add section to `CLI_USAGE.md`:
  ```markdown
  ## create — Create a new database

  Creates a new blunderDB database file.

  **Usage:**
  ```bash
  blunderdb create --db <path> [--user <name>] [--description <text>] [--force]
  ```

  **Flags:**
  - `--db` — Path to the database file to create (required)
  - `--user` — Set the database owner name
  - `--description` — Set a description for the database
  - `--force` — Overwrite if file already exists
  ```

### 3. Document `match` command

- [ ] Add section showing how to display match data with different formats

### 4. Document `info` command

- [ ] Add section showing how to view database metadata and statistics

### 5. Document `edit` command

- [ ] Add section showing how to edit database metadata

### 6. Document `verify` command

- [ ] Add section showing how to verify database integrity and compare with source files

### 7. Document `batch` import subtype

- [ ] Add to the existing `import` section a subsection for `--type batch`

### 8. Update CLI_README.md

- [ ] Add all missing commands to the quick-reference list
- [ ] Ensure the command list matches the full set in `printUsage()`

### 9. Verify accuracy

- [ ] Run each documented command with `--help` or test flags to confirm behavior matches documentation
- [ ] Ensure example commands work:
  ```bash
  ./blunderdb create --db test.db --user "Test"
  ./blunderdb info --db test.db
  ./blunderdb edit --db test.db --user "New Name"
  ./blunderdb verify --db test.db
  ```

## Acceptance criteria

- [ ] All 12 CLI subcommands are documented in `CLI_USAGE.md`
- [ ] `CLI_README.md` lists all subcommands
- [ ] Each documented command includes: description, usage syntax, flags, example
- [ ] Examples are verified to work

## Rollback

`git revert` — documentation-only changes.
