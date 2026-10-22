// Package issuancetest keeps test binaries away from the real Issuer identity.
package issuancetest

import (
	"fmt"
	"os"

	"github.com/adrg/xdg"
)

// Run points XDG_CONFIG_HOME at a throwaway directory for the whole test binary, then runs
// the tests. A test that exports without isolating the identity itself would otherwise create
// or reuse the identity of whoever runs the suite. Call it from TestMain: os.Exit(issuancetest.Run(m.Run)).
func Run(run func() int) int {
	dir, err := os.MkdirTemp("", "blunderdb-xdg-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "issuancetest:", err)
		return 1
	}
	defer os.RemoveAll(dir)
	prev, had := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", dir)
	xdg.Reload()
	defer func() {
		if had {
			os.Setenv("XDG_CONFIG_HOME", prev)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		xdg.Reload()
	}()
	return run()
}
