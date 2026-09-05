package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// raceTestTwoSided opens the TS-06-06 table these measurements compare against.
// Generated and cached by bearofftest since ADR-0027 took the file out of the
// binary; it is the same bytes gnubg produces, so the measurements are
// comparable with the ones recorded before the change.
func raceTestTwoSided(t *testing.T) *race.TwoSided {
	t.Helper()
	ts, err := race.OpenTwoSided(bearofftest.TwoSidedPath(t))
	if err != nil {
		t.Fatalf("opening the generated TS-06-06: %v", err)
	}
	t.Cleanup(func() { ts.Close() })
	return ts
}
