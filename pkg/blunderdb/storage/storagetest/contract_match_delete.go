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

// testMatchDeleteGamesKeepsTheMatch pins the two halves a replacement needs
// (ADR-0045 §2): DeleteGames empties a match of its games and moves WITHOUT
// removing the match — its id is what a tournament, a collection and the
// last-visited position point at — and PurgeOrphanPositions, asked separately
// and afterwards, drops only what nothing holds.
//
// The two are deliberately not one call: between them the caller writes the
// corrected games, which is what keeps a position both versions share on its
// existing row, analysis and id included.
func testMatchDeleteGamesKeepsTheMatch(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", GameCount: 2}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	var gameIDs []int64
	for n := 1; n <= 2; n++ {
		g := domain.Game{MatchID: matchID, GameNumber: int32(n)}
		id, err := s.Matches().CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame %d: %v", n, err)
		}
		gameIDs = append(gameIDs, id)
	}

	// One position only this match reaches, one it shares with a second match.
	own := provenancePos(1)
	ownID, err := s.Positions().Save(ctx, "", &own)
	if err != nil {
		t.Fatalf("Save own position: %v", err)
	}
	shared := provenancePos(2)
	sharedID, err := s.Positions().Save(ctx, "", &shared)
	if err != nil {
		t.Fatalf("Save shared position: %v", err)
	}
	for i, posID := range []int64{ownID, sharedID} {
		if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{
			GameID: gameIDs[i], MoveNumber: 1, MoveType: "checker", PositionID: posID, Player: 1,
		}); err != nil {
			t.Fatalf("CreateMove %d: %v", i, err)
		}
	}
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
		GameID: gameID2, MoveNumber: 1, MoveType: "checker", PositionID: sharedID, Player: 1,
	}); err != nil {
		t.Fatalf("CreateMove (second match): %v", err)
	}

	posIDs, err := s.Matches().DeleteGames(ctx, "", matchID)
	if err != nil {
		t.Fatalf("DeleteGames: %v", err)
	}
	if len(posIDs) != 2 {
		t.Errorf("DeleteGames returned %d position ids, want 2 (%v)", len(posIDs), posIDs)
	}

	// The match survives, with its id and its header.
	got, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get match after DeleteGames: %v", err)
	}
	if got.ID != matchID || got.Player1Name != "Alice" {
		t.Errorf("match changed: id=%d player1=%q", got.ID, got.Player1Name)
	}
	for _, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		t.Error("game not deleted")
	}
	for _, gid := range gameIDs {
		for _, err := range s.Matches().Moves(ctx, "", gid) {
			if err != nil {
				t.Fatalf("Moves: %v", err)
			}
			t.Error("move not cascade-deleted")
		}
	}

	// DeleteGames purges nothing on its own: both positions are still there.
	for _, id := range []int64{ownID, sharedID} {
		if _, err := s.Positions().Load(ctx, "", id); err != nil {
			t.Errorf("DeleteGames must not purge position %d: %v", id, err)
		}
	}

	if err := s.Matches().PurgeOrphanPositions(ctx, "", posIDs); err != nil {
		t.Fatalf("PurgeOrphanPositions: %v", err)
	}
	if _, err := s.Positions().Load(ctx, "", ownID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("orphan position survived the purge: got %v, want ErrNotFound", err)
	}
	if _, err := s.Positions().Load(ctx, "", sharedID); err != nil {
		t.Errorf("position held by a second match was purged: %v", err)
	}

	// An empty list, and an id that no longer exists, are both no-ops.
	if err := s.Matches().PurgeOrphanPositions(ctx, "", nil); err != nil {
		t.Errorf("PurgeOrphanPositions(nil): %v", err)
	}
	if err := s.Matches().PurgeOrphanPositions(ctx, "", []int64{ownID}); err != nil {
		t.Errorf("PurgeOrphanPositions on an already-gone id: %v", err)
	}
}

// testMatchReplaceHeader pins what ReplaceHeader writes and, above all, what it
// leaves alone: the id, the tournament, the match comment and the last-visited
// position are the reasons a replacement keeps the row instead of making a new
// one.
func testMatchReplaceHeader(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{
		Player1Name: "Alice", Player2Name: "Bob",
		Event: "Old event", Location: "Old place", Round: "Round 1",
		MatchLength: 5, GameCount: 1, MatchHash: "h-old", CanonicalHash: "c-old",
	}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if err := s.Matches().UpdateComment(ctx, "", matchID, "my note on this match"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if err := s.Matches().SetLastVisitedPosition(ctx, "", matchID, 7); err != nil {
		t.Fatalf("SetLastVisitedPosition: %v", err)
	}
	if err := s.Tournaments().SetMatchByName(ctx, "", matchID, "Open de Paris"); err != nil {
		t.Fatalf("SetMatchByName: %v", err)
	}

	next := domain.Match{
		Player1Name: "Alice Doe", Player2Name: "Robert",
		Event: "New event", Location: "New place", Round: "Final",
		MatchLength: 7, GameCount: 3, MatchHash: "h-new", CanonicalHash: "c-new",
	}
	if err := s.Matches().ReplaceHeader(ctx, "", matchID, &next); err != nil {
		t.Fatalf("ReplaceHeader: %v", err)
	}

	got, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != matchID {
		t.Errorf("id changed: %d, want %d", got.ID, matchID)
	}
	for _, tc := range []struct{ name, got, want string }{
		{"player1", got.Player1Name, "Alice Doe"},
		{"player2", got.Player2Name, "Robert"},
		{"event", got.Event, "New event"},
		{"location", got.Location, "New place"},
		{"round", got.Round, "Final"},
		{"match_hash", got.MatchHash, "h-new"},
		{"canonical_hash", got.CanonicalHash, "c-new"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
	if got.MatchLength != 7 {
		t.Errorf("match_length = %d, want 7", got.MatchLength)
	}
	if got.GameCount != 3 {
		t.Errorf("game_count = %d, want 3", got.GameCount)
	}

	// Untouched, on purpose.
	if got.Comment != "my note on this match" {
		t.Errorf("match comment lost: %q", got.Comment)
	}
	if got.LastVisitedPosition != 7 {
		t.Errorf("last visited position = %d, want 7", got.LastVisitedPosition)
	}
	if got.TournamentID == nil || got.TournamentName != "Open de Paris" {
		t.Errorf("tournament lost: id=%v name=%q", got.TournamentID, got.TournamentName)
	}

	// FindByHash follows the new hashes, and no longer the old ones.
	if id, found, err := s.Matches().FindByHash(ctx, "", "h-new", ""); err != nil || !found || id != matchID {
		t.Errorf("FindByHash(h-new) = (%d, %v, %v), want (%d, true, nil)", id, found, err, matchID)
	}
	if _, found, err := s.Matches().FindByHash(ctx, "", "h-old", ""); err != nil || found {
		t.Errorf("FindByHash(h-old) = (found %v, %v), want not found", found, err)
	}
}
