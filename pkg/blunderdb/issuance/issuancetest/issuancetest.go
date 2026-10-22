// Package issuancetest keeps test binaries away from the real Issuer identity.
package issuancetest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// Run points XDG_CONFIG_HOME at a throwaway directory for the whole test binary, then runs
// the tests. It fails the binary (non-zero code) if the real configuration directory changed
// meanwhile, or if the identity directory ever resolved outside the throwaway one. Call it
// from TestMain: os.Exit(issuancetest.Run(m.Run)).
func Run(run func() int) int {
	return runGuarded(issuance.ConfigDir(), run)
}

func runGuarded(realDir string, run func() int) int {
	before := snapshot(realDir)
	tmp, err := os.MkdirTemp("", "blunderdb-xdg-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "issuancetest:", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	prev, had := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	xdg.Reload()
	if dir := issuance.ConfigDir(); !strings.HasPrefix(dir, tmp+string(filepath.Separator)) {
		fmt.Fprintf(os.Stderr, "issuancetest: identity directory %s is outside %s\n", dir, tmp)
		return 1
	}
	code := run()
	if had {
		os.Setenv("XDG_CONFIG_HOME", prev)
	} else {
		os.Unsetenv("XDG_CONFIG_HOME")
	}
	xdg.Reload()
	if after := snapshot(realDir); after != before {
		fmt.Fprintf(os.Stderr, "issuancetest: tests modified the real identity directory %s\n", realDir)
		return 1
	}
	return code
}

// snapshot fingerprints a directory by names, sizes and modification times.
func snapshot(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "absent"
	}
	var b strings.Builder
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%s|%d|%d\n", e.Name(), info.Size(), info.ModTime().UnixNano())
	}
	return b.String()
}
