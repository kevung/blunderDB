# 02 — Add `go vet` + `golangci-lint` to CI

**Goal:** Static analysis runs on every push/PR. Catches shadow variables, unchecked errors, inefficient patterns, and 50+ other issues before they reach main.

**Depends on:** 01 (CI test step exists).

**Impact:** High — zero static analysis today.

## Context

- No linting or vetting in CI currently
- `go vet` is built-in, zero config
- `golangci-lint` is the standard Go meta-linter (wraps staticcheck, errcheck, gosimple, govet, etc.)
- The codebase has 746 `fmt.Println` calls and known patterns that linters will flag — initial run will likely produce many warnings

## Tasks

### 1. Add `go vet` to CI

- [x] Add step to `.github/workflows/build.yml` (in the test job or before build):
  ```yaml
  - name: Go vet
    run: go vet ./...
    shell: bash
  ```
- [x] Fix any issues `go vet` reports (none found) (expected: few or none — `go vet` is conservative)

### 2. Add `golangci-lint` to CI

- [x] Add step using the official GitHub Action (golangci-lint-action@v7):
  ```yaml
  - name: golangci-lint
    uses: golangci/golangci-lint-action@v6
    with:
      version: latest
  ```
- [x] Create `.golangci.yml` configuration file in repo root (v2 format)

### 3. Configure initial linter set

- [x] Start with a conservative set to avoid overwhelming noise:
  ```yaml
  # .golangci.yml
  linters:
    enable:
      - errcheck      # unchecked errors
      - govet         # suspicious constructs
      - staticcheck   # advanced static analysis
      - unused        # unused code
      - ineffassign   # ineffective assignments
      - gosimple      # simplifiable code
    disable:
      - errcheckfuncs # too noisy initially with 746 fmt.Println patterns
  
  linters-settings:
    errcheck:
      exclude-functions:
        - fmt.Println   # suppress until task 07 (slog migration)
        - fmt.Printf
        - fmt.Fprintf
  
  issues:
    max-issues-per-linter: 50
    max-same-issues: 10
  ```
- [x] Exclude `fmt.Print*` from errcheck initially (task 07 will remove them)
- [x] Exclude auto-generated `frontend/wailsjs/` from linting

### 4. Fix initial lint findings

- [x] Run locally: `golangci-lint run ./...`
- [x] Fix genuine issues (removed 7 unused functions/types, fixed 10 ineffassign, fixed 2 SA9003)
- [x] Suppress false positives via config exclusions (errcheck for db.go and test files)
- [x] Target: zero lint errors on the initial commit

### 5. Optional: add nolint directive policy

- [x] Require `//nolint:lintername // reason` format (not bare `//nolint`)
- [x] Enable `nolintlint` linter to enforce this:
  ```yaml
  linters:
    enable:
      - nolintlint
  linters-settings:
    nolintlint:
      require-explanation: true
      require-specific: true
  ```

## Acceptance criteria

- [x] `go vet ./...` runs in CI, blocks on failure
- [x] `golangci-lint` runs in CI, blocks on failure
- [x] `.golangci.yml` exists with documented linter choices
- [x] All current code passes lint (zero warnings)
- [x] `frontend/wailsjs/` excluded from analysis

## Rollback

Revert workflow + config: `git revert`. No code changes required if lint fixes were in a separate commit.
