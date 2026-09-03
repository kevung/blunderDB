package sqlshared

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestGetMatchIDsForTournamentPropagatesScanError is B.6's (#174) test on the
// tournament-filter helper: a scan failure on one row used to be silently
// skipped (`continue`), returning a match list one short of the truth
// instead of reporting that the query could not be trusted.
func TestGetMatchIDsForTournamentPropagatesScanError(t *testing.T) {
	f := &fakeExecer{queryCallToFail: 1}
	if _, err := getMatchIDsForTournament(context.Background(), f, 1); err == nil {
		t.Fatal("getMatchIDsForTournament with a corrupted row = nil error; want a non-nil error")
	}
}

// TestGetPlayer1MovesForPositionPropagatesErrors covers both failure shapes:
// the Query itself failing (a locked database) and a Scan failing on one
// row — both used to come back as (nil, nil), indistinguishable from "this
// position recorded no moves".
func TestGetPlayer1MovesForPositionPropagatesErrors(t *testing.T) {
	t.Run("query failure", func(t *testing.T) {
		f := &fakeExecer{queryErrToFail: 1}
		if _, _, err := getPlayer1MovesForPosition(context.Background(), f, 1); err == nil {
			t.Fatal("getPlayer1MovesForPosition with a failing query = nil error; want a non-nil error")
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		f := &fakeExecer{queryCallToFail: 1}
		if _, _, err := getPlayer1MovesForPosition(context.Background(), f, 1); err == nil {
			t.Fatal("getPlayer1MovesForPosition with a corrupted row = nil error; want a non-nil error")
		}
	})
}

// TestIsPlayer1TakePassCubeActionPropagatesError: this predicate used to
// report "not a take/pass" (false) on a database error, indistinguishable
// from a position genuinely played some other way.
func TestIsPlayer1TakePassCubeActionPropagatesError(t *testing.T) {
	f := &fakeExecer{queryErrToFail: 1}
	pos := &domain.Position{ID: 1}
	ok, err := isPlayer1TakePassCubeAction(context.Background(), f, pos)
	if err == nil {
		t.Fatal("isPlayer1TakePassCubeAction with a failing query = nil error; want a non-nil error")
	}
	if ok {
		t.Error("isPlayer1TakePassCubeAction reported true alongside an error")
	}
}

// TestMatchesMoveErrorFilterPropagatesError: a query failure while looking
// up player 1's recorded moves used to report "does not match" instead of
// the outage it was.
func TestMatchesMoveErrorFilterPropagatesError(t *testing.T) {
	f := &fakeExecer{queryErrToFail: 1}
	pos := &domain.Position{ID: 1}
	analysis := &domain.PositionAnalysis{
		AnalysisType:    "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "13/11"}}},
	}
	ok, err := matchesMoveErrorFilter(context.Background(), f, pos, analysis, "E>0")
	if err == nil {
		t.Fatal("matchesMoveErrorFilter with a failing query = nil error; want a non-nil error")
	}
	if ok {
		t.Error("matchesMoveErrorFilter reported a match alongside an error")
	}
}

// TestMatchesSearchTextPropagatesError: a locked database used to make a
// position silently disappear from a "t\"…\"" search instead of failing the
// search outright.
func TestMatchesSearchTextPropagatesError(t *testing.T) {
	f := &fakeExecer{queryErrToFail: 1}
	pos := &domain.Position{ID: 1}
	ok, err := matchesSearchText(context.Background(), f, pos, "blunder")
	if err == nil {
		t.Fatal("matchesSearchText with a failing query = nil error; want a non-nil error")
	}
	if ok {
		t.Error("matchesSearchText reported a match alongside an error")
	}
}

// TestFindPropagatesTournamentLookupFailure is the search-level regression
// for the same bug: SearchStore.find used to swallow getMatchIDsForTournament's
// error (err == nil check dropped the branch entirely), silently narrowing a
// tournament filter to "no matches in this tournament" whenever the lookup
// query failed — a locked database made a tournament look empty instead of
// failing the search.
func TestFindPropagatesTournamentLookupFailure(t *testing.T) {
	f := &fakeExecer{queryErrToFail: 1}
	store := &SearchStore{DB: f}
	_, err := store.find(context.Background(), "t", domain.SearchFilters{TournamentIDsFilter: "5"})
	if err == nil {
		t.Fatal("find with a failing tournament lookup = nil error; want a non-nil error")
	}
}
