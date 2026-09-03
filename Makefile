.PHONY: dev build check check-fast check-all test lint vet gofmt golangci vuln \
        test-go test-pg test-e2e test-frontend lint-frontend release-check

# VERSION feeds `blunderdb version`'s app-version line (internal/cli.appVersion,
# see internal/cli/version.go). It tracks the nearest git tag, same as CI
# (.github/workflows/build.yml); "dev" if there is no tag/repo (e.g. a source
# tarball build).
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# engine/gammonnet runs 2-ply searches in its tests: fine natively, but the
# race detector's slowdown pushes it past any sane timeout, so it runs on its
# own without -race — the same split as the CI `test` job.
GAMMONNET := ./pkg/blunderdb/engine/gammonnet/
GO_PKGS_RACE = $(shell go list ./... | grep -v '/pkg/blunderdb/engine/gammonnet$$' | grep -v node_modules)

# Tracked Go files gofmt itself has an opinion on (skip the generated
# frontend/wailsjs and the frontend/dist stub — neither is Go anyway, but
# both can appear under a naive find).
GOFMT_FILES = $(shell git ls-files '*.go' | grep -v '^frontend/')

dev:
	wails dev -tags webkit2_41

build:
	wails build -tags webkit2_41 -ldflags "-X github.com/kevung/blunderdb/internal/cli.appVersion=$(VERSION)"

# main.go embeds frontend/dist; a checkout that never built the frontend has
# no such directory and every Go command dies on the embed pattern. A stub is
# enough for the backend targets (CI does the same).
frontend/dist:
	mkdir -p $@ && touch $@/.gitkeep

# check-fast is what .githooks/pre-commit runs on every commit: seconds, no
# network, no Docker — gofmt, `go vet`, and the frontend's own fast gates
# (eslint, prettier --check). Everything that needs a full test run
# (test-go, golangci, govulncheck, PG, e2e) waits for `check`/`check-all`.
check-fast: gofmt vet lint-frontend

# check is the local pre-push loop: everything CI's build.yml runs on every
# push EXCEPT what needs Docker (test-pg) or a browser (test-e2e), or is only
# meaningful at release time (release-check) — check-all adds those three.
# `golangci` here does not need a separate gofmt pass: .golangci.yml enables
# the gofmt/goimports formatters, so `golangci-lint run` already catches what
# check-fast's `gofmt` target catches (plus goimports' import-grouping).
check: vet golangci vuln test-go test-frontend lint-frontend

# check-all is full CI parity: check, plus the PostgreSQL contract suite
# (needs Docker — see test-pg, which says so loudly if it isn't there), the
# Playwright end-to-end suite (needs a browser), and the release
# version-string check. golangci-lint's second pass with
# `--build-tags postgres` (E.2, #218) belongs here too once it exists as a
# target — it does not on this branch.
check-all: check test-pg test-e2e release-check

test: test-go test-frontend

lint: vet golangci lint-frontend

vet: frontend/dist
	go vet ./...

# gofmt -l lists files gofmt disagrees with; a non-empty list is a failure.
# Kept as its own target even though `golangci` also catches this (via the
# gofmt formatter enabled in .golangci.yml), because check-fast must not pay
# for a full golangci-lint run on every commit.
gofmt:
	@bad="$$(gofmt -l $(GOFMT_FILES))"; \
	if [ -n "$$bad" ]; then \
		echo "gofmt: the following files are not gofmt-clean:" >&2; \
		echo "$$bad" >&2; \
		echo "run: gofmt -w $$bad" >&2; \
		exit 1; \
	fi

test-go: frontend/dist
	go test -race -count=1 $(GO_PKGS_RACE)
	go test -count=1 $(GAMMONNET)

# test-pg replays the storage contract and the migrate package against a real
# PostgreSQL via testcontainers (pkg/blunderdb/storage/postgres/postgres_test.go)
# — the same command and BLUNDERDB_REQUIRE_PG=1 env var as CI's test-postgres
# job (.github/workflows/build.yml), so a missing or broken Docker daemon
# fails loudly here too instead of the suite silently skipping itself (the
# skip/fatal split is in postgres_test.go).
test-pg: frontend/dist
	BLUNDERDB_REQUIRE_PG=1 go test -tags postgres -count=1 \
		./pkg/blunderdb/storage/postgres/... ./pkg/blunderdb/migrate/...

# test-e2e is the Playwright suite (frontend/tests/e2e): it drives a real
# browser against the built app, so it stays out of `check` and only runs as
# part of check-all/CI.
test-e2e:
	cd frontend && npm run test:e2e

golangci: frontend/dist
	golangci-lint run ./...

vuln: frontend/dist
	govulncheck ./...

lint-frontend:
	cd frontend && npm run lint && npm run format:check

test-frontend:
	cd frontend && npm test

# The three version strings (doc/source/conf.py, frontend metaStore.js,
# wails.json) must agree; scripts/release.sh --check exits non-zero if not.
release-check:
	scripts/release.sh --check
