// Contract cases for transcriptions: the draft's CRUD, the order the library
// list gets, and what deleting the produced match does to the draft.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testTranscriptionCRUD pins the draft's life cycle: an insert returns an id,
// Get reads back every field, Save on an existing row rewrites in place
// rather than appending, and Delete removes it once and reports ErrNotFound
// after that.
func testTranscriptionCRUD(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ts := s.Transcriptions()

	id, err := ts.Save(ctx, "", &storage.Transcription{
		FormatVersion: "1",
		Label:         "Alice — Bob",
		Document:      `{"actions":[]}`,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}

	got, err := ts.Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != id || got.FormatVersion != "1" || got.Label != "Alice — Bob" || got.Document != `{"actions":[]}` {
		t.Errorf("Get: got %+v", got)
	}
	if got.MatchID != 0 {
		t.Errorf("a draft that has produced no match must read back MatchID 0, got %d", got.MatchID)
	}
	if got.CreatedAt == "" || got.UpdatedAt == "" {
		t.Errorf("timestamps are the store's to write: got created=%q updated=%q", got.CreatedAt, got.UpdatedAt)
	}

	// Saving an existing draft rewrites its row: the document is opaque and
	// is replaced whole, and no second row appears.
	got.Document = `{"actions":[{"kind":"roll"}]}`
	got.Label = "Alice — Bob, finale"
	if _, err := ts.Save(ctx, "", got); err != nil {
		t.Fatalf("Save (update): %v", err)
	}
	again, err := ts.Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if again.Document != `{"actions":[{"kind":"roll"}]}` || again.Label != "Alice — Bob, finale" {
		t.Errorf("update did not take: got %+v", again)
	}
	if n := countTranscriptions(t, ts); n != 1 {
		t.Errorf("after an update the scope holds %d drafts, want 1", n)
	}

	if err := ts.Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := ts.Get(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want ErrNotFound", err)
	}
	if err := ts.Delete(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Delete twice: got %v, want ErrNotFound", err)
	}
	if _, err := ts.Save(ctx, "", &storage.Transcription{ID: id, FormatVersion: "1", Document: "{}"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Save onto a deleted draft: got %v, want ErrNotFound", err)
	}
}

// testTranscriptionListNewestFirst pins the order the library list reads: the
// draft touched last comes first, which is the one the user was typing in.
func testTranscriptionListNewestFirst(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ts := s.Transcriptions()

	var ids []int64
	for _, label := range []string{"un", "deux", "trois"} {
		id, err := ts.Save(ctx, "", &storage.Transcription{FormatVersion: "1", Label: label, Document: "{}"})
		if err != nil {
			t.Fatalf("Save %q: %v", label, err)
		}
		ids = append(ids, id)
	}

	var got []int64
	for tr, err := range ts.List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		got = append(got, tr.ID)
	}
	// Three saves inside one second share a timestamp on SQLite, so the id
	// tiebreak is what makes the order total — and it must still be newest
	// first.
	want := []int64{ids[2], ids[1], ids[0]}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("List order: got %v, want %v", got, want)
	}
}

// testTranscriptionMatchLink pins ADR-0045's link between a draft and the
// Match it produced: the draft points at the match, an unknown match is
// refused, and deleting the match sets the link back to NULL instead of
// destroying the typing that produced it.
func testTranscriptionMatchLink(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ts := s.Transcriptions()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, MatchHash: "tr-h1"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}

	id, err := ts.Save(ctx, "", &storage.Transcription{
		FormatVersion: "1", MatchID: matchID, Label: "Alice — Bob", Document: "{}",
	})
	if err != nil {
		t.Fatalf("Save linked draft: %v", err)
	}
	linked, err := ts.Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get linked draft: %v", err)
	}
	if linked.MatchID != matchID {
		t.Fatalf("Get linked draft: got matchID=%d, want %d", linked.MatchID, matchID)
	}

	// A match id that names nothing is a dangling reference, not a silent
	// write (Errors/DanglingReferenceIsNotFound applies here too).
	if _, err := ts.Save(ctx, "", &storage.Transcription{
		FormatVersion: "1", MatchID: matchID + 10_000, Document: "{}",
	}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Save with an unknown match: got %v, want ErrNotFound", err)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade match: %v", err)
	}
	survivor, err := ts.Get(ctx, "", id)
	if err != nil {
		t.Fatalf("the draft must survive the deletion of its match: %v", err)
	}
	if survivor.MatchID != 0 {
		t.Errorf("after the match is deleted the link is NULL: got matchID=%d, want 0", survivor.MatchID)
	}
	if survivor.Document != "{}" {
		t.Errorf("the typing must be untouched: got document %q", survivor.Document)
	}
}

func countTranscriptions(t *testing.T, ts storage.TranscriptionStore) int {
	t.Helper()
	n := 0
	for _, err := range ts.List(context.Background(), "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		n++
	}
	return n
}
