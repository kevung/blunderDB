// Contract cases for match deletion: the cascade and the retention predicate that
// decides which positions survive it.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testMatchDeleteCascade(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID, Player: 1}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}

	if _, err := s.Matches().Get(ctx, "", matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("after delete Get match: got %v, want ErrNotFound", err)
	}
	for _, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		t.Error("game not cascade-deleted")
	}
	for _, err := range s.Matches().Moves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		t.Error("move not cascade-deleted")
	}
	if _, err := s.Positions().Load(ctx, "", posID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("orphan position not deleted: got %v, want ErrNotFound", err)
	}
}

// testMatchDeleteCascadeRetention pins what survives deleting a match. Every
// position below occurs in the match; they differ only in what else holds them.
// Before the individually-imported flag existed, a position the user had
// imported on its own was purged here as an orphan, silently, along with its
// Anki card.
//
// A comment does NOT hold a position, and that is deliberate: match importers
// attach the source file's per-move notes as comments (ingest/xg.go), so a
// comment is not evidence the user did anything — holding on it would keep a
// whole annotated match alive after the user deleted it.
func testMatchDeleteCascadeRetention(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	// Every position is reached by one of the match's moves.
	inMatch := func(n int, individual bool) int64 {
		p := provenancePos(n)
		p.IndividuallyImported = individual
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(n), MoveType: "checker", PositionID: id, Player: 1}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove %d: %v", n, err)
		}
		return id
	}

	purged := inMatch(1, false)    // held by nothing but the match…
	individual := inMatch(2, true) // …the user brought this one in themselves
	inCollection := inMatch(3, false)
	commented := inMatch(4, false)     // …with a note that came in with the match
	userCommented := inMatch(9, false) // …with a note the user wrote (#263)
	inDeck := inMatch(5, false)
	ankiCard := inMatch(7, false)

	// The user flagged this one for study in the source tool (docs/adr/0006):
	// same reasoning as individually_imported — the retention predicate must
	// keep it, or deleting a match would delete the very positions the `fl`
	// filter exists to surface.
	flaggedPos := provenancePos(6)
	flaggedPos.Flagged = true
	flagged, err := s.Positions().Save(ctx, "", &flaggedPos)
	if err != nil {
		t.Fatalf("Save flagged position: %v", err)
	}
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{
		GameID: gameID, MoveNumber: 6, MoveType: "checker", PositionID: flagged, Player: 1,
	}); err != nil {
		t.Fatalf("CreateMove (flagged): %v", err)
	}

	// The most common real-world case: a position (typically an opening
	// position) that recurs in a second, still-live match. This is the FIRST
	// clause of positionIsHeldSQL, not one of the "extra" holders below.
	sharedWithSecondMatch := inMatch(8, false)
	m2 := domain.Match{Player1Name: "Eve", Player2Name: "Frank"}
	matchID2, err := s.Matches().Save(ctx, "", &m2)
	if err != nil {
		t.Fatalf("Save second match: %v", err)
	}
	g2 := domain.Game{MatchID: matchID2, GameNumber: 1}
	gameID2, err := s.Matches().CreateGame(ctx, "", &g2)
	if err != nil {
		t.Fatalf("CreateGame (second match): %v", err)
	}
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{
		GameID: gameID2, MoveNumber: 1, MoveType: "checker", PositionID: sharedWithSecondMatch, Player: 1,
	}); err != nil {
		t.Fatalf("CreateMove (second match): %v", err)
	}

	// An analysis never holds a position: it arrives with the match, and every
	// match position has one, so holding on it would mean never purging
	// anything. A comment holds one only when the USER wrote it (#263): an
	// imported per-move note is the file's sentence, not theirs.
	if err := s.Analyses().Save(ctx, "", purged, &domain.PositionAnalysis{}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	coll, err := s.Collections().Create(ctx, "", "keep", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := s.Collections().AddPosition(ctx, "", coll, inCollection); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}
	if _, err := s.Comments().AddFrom(ctx, "", commented, "note that came in with the match", domain.CommentOriginXG); err != nil {
		t.Fatalf("AddFrom comment: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", userCommented, "I keep missing this one"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	deck, err := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deck, []int64{inDeck, ankiCard}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}

	for _, tc := range []struct {
		name string
		id   int64
		kept bool
	}{
		{"held by nothing (analysis only)", purged, false},
		{"commented by the importer, not the user", commented, false},
		{"commented by the user (#263)", userCommented, true},
		{"individually imported", individual, true},
		{"in a collection", inCollection, true},
		{"in an Anki deck", inDeck, true},
		{"flagged (ADR-0006)", flagged, true},
		{"referenced by an Anki card", ankiCard, true},
		{"shared with a second, still-live match", sharedWithSecondMatch, true},
	} {
		_, err := s.Positions().Load(ctx, "", tc.id)
		switch {
		case tc.kept && err != nil:
			t.Errorf("position %s was purged with the match: %v", tc.name, err)
		case !tc.kept && !errors.Is(err, storage.ErrNotFound):
			t.Errorf("position %s survived the match: got %v, want ErrNotFound", tc.name, err)
		}
	}
}
