package ingest

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The daemon's .db import applies the desktop's rule: a source from before
// 2.41.0 leaves its gammonNet verdicts on take/pass positions behind, and
// brings every other verdict; a 2.41.0 source brings them all.
func TestDBImportStaleResponseVerdicts(t *testing.T) {
	ctx := context.Background()
	response := func(variant int) domain.Position {
		p := domain.InitializePosition()
		p.DecisionType = domain.CubeAction
		p.Board.Points[7+variant] = domain.Point{Checkers: 1, Color: domain.Black}
		p.Cube = domain.Cube{Value: 1, Owner: domain.None}
		return p
	}
	for _, c := range []struct {
		version   string
		gnArrives bool
	}{{"2.40.0", false}, {domain.DatabaseVersion, true}} {
		srcPath := filepath.Join(t.TempDir(), "source.db")
		src, err := sqlite.Open(ctx, srcPath, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := map[uint64]bool{}
		for i, e := range []struct {
			pos    domain.Position
			engine string
			keep   bool
		}{
			{response(0), "gammonNet v1.6.0", c.gnArrives},
			{response(1), "XG", true},
			{domain.InitializePosition(), "gammonNet v1.6.0", true},
		} {
			id, err := src.Positions().Save(ctx, "", &e.pos)
			if err != nil {
				t.Fatal(err)
			}
			a := &domain.PositionAnalysis{AnalysisType: "DoublingCube", AnalysisEngineVersion: e.engine,
				DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "0-ply", AnalysisEngine: e.engine, BestCubeAction: "Double, Take"}}
			if err := src.Analyses().Save(ctx, "", id, a); err != nil {
				t.Fatalf("analysis %d: %v", i, err)
			}
			want[engine.ZobristHash(&e.pos)] = e.keep
		}
		src.Close()
		raw, err := sql.Open("sqlite", srcPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, c.version); err != nil {
			t.Fatal(err)
		}
		_ = raw.Close()

		target, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := (DBImporter{S: target}).Import(ctx, "", Source{Format: FormatNativeDB, Path: srcPath}, nil); err != nil {
			t.Fatalf("%s: import: %v", c.version, err)
		}
		ids, err := target.Positions().ListIDs(ctx, "", storage.ListOpts{})
		if err != nil {
			t.Fatal(err)
		}
		positions, err := target.Positions().LoadByIDs(ctx, "", ids)
		if err != nil {
			t.Fatal(err)
		}
		if len(positions) != len(want) {
			t.Fatalf("%s: %d positions imported, want %d", c.version, len(positions), len(want))
		}
		for i := range positions {
			_, err := target.Analyses().Load(ctx, "", positions[i].ID)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				t.Fatal(err)
			}
			if got, w := err == nil, want[engine.ZobristHash(&positions[i])]; got != w {
				t.Errorf("%s: position %d analysis imported %v, want %v", c.version, positions[i].ID, got, w)
			}
		}
		target.Close()
	}
}
