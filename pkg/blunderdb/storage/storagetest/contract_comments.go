// Contract cases for comments: the per-position wall, the concatenated text
// and the database-wide listing and search.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// drainComments collects the entries of a comment stream.
func drainComments(t *testing.T, what string, seq func(func(*domain.CommentEntry, error) bool)) []domain.CommentEntry {
	t.Helper()
	var out []domain.CommentEntry
	for e, err := range seq {
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		out = append(out, *e)
	}
	return out
}

// commentIDs projects the ids of a comment slice.
func commentIDs(entries []domain.CommentEntry) []int64 {
	ids := make([]int64, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	return ids
}

// testCommentCRUD pins the life cycle of one position's comment wall: Add,
// Update (same id, modified_at set), Delete, DeleteForPosition, and the two
// read shapes — ByPosition streams the entries most recent first, Text joins
// them oldest first with blank lines. An empty entry is not a comment: it is
// stored (Add succeeds) but invisible to both reads, matching the search
// filter's notion of "has a comment".
func testCommentCRUD(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	comments := s.Comments()

	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	posA, posB := save(1), save(2)

	first, err := comments.Add(ctx, "", posA, "first note")
	if err != nil {
		t.Fatalf("Add first: %v", err)
	}
	if first == 0 {
		t.Fatal("Add returned id 0")
	}
	second, err := comments.Add(ctx, "", posA, "second note")
	if err != nil {
		t.Fatalf("Add second: %v", err)
	}
	if _, err := comments.Add(ctx, "", posA, ""); err != nil {
		t.Fatalf("Add empty entry: %v", err)
	}
	other, err := comments.Add(ctx, "", posB, "other position")
	if err != nil {
		t.Fatalf("Add on posB: %v", err)
	}

	// ByPosition: most recent first, the empty entry skipped, PositionID set.
	wall := drainComments(t, "ByPosition", comments.ByPosition(ctx, "", posA))
	if got := commentIDs(wall); len(got) != 2 || got[0] != second || got[1] != first {
		t.Errorf("ByPosition ids: got %v, want [%d %d] (most recent first, empty entry hidden)", got, second, first)
	}
	for _, e := range wall {
		if e.PositionID != posA {
			t.Errorf("entry %d PositionID: got %d, want %d", e.ID, e.PositionID, posA)
		}
		if e.CreatedAt == "" {
			t.Errorf("entry %d CreatedAt is empty", e.ID)
		}
		if e.ModifiedAt != "" {
			t.Errorf("entry %d ModifiedAt before any Update: got %q, want empty", e.ID, e.ModifiedAt)
		}
	}

	// Text: oldest first, blank-line separated, empty entry skipped.
	if text, err := comments.Text(ctx, "", posA); err != nil {
		t.Fatalf("Text: %v", err)
	} else if text != "first note\n\nsecond note" {
		t.Errorf("Text: got %q, want %q", text, "first note\n\nsecond note")
	}

	// Update keeps the id, changes the text, stamps modified_at.
	if err := comments.Update(ctx, "", first, "first note, edited"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	wall = drainComments(t, "ByPosition after Update", comments.ByPosition(ctx, "", posA))
	if got := commentIDs(wall); len(got) != 2 || got[0] != second || got[1] != first {
		t.Errorf("ByPosition ids after Update: got %v, want [%d %d]", got, second, first)
	}
	for _, e := range wall {
		if e.ID == first {
			if e.Text != "first note, edited" {
				t.Errorf("Update text: got %q, want %q", e.Text, "first note, edited")
			}
			if e.ModifiedAt == "" {
				t.Error("ModifiedAt after Update is empty")
			}
		}
	}
	if text, _ := comments.Text(ctx, "", posA); text != "first note, edited\n\nsecond note" {
		t.Errorf("Text after Update: got %q", text)
	}

	// Updating an entry to "" hides it: the wall treats it as absent.
	if err := comments.Update(ctx, "", second, ""); err != nil {
		t.Fatalf("Update to empty: %v", err)
	}
	if got := commentIDs(drainComments(t, "ByPosition after blanking", comments.ByPosition(ctx, "", posA))); len(got) != 1 || got[0] != first {
		t.Errorf("ByPosition after blanking %d: got %v, want [%d]", second, got, first)
	}

	// Delete removes exactly one entry.
	if err := comments.Delete(ctx, "", first); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if text, _ := comments.Text(ctx, "", posA); text != "" {
		t.Errorf("Text after deleting the last visible entry: got %q, want empty", text)
	}
	if n := len(drainComments(t, "ByPosition after Delete", comments.ByPosition(ctx, "", posA))); n != 0 {
		t.Errorf("ByPosition after Delete: got %d entries, want 0", n)
	}

	// DeleteForPosition clears one position and leaves the others alone.
	if _, err := comments.Add(ctx, "", posA, "third note"); err != nil {
		t.Fatalf("Add third: %v", err)
	}
	if err := comments.DeleteForPosition(ctx, "", posA); err != nil {
		t.Fatalf("DeleteForPosition: %v", err)
	}
	if n := len(drainComments(t, "ByPosition after DeleteForPosition", comments.ByPosition(ctx, "", posA))); n != 0 {
		t.Errorf("ByPosition after DeleteForPosition: got %d entries, want 0", n)
	}
	if got := commentIDs(drainComments(t, "ByPosition posB", comments.ByPosition(ctx, "", posB))); len(got) != 1 || got[0] != other {
		t.Errorf("posB wall after clearing posA: got %v, want [%d]", got, other)
	}
	if text, _ := comments.Text(ctx, "", posB); text != "other position" {
		t.Errorf("posB Text after clearing posA: got %q", text)
	}
}

// testCommentUpsert pins the single-comment view Upsert edits: with no
// visible entry it appends one, otherwise it rewrites the oldest non-empty
// entry in place (same id, modified_at stamped) and leaves the rest of the
// wall alone. That entry is the last item of ByPosition, which is how the
// desktop reads it back before editing. An empty entry is not a comment, so
// a position that only has blank entries gets a new one; a position that
// does not exist is ErrNotFound, like Add.
func testCommentUpsert(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	comments := s.Comments()

	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	pos := save(1)

	// No entry yet: Upsert appends one.
	primary, err := comments.Upsert(ctx, "", pos, "first")
	if err != nil {
		t.Fatalf("Upsert on an empty wall: %v", err)
	}
	if primary == 0 {
		t.Fatal("Upsert returned id 0")
	}
	if got := commentIDs(drainComments(t, "ByPosition", comments.ByPosition(ctx, "", pos))); len(got) != 1 || got[0] != primary {
		t.Errorf("wall after first Upsert: got %v, want [%d]", got, primary)
	}
	if text, _ := comments.Text(ctx, "", pos); text != "first" {
		t.Errorf("Text after first Upsert: got %q, want %q", text, "first")
	}

	// A visible entry exists: rewritten in place.
	again, err := comments.Upsert(ctx, "", pos, "first, rewritten")
	if err != nil {
		t.Fatalf("Upsert again: %v", err)
	}
	if again != primary {
		t.Errorf("Upsert on an existing entry: got id %d, want %d (rewrite, not append)", again, primary)
	}
	wall := drainComments(t, "ByPosition after rewrite", comments.ByPosition(ctx, "", pos))
	if len(wall) != 1 || wall[0].Text != "first, rewritten" {
		t.Errorf("wall after rewrite: got %+v, want one entry reading %q", wall, "first, rewritten")
	} else if wall[0].ModifiedAt == "" {
		t.Error("ModifiedAt after Upsert rewrite is empty")
	}

	// Several entries: only the oldest non-empty one is rewritten; the
	// primary is the last item of ByPosition.
	second, err := comments.Add(ctx, "", pos, "second")
	if err != nil {
		t.Fatalf("Add second: %v", err)
	}
	if id, err := comments.Upsert(ctx, "", pos, "primary"); err != nil || id != primary {
		t.Errorf("Upsert with two entries: got id %d, err %v; want %d", id, err, primary)
	}
	wall = drainComments(t, "ByPosition with two entries", comments.ByPosition(ctx, "", pos))
	if len(wall) != 2 || wall[0].ID != second || wall[0].Text != "second" ||
		wall[1].ID != primary || wall[1].Text != "primary" {
		t.Errorf("wall after Upsert with two entries: got %+v, want [%d %q] then [%d %q]",
			wall, second, "second", primary, "primary")
	}

	// Once the primary is deleted the next oldest entry takes its place.
	if err := comments.Delete(ctx, "", primary); err != nil {
		t.Fatalf("Delete primary: %v", err)
	}
	if id, err := comments.Upsert(ctx, "", pos, "promoted"); err != nil || id != second {
		t.Errorf("Upsert after deleting the primary: got id %d, err %v; want %d", id, err, second)
	}
	if text, _ := comments.Text(ctx, "", pos); text != "promoted" {
		t.Errorf("Text after promotion: got %q, want %q", text, "promoted")
	}

	// An empty entry is not a comment: it is neither rewritten nor read back.
	other := save(2)
	blank, err := comments.Add(ctx, "", other, "")
	if err != nil {
		t.Fatalf("Add blank: %v", err)
	}
	fresh, err := comments.Upsert(ctx, "", other, "note")
	if err != nil {
		t.Fatalf("Upsert over a blank entry: %v", err)
	}
	if fresh == blank {
		t.Errorf("Upsert rewrote the blank entry %d; want a new entry", blank)
	}
	if got := commentIDs(drainComments(t, "ByPosition other", comments.ByPosition(ctx, "", other))); len(got) != 1 || got[0] != fresh {
		t.Errorf("wall of the other position: got %v, want [%d]", got, fresh)
	}

	// Like Add, an unknown position is the caller's mistake, not a 500.
	if _, err := comments.Upsert(ctx, "", 987654321, "orphan"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Upsert on an unknown position: got %v, want ErrNotFound", err)
	}
}

// testCommentSearchAcrossPositions pins the database-wide reads: ListAll
// streams every non-empty entry most recent first whatever its position, and
// Search narrows that stream to the entries whose text contains the query,
// case-insensitively, keeping the same order. A query that matches nothing
// yields an empty stream, not an error.
func testCommentSearchAcrossPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	comments := s.Comments()

	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	posA, posB, posC := save(1), save(2), save(3)

	add := func(pos int64, text string) int64 {
		id, err := comments.Add(ctx, "", pos, text)
		if err != nil {
			t.Fatalf("Add %q on %d: %v", text, pos, err)
		}
		return id
	}
	a1 := add(posA, "Blunder: should have hit")
	b1 := add(posB, "clean take")
	add(posC, "")
	c1 := add(posC, "another blunder, missed the double")
	a2 := add(posA, "revisited: still wrong")

	all := drainComments(t, "ListAll", comments.ListAll(ctx, ""))
	if got := commentIDs(all); len(got) != 4 ||
		got[0] != a2 || got[1] != c1 || got[2] != b1 || got[3] != a1 {
		t.Errorf("ListAll ids: got %v, want [%d %d %d %d] (most recent first, empty entry hidden)",
			got, a2, c1, b1, a1)
	}
	byID := make(map[int64]domain.CommentEntry, len(all))
	for _, e := range all {
		byID[e.ID] = e
	}
	if byID[c1].PositionID != posC || byID[b1].PositionID != posB {
		t.Errorf("ListAll PositionID: c1 on %d (want %d), b1 on %d (want %d)",
			byID[c1].PositionID, posC, byID[b1].PositionID, posB)
	}

	// Substring, case-insensitive, across positions, most recent first.
	if got := commentIDs(drainComments(t, "Search blunder", comments.Search(ctx, "", "BLUNDER"))); len(got) != 2 ||
		got[0] != c1 || got[1] != a1 {
		t.Errorf("Search BLUNDER: got %v, want [%d %d]", got, c1, a1)
	}
	if got := commentIDs(drainComments(t, "Search take", comments.Search(ctx, "", "take"))); len(got) != 1 || got[0] != b1 {
		t.Errorf("Search take: got %v, want [%d]", got, b1)
	}
	if got := drainComments(t, "Search miss", comments.Search(ctx, "", "no such text")); len(got) != 0 {
		t.Errorf("Search with no match: got %v, want nothing", commentIDs(got))
	}

	// An entry blanked by Update leaves both streams.
	if err := comments.Update(ctx, "", a1, ""); err != nil {
		t.Fatalf("Update to empty: %v", err)
	}
	if got := commentIDs(drainComments(t, "Search after blanking", comments.Search(ctx, "", "blunder"))); len(got) != 1 || got[0] != c1 {
		t.Errorf("Search after blanking %d: got %v, want [%d]", a1, got, c1)
	}
	if n := len(drainComments(t, "ListAll after blanking", comments.ListAll(ctx, ""))); n != 3 {
		t.Errorf("ListAll after blanking: got %d entries, want 3", n)
	}
}

// testCommentPositionDeleteCascades pins the promise on PositionStore.Delete:
// a position takes its comments with it, and only its own.
func testCommentPositionDeleteCascades(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	comments := s.Comments()

	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	doomed, kept := save(1), save(2)
	if _, err := comments.Add(ctx, "", doomed, "on the doomed position"); err != nil {
		t.Fatalf("Add on doomed: %v", err)
	}
	survivor, err := comments.Add(ctx, "", kept, "on the kept position")
	if err != nil {
		t.Fatalf("Add on kept: %v", err)
	}

	if err := s.Positions().Delete(ctx, "", doomed); err != nil {
		t.Fatalf("Positions.Delete: %v", err)
	}

	if n := len(drainComments(t, "ByPosition doomed", comments.ByPosition(ctx, "", doomed))); n != 0 {
		t.Errorf("comments of the deleted position: got %d, want 0", n)
	}
	if got := commentIDs(drainComments(t, "ListAll", comments.ListAll(ctx, ""))); len(got) != 1 || got[0] != survivor {
		t.Errorf("ListAll after position delete: got %v, want [%d]", got, survivor)
	}
	if got := commentIDs(drainComments(t, "Search", comments.Search(ctx, "", "position"))); len(got) != 1 || got[0] != survivor {
		t.Errorf("Search after position delete: got %v, want [%d]", got, survivor)
	}
}
