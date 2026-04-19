# 01 — Add `go test -race` to CI

**Goal:** The entire Go test suite runs on every push and PR, with race detection enabled. Currently zero tests run in CI.

**Depends on:** Nothing.

**Impact:** Critical — the 11k-line test suite only runs locally today.

## Context

- CI workflow: `.github/workflows/build.yml`
- 4 build targets: `ubuntu-latest`, `ubuntu-22.04`, `macos-latest`, `windows-latest`
- Go 1.23.1, tests require no CGO (pure-Go SQLite via `modernc.org/sqlite`)
- Test suite: 127 tests + 10 benchmarks across `./...` (root + `./tests/`)
- Some tests use fixture files under `testdata/` and `gnubg/`
- Tests currently pass locally: `go test -count=1 -timeout 300s ./...`

## Tasks

### 1. Add test step to CI workflow

- [x] Edit `.github/workflows/build.yml`
- [x] Add a test step **after** Go setup and **before** Wails build:
  ```yaml
  - name: Run Go tests
    run: go test -timeout 120s -race -count=1 -short ./...
    shell: bash
  ```
- [x] Place it in a dedicated `test` job (runs in parallel with `build`)
- [x] The `-race` flag adds ~2× overhead but catches data races in the mutex patterns
- [x] `-short` skips `TestSchemaBenchmark_CrossVersion` (needs 245 MB fixture files, 10+ min) and other explicitly-skipped slow tests

### 2. Handle platform-specific test behavior

- [x] Tests pass on `ubuntu-latest` with `-short` in ~10 s
- [ ] Verify tests pass on all 4 CI platforms by pushing to a branch and checking
- [ ] If any tests fail on Windows (path separators, temp file handling), gate them with `runtime.GOOS` or `testing.Short()`

### 3. Optional: separate test job from build job

- [x] Dedicated `test` job added that runs in parallel with the `build` job

## Acceptance criteria

- [x] Every push to `main` and every PR triggers Go tests
- [x] Test failures block the CI status (red badge on PR)
- [x] Race detector is enabled on at least one CI platform
- [x] Fixed stale `TestSchemaV200_DatabaseVersion` (expected `2.2.0`, now uses `DatabaseVersion` constant)
- [ ] All existing tests pass on all CI platforms (needs CI run to confirm)

## Rollback

Revert the workflow file change: `git revert`.
