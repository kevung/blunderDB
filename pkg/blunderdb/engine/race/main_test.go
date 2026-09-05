package race

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// The one-sided table is no longer compiled in (ADR-0027), so nothing loads it
// on its own any more: the application generates it in the background and
// points the engine at it. Tests do the same, once for the package.
func TestMain(m *testing.M) {
	t := &testing.T{}
	if err := engine.LoadOneSided(bearofftest.OneSidedPath(t)); err != nil {
		panic("race tests: loading the generated one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}
