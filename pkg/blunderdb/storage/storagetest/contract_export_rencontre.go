package storagetest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// testExportCarriesRencontre: a Rencontre leaves as soon as one of its
// Tournaments does — name, dates (the source's, not the export's), table
// count, table properties, the rooms of the exported member — without its
// output folder and without the members that stay behind. A Rencontre none of
// whose Tournaments leaves stays behind, and the plan counts what is written.
func testExportCarriesRencontre(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	rs := s.Rencontres()
	rid, err := rs.Create(ctx, "", domain.Rencontre{
		Name: "Festival", StartsOn: "2026-10-03", EndsOn: "2026-10-04", Tables: 14, OutputDir: "/home/producteur/salle",
	})
	if err != nil {
		t.Fatal(err)
	}
	main, _ := launchedDirection(t, s, "Principal")
	speed, _ := s.Tournaments().Create(ctx, "", "Speed", "2026-10-03", "")
	for _, tid := range []int64{main, speed} {
		if err := rs.Attach(ctx, "", tid, rid); err != nil {
			t.Fatal(err)
		}
	}
	tables := []domain.TableSetting{
		{Number: 1, Name: "Finale", Room: "A", Reserved: true, AssignedTo: []string{"Anna"}},
		{Number: 5, Room: "B", AssignedTo: []string{}},
	}
	if err := rs.SetTableSettings(ctx, "", rid, tables); err != nil {
		t.Fatal(err)
	}
	if err := rs.SetEventRooms(ctx, "", main, []string{"A", "B"}); err != nil {
		t.Fatal(err)
	}
	ownTables := []domain.TableSetting{{Number: 2, Name: "Seule", AssignedTo: []string{"Bruno"}}}
	if err := rs.SetTournamentTableSettings(ctx, "", main, ownTables); err != nil {
		t.Fatal(err)
	}
	srcR, err := rs.Get(ctx, "", rid)
	if err != nil {
		t.Fatal(err)
	}
	// A second Rencontre whose only Tournament stays behind.
	rid2, _ := rs.Create(ctx, "", domain.Rencontre{Name: "Autre", Tables: 4})
	away, _ := s.Tournaments().Create(ctx, "", "Ailleurs", "2026-10-05", "")
	if err := rs.Attach(ctx, "", away, rid2); err != nil {
		t.Fatal(err)
	}

	opts := ingest.WholeTenant(ingest.FormatSQLite)
	opts.Selection = ingest.Selection{TournamentIDs: []int64{main}}
	plan, err := ingest.PlanExport(ctx, s, "", opts)
	if err != nil {
		t.Fatalf("PlanExport: %v", err)
	}
	path := filepath.Join(t.TempDir(), "out.db")
	report, err := ingest.ExportSQLite(ctx, s, "", path, opts)
	if err != nil {
		t.Fatalf("ExportSQLite: %v", err)
	}
	if plan.Rencontres != 1 || report.Rencontres != 1 {
		t.Errorf("rencontres: planned %d, wrote %d; want 1", plan.Rencontres, report.Rencontres)
	}
	if m := plan.Mismatch(report); m != "" {
		t.Errorf("plan and report disagree on %s", m)
	}

	out, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	list, err := out.Rencontres().List(ctx, "")
	if err != nil || len(list) != 1 {
		t.Fatalf("rencontres in the file = %d, %v; want one", len(list), err)
	}
	got := list[0]
	if got.OutputDir != "" {
		t.Errorf("the output folder left with the export: %q", got.OutputDir)
	}
	if got.Name != srcR.Name || got.StartsOn != srcR.StartsOn || got.EndsOn != srcR.EndsOn || got.Tables != srcR.Tables {
		t.Errorf("rencontre = %+v, want the source's %+v", got, srcR)
	}
	if got.CreatedAt != srcR.CreatedAt || got.UpdatedAt != srcR.UpdatedAt {
		t.Errorf("dates = %q/%q, want the source's %q/%q", got.CreatedAt, got.UpdatedAt, srcR.CreatedAt, srcR.UpdatedAt)
	}
	if !reflect.DeepEqual(got.TableSettings, tables) {
		t.Errorf("table settings = %+v, want %+v", got.TableSettings, tables)
	}
	if len(got.TournamentIDs) != 1 || !reflect.DeepEqual(got.EventRooms[got.TournamentIDs[0]], []string{"A", "B"}) {
		t.Errorf("members %v, rooms %v; want the one exported member with rooms A, B", got.TournamentIDs, got.EventRooms)
	}
	if own, err := out.Rencontres().TournamentTableSettings(ctx, "", got.TournamentIDs[0]); err != nil || !reflect.DeepEqual(own, ownTables) {
		t.Errorf("the tournament's own tables = %+v, %v; want %+v", own, err, ownTables)
	}
}
