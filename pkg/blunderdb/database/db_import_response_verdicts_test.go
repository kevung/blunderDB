package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A source from before 2.41.0 holds gammonNet's verdicts on take/pass
// positions scored as the answerer's centred-cube decision. Importing it
// leaves them behind, as opening it would drop them, whether the position
// is new here or already held; other engines' verdicts and gammonNet's on
// other positions come along. A 2.41.0 source's verdicts all come along.
func TestImportDatabase_StaleResponseVerdicts(t *testing.T) {
	ctx := context.Background()
	response := func(variant int) domain.Position {
		p := initialPosition()
		p.DecisionType = domain.CubeAction
		p.Board.Points[7+variant] = domain.Point{Checkers: 1, Color: domain.Black}
		p.Cube = domain.Cube{Value: 1, Owner: domain.None}
		return p
	}
	analyse := func(d *Database, id int64, label string) {
		t.Helper()
		a := &domain.PositionAnalysis{AnalysisType: "DoublingCube", AnalysisEngineVersion: label,
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "0-ply", AnalysisEngine: label, BestCubeAction: "Double, Take"}}
		if err := d.store.Analyses().Save(ctx, "", id, a); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		version   string
		gnArrives bool
	}{{"2.40.0", false}, {DatabaseVersion, true}} {
		src := newTestDB(t)
		ids := map[string]int64{}
		for name, p := range map[string]domain.Position{
			"gnNew": response(0), "gnHeld": response(1), "xg": response(2), "gnOther": initialPosition(),
		} {
			id, err := src.SavePosition(&p)
			if err != nil {
				t.Fatal(err)
			}
			ids[name] = id
		}
		analyse(src, ids["gnNew"], "gammonNet v1.6.0")
		analyse(src, ids["gnHeld"], "gammonNet v1.6.0")
		analyse(src, ids["xg"], "XG")
		analyse(src, ids["gnOther"], "gammonNet v1.6.0")
		path := exportTo(t, src, filepath.Join(t.TempDir(), "source.db"), ExportOptions{AllPositions: true, IncludeAnalysis: true})
		raw, err := openExistingSQLite(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, c.version); err != nil {
			t.Fatal(err)
		}
		_ = raw.Close()

		dst := newTestDB(t)
		held := response(1)
		if _, err := dst.SavePosition(&held); err != nil {
			t.Fatal(err)
		}
		if _, err := dst.CommitImportDatabase(path); err != nil {
			t.Fatalf("%s: import: %v", c.version, err)
		}
		for name, want := range map[string]bool{"gnNew": c.gnArrives, "gnHeld": c.gnArrives, "xg": true, "gnOther": true} {
			p, err := src.store.Positions().Load(ctx, "", ids[name])
			if err != nil {
				t.Fatal(err)
			}
			id, found, err := heldByZobrist(dst.store.Positions(), p)
			if err != nil || !found {
				t.Fatalf("%s: %s not imported: %v", c.version, name, err)
			}
			_, err = dst.store.Analyses().Load(ctx, "", id)
			if got := err == nil; got != want || (err != nil && !errors.Is(err, storage.ErrNotFound)) {
				t.Errorf("%s: %s analysis imported %v (%v), want %v", c.version, name, got, err, want)
			}
		}
	}
}
