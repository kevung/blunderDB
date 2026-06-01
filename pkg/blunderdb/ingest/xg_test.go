package ingest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/xgparser/xgparser"
)

// xgFixtures are the repo-root testdata .xg files, relative to this package dir.
func xgFixtures() []string {
	names := []string{
		"charlot1-charlot2_7p_2025-11-08-2305.xg",
		"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
		"match_with_comment.xg",
		"test.xg",
	}
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = filepath.Join("..", "..", "..", "testdata", n)
	}
	return out
}

// positionKey serialises a stored position (ID-independent) so the same logical
// position lines up across two independently-imported databases.
func positionKey(p *domain.Position) string {
	c := *p
	c.ID = 0
	b, _ := json.Marshal(c)
	return string(b)
}

// normaliseAnalysisForCompare zeroes the fields that legitimately differ between
// two import runs (timestamps and the row-local position id) so the rest can be
// compared field-by-field.
func normaliseAnalysisForCompare(a *domain.PositionAnalysis) *domain.PositionAnalysis {
	if a == nil {
		return nil
	}
	c := *a
	c.PositionID = 0
	c.CreationDate = time.Time{}
	c.LastModifiedDate = time.Time{}
	return &c
}

// legacyImport imports path via the SQLite-only Database wrapper into a temp
// file, then reopens that file through the same storage layer the ingest path
// uses and returns the *raw stored* position→analysis map. Reading raw (rather
// than via Database.LoadAnalysis, which enriches played moves/cube actions from
// the move table and back-fills the deprecated singular fields at read time)
// keeps the comparison symmetric: both sides decode exactly what was persisted.
func legacyImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("legacy SetupDatabase: %v", err)
	}
	if _, err := db.ImportXGMatch(path); err != nil {
		db.Close()
		t.Fatalf("legacy ImportXGMatch(%s): %v", path, err)
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

// posData is the raw stored data for one position: its analysis (nil if none)
// and the sorted set of its comment texts.
type posData struct {
	analysis *domain.PositionAnalysis
	comments []string
}

// rawAnalyses reads every position with its raw stored analysis and comments.
func rawAnalyses(t *testing.T, ctx context.Context, s storage.Storage) map[string]*posData {
	t.Helper()
	var positions []*domain.Position
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("Positions().List: %v", err)
		}
		pc := *p
		positions = append(positions, &pc)
	}
	out := make(map[string]*posData, len(positions))
	for _, p := range positions {
		d := &posData{}
		if a, err := s.Analyses().Load(ctx, "", p.ID); err == nil {
			d.analysis = a
		}
		for c, err := range s.Comments().ByPosition(ctx, "", p.ID) {
			if err != nil {
				t.Fatalf("Comments().ByPosition: %v", err)
			}
			d.comments = append(d.comments, c.Text)
		}
		sort.Strings(d.comments)
		out[positionKey(p)] = d
	}
	return out
}

// ingestImport imports path via MapXG + WriteMatch into a fresh Storage and
// returns the same position→analysis map.
func ingestImport(t *testing.T, path string) map[string]*posData {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	graph, err := MapXG(path)
	if err != nil {
		t.Fatalf("MapXG(%s): %v", path, err)
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

// TestXGImportParity is the parity gate for the XG mapper: for every fixture, an
// import through the new backend-agnostic MapXG+WriteMatch path must produce the
// exact same positions and analyses (field-by-field, ignoring timestamps) as the
// legacy Database.ImportXGMatch path.
func TestXGImportParity(t *testing.T) {
	for _, path := range xgFixtures() {
		t.Run(filepath.Base(path), func(t *testing.T) {
			legacy := legacyImport(t, path)
			fresh := ingestImport(t, path)

			if len(legacy) != len(fresh) {
				t.Fatalf("position count mismatch: legacy=%d ingest=%d", len(legacy), len(fresh))
			}

			// Same set of positions.
			for key := range legacy {
				if _, ok := fresh[key]; !ok {
					t.Fatalf("position present in legacy but missing from ingest:\n%s", key)
				}
			}

			// Same analysis and comments per position.
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
						lb, _ := json.MarshalIndent(ln, "", "  ")
						fb, _ := json.MarshalIndent(fn, "", "  ")
						t.Errorf("analysis mismatch for position:\n%s\n--- legacy ---\n%s\n--- ingest ---\n%s",
							key, lb, fb)
					}
				}
				if !reflect.DeepEqual(ld.comments, fd.comments) {
					mismatches++
					if mismatches <= 5 {
						t.Errorf("comment mismatch for position:\n%s\n legacy=%q ingest=%q",
							key, ld.comments, fd.comments)
					}
				}
			}
			if mismatches > 0 {
				t.Fatalf("%d mismatch(es)", mismatches)
			}
		})
	}
}

// TestXGHashParity keeps ingest's copied hash functions in lock-step with the
// canonical database implementations.
func TestXGHashParity(t *testing.T) {
	for _, path := range xgFixtures() {
		imp := xgparser.NewImport(path)
		segments, err := imp.GetFileSegments()
		if err != nil {
			t.Fatalf("GetFileSegments(%s): %v", path, err)
		}
		match, err := xgparser.ParseXG(segments)
		if err != nil {
			t.Fatalf("ParseXG(%s): %v", path, err)
		}
		if got, want := computeMatchHash(match), database.ComputeMatchHash(match); got != want {
			t.Errorf("%s: match hash mismatch\n ingest=%s\n   db  =%s", filepath.Base(path), got, want)
		}
		if got, want := computeCanonicalMatchHashFromXG(match), database.ComputeCanonicalMatchHashFromXG(match); got != want {
			t.Errorf("%s: canonical hash mismatch\n ingest=%s\n   db  =%s", filepath.Base(path), got, want)
		}
	}
}
