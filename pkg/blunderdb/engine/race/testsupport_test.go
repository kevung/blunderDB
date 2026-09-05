package race

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// testTwoSided opens the TS-06-06 table for a test.
//
// It used to be EmbeddedTwoSided(), which could not fail because the file was
// compiled in. Since ADR-0027 the table is generated on the machine that wants
// it, so a test asks bearofftest for one — generated once and cached across
// packages, not committed back into the repository.
func testTwoSided(t *testing.T) *TwoSided {
	t.Helper()
	ts, err := OpenTwoSided(bearofftest.TwoSidedPath(t))
	if err != nil {
		t.Fatalf("opening the generated TS-06-06: %v", err)
	}
	t.Cleanup(func() { ts.Close() })
	return ts
}
