package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestSQLiteExportRoundTrip populates a source store, exports it to a real
// SQLite file, reopens that file as a SQLite storage, and verifies the families
// round-trip. This is the core M4-T07 guarantee: the export is a valid,
// Desktop-openable blunderDB database.
func TestSQLiteExportRoundTrip(t *testing.T) {
	ctx := context.Background()

	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	// A position with a comment.
	p := domain.InitializePosition()
	pid, err := src.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Comments().Add(ctx, "", pid, "study this"); err != nil {
		t.Fatal(err)
	}

	// A collection holding it.
	cid, err := src.Collections().Create(ctx, "", "Openings", "my openings")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Collections().AddPosition(ctx, "", cid, pid); err != nil {
		t.Fatal(err)
	}

	// A deck sourced from the collection.
	if _, err := src.Anki().CreateDeck(ctx, "", "Deck", "", "collection", cid, ""); err != nil {
		t.Fatal(err)
	}

	// A saved filter.
	if _, err := src.Filters().Save(ctx, "", "winners", "winrate>50"); err != nil {
		t.Fatal(err)
	}

	// Export to a real file.
	outPath := filepath.Join(t.TempDir(), "export.sqlite")
	out, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := (SQLiteExporter{S: src}).Export(ctx, "", out, WholeTenant(FormatSQLite)); err != nil {
		out.Close()
		t.Fatalf("export: %v", err)
	}
	out.Close()

	// Reopen the exported file as a SQLite database (as Desktop would).
	dst, err := sqlite.Open(ctx, outPath, nil)
	if err != nil {
		t.Fatalf("reopen export: %v", err)
	}
	defer dst.Close()

	// Schema version present → it's a valid blunderDB file.
	if v, err := dst.Version(ctx); err != nil || v == "" {
		t.Fatalf("export has no schema version (v=%q err=%v)", v, err)
	}

	counts, err := dst.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Positions != 1 {
		t.Fatalf("exported positions = %d, want 1", counts.Positions)
	}

	// Comment carried across (the position id is reassigned, so read by listing).
	var sawComment bool
	for c, err := range dst.Comments().ListAll(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if c.Text == "study this" {
			sawComment = true
		}
	}
	if !sawComment {
		t.Fatal("exported file missing the comment")
	}

	// Collection + its membership.
	var collID int64
	var collCount int
	for c, err := range dst.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		collCount++
		collID = c.ID
	}
	if collCount != 1 {
		t.Fatalf("exported collections = %d, want 1", collCount)
	}
	var members int
	for _, err := range dst.Collections().Positions(ctx, "", collID, storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		members++
	}
	if members != 1 {
		t.Fatalf("exported collection membership = %d, want 1", members)
	}

	// Deck + filter present.
	var deckCount int
	for _, err := range dst.Anki().ListDecks(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		deckCount++
	}
	if deckCount != 1 {
		t.Fatalf("exported decks = %d, want 1", deckCount)
	}
	var filterCount int
	for _, err := range dst.Filters().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		filterCount++
	}
	if filterCount != 1 {
		t.Fatalf("exported filters = %d, want 1", filterCount)
	}
}

// TestSQLiteExportEmptyTenant ensures an empty tenant still produces a valid,
// openable (but empty) SQLite file.
func TestSQLiteExportEmptyTenant(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	outPath := filepath.Join(t.TempDir(), "empty.sqlite")
	out, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := (SQLiteExporter{S: src}).Export(ctx, "", out, WholeTenant(FormatSQLite)); err != nil {
		out.Close()
		t.Fatalf("export: %v", err)
	}
	out.Close()

	dst, err := sqlite.Open(ctx, outPath, nil)
	if err != nil {
		t.Fatalf("reopen empty export: %v", err)
	}
	defer dst.Close()
	counts, err := dst.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Positions != 0 {
		t.Fatalf("empty export positions = %d, want 0", counts.Positions)
	}
}

// TestSQLiteExportTranscriptions pins the rule the transcription step was
// added under (#334): a draft travels with a WHOLE export and with nothing
// else. No Selection flag addresses drafts, so an export that filters
// anything carries none — the recipient's file holds the empty table
// sqlite.Bootstrap created, not a silent subset.
func TestSQLiteExportTranscriptions(t *testing.T) {
	ctx := context.Background()

	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	p := domain.InitializePosition()
	if _, err := src.Positions().Save(ctx, "", &p); err != nil {
		t.Fatal(err)
	}
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, MatchHash: "h-tr"}
	matchID, err := src.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Transcriptions().Save(ctx, "", &storage.Transcription{
		FormatVersion: "1", MatchID: matchID, Label: "Alice — Bob", Document: `{"actions":[]}`,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Transcriptions().Save(ctx, "", &storage.Transcription{
		FormatVersion: "1", Label: "brouillon libre", Document: `{"actions":[1]}`,
	}); err != nil {
		t.Fatal(err)
	}

	exportTo := func(t *testing.T, opts ExportOptions) storage.Storage {
		t.Helper()
		outPath := filepath.Join(t.TempDir(), "export.sqlite")
		out, err := os.Create(outPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := (SQLiteExporter{S: src}).Export(ctx, "", out, opts); err != nil {
			out.Close()
			t.Fatalf("export: %v", err)
		}
		out.Close()
		dst, err := sqlite.Open(ctx, outPath, nil)
		if err != nil {
			t.Fatalf("reopen export: %v", err)
		}
		t.Cleanup(func() { dst.Close() })
		return dst
	}

	list := func(t *testing.T, s storage.Storage) []*storage.Transcription {
		t.Helper()
		var out []*storage.Transcription
		for tr, err := range s.Transcriptions().List(ctx, "") {
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, tr)
		}
		return out
	}

	t.Run("whole export carries them", func(t *testing.T) {
		dst := exportTo(t, WholeTenant(FormatSQLite))
		got := list(t, dst)
		if len(got) != 2 {
			t.Fatalf("exported transcriptions = %d, want 2", len(got))
		}
		var linked, free *storage.Transcription
		for _, tr := range got {
			if tr.Label == "Alice — Bob" {
				linked = tr
			} else {
				free = tr
			}
		}
		if linked == nil || free == nil {
			t.Fatalf("exported drafts do not carry their labels: %+v", got)
		}
		if linked.Document != `{"actions":[]}` || linked.FormatVersion != "1" {
			t.Errorf("the document travels verbatim with its own version: got %+v", linked)
		}
		if free.MatchID != 0 {
			t.Errorf("a draft that produced no match keeps none: got matchID=%d", free.MatchID)
		}
		// The link points at the match as the EXPORT numbered it, not at the
		// source's id: ids are reassigned, and a raw copy would dangle.
		var exportedMatch int64
		for em, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			exportedMatch = em.ID
		}
		if exportedMatch == 0 {
			t.Fatal("the export carries no match to link to")
		}
		if linked.MatchID != exportedMatch {
			t.Errorf("draft link: got matchID=%d, want the exported match %d", linked.MatchID, exportedMatch)
		}
	})

	t.Run("a filtered export carries none", func(t *testing.T) {
		opts := WholeTenant(FormatSQLite)
		opts.Selection.AllMatches = false
		opts.Selection.MatchIDs = []int64{matchID}
		if got := list(t, exportTo(t, opts)); len(got) != 0 {
			t.Fatalf("a filtered export carried %d transcriptions, want 0", len(got))
		}
	})
}
