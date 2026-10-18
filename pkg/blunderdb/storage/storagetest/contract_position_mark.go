package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testPositionSetIndividuallyImported: lifting the mark leaves the match that
// holds the position, its stats and its analysis as they were; an unknown
// position is refused by name.
func testPositionSetIndividuallyImported(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	_, posIDs := statsFixtureMatch(t, s, 0, "Ann", "Abe")
	id := posIDs[0]

	before, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	marked := func() bool {
		t.Helper()
		p, err := s.Positions().Load(ctx, "", id)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		return p.IndividuallyImported
	}
	for _, want := range []bool{true, true, false, false} {
		if err := s.Positions().SetIndividuallyImported(ctx, "", id, want); err != nil {
			t.Fatalf("SetIndividuallyImported(%v): %v", want, err)
		}
		if got := marked(); got != want {
			t.Errorf("after SetIndividuallyImported(%v): mark = %v", want, got)
		}
	}
	if _, err := s.Analyses().Load(ctx, "", id); err != nil {
		t.Errorf("the analysis did not survive: %v", err)
	}
	after, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute after: %v", err)
	}
	if after.Totals != before.Totals {
		t.Errorf("totals moved: %+v, was %+v", after.Totals, before.Totals)
	}
	if err := s.Positions().SetIndividuallyImported(ctx, "", id+1000000, true); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("unknown position: err = %v, want ErrNotFound", err)
	}
}
