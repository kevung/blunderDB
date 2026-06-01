package ingest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/gnubgparser"
)

// gnubgFixtures are the repo-root testdata GnuBG files (SGF carries analysis;
// MAT is moves-only), relative to this package dir.
func gnubgFixtures() []string {
	names := []string{
		"test.sgf",
		"charlot1-charlot2_7p_2025-11-08-2305.sgf",
		"test.mat",
		"charlot1-charlot2_7p_2025-11-08-2305.mat",
	}
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = filepath.Join("..", "..", "..", "testdata", n)
	}
	return out
}

// legacyGnuBGImport imports path via Database.ImportGnuBGMatch into a temp file,
// then reopens it through the storage layer for a raw read (see legacyImport).
func legacyGnuBGImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("legacy SetupDatabase: %v", err)
	}
	if _, err := db.ImportGnuBGMatch(path); err != nil {
		db.Close()
		t.Fatalf("legacy ImportGnuBGMatch(%s): %v", path, err)
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

// ingestGnuBGImport imports path via MapGnuBG + WriteMatch into a fresh Storage.
func ingestGnuBGImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	graph, err := MapGnuBG(path)
	if err != nil {
		t.Fatalf("MapGnuBG(%s): %v", path, err)
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

// TestGnuBGImportParity is the parity gate for the GnuBG mapper: MapGnuBG +
// WriteMatch must produce the same positions and analyses (field-by-field,
// timestamps ignored) as database.ImportGnuBGMatch, for both SGF (with
// analysis) and MAT (moves-only) fixtures.
func TestGnuBGImportParity(t *testing.T) {
	for _, path := range gnubgFixtures() {
		t.Run(filepath.Base(path), func(t *testing.T) {
			legacy := legacyGnuBGImport(t, path)
			fresh := ingestGnuBGImport(t, path)

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

// TestGnuBGHashParity locks the copied GnuBG hashes to the database originals.
func TestGnuBGHashParity(t *testing.T) {
	for _, path := range gnubgFixtures() {
		var match *gnubgparser.Match
		var err error
		if filepath.Ext(path) == ".sgf" {
			match, err = gnubgparser.ParseSGFFile(path)
		} else {
			match, err = gnubgparser.ParseMATFile(path)
		}
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if got, want := computeGnuBGMatchHash(match), database.ComputeGnuBGMatchHash(match); got != want {
			t.Errorf("%s: match hash mismatch\n ingest=%s\n   db  =%s", filepath.Base(path), got, want)
		}
		if got, want := computeCanonicalMatchHashFromGnuBG(match), database.ComputeCanonicalMatchHashFromGnuBG(match); got != want {
			t.Errorf("%s: canonical hash mismatch\n ingest=%s\n   db  =%s", filepath.Base(path), got, want)
		}
	}
}
