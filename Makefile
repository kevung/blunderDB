.PHONY: dev build check test lint vet golangci vuln test-go test-frontend lint-frontend release-check

# VERSION feeds `blunderdb version`'s app-version line (internal/cli.appVersion,
# see internal/cli/version.go). It tracks the nearest git tag, same as CI
# (.github/workflows/build.yml); "dev" if there is no tag/repo (e.g. a source
# tarball build).
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# engine/gammonnet runs 2-ply searches in its tests: fine natively, but the
# race detector's slowdown pushes it past any sane timeout, so it runs on its
# own without -race — the same split as the CI `test` job.
GAMMONNET := ./pkg/blunderdb/engine/gammonnet/
GO_PKGS_RACE = $(shell go list ./... | grep -v '/pkg/blunderdb/engine/gammonnet$$')

dev:
	wails dev -tags webkit2_41

build:
	wails build -tags webkit2_41 -ldflags "-X github.com/kevung/blunderdb/internal/cli.appVersion=$(VERSION)"

# main.go embeds frontend/dist; a checkout that never built the frontend has
# no such directory and every Go command dies on the embed pattern. A stub is
# enough for the backend targets (CI does the same).
frontend/dist:
	mkdir -p $@ && touch $@/.gitkeep

# Everything CI enforces, in CI's order. Run before pushing anything nontrivial.
check: vet test-go golangci vuln lint-frontend test-frontend

test: test-go test-frontend

lint: vet golangci lint-frontend

vet: frontend/dist
	go vet ./...

test-go: frontend/dist
	go test -race -count=1 $(GO_PKGS_RACE)
	go test -count=1 $(GAMMONNET)

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
