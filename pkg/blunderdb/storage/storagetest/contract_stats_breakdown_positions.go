package storagetest

import (
	"context"
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsBreakdownPositions pins the clickable figures of the breakdowns:
// every row's position count is the length of the list its drill-down loads,
// blunders included, and a position reached by several decisions counts once.
func testStatsBreakdownPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// The same two positions played in two matches: four decisions, two
	// positions.
	_, posIDs := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	statsFixtureMatch(t, s, 0, "Alice", "Bob")
	if _, err := s.Comments().Add(ctx, "", posIDs[0], "#timing"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	// Every fixture decision carries an error; this threshold makes each one
	// a blunder, so the blunder column has something to count.
	if err := s.LibrarySettings().Save(ctx, "", storage.LibrarySettings{ErrorThresholdMP: 1, BlunderThresholdMP: 2}); err != nil {
		t.Fatalf("Save settings: %v", err)
	}

	// The figure must equal the list under every filter, not only the open
	// one: the player and the decision type narrow the decisions a position
	// is counted for.
	for name, filter := range map[string]storage.StatsFilter{
		"no filter":       {DecisionType: -1},
		"player":          {DecisionType: -1, PlayerName: "Alice"},
		"decision type":   {DecisionType: 0},
		"player and type": {DecisionType: 0, PlayerName: "Bob"},
		"other decision":  {DecisionType: 1},
	} {
		t.Run(name, func(t *testing.T) { checkBreakdownPositions(t, s, filter, name != "other decision") })
	}

	// A row the selection does not hold loads nothing.
	ids, err := s.Stats().PositionIDsBySelection(ctx, "", storage.StatsFilter{DecisionType: -1}, storage.SelectionSpec{
		Kind: "breakdown", Breakdown: storage.BreakdownTag, BreakdownKey: "#absent"})
	if err != nil || len(ids) != 0 {
		t.Errorf("an absent tag loads %v (err %v), want nothing", ids, err)
	}
}

func checkBreakdownPositions(t *testing.T, s storage.Storage, filter storage.StatsFilter, wantRows bool) {
	ctx := context.Background()
	res, err := s.Stats().Compute(ctx, "", filter)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	counts, err := s.Stats().BreakdownPositionCounts(ctx, "", filter)
	if err != nil {
		t.Fatalf("BreakdownPositionCounts: %v", err)
	}

	type row struct {
		dim, key            string
		decisions, blunders int
	}
	var rows []row
	for _, p := range res.PerPhase {
		rows = append(rows, row{storage.BreakdownPhase, p.Phase, p.NumDecisions, p.BlunderCount})
	}
	for _, g := range res.PerGameType {
		rows = append(rows, row{storage.BreakdownGameType, g.GameType, g.NumDecisions, g.BlunderCount})
	}
	for _, tag := range res.PerTag {
		rows = append(rows, row{storage.BreakdownTag, tag.Tag, tag.NumDecisions, tag.BlunderCount})
	}
	for _, c := range res.PerScore {
		key := fmt.Sprintf("%d-%d", c.MoverAway, c.OpponentAway)
		if c.Money {
			key = "money"
		}
		rows = append(rows, row{storage.BreakdownScore, key, c.NumDecisions, c.BlunderCount})
	}
	if len(rows) == 0 {
		if wantRows {
			t.Fatal("the breakdowns are empty though decisions were counted")
		}
		return
	}

	sawDuplicate := false
	for _, r := range rows {
		c, ok := counts[r.dim][r.key]
		if !ok {
			t.Errorf("%s %q: no position count", r.dim, r.key)
			continue
		}
		for _, onlyBlunders := range []bool{false, true} {
			ids, err := s.Stats().PositionIDsBySelection(ctx, "", filter, storage.SelectionSpec{
				Kind: "breakdown", Breakdown: r.dim, BreakdownKey: r.key, OnlyBlunders: onlyBlunders})
			if err != nil {
				t.Fatalf("PositionIDsBySelection %s %q: %v", r.dim, r.key, err)
			}
			want := c.Positions
			if onlyBlunders {
				want = c.Blunders
			}
			if len(ids) != want {
				t.Errorf("%s %q (blunders only: %v): the figure says %d positions, the drill-down loads %d",
					r.dim, r.key, onlyBlunders, want, len(ids))
			}
		}
		if c.Positions > r.decisions || c.Blunders > r.blunders {
			t.Errorf("%s %q: %+v positions exceed %d decisions / %d blunders", r.dim, r.key, c, r.decisions, r.blunders)
		}
		if c.Positions < r.decisions {
			sawDuplicate = true
		}
	}
	if !sawDuplicate && filter.PlayerName == "" && filter.DecisionType == -1 {
		t.Error("no row counts fewer positions than decisions, though each position was played twice")
	}
}
