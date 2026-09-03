package sqlshared

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

// TestLoadPlayer1MovesPropagatesErrors covers both failure shapes on the
// batched preload (B.10, #178, folding what was getPlayer1MovesForPosition's
// one-query-per-position into one query per chunk of ids): the Query itself
// failing (a locked database) and a Scan failing on one row — both used to
// come back as (nil, nil) from the per-row helper, indistinguishable from
// "this position recorded no moves", and must still fail loudly now that the
// query runs once for every candidate instead of once per candidate.
func TestLoadPlayer1MovesPropagatesErrors(t *testing.T) {
	t.Run("query failure", func(t *testing.T) {
		f := &fakeExecer{queryErrToFail: 1}
		if _, err := loadPlayer1Moves(context.Background(), f, []int64{1}); err == nil {
			t.Fatal("loadPlayer1Moves with a failing query = nil error; want a non-nil error")
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		f := &fakeExecer{queryCallToFail: 1}
		if _, err := loadPlayer1Moves(context.Background(), f, []int64{1}); err == nil {
			t.Fatal("loadPlayer1Moves with a corrupted row = nil error; want a non-nil error")
		}
	})
}

// TestLoadCommentTextsPropagatesErrors is the same regression for the batched
// comment preload (B.10, #178, folding loadCommentText into loadCommentTexts):
// a locked database must fail the search outright, not silently answer every
// id with "no comment".
func TestLoadCommentTextsPropagatesErrors(t *testing.T) {
	t.Run("query failure", func(t *testing.T) {
		f := &fakeExecer{queryErrToFail: 1}
		if _, err := loadCommentTexts(context.Background(), f, []int64{1}); err == nil {
			t.Fatal("loadCommentTexts with a failing query = nil error; want a non-nil error")
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		f := &fakeExecer{queryCallToFail: 1}
		if _, err := loadCommentTexts(context.Background(), f, []int64{1}); err == nil {
			t.Fatal("loadCommentTexts with a corrupted row = nil error; want a non-nil error")
		}
	})
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
	_, err := store.find(context.Background(), "t", domain.SearchFilters{TournamentIDsFilter: "5"}, storage.ListOpts{})
	if err == nil {
		t.Fatal("find with a failing tournament lookup = nil error; want a non-nil error")
	}
}
