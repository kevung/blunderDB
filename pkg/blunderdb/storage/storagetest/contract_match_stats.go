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
		// The split, the error count and the Snowie parts against the same
		// direct oracle: the players table and the Stats panel read them.
		for _, id := range []int64{matchA, matchB, matchC, matchD} {
			detail, err := s.Stats().MatchDetail(ctx, "", id)
			if err != nil {
				continue
			}
			r1, r2 := bySeat[[2]int64{id, 1}], bySeat[[2]int64{id, 2}]
			for seat, want := range map[int]storage.MatchPlayerDetailStats{1: detail.Player1, 2: detail.Player2} {
				got := bySeat[[2]int64{id, int64(seat)}]
				snowie := 0.0
				if n := r1.SnowieMoves + r2.SnowieMoves; n > 0 {
					snowie = 500 * float64(got.SnowieErrorMP) / 1000 / float64(n)
				}
				if got.CheckerErrorMP != int64(math.Round(want.CheckerEquityError*1000)) ||
					got.CheckerErrorMP+got.CubeErrorMP != got.ErrorMP ||
					got.Errors != want.TotalErrors || math.Abs(snowie-want.SnowieER) > 1e-9 ||
					got.CheckerMoves < got.SnowieMoves {
					t.Errorf("%s: match %d seat %d: row %+v, direct checker error=%v errors=%d snowie=%v (from rows %v)",
						stage, id, seat, got, want.CheckerEquityError, want.TotalErrors, want.SnowieER, snowie)
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
	if _, err := s.Stats().MatchSeries(ctx, "", storage.StatsFilter{DecisionType: -1, MinAnalysisDepth: 2}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("MatchSeries with a provenance filter: err %v, want ErrInvalid", err)
	}
	// By decision type, the table's split against MatchDetail's.
	for _, dt := range []int{0, 1} {
		series, err := s.Stats().MatchSeries(ctx, "", storage.StatsFilter{DecisionType: dt})
		if err != nil {
			t.Fatalf("MatchSeries(decision type %d): %v", dt, err)
		}
		for _, g := range series {
			detail, err := s.Stats().MatchDetail(ctx, "", g.ID)
			if err != nil {
				t.Fatalf("MatchDetail(%d): %v", g.ID, err)
			}
			var n int
			var errSum float64
			for _, p := range []storage.MatchPlayerDetailStats{detail.Player1, detail.Player2} {
				if dt == 0 {
					n += p.CheckerDecisions
					errSum += p.PRChecker * float64(p.CheckerDecisions)
				} else {
					c := p.TotalDecisions - p.CheckerDecisions
					n += c
					errSum += p.PRCube * float64(c)
				}
			}
			if g.NumDecisions != n || math.Abs(g.PR-errSum/float64(max(n, 1))) > 1e-9 {
				t.Errorf("decision type %d, match %d: series %+v, direct %d decisions PR %v", dt, g.ID, g, n, errSum/float64(max(n, 1)))
			}
		}
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

// testHeadToHeadWindowsRanking holds the corpus views read from match_stats
// to MatchDetail and Compute, the direct calculations.
func testHeadToHeadWindowsRanking(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, _ := statsFixtureMatch(t, s, 2, "Bob", "Alice")
	statsFixtureMatch(t, s, 4, "Alice", "Carol")
	all := storage.StatsFilter{DecisionType: -1}

	h, err := s.Stats().HeadToHead(ctx, "", "Alice", "Bob", all)
	if err != nil {
		t.Fatalf("HeadToHead: %v", err)
	}
	if len(h.Matches) != 2 {
		t.Fatalf("HeadToHead matches = %d, want 2 (the Carol match is not one)", len(h.Matches))
	}
	var sumA, sumB float64
	for _, m := range h.Matches {
		detail, err := s.Stats().MatchDetail(ctx, "", m.ID)
		if err != nil {
			t.Fatalf("MatchDetail(%d): %v", m.ID, err)
		}
		a, b := detail.Player1, detail.Player2
		if m.ID == matchB {
			a, b = b, a
		}
		if m.DecisionsA != a.TotalDecisions || m.DecisionsB != b.TotalDecisions ||
			math.Abs(m.PRA-a.PR) > 1e-9 || math.Abs(m.PRB-b.PR) > 1e-9 {
			t.Errorf("match %d: %+v, direct A %d/%v B %d/%v", m.ID, m, a.TotalDecisions, a.PR, b.TotalDecisions, b.PR)
		}
		sumA += a.PR * float64(a.TotalDecisions)
		sumB += b.PR * float64(b.TotalDecisions)
	}
	if h.DecisionsA > 0 && math.Abs(h.PRA-sumA/float64(h.DecisionsA)) > 1e-9 {
		t.Errorf("PR A over the record = %v, want %v", h.PRA, sumA/float64(h.DecisionsA))
	}
	if h.DecisionsB > 0 && math.Abs(h.PRB-sumB/float64(h.DecisionsB)) > 1e-9 {
		t.Errorf("PR B over the record = %v, want %v", h.PRB, sumB/float64(h.DecisionsB))
	}
	if _, err := s.Stats().HeadToHead(ctx, "", "Alice", "Alice", all); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("HeadToHead of a player with himself: %v, want ErrInvalid", err)
	}
	if _, err := s.Stats().HeadToHead(ctx, "", "Alice", "Bob", storage.StatsFilter{DecisionType: -1, AnalysisEngine: "x"}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("HeadToHead with a provenance filter: %v, want ErrInvalid", err)
	}

	// Every fixture match is dated June 2025: a quarter window is one point
	// covering April..June, with Compute's PR for the player.
	alice := storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"}
	res, err := s.Stats().Compute(ctx, "", alice)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	win, err := s.Stats().PRByWindow(ctx, "", alice, 3)
	if err != nil {
		t.Fatalf("PRByWindow: %v", err)
	}
	if len(win) != 1 || win[0].From != "2025-04" || win[0].To != "2025-06" ||
		win[0].NumDecisions != res.Totals.NumDecisions || win[0].NumMatches != 3 || math.Abs(win[0].PR-res.PRGlobal) > 1e-9 {
		t.Errorf("PRByWindow = %+v, want one April..June point with %d decisions, 3 matches, PR %v", win, res.Totals.NumDecisions, res.PRGlobal)
	}
	if _, err := s.Stats().PRByWindow(ctx, "", alice, 0); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("PRByWindow of 0 months: %v, want ErrInvalid", err)
	}

	// The ranking is the players table's order by PR above a floor.
	rows, err := s.Stats().PlayerTable(ctx, "", all)
	if err != nil {
		t.Fatalf("PlayerTable: %v", err)
	}
	ranked := storage.RankPlayers(rows, 1)
	for i := 1; i < len(ranked); i++ {
		if ranked[i].PR < ranked[i-1].PR || ranked[i].Rank < ranked[i-1].Rank {
			t.Errorf("ranking out of order at %d: %+v", i, ranked)
		}
	}
	if len(storage.RankPlayers(rows, 1<<30)) != 0 {
		t.Error("a floor above every player's decisions still ranks someone")
	}
}
