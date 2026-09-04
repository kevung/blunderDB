package sqlshared

import (
	"context"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Direct unit tests of buildWhere (B.15, #183): inputs in, SQL text and
// bound arguments out, with no database underneath — fakeDialect answers
// every dialect question with SQLite-shaped text (fakeexecer_test.go) and
// the two filters that would make buildWhere issue a real query
// (TournamentIDsFilter, MoveErrorFilter) are left unset except where a test
// is specifically about one of them, in which case fakeExecer's zero
// responses (no rows, no error) are exactly what an empty database gives.

func buildWhereStore() *SearchStore {
	return &SearchStore{DB: &fakeExecer{}}
}

func TestBuildWhereEmptyFilterIsJustTenant(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if wc.where != "1=1" {
		t.Errorf("where = %q, want the bare tenant predicate %q", wc.where, "1=1")
	}
	if len(wc.args) != 0 {
		t.Errorf("args = %v, want none", wc.args)
	}
	if wc.needAnalysis {
		t.Error("needAnalysis: want false, no filter asked for the analysis blob")
	}
	if !wc.useSQLFilters {
		t.Error("useSQLFilters: want true, MirrorFilter is off")
	}
}

func TestBuildWhereIndividuallyImportedAndFlagged(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{
		IndividuallyImportedFilter: true,
		FlaggedFilter:              true,
	})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "p.individually_imported = 1") {
		t.Errorf("where = %q, want the individually-imported predicate", wc.where)
	}
	if !strings.Contains(wc.where, "p.flagged = 1") {
		t.Errorf("where = %q, want the flagged predicate", wc.where)
	}
}

func TestBuildWhereRestrictToPositionIDsBindsEveryID(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{
		RestrictToPositionIDs: "3,7,11",
	})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "p.id IN (?,?,?)") {
		t.Errorf("where = %q, want a 3-placeholder IN clause", wc.where)
	}
	if len(wc.args) != 3 || wc.args[0] != int64(3) || wc.args[1] != int64(7) || wc.args[2] != int64(11) {
		t.Errorf("args = %v, want [3 7 11]", wc.args)
	}
}

func TestBuildWhereRestrictToPositionIDsEmptyListExcludesEverything(t *testing.T) {
	// A restriction string that parses to no ids at all (rather than an unset
	// filter) must exclude every row, not silently apply no restriction.
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{
		RestrictToPositionIDs: "not-a-number",
	})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "AND 0=1") {
		t.Errorf("where = %q, want the always-false clause", wc.where)
	}
}

func TestBuildWherePlayerFilterUsesILikeAndBindsTwice(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{
		PlayerFilter: "Dupont",
	})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if strings.Count(wc.where, "LIKE") != 2 {
		t.Errorf("where = %q, want two LIKE occurrences (player1 and player2)", wc.where)
	}
	if len(wc.args) != 2 || wc.args[0] != "Dupont" || wc.args[1] != "Dupont" {
		t.Errorf("args = %v, want [\"Dupont\" \"Dupont\"]", wc.args)
	}
}

func TestBuildWhereDecisionTypeAndCubeResponse(t *testing.T) {
	f := domain.SearchFilters{
		Filter:             domain.Position{DecisionType: domain.CubeAction, PlayerOnRoll: 0},
		DecisionTypeFilter: true,
		CubeResponseFilter: "takepass",
	}
	wc, err := buildWhereStore().buildWhere(context.Background(), "", f)
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "p.decision_type = ? AND p.player_on_roll = ?") {
		t.Errorf("where = %q, want the decision-type predicate", wc.where)
	}
	if !strings.Contains(wc.where, "p.is_cube_response = 1") {
		t.Errorf("where = %q, want the take/pass sub-type predicate", wc.where)
	}
	if len(wc.args) != 2 || wc.args[0] != domain.CubeAction || wc.args[1] != 0 {
		t.Errorf("args = %v, want [CubeAction 0]", wc.args)
	}
}

