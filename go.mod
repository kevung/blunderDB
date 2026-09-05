module github.com/kevung/blunderdb

go 1.25.13

// frontend/node_modules ships a stray Go package (flatted/golang) that
// `go test ./...` picks up as part of this module. Go 1.25's `ignore`
// directive would exclude it, but the Wails CLI (v2.10.x) parses go.mod with
// an older x/mod and aborts on "unknown directive: ignore" — it broke every
// platform build of the 0.34.0 tag. Filter with `go list ./... | grep -v
// node_modules` where it matters (Makefile) instead.

// testcontainers-go is a DIRECT dependency for tests only — eight test files
// behind the `postgres` build tag — and it drags 112 edges into the module
// graph. Reviewed for B.19 (#187) on 2026-09-04 and ACCEPTED as is, rather
// than split into a `tests/postgres/` submodule or a go.work: `go list -deps
// ./cmd/serve` shows zero testcontainers packages, so the shipped daemon
// carries none of it, and govulncheck reports on the call graph it can reach,
// not on go.sum. The cost is a longer `go mod download` in CI; the price of a
// submodule would be a second module to keep in step at every bump, for a
// dependency no released artefact contains.
require (
	github.com/adrg/xdg v0.5.3
	github.com/jackc/pgx/v5 v5.10.0
	github.com/kevung/bgfparser v1.2.0
	github.com/kevung/gnubgparser v1.6.0
	github.com/kevung/xgparser v1.4.0
	github.com/klauspost/compress v1.19.2
	github.com/open-spaced-repetition/go-fsrs/v3 v3.3.1
	github.com/testcontainers/testcontainers-go v0.44.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0
	github.com/wailsapp/wails/v2 v2.15.0
	golang.org/x/crypto v0.54.0
	golang.org/x/net v0.56.0
	golang.org/x/sys v0.47.0
	golang.org/x/text v0.40.0
	modernc.org/sqlite v1.58.0
)

require (
	dario.cat/mergo v1.0.2 // indirect
	git.sr.ht/~jackmordaunt/go-toast/v2 v2.0.3 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/bep/debounce v1.2.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/labstack/echo/v4 v4.15.3 // indirect
	github.com/labstack/gommon v0.5.0 // indirect
	github.com/leaanthony/go-ansi-parser v1.6.1 // indirect
	github.com/leaanthony/gosod v1.0.4 // indirect
	github.com/leaanthony/slicer v1.6.0 // indirect
	github.com/leaanthony/u v1.1.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20260330125221-c963978e514e // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.3.0 // indirect
	github.com/moby/moby/api v1.55.0 // indirect
	github.com/moby/moby/client v0.5.0 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/moby/sys/sequential v0.7.0 // indirect
	github.com/moby/sys/user v0.4.1 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/shirou/gopsutil/v4 v4.26.6 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/tkrajina/go-reflector v0.5.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/wailsapp/go-webview2 v1.0.23 // indirect
	github.com/wailsapp/mimetype v1.4.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.75.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

// Pins go-webview2 (Windows only) below the v1.0.23 the module graph would
// otherwise resolve. The pin predates the current Wails version and its reason
// was never recorded; lift it only after a Windows build and a smoke test of
// the packaged .exe, since it cannot be exercised on Linux.
//
// Reviewed for B.19 (#187) on 2026-09-04 and KEPT. Lifting it is not a
// judgement call that can be made from here: the package is `//go:build
// windows`, so nothing on this machine or in the Linux CI compiles it, and the
// only evidence that would settle it is a packaged .exe that opens a window.
// That evidence is cheap to gather at the next Windows release and expensive
// to fake; until someone has it, an unexplained pin that works beats an
// unexplained bump that might not.
replace github.com/wailsapp/go-webview2 => github.com/wailsapp/go-webview2 v1.0.16
