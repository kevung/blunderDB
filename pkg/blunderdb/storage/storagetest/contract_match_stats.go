package storagetest

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMatchStatsOracle holds the materialised match_stats rows to the direct
// calculation on the same library, through every write that can make them
// stale: a row that drifts from the decisions it summarises would be read
// silently by every PR the panels show. The oracles are MatchDetail (per
// match, per seat) and Compute (a player's PR over all his matches) — the
// two independent paths the table stands in for.
func testMatchStatsOracle(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchA, posA := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, _ := statsFixtureMatch(t, s, 2, "Alice", "Carol")
	matchC, _ := statsFixtureMatch(t, s, 4, "Dave", "Alice")

	// A match reaching A's player-1 position, as an opening is shared.
	shared := domain.Match{Player1Name: "Erin", Player2Name: "Bob", MatchLength: 7}
	matchD, err := s.Matches().Save(ctx, "", &shared)
	if err != nil {
		t.Fatalf("Save match D: %v", err)
	}
	gameD, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchD, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame D: %v", err)
	}
	luck := int32(-37)
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameD, MoveNumber: 1, MoveType: "checker",
		PositionID: posA[0], Player: 1, CheckerMove: "13/11 24/23", LuckMP: &luck}); err != nil {
		t.Fatalf("CreateMove D: %v", err)
	}

	check := func(stage string) {
		t.Helper()
		rows, err := s.Stats().MatchStats(ctx, "", nil)
		if err != nil {
			t.Fatalf("%s: MatchStats: %v", stage, err)
		}
		bySeat := map[[2]int64]storage.MatchStatsRow{}
		for _, r := range rows {
			bySeat[[2]int64{r.MatchID, int64(r.Seat)}] = r
		}
		for _, id := range []int64{matchA, matchB, matchC, matchD} {
			detail, err := s.Stats().MatchDetail(ctx, "", id)
			if err != nil {
				t.Fatalf("%s: MatchDetail(%d): %v", stage, id, err)
			}
			for seat, want := range map[int]storage.MatchPlayerDetailStats{1: detail.Player1, 2: detail.Player2} {
				got, ok := bySeat[[2]int64{id, int64(seat)}]
				if !ok {
					t.Errorf("%s: match %d seat %d: no row", stage, id, seat)
					continue
				}
				if got.Decisions != want.TotalDecisions || got.CheckerDecisions != want.CheckerDecisions ||
					got.CubeDecisions != want.TotalDecisions-want.CheckerDecisions ||
					got.Blunders != want.TotalBlunders || math.Abs(got.PR-want.PR) > 1e-9 {
					t.Errorf("%s: match %d seat %d: row %+v, direct decisions=%d checker=%d blunders=%d PR=%v",
						stage, id, seat, got, want.TotalDecisions, want.CheckerDecisions, want.TotalBlunders, want.PR)
				}
			}
		}
		if len(bySeat) != len(rows) || len(rows)%2 != 0 {
			t.Errorf("%s: %d rows, want two distinct seats per match", stage, len(rows))
		}

		// A player's PR over every match he sat in, from the rows, equals Compute's.
		res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
		if err != nil {
			t.Fatalf("%s: Compute: %v", stage, err)
		}
		var sum int64
		var n int
		for _, id := range []int64{matchA, matchB, matchC, matchD} {
			m, err := s.Matches().Get(ctx, "", id)
			if err != nil {
				continue // deleted
			}
			for seat, name := range map[int64]string{1: m.Player1Name, 2: m.Player2Name} {
				if r, ok := bySeat[[2]int64{id, seat}]; ok && name == "Alice" {
					sum += r.ErrorMP
					n += r.Decisions
				}
			}
		}
		if n != res.Totals.NumDecisions || math.Abs(500*float64(sum)/1000/float64(max(n, 1))-res.PRGlobal) > 1e-9 {
			t.Errorf("%s: Alice from rows: %d decisions, %d mp; Compute: %d decisions, PR %v",
				stage, n, sum, res.Totals.NumDecisions, res.PRGlobal)
		}
	}

	check("fresh")
	// MatchSeries (Compute's PerMatch) against MatchDetail, seat by seat.
	checkSeries := func(stage string, f storage.StatsFilter) {
		t.Helper()
		series, err := s.Stats().MatchSeries(ctx, "", f)
		if err != nil {
			t.Fatalf("%s: MatchSeries: %v", stage, err)
		}
		got := map[int64]storage.MatchStats{}
		for _, m := range series {
			got[m.ID] = m
		}
		for _, id := range []int64{matchA, matchB, matchC, matchD} {
			m, err := s.Matches().Get(ctx, "", id)
			if err != nil {
				continue
			}
			detail, err := s.Stats().MatchDetail(ctx, "", id)
			if err != nil {
				t.Fatalf("%s: MatchDetail(%d): %v", stage, id, err)
			}
			var n int
			var errSum float64
			for name, p := range map[string]storage.MatchPlayerDetailStats{m.Player1Name: detail.Player1, m.Player2Name: detail.Player2} {
				if f.PlayerName == "" || f.PlayerName == name {
					n += p.TotalDecisions
					errSum += p.PR * float64(p.TotalDecisions)
				}
			}
			g, ok := got[id]
			if n == 0 {
				if ok {
					t.Errorf("%s: match %d in the series without a counted decision", stage, id)
				}
				continue
			}
			if !ok || g.NumDecisions != n || math.Abs(g.PR-errSum/float64(n)) > 1e-9 {
				t.Errorf("%s: match %d: series %+v, direct %d decisions PR %v", stage, id, g, n, errSum/float64(n))
			}
		}
	}
	checkSeries("all players", storage.StatsFilter{DecisionType: -1})
	checkSeries("Alice", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if _, err := s.Stats().MatchSeries(ctx, "", storage.StatsFilter{DecisionType: 0}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("MatchSeries by decision type: err %v, want ErrInvalid", err)
	}

	rows, err := s.Stats().MatchStats(ctx, "", []int64{matchD})
	if err != nil {
		t.Fatalf("MatchStats(D): %v", err)
	}
	if len(rows) != 2 || rows[0].LuckMP != int64(luck) || rows[0].LuckRolls != 1 || rows[1].LuckRolls != 0 {
		t.Errorf("match D luck: got %+v, want seat 1 luck %d over one roll", rows, luck)
	}

	// The shared position's analysis changes: A and D must both follow.
	worse := 0.25
	a := domain.PositionAnalysis{
		PlayedMoves: []string{"13/11 24/23"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/6 6/4", Equity: 0.50},
			{Move: "13/11 24/23", Equity: 0.25, EquityError: &worse},
		}},
	}
	if err := s.Analyses().Save(ctx, "", posA[0], &a); err != nil {
		t.Fatalf("re-save shared analysis: %v", err)
	}
	check("shared analysis changed")
	checkSeries("shared analysis changed", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})

	if err := s.Matches().SwapPlayers(ctx, "", matchB); err != nil {
		t.Fatalf("SwapPlayers: %v", err)
	}
	check("seats swapped")
	checkSeries("seats swapped", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})

	settings := storage.DefaultLibrarySettings()
	settings.ErrorThresholdMP, settings.BlunderThresholdMP = 20, 40
	if err := s.LibrarySettings().Save(ctx, "", settings); err != nil {
		t.Fatalf("Save library settings: %v", err)
	}
	check("blunder threshold moved")
	checkSeries("blunder threshold moved", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})

	if err := s.Matches().DeleteCascade(ctx, "", matchC); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}
	gone, err := s.Stats().MatchStats(ctx, "", []int64{matchC})
	if err != nil || len(gone) != 0 {
		t.Errorf("deleted match: rows %+v, err %v; want none", gone, err)
	}

	n, err := s.Stats().RebuildMatchStats(ctx, "", nil)
	if err != nil || n != 3 {
		t.Errorf("RebuildMatchStats: %d matches, err %v; want 3", n, err)
	}
}
