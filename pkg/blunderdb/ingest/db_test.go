package ingest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// makeSourceDB builds a populated native .db file by importing an XG fixture
// (which exercises positions, analyses and comments) via the legacy Database,
// and returns its path.
func makeSourceDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "source.db")
	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	xg := filepath.Join("..", "..", "..", "testdata", "match_with_comment.xg")
	if _, err := db.ImportXGMatch(xg); err != nil {
		db.Close()
		t.Fatalf("seed ImportXGMatch: %v", err)
	}
	db.Close()
	return dbPath
}

// TestDBImportParity verifies the native .db importer faithfully copies a source
// database's position library (positions + analyses + comments) into a fresh
// Storage: importing source.db into an empty target must reproduce exactly the
// source's positions, analyses (field-by-field, timestamps ignored) and
// comments.
func TestDBImportParity(t *testing.T) {
	ctx := context.Background()
	srcPath := makeSourceDB(t)

	// Expected = the raw stored content of the source database.
	srcStore, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	expected := rawAnalyses(t, ctx, srcStore)
	srcStore.Close()

	// Import source.db into a fresh in-memory target via DBImporter.
	target, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	defer target.Close()

	sum, err := DBImporter{S: target}.Import(ctx, "", Source{Format: FormatNativeDB, Path: srcPath}, nil)
	if err != nil {
		t.Fatalf("DBImporter.Import: %v", err)
	}
	if sum.SavedPositions != len(expected) {
		t.Fatalf("saved positions = %d, want %d", sum.SavedPositions, len(expected))
	}

	got := rawAnalyses(t, ctx, target)

	if len(got) != len(expected) {
		t.Fatalf("position count mismatch: source=%d imported=%d", len(expected), len(got))
	}

	var mismatches int
	for key, ed := range expected {
		gd, ok := got[key]
		if !ok {
			t.Fatalf("position present in source but missing after import:\n%s", key)
		}
		if (ed.analysis == nil) != (gd.analysis == nil) {
			t.Errorf("analysis presence mismatch:\n%s\n source=%v imported=%v",
				key, ed.analysis != nil, gd.analysis != nil)
			mismatches++
			continue
		}
		en := normaliseAnalysisForCompare(ed.analysis)
		gn := normaliseAnalysisForCompare(gd.analysis)
		if !reflect.DeepEqual(en, gn) {
			mismatches++
			if mismatches <= 3 {
				t.Errorf("analysis mismatch:\n%s\n source=%+v\n imported=%+v", key, en, gn)
			}
		}
		if !reflect.DeepEqual(ed.comments, gd.comments) {
			mismatches++
			if mismatches <= 3 {
				t.Errorf("comment mismatch:\n%s\n source=%q imported=%q", key, ed.comments, gd.comments)
			}
		}
	}
	if mismatches > 0 {
		t.Fatalf("%d mismatch(es)", mismatches)
	}
}

// TestDBImportDedup verifies a second import of the same .db adds nothing new
// (positions dedup by content; analyses/comments already present).
func TestDBImportDedup(t *testing.T) {
	ctx := context.Background()
	srcPath := makeSourceDB(t)

	target, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	defer target.Close()

	imp := DBImporter{S: target}
	if _, err := imp.Import(ctx, "", Source{Format: FormatNativeDB, Path: srcPath}, nil); err != nil {
		t.Fatalf("first import: %v", err)
	}
	first, err := target.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := imp.Import(ctx, "", Source{Format: FormatNativeDB, Path: srcPath}, nil); err != nil {
		t.Fatalf("second import: %v", err)
	}
	second, err := target.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Positions != second.Positions {
		t.Fatalf("positions grew on re-import: %d → %d", first.Positions, second.Positions)
	}
}
