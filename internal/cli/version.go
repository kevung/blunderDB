package cli

// appVersion is the blunderDB application version (e.g. "0.32.0"), distinct
// from DatabaseVersion (the SQLite schema version — see
// pkg/blunderdb/domain.DatabaseVersion). It is injected at build time via
//
//	-ldflags "-X github.com/kevung/blunderdb/internal/cli.appVersion=<version>"
//
// `make build` and the release CI matrix (.github/workflows/build.yml) both
// set it from `git describe --tags`, the same value scripts/release.sh tags
// the repo with — so there is nothing for release.sh to keep in sync here, the
// git tag is the single source of truth. A binary built without that flag
// (e.g. a bare `go build`) reports "dev" instead of a stale hardcoded number.
var appVersion = "dev"
