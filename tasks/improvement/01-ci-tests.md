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

- [ ] Edit `.github/workflows/build.yml`
- [ ] Add a test step **after** Go setup and **before** Wails build:
  ```yaml
  - name: Run Go tests
    run: go test -timeout 300s -race -count=1 ./...
    shell: bash
  ```
- [ ] Place it after the `Setup GoLang` + `go version` step, before `Install Wails`
- [ ] The `-race` flag adds ~2× overhead but catches data races in the mutex patterns
- [ ] `-timeout 300s` matches the local test timeout (some benchmarks are slow)

### 2. Handle platform-specific test behavior

- [ ] Verify tests pass on all 4 CI platforms by pushing to a branch and checking
- [ ] If any tests fail on Windows (path separators, temp file handling), gate them with `runtime.GOOS` or `testing.Short()`
- [ ] If the race detector causes excessive slowdown on CI, consider running `-race` on one platform only (e.g. `ubuntu-latest`) and non-race on others

### 3. Optional: separate test job from build job

- [ ] Consider adding a dedicated `test` job that runs in parallel with the `build` job, to avoid slowing down builds:
  ```yaml
  jobs:
    test:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with: { go-version: '1.23.1' }
        - run: go test -timeout 300s -race -count=1 ./...
    build:
      # ... existing build matrix ...
  ```
- [ ] This is optional but recommended — tests don't need Wails, Node, or webkit deps

## Acceptance criteria

- [ ] Every push to `main` and every PR triggers Go tests
- [ ] Test failures block the CI status (red badge on PR)
- [ ] Race detector is enabled on at least one CI platform
- [ ] All 127 existing tests pass on all CI platforms

## Rollback

Revert the workflow file change: `git revert`.
