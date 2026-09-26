package cli

// appVersion is the blunderDB application version (e.g. "0.32.0"), distinct
// from domain.DatabaseVersion (the schema). It is injected at build time from
// `git describe --tags` via
//
//	-ldflags "-X github.com/kevung/blunderdb/internal/cli.appVersion=<version>"
//
// and reads "dev" in a build without that flag.
var appVersion = "dev"
