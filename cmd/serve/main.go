// Command serve is the headless entrypoint of the blunderDB engine.
//
// It builds WITHOUT the Wails GUI or the embedded frontend (unlike the root
// main.go), so it compiles as a pure-Go static binary (CGO disabled) suitable
// for a minimal container image. It is functionally identical to
// `blunderdb serve …`: it forwards its arguments to server.RunServe.
//
// SECURITY: the daemon performs NO authentication; it trusts the X-Tenant-ID
// header and must run behind an authenticating reverse-proxy (gammonGo).
package main

import (
	"fmt"
	"os"

	"github.com/kevung/blunderdb/internal/server"
)

func main() {
	if err := server.RunServe(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
