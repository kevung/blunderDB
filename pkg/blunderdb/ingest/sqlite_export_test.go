package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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
	if err := (SQLiteExporter{S: src}).Export(ctx, "", out, ExportOptions{Format: FormatSQLite}); err != nil {
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
	for c, err := range dst.Comments().ListAll(ctx, "") {
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
	for _, err := range dst.Collections().Positions(ctx, "", collID) {
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
	if err := (SQLiteExporter{S: src}).Export(ctx, "", out, ExportOptions{Format: FormatSQLite}); err != nil {
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
