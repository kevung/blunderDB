package gui

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// TestMain loads the generated bearoff tables once (ADR-0027) for the race
// regime tests. Failures panic: a TestMain has no *testing.T.
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
