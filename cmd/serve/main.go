// Command serve is the headless entrypoint of the blunderDB engine.
//
// It builds without Wails or the frontend, as a static CGO-free binary for
// the container image, and behaves as `blunderdb serve …`. `healthcheck` is
// the image's readiness probe (distroless ships no curl).
//
// SECURITY: the daemon performs NO authentication; it trusts the X-Tenant-ID
// header and must run behind an authenticating reverse-proxy (gammonGo).
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/internal/server"

	// Registers the legacy SQLite migration chain (migrate_hook.go),
	// without which Storage.Migrate refuses a non-current database.
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
