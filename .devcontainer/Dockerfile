# Minimal devcontainer for blunderDB backend + frontend work (E.5, #221).
# Not CI: CI pins Node 23.4.0 and matrix-builds four OSes
# (.github/workflows/build.yml); this image picks the current Node LTS (22)
# instead, since a devcontainer is meant to stay usable for a long time
# without chasing a fast-moving non-LTS release. It is not used to produce
# release binaries — do a real build (`make build`) or CI for that.
FROM golang:1.25.13-bookworm

# webkit2gtk-4.1 matches CLAUDE.md: it is the package `wails dev`/`wails build
# -tags webkit2_41` link against on Arch and ubuntu-latest; CI's ubuntu-22.04
# leg instead targets plain `wails build` against webkit2gtk-4.0.
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential pkg-config git curl ca-certificates gnupg \
        libgtk-3-dev libwebkit2gtk-4.1-dev \
    && rm -rf /var/lib/apt/lists/*

# Node 22 LTS via NodeSource (Debian's own nodejs package is far too old for
# the frontend's Vite/Svelte 5 toolchain).
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# The Wails CLI itself (library version pinned by frontend's go.mod/wails.json).
RUN go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2

# govulncheck: `make check`/`make check-all` both call it.
RUN go install golang.org/x/vuln/cmd/govulncheck@latest

# golangci-lint v2.11.4 — pinned to match .golangci.yml's documented version
# (CLAUDE.md); the install script drops the binary in $(go env GOPATH)/bin,
# already on PATH from the base image.
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
        | sh -s -- -b "$(go env GOPATH)/bin" v2.11.4

WORKDIR /workspaces/blunderDB
