package gui

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// The bearoff tables are generated, not embedded (ADR-0027), so nothing loads
// them on its own. The application does it in the background at start-up;
// these tests do it once for the package, since several of them exercise the
// race regimes that need a table to be exact.
func TestMain(m *testing.M) {
	t := &testing.T{}
	// Both tables: the race regimes need the two-sided one resolved, and the
	// EPC the one-sided one loaded.
	race.SetDataDir(bearofftest.DataDir(t))
	race.Invalidate()
	if err := engine.LoadOneSided(bearofftest.OneSidedPath(t)); err != nil {
		panic("gui tests: loading the generated one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}
