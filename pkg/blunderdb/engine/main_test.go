package engine

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// The one-sided table left the binary with ADR-0027: nothing loads it on its
// own any more. The application generates it in the background and points the
// engine at it; tests do the same, once for the package.
func TestMain(m *testing.M) {
	t := &testing.T{}
	if err := LoadOneSided(bearofftest.OneSidedPath(t)); err != nil {
		panic("engine tests: loading the generated one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}
