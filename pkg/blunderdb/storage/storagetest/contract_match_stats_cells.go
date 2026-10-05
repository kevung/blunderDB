package storagetest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/statsequal"
)

// testMatchStatsSharedPositions: a position reached by several matches is
// one position of the totals, whichever order the matches were computed in.
// The second match reaches the first's positions after the first was
// computed with them private; the totals must not count them twice.
func testMatchStatsSharedPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	totals := func(label string, f storage.StatsFilter, positions, decisions, matches int) {
		t.Helper()
		res, err := s.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("%s: Compute: %v", label, err)
		}
		got := res.Totals
		if got.NumPositions != positions || got.NumDecisions != decisions || got.NumMatches != matches {
			t.Errorf("%s: positions %d, decisions %d, matches %d; want %d, %d, %d",
				label, got.NumPositions, got.NumDecisions, got.NumMatches, positions, decisions, matches)
		}
	}
	all := storage.StatsFilter{DecisionType: -1}

	statsFixtureMatch(t, s, 0, "Ann", "Abe")
	totals("one match", all, 2, 2, 1)

	matchB, _ := statsFixtureMatch(t, s, 0, "Bea", "Bob")
	totals("a second match on the same positions", all, 2, 4, 2)
	totals("one seat of each", storage.StatsFilter{DecisionType: -1, PlayerName: "Ann", PlayerAliases: []string{"Bea"}}, 1, 2, 2)

	statsFixtureMatch(t, s, 10, "Cid", "Cal")
	totals("a third match on other positions", all, 4, 6, 3)

	if err := s.Matches().DeleteCascade(ctx, "", matchB); err != nil {
		t.Fatal(err)
	}
	totals("the second match deleted", all, 4, 4, 2)
	totals("its players gone", storage.StatsFilter{DecisionType: -1, PlayerName: "Bea"}, 0, 0, 0)
}

// statsMatchRebuild fails when the figures Compute reads from match_stats and
// its cells differ from the figures of a table rebuilt from scratch: a write
// that changed what a row summarises without dropping it left it stale.
func statsMatchRebuild(t *testing.T, s storage.Storage, label string) {
	t.Helper()
	ctx := context.Background()
	f := storage.StatsFilter{DecisionType: -1}
	got, err := s.Stats().Compute(ctx, "", f)
	if err != nil {
		t.Fatalf("%s: Compute: %v", label, err)
	}
	if _, err := s.Stats().RebuildMatchStats(ctx, "", nil); err != nil {
		t.Fatalf("%s: RebuildMatchStats: %v", label, err)
	}
	want, err := s.Stats().Compute(ctx, "", f)
	if err != nil {
		t.Fatalf("%s: Compute after rebuild: %v", label, err)
	}
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(want)
	if ok, err := statsequal.JSON(string(g), string(w)); err != nil || !ok {
		t.Errorf("%s: stale per-match figures (%v)\n kept    %s\n rebuilt %s", label, err, g, w)
	}
}

// testMatchStatsPositionUpdateInvalidates: rewriting a position rewrites the
// scores, phase and decision type the cells count by.
func testMatchStatsPositionUpdateInvalidates(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	_, pos := statsFixtureMatch(t, s, 0, "Ann", "Abe")
	statsMatchRebuild(t, s, "filled")
	p, err := s.Positions().Load(ctx, "", pos[0])
	if err != nil {
		t.Fatal(err)
	}
	p.Score = [2]int{2, 5}
	if err := s.Positions().Update(ctx, "", p); err != nil {
		t.Fatal(err)
	}
	statsMatchRebuild(t, s, "after the score edit")
}

// testMatchStatsBestCubeActionInvalidates: the best cube action is the key of
// a cube cell; an analysis that changes it alone must drop the match's rows.
func testMatchStatsBestCubeActionInvalidates(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	m := domain.Match{Player1Name: "Ann", Player2Name: "Abe", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
	if err != nil {
		t.Fatal(err)
	}
	pos := statsDecisionPos(t, 0)
	pos.DecisionType = domain.CubeAction
	pos.Dice = [2]int{0, 0}
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "cube",
		PositionID: posID, Player: 1, CubeAction: "Double"}); err != nil {
		t.Fatal(err)
	}
	analysis := func(best string) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{
			PlayedCubeActions: []string{"Double"},
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
				CubefulNoDoubleEquity: 0.60, CubefulDoubleTakeEquity: 0.40, CubefulDoublePassEquity: 1.0,
				CubefulDoubleTakeError: -0.20, BestCubeAction: best,
			},
		}
	}
	if err := s.Analyses().Save(ctx, "", posID, analysis("No Double")); err != nil {
		t.Fatal(err)
	}
	statsMatchRebuild(t, s, "filled")
	if err := s.Analyses().Save(ctx, "", posID, analysis("Too Good, Pass")); err != nil {
		t.Fatal(err)
	}
	statsMatchRebuild(t, s, "after the best action changed alone")
}

// testMatchStatsMETClearedInvalidates: the analysis upsert drops met_id for a
// verdict that is not gammonNet's, and met_id is a dimension of the cells.
func testMatchStatsMETClearedInvalidates(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	_, pos := statsFixtureMatch(t, s, 0, "Ann", "Abe")
	mt := s.MatchEquityTables()
	id, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "Rockwell-Kazaross", Digest: "rk", Source: "<met/>"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mt.SetCurrent(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	if err := mt.TagAnalyses(ctx, "", id, pos[:]); err != nil {
		t.Fatal(err)
	}
	statsMatchRebuild(t, s, "filled")
	a, err := s.Analyses().Load(ctx, "", pos[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Analyses().Save(ctx, "", pos[0], a); err != nil {
		t.Fatal(err)
	}
	if got, err := mt.OfAnalysis(ctx, "", pos[0]); err != nil || got != 0 {
		t.Fatalf("OfAnalysis after the upsert = %d, %v; want 0", got, err)
	}
	statsMatchRebuild(t, s, "after the upsert cleared met_id")
}
