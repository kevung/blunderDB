.PHONY: dev build

# VERSION feeds `blunderdb version`'s app-version line (internal/cli.appVersion,
# see internal/cli/version.go). It tracks the nearest git tag, same as CI
# (.github/workflows/build.yml); "dev" if there is no tag/repo (e.g. a source
# tarball build).
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

dev:
	wails dev -tags webkit2_41

build:
	wails build -tags webkit2_41 -ldflags "-X github.com/kevung/blunderdb/internal/cli.appVersion=$(VERSION)"
