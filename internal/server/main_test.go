package server

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// TestMain provides the EPC's one-sided bearoff table (ADR-0027) from the
// repository fixture via bearofftest. Failures panic: a TestMain has no
// *testing.T, and a zero-value one hides errors behind runtime.Goexit.
func TestMain(m *testing.M) {
	path, err := bearofftest.EnsureOneSided()
	if err != nil {
		panic("server tests: " + err.Error())
	}
	if err := engine.LoadOneSided(path); err != nil {
		panic("server tests: loading the one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}
