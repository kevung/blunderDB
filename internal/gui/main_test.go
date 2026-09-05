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
// race regimes that need a table to be exact. Failures panic: a TestMain has
// no *testing.T, and a zero-value one turns any failure into an unreadable
// "main called runtime.Goexit".
func TestMain(m *testing.M) {
	// Both tables: the race regimes need the two-sided one resolved, and the
	// EPC the one-sided one loaded.
	dir, err := bearofftest.EnsureDataDir()
	if err != nil {
		panic("gui tests: " + err.Error())
	}
	race.SetDataDir(dir)
	race.Invalidate()
	oneSided, err := bearofftest.EnsureOneSided()
	if err != nil {
		panic("gui tests: " + err.Error())
	}
	if err := engine.LoadOneSided(oneSided); err != nil {
		panic("gui tests: loading the one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}
