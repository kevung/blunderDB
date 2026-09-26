package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// raceTestTwoSided opens the TS-06-06 table these measurements compare against,
// generated and cached by bearofftest (ADR-0027), byte-identical to gnubg's.
func raceTestTwoSided(t *testing.T) *race.TwoSided {
	t.Helper()
	ts, err := race.OpenTwoSided(bearofftest.TwoSidedPath(t))
	if err != nil {
		t.Fatalf("opening the generated TS-06-06: %v", err)
	}
	t.Cleanup(func() { ts.Close() })
	return ts
}
