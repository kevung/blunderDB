package storagetest

import (
	"context"
	"errors"
	"testing"

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
