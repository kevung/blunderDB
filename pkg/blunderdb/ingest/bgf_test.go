package ingest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/bgfparser"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func bgfFixtures() []string {
	names := []string{
		"TachiAI_V_player_Nov_2__2025__16_55.bgf",
	}
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = filepath.Join("..", "..", "..", "testdata", n)
	}
	return out
}

func legacyBGFImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("legacy SetupDatabase: %v", err)
	}
	if _, err := db.ImportBGFMatch(path); err != nil {
		db.Close()
		t.Fatalf("legacy ImportBGFMatch(%s): %v", path, err)
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

func ingestBGFImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	graph, err := MapBGF(path)
	if err != nil {
		t.Fatalf("MapBGF(%s): %v", path, err)
	}
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if _, err := WriteMatch(ctx, tx, "", graph, nil); err != nil {
		tx.Rollback()
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return rawAnalyses(t, ctx, s)
}

// TestBGFImportParity is the parity gate for the BGF mapper: MapBGF + WriteMatch
// must produce the same positions and analyses (field-by-field, timestamps
// ignored) as database.ImportBGFMatch.
func TestBGFImportParity(t *testing.T) {
	for _, path := range bgfFixtures() {
		t.Run(filepath.Base(path), func(t *testing.T) {
			legacy := legacyBGFImport(t, path)
			fresh := ingestBGFImport(t, path)

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

// TestBGFHashParity locks the copied BGF hashes to the database originals.
func TestBGFHashParity(t *testing.T) {
	for _, path := range bgfFixtures() {
		match, err := bgfparser.ParseBGF(path)
		if err != nil {
			t.Fatalf("ParseBGF(%s): %v", path, err)
		}
		if got, want := computeBGFMatchHash(match), database.ComputeBGFMatchHash(match); got != want {
			t.Errorf("%s: match hash mismatch\n ingest=%s\n   db  =%s", filepath.Base(path), got, want)
		}
		if got, want := computeCanonicalMatchHashFromBGF(match), database.ComputeCanonicalMatchHashFromBGF(match); got != want {
			t.Errorf("%s: canonical hash mismatch\n ingest=%s\n   db  =%s", filepath.Base(path), got, want)
		}
	}
}
