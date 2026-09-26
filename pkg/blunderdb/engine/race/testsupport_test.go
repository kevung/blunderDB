package race

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// testTwoSided opens the TS-06-06 table for a test.
//
// The table is generated, not embedded (ADR-0027), so a test asks
// bearofftest for one, cached across packages.

func testTwoSided(t *testing.T) *TwoSided {
	t.Helper()
	ts, err := OpenTwoSided(bearofftest.TwoSidedPath(t))
	if err != nil {
		t.Fatalf("opening the generated TS-06-06: %v", err)
	}
	t.Cleanup(func() { ts.Close() })
	return ts
}
