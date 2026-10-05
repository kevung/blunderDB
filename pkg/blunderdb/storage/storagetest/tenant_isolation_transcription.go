package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// checkTranscriptionRevisionIsolation: another tenant can neither read a
// draft nor advance its revision, whatever revision it names.
func checkTranscriptionRevisionIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	row := &storage.Transcription{FormatVersion: "3", Document: "{}"}
	id, err := s.Transcriptions().Save(ctx, a, row)
	if err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}
	if _, err := s.Transcriptions().Get(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, draft of %s): got %v, want ErrNotFound", b, a, err)
	}
	if _, err := s.Transcriptions().Touch(ctx, b, id, 1); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Touch(%s, draft of %s): got %v, want ErrNotFound", b, a, err)
	}
	foreign := &storage.Transcription{ID: id, FormatVersion: "3", Document: `{"x":1}`, Revision: 1}
	if _, err := s.Transcriptions().Save(ctx, b, foreign); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Save(%s, draft of %s): got %v, want ErrNotFound", b, a, err)
	}
	if got, err := s.Transcriptions().Get(ctx, a, id); err != nil || got.Revision != 1 || got.Document != "{}" {
		t.Errorf("the owner's draft must be untouched: got %+v, %v", got, err)
	}
}

// checkDuelIsolation: another tenant can neither read a Duel nor rewrite or
// delete it, nor read or write the origin of a Match it does not hold.
func checkDuelIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	id, err := s.Duels().Save(ctx, a, &storage.Duel{FormatVersion: "1", Document: "{}", DiceSeed: "seed"})
	if err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}
	if _, err := s.Duels().Get(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, duel of %s): got %v, want ErrNotFound", b, a, err)
	}
	for row, err := range s.Duels().List(ctx, b) {
		if err == nil && row.ID == id {
			t.Errorf("List(%s) shows the duel of %s", b, a)
		}
	}
	foreign := &storage.Duel{ID: id, FormatVersion: "1", Document: `{"x":1}`, Revision: 1}
	if _, err := s.Duels().Save(ctx, b, foreign); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Save(%s, duel of %s): got %v, want ErrNotFound", b, a, err)
	}
	if err := s.Duels().Delete(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Delete(%s, duel of %s): got %v, want ErrNotFound", b, a, err)
	}
	if got, err := s.Duels().Get(ctx, a, id); err != nil || got.Document != "{}" {
		t.Errorf("the owner's duel must be untouched: got %+v, %v", got, err)
	}

	matchID, err := s.Matches().Save(ctx, a, &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3, MatchHash: "duel-iso"})
	if err != nil {
		t.Fatalf("Save match(%s): %v", a, err)
	}
	if err := s.Duels().SetOrigin(ctx, b, &storage.MatchOrigin{MatchID: matchID, DiceSeed: "x"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetOrigin(%s, match of %s): got %v, want ErrNotFound", b, a, err)
	}
	if err := s.Duels().SetOrigin(ctx, a, &storage.MatchOrigin{MatchID: matchID, DiceSeed: "seed"}); err != nil {
		t.Fatalf("SetOrigin(%s): %v", a, err)
	}
	if _, err := s.Duels().Origin(ctx, b, matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Origin(%s, match of %s): got %v, want ErrNotFound", b, a, err)
	}
}
