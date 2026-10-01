// Contract cases for a draft's revision (ADR-0057 rule 4) and for the count
// of what reopening an imported Match as a draft drops (ADR-0045 §2).
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testTranscriptionRevision pins the revision as a compare-and-swap: an
// insert starts at 1, every write advances it, a write naming a revision the
// row no longer has is ErrConflict and changes nothing, a missing row is
// ErrNotFound, and revision 0 rewrites unconditionally.
func testTranscriptionRevision(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ts := s.Transcriptions()

	row := &storage.Transcription{FormatVersion: "3", Label: "a", Document: `{"n":1}`}
	id, err := ts.Save(ctx, "", row)
	if err != nil || row.Revision != 1 {
		t.Fatalf("insert: revision %d, %v; want 1, nil", row.Revision, err)
	}
	row.ID = id
	row.Document = `{"n":2}`
	if _, err := ts.Save(ctx, "", row); err != nil || row.Revision != 2 {
		t.Fatalf("rewrite at revision 1: revision %d, %v; want 2, nil", row.Revision, err)
	}

	stale := &storage.Transcription{ID: id, FormatVersion: "3", Document: `{"n":"stale"}`, Revision: 1}
	if _, err := ts.Save(ctx, "", stale); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("rewrite at a stale revision: got %v, want ErrConflict", err)
	}
	if got, err := ts.Get(ctx, "", id); err != nil || got.Document != `{"n":2}` || got.Revision != 2 {
		t.Fatalf("a refused write must change nothing: got %+v, %v", got, err)
	}

	rev, err := ts.Touch(ctx, "", id, 2)
	if err != nil || rev != 3 {
		t.Fatalf("Touch at revision 2: %d, %v; want 3, nil", rev, err)
	}
	if _, err := ts.Touch(ctx, "", id, 2); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("Touch at a stale revision: got %v, want ErrConflict", err)
	}
	if got, _ := ts.Get(ctx, "", id); got.Document != `{"n":2}` {
		t.Fatalf("Touch must not change the document: got %q", got.Document)
	}

	repair := &storage.Transcription{ID: id, FormatVersion: "3", Document: `{"n":3}`}
	if _, err := ts.Save(ctx, "", repair); err != nil || repair.Revision != 4 {
		t.Fatalf("unconditional rewrite: revision %d, %v; want 4, nil", repair.Revision, err)
	}

	if err := ts.Delete(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	if _, err := ts.Touch(ctx, "", id, 4); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Touch on a deleted draft: got %v, want ErrNotFound", err)
	}
	if _, err := ts.Save(ctx, "", &storage.Transcription{ID: id, FormatVersion: "3", Document: "{}", Revision: 4}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Save on a deleted draft: got %v, want ErrNotFound", err)
	}
}

// testTranscriptionAnnotations pins the count of what a .mat cannot carry:
// the position and move analyses and the comments of the Match's own
// positions, and nothing of another Match.
func testTranscriptionAnnotations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchID := annotatedMatch(t, s, "ann-1", 0)
	_ = annotatedMatch(t, s, "ann-2", 1)

	analyses, comments, err := s.Transcriptions().Annotations(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Annotations: %v", err)
	}
	if analyses != 1 || comments != 2 {
		t.Errorf("Annotations = %d analyses, %d comments; want 1, 2", analyses, comments)
	}
	empty, err := s.Matches().Save(ctx, "", &domain.Match{Player1Name: "C", Player2Name: "D", MatchLength: 3, MatchHash: "ann-3"})
	if err != nil {
		t.Fatal(err)
	}
	if a, c, err := s.Transcriptions().Annotations(ctx, "", empty); err != nil || a+c != 0 {
		t.Errorf("Annotations of a bare match = %d, %d, %v; want 0, 0, nil", a, c, err)
	}
}

// annotatedMatch saves a one-move Match whose position holds one analysis and
// two comments.
func annotatedMatch(t *testing.T, s storage.Storage, hash string, slot int) int64 {
	t.Helper()
	ctx := context.Background()
	matchID, err := s.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 5, MatchHash: hash})
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	pos := statsDecisionPos(t, slot)
	pos.DecisionType = domain.CheckerAction
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker",
		PositionID: posID, Player: 1, CheckerMove: "13/11 24/23"}); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	a := domain.PositionAnalysis{AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Index: 0, Move: "13/11 24/23", Equity: 0.1}}}}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	for _, text := range []string{"one", "two"} {
		if _, err := s.Comments().Add(ctx, "", posID, text); err != nil {
			t.Fatalf("Add comment: %v", err)
		}
	}
	return matchID
}
