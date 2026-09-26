package sqlshared

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestGetMatchIDsForTournamentPropagatesScanError: a scan failure on one row
// must be reported, not skipped into a match list one short of the truth.
func TestGetMatchIDsForTournamentPropagatesScanError(t *testing.T) {
	f := &fakeExecer{queryCallToFail: 1}
	if _, err := getMatchIDsForTournament(context.Background(), f, 1); err == nil {
		t.Fatal("getMatchIDsForTournament with a corrupted row = nil error; want a non-nil error")
	}
}

// TestLoadPlayer1MovesPropagatesErrors covers both failure shapes on the
// batched preload: the Query itself failing and a Scan failing on one row.
// Neither may read as "this position recorded no moves".
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

// TestLoadCommentTextsPropagatesErrors: the same for the batched comment
// preload: a locked database must fail the search outright, not silently answer every
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

// TestFindPropagatesTournamentLookupFailure: at search level, a failed
// tournament lookup must fail the search, not make the tournament look empty.
func TestFindPropagatesTournamentLookupFailure(t *testing.T) {
	f := &fakeExecer{queryErrToFail: 1}
	store := &SearchStore{DB: f}
	_, err := store.find(context.Background(), "t", domain.SearchFilters{TournamentIDsFilter: "5"}, storage.ListOpts{})
	if err == nil {
		t.Fatal("find with a failing tournament lookup = nil error; want a non-nil error")
	}
}
