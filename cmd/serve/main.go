// Command serve is the headless entrypoint of the blunderDB engine.
//
// It builds WITHOUT the Wails GUI or the embedded frontend (unlike the root
// main.go), so it compiles as a pure-Go static binary (CGO disabled) suitable
// for a minimal container image. It is functionally identical to
// `blunderdb serve …`: it forwards its arguments to server.RunServe. The one
// other word it understands is `healthcheck`, the readiness probe the
// container image's HEALTHCHECK runs (distroless ships no curl): it forwards
// the rest to server.RunHealthcheck exactly as `blunderdb healthcheck` does.
//
// SECURITY: the daemon performs NO authentication; it trusts the X-Tenant-ID
// header and must run behind an authenticating reverse-proxy (gammonGo).
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/internal/server"

	// Blank-imported so its init() registers the legacy SQLite migration
	// chain with storage/sqlite (see pkg/blunderdb/database/migrate_hook.go).
	// Without it, storage/sqlite.Storage.Migrate refuses to touch a
	// non-fresh database that isn't already current — see that comment.
	// The database package is pure Go (no CGO, no Wails), so this keeps
	// `serve` a static CGO_ENABLED=0 binary.
	_ "github.com/kevung/blunderdb/pkg/blunderdb/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// run dispatches on the first argument: `healthcheck` probes a daemon,
// anything else is a serve flag.
func run(args []string) error {
	if len(args) > 0 && strings.EqualFold(args[0], "healthcheck") {
		return server.RunHealthcheck(args[1:])
	}
	return server.RunServe(args)
}
