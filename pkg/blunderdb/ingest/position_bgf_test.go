package ingest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// bgfTextFixtures are BGBlitz single-position text exports (a checker position
// with move evaluations, plus cube decisions in several languages).
func bgfTextFixtures() []string {
	names := []string{
		"01_checkerPosition_FR.txt",
		"04_DP_EN.txt",
		"02_NDT_FR.txt",
		"06_RT_FR.txt",
	}
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = filepath.Join("..", "..", "..", "testdata", "bgf_positions", n)
	}
	return out
}

func legacyBGFTextImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("legacy SetupDatabase: %v", err)
	}
	if _, err := db.ImportBGFPosition(path); err != nil {
		db.Close()
		t.Fatalf("legacy ImportBGFPosition(%s): %v", path, err)
	}
	db.Close()

	ctx := context.Background()
	s, err := sqlite.Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("reopen legacy db: %v", err)
	}
	defer s.Close()
	return rawAnalyses(t, ctx, s)
}

func ingestBGFTextImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	if _, err := (PositionImporter{S: s}).Import(ctx, "", Source{Format: FormatPosition, Path: path}, nil); err != nil {
		t.Fatalf("PositionImporter.Import(%s): %v", path, err)
	}
	return rawAnalyses(t, ctx, s)
}

// TestBGFTextImportParity is the parity gate for the BGBlitz text single-position
// mapper: PositionImporter must produce the same position and analysis
// (field-by-field, timestamps ignored) as database.ImportBGFPosition.
func TestBGFTextImportParity(t *testing.T) {
	for _, path := range bgfTextFixtures() {
		t.Run(filepath.Base(path), func(t *testing.T) {
			legacy := legacyBGFTextImport(t, path)
			fresh := ingestBGFTextImport(t, path)

			if len(legacy) != len(fresh) {
				t.Fatalf("position count mismatch: legacy=%d ingest=%d", len(legacy), len(fresh))
			}
			for key := range legacy {
				if _, ok := fresh[key]; !ok {
					t.Fatalf("position present in legacy but missing from ingest:\n%s", key)
				}
			}

			var mismatches int
			for key, ld := range legacy {
				fd := fresh[key]
				la, fa := ld.analysis, fd.analysis
				if (la == nil) != (fa == nil) {
					t.Errorf("analysis presence mismatch for position:\n%s\n legacy=%v ingest=%v",
						key, la != nil, fa != nil)
					mismatches++
					continue
				}
				ln := normaliseAnalysisForCompare(la)
				fn := normaliseAnalysisForCompare(fa)
				if !reflect.DeepEqual(ln, fn) {
					mismatches++
					if mismatches <= 3 {
						t.Errorf("analysis mismatch for position:\n%s\n legacy=%+v\n ingest=%+v", key, ln, fn)
					}
				}
			}
			if mismatches > 0 {
				t.Fatalf("%d mismatch(es)", mismatches)
			}
		})
	}
}
