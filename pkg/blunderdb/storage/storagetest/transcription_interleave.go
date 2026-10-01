package storagetest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// interleaveExecer runs `between` once, right after the next statement that
// rewrites a transcription row has completed, before the store reads
// anything else: the window where another writer can slip in.
type interleaveExecer struct {
	sqlshared.Execer
	between *func()
}

func (e interleaveExecer) fire(query string) {
	if f := *e.between; f != nil && strings.Contains(query, "UPDATE transcription") {
		*e.between = nil
		f()
	}
}

func (e interleaveExecer) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	n, err := e.Execer.Exec(ctx, query, args...)
	e.fire(query)
	return n, err
}

func (e interleaveExecer) QueryRow(ctx context.Context, query string, args ...any) sqlshared.Row {
	return interleaveRow{e.Execer.QueryRow(ctx, query, args...), func() { e.fire(query) }}
}

type interleaveRow struct {
	sqlshared.Row
	after func()
}

func (r interleaveRow) Scan(dest ...any) error {
	err := r.Row.Scan(dest...)
	r.after()
	return err
}

// RunTranscriptionInterleave pins that a write's revision is the one THAT
// write produced. Writer A rewrites the draft; writer B, on another
// connection, rewrites it before A reads anything back. A must come out at
// B's predecessor, so A's next write is refused (a conflict) and B's write
// survives. A revision read back after the fact would hand A B's revision and
// let A overwrite B silently. shared is s's own execer.
func RunTranscriptionInterleave(t *testing.T, s storage.Storage, shared sqlshared.Execer) {
	ctx := context.Background()
	var between func()
	a := &sqlshared.TranscriptionStore{DB: interleaveExecer{shared, &between}}
	b := s.Transcriptions()

	row := &storage.Transcription{FormatVersion: "3", Document: `{"w":"init"}`}
	id, err := b.Save(ctx, "", row)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	t.Run("Save", func(t *testing.T) {
		cur, err := b.Get(ctx, "", id)
		if err != nil {
			t.Fatal(err)
		}
		mine := &storage.Transcription{ID: id, FormatVersion: "3", Document: `{"w":"a1"}`, Revision: cur.Revision}
		between = func() {
			theirs := &storage.Transcription{ID: id, FormatVersion: "3", Document: `{"w":"b"}`, Revision: cur.Revision + 1}
			if _, err := b.Save(ctx, "", theirs); err != nil {
				t.Errorf("writer B: %v", err)
			}
		}
		if _, err := a.Save(ctx, "", mine); err != nil {
			t.Fatalf("writer A: %v", err)
		}
		if mine.Revision != cur.Revision+1 {
			t.Fatalf("writer A came out at revision %d, want %d (its own write's)", mine.Revision, cur.Revision+1)
		}
		mine.Document = `{"w":"a2"}`
		if _, err := a.Save(ctx, "", mine); !errors.Is(err, storage.ErrConflict) {
			t.Fatalf("writer A's next write: got %v, want ErrConflict", err)
		}
		if got, err := b.Get(ctx, "", id); err != nil || got.Document != `{"w":"b"}` {
			t.Fatalf("writer B's write must survive: got %+v, %v", got, err)
		}
	})

	t.Run("Touch", func(t *testing.T) {
		cur, err := b.Get(ctx, "", id)
		if err != nil {
			t.Fatal(err)
		}
		between = func() {
			theirs := &storage.Transcription{ID: id, FormatVersion: "3", Document: `{"w":"b2"}`, Revision: cur.Revision + 1}
			if _, err := b.Save(ctx, "", theirs); err != nil {
				t.Errorf("writer B: %v", err)
			}
		}
		rev, err := a.Touch(ctx, "", id, cur.Revision)
		if err != nil {
			t.Fatalf("writer A: %v", err)
		}
		if rev != cur.Revision+1 {
			t.Fatalf("writer A's Touch came out at revision %d, want %d", rev, cur.Revision+1)
		}
		if _, err := a.Touch(ctx, "", id, rev); !errors.Is(err, storage.ErrConflict) {
			t.Fatalf("writer A's next Touch: got %v, want ErrConflict", err)
		}
	})
}