func TestBuildWhereIncludeCubeAndScore(t *testing.T) {
	f := domain.SearchFilters{
		Filter:       domain.Position{Cube: domain.Cube{Value: 2, Owner: 1}, Score: [2]int{3, 5}},
		IncludeCube:  true,
		IncludeScore: true,
	}
	wc, err := buildWhereStore().buildWhere(context.Background(), "", f)
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "p.cube_value = ? AND p.cube_owner = ?") {
		t.Errorf("where = %q, want the cube predicate", wc.where)
	}
	if !strings.Contains(wc.where, "p.score_1 = ? AND p.score_2 = ?") {
		t.Errorf("where = %q, want the score predicate", wc.where)
	}
	wantArgs := []any{2, 1, 3, 5}
	if len(wc.args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", wc.args, wantArgs)
	}
	for i, want := range wantArgs {
		if wc.args[i] != want {
			t.Errorf("args[%d] = %v, want %v", i, wc.args[i], want)
		}
	}
}

// TestBuildWhereMirrorFilterDisablesSQLPushdown pins useSQLFilters: with
// MirrorFilter set, board/cube/score/dice predicates move entirely to
// applyGoFilters (both the position and its mirror have to be tried), so
// none of them appear in the WHERE text buildWhere returns.
func TestBuildWhereMirrorFilterDisablesSQLPushdown(t *testing.T) {
	f := domain.SearchFilters{
		Filter:       domain.Position{Cube: domain.Cube{Value: 2, Owner: 1}},
		IncludeCube:  true,
		MirrorFilter: true,
	}
	wc, err := buildWhereStore().buildWhere(context.Background(), "", f)
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if wc.useSQLFilters {
		t.Error("useSQLFilters: want false when MirrorFilter is set")
	}
	if strings.Contains(wc.where, "cube_value") {
		t.Errorf("where = %q, want no cube predicate: MirrorFilter defers it to applyGoFilters", wc.where)
	}
	if !wc.needAnalysis {
		t.Error("needAnalysis: want true, MirrorFilter always needs the decoded analysis")
	}
}

// TestBuildWhereNeedAnalysisTriggers pins the four filters that gate
// decoding the compressed analysis blob — see buildWhere's own comment on
// needAnalysis for why MoveErrorFilter is deliberately not a fifth.
func TestBuildWhereNeedAnalysisTriggers(t *testing.T) {
	tests := []struct {
		name string
		f    domain.SearchFilters
		want bool
	}{
		{"none", domain.SearchFilters{}, false},
		{"move pattern", domain.SearchFilters{MovePatternFilter: "6-4"}, true},
		{"mirror", domain.SearchFilters{MirrorFilter: true}, true},
		{"date", domain.SearchFilters{DateFilter: "d2026-01-01,2026-12-31"}, true},
		{"equity", domain.SearchFilters{EquityFilter: "e>0.1"}, true},
		{"move error alone", domain.SearchFilters{MoveErrorFilter: "E>0.1"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wc, err := buildWhereStore().buildWhere(context.Background(), "", tc.f)
			if err != nil {
				t.Fatalf("buildWhere: %v", err)
			}
			if wc.needAnalysis != tc.want {
				t.Errorf("needAnalysis = %v, want %v", wc.needAnalysis, tc.want)
			}
		})
	}
}

// TestBuildWhereWinGammonRateAddsAnalysisSubquery pins the one pair of rate
// filters (win/gammon) that buildWhere pushes to SQL through an IN-subquery
// rather than leaving to applyGoFilters (see the type's own comment on why:
// the ORDER BY plan). Backgammon/player-2 rates go through applyGoFilters's
// rateFilterChecks instead and add nothing here.
func TestBuildWhereWinGammonRateAddsAnalysisSubquery(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{
		WinRateFilter: "w>0.55",
	})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "p.id IN (SELECT position_id FROM analysis WHERE") {
		t.Errorf("where = %q, want the win/gammon-rate analysis subquery", wc.where)
	}
	if !strings.Contains(wc.where, "player1_win_rate") {
		t.Errorf("where = %q, want a player1_win_rate comparison", wc.where)
	}
}

func TestBuildWhereEffIncludeClearsPointsSharedWithExclude(t *testing.T) {
	// Point 5 is both included and excluded; EffectiveIncludeFilter must drop
	// it from the include side so "Except" wins, exactly as find's own
	// comment on effInclude promises.
	include := domain.Position{}
	include.Board.Points[5] = domain.Point{Checkers: 2, Color: domain.White}
	exclude := domain.Position{}
	exclude.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.White}

	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{
		Filter:        include,
		ExcludeFilter: exclude,
	})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if wc.effInclude.Board.Points[5].Checkers != 0 {
		t.Errorf("effInclude point 5 = %+v, want cleared (shared with ExcludeFilter)", wc.effInclude.Board.Points[5])
	}
}
