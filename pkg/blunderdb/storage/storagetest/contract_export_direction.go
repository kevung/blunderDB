package storagetest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// launchedDirection directs a new Tournament of eight players and launches
// its first round: a journal with pairings a library Match can fill.
func launchedDirection(t *testing.T, s storage.Storage, name string) (int64, *direction.Direction) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	tid, err := s.Tournaments().Create(ctx, "", name, "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	var cfg tournoi.Config
	if err := json.Unmarshal([]byte(`{"name":"`+name+`","tables":{"count":8},
		"phases":[{"kind":"swiss_lives","length":7,"lives":2}]}`), &cfg); err != nil {
		t.Fatal(err)
	}
	store := storage.BindDirection(s.Directions(), "")
	d, err := direction.Create(ctx, store, tid, cfg, 7, now)
	if err != nil {
		t.Fatalf("direction.Create: %v", err)
	}
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if err := d.Enter(ctx, tournoi.Player{ID: tournoi.PlayerID(id), Name: "Joueur " + id}, now); err != nil {
			t.Fatalf("Enter: %v", err)
		}
	}
	// The draw comes first in a Swiss: launching may take more than one pass.
	for i := 0; i < 4 && len(d.State().MatchOrder) == 0; i++ {
		for _, a := range d.ProposeAt(now) {
			ev, err := d.EventFor(a, now)
			if err != nil {
				t.Fatalf("EventFor: %v", err)
			}
			if err := d.Apply(ctx, ev); err != nil {
				t.Fatalf("Apply: %v", err)
			}
		}
	}
	if len(d.State().MatchOrder) < 2 {
		t.Fatalf("%d pairings launched, want at least 2", len(d.State().MatchOrder))
	}
	if err := d.SetOutputDir(ctx, "/home/producteur/affichage"); err != nil {
		t.Fatal(err)
	}
	return tid, d
}

// testExportCarriesDirection: an exported Tournament takes its Direction
// with it — the record without its output folder, the journal event for
// event, the pairs — and the journal replayed in the file finds its Matches
// in their Slots. A Match whose Tournament stays behind leaves without its
// Slot, and the plan counts the Directions the export writes.
func testExportCarriesDirection(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tid, d := launchedDirection(t, s, "Open de Lyon")
	slotA, slotB := string(d.State().MatchOrder[0]), string(d.State().MatchOrder[1])
	inSlotA := newTournamentMatchOf(t, s, tid, "Joueur a", "Joueur b", 7)
	inSlotB := newTournamentMatchOf(t, s, tid, "Joueur c", "Joueur d", 7)
	ds := s.Directions()
	for slot, m := range map[string]int64{slotA: inSlotA, slotB: inSlotB} {
		if err := ds.AttachSlot(ctx, "", tid, slot, m); err != nil {
			t.Fatalf("AttachSlot: %v", err)
		}
	}
	pair := []storage.PairMember{{Name: "Anna", Club: "Lyon", Rating: 1600}, {Name: "Bruno"}}
	if err := ds.SetPair(ctx, "", tid, "a", pair); err != nil {
		t.Fatalf("SetPair: %v", err)
	}
	srcRec, err := ds.Get(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	srcEvents, err := ds.LoadEvents(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	// A second directed Tournament, left behind: its Match leaves alone.
	otherTID, other := launchedDirection(t, s, "Open de Paris")
	lone := newTournamentMatchOf(t, s, otherTID, "Joueur e", "Joueur f", 7)
	if err := ds.AttachSlot(ctx, "", otherTID, string(other.State().MatchOrder[0]), lone); err != nil {
		t.Fatalf("AttachSlot: %v", err)
	}

	opts := ingest.WholeTenant(ingest.FormatSQLite)
	opts.Selection = ingest.Selection{
		TournamentIDs: []int64{tid}, TournamentMatches: true, MatchIDs: []int64{lone},
	}
	plan, err := ingest.PlanExport(ctx, s, "", opts)
	if err != nil {
		t.Fatalf("PlanExport: %v", err)
	}
	path := filepath.Join(t.TempDir(), "out.db")
	report, err := ingest.ExportSQLite(ctx, s, "", path, opts)
	if err != nil {
		t.Fatalf("ExportSQLite: %v", err)
	}
	if report.Directions != 1 || plan.Directions != 1 {
		t.Errorf("directions: planned %d, wrote %d; want 1", plan.Directions, report.Directions)
	}
	if report.Slots != 2 || plan.Slots != 2 {
		t.Errorf("slots: planned %d, wrote %d; want 2", plan.Slots, report.Slots)
	}
	if m := plan.Mismatch(report); m != "" {
		t.Errorf("plan and report disagree on %s", m)
	}

	out, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	recs, err := out.Directions().List(ctx, "")
	if err != nil || len(recs) != 1 {
		t.Fatalf("directions in the file = %+v, %v; want one", recs, err)
	}
	got := recs[0]
	newTID := got.TournamentID
	if got.OutputDir != "" {
		t.Errorf("the output folder left with the export: %q", got.OutputDir)
	}
	if got.State != srcRec.State || got.Config != srcRec.Config || got.FormatVersion != srcRec.FormatVersion ||
		got.EngineVersion != srcRec.EngineVersion || !got.CreatedAt.Equal(srcRec.CreatedAt) || !got.UpdatedAt.Equal(srcRec.UpdatedAt) {
		t.Errorf("record = %+v, want the source's %+v", got, srcRec)
	}
	gotEvents, err := out.Directions().LoadEvents(ctx, "", newTID)
	if err != nil || len(gotEvents) != len(srcEvents) {
		t.Fatalf("journal = %d events, %v; want %d", len(gotEvents), err, len(srcEvents))
	}
	for i, ev := range gotEvents {
		want := srcEvents[i]
		if ev.Seq != want.Seq || ev.Kind != want.Kind || !ev.Time.Equal(want.Time) || string(ev.Payload) != string(want.Payload) {
			t.Errorf("event %d = %+v, want %+v", i, ev, want)
		}
	}
	pairs, err := out.Directions().Pairs(ctx, "", newTID)
	if err != nil || !slices.Equal(pairs["a"], pair) {
		t.Errorf("pairs = %+v, %v; want a = %+v", pairs, err, pair)
	}

	replayed, err := direction.Open(ctx, storage.BindDirection(out.Directions(), ""), newTID)
	if err != nil {
		t.Fatalf("replaying the exported journal: %v", err)
	}
	if !slices.Equal(replayed.State().MatchOrder, d.State().MatchOrder) {
		t.Errorf("replayed pairings %v, want %v", replayed.State().MatchOrder, d.State().MatchOrder)
	}
	filled, err := out.Directions().FilledSlots(ctx, "", newTID)
	if err != nil {
		t.Fatal(err)
	}
	bySlot := map[string]string{}
	for _, f := range filled {
		if _, ok := replayed.State().Matches[tournoi.MatchID(f.SlotID)]; !ok {
			t.Errorf("slot %s fills no pairing of the replayed journal", f.SlotID)
		}
		bySlot[f.SlotID] = f.Player1
	}
	if bySlot[slotA] != "Joueur a" || bySlot[slotB] != "Joueur c" || len(bySlot) != 2 {
		t.Errorf("filled slots = %v, want %s → Joueur a, %s → Joueur c", bySlot, slotA, slotB)
	}

	for m, err := range out.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if m.Player1Name != "Joueur e" {
			continue
		}
		if _, slot, err := out.Directions().SlotOf(ctx, "", m.ID); err != nil || slot != "" {
			t.Errorf("the match left behind by its tournament kept slot %q (%v)", slot, err)
		}
	}
}
