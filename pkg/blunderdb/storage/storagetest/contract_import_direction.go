package storagetest

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// testImportReadsDirectionBack: a file exported with a directed Tournament
// gives its Direction back on import — record, journal, pairs, own tables,
// Slots filled by the remapped Matches, its Rencontre — into a blank library
// and into one whose Tournament lost its Direction. A library whose
// Tournament is still directed keeps its own: the import duplicates nothing.
func testImportReadsDirectionBack(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tid, d := launchedDirection(t, s, "Open de Lyon")
	slotA, slotB := string(d.State().MatchOrder[0]), string(d.State().MatchOrder[1])
	inSlotA := fingerprintedTournamentMatch(t, s, tid, "Joueur a", "Joueur b")
	inSlotB := fingerprintedTournamentMatch(t, s, tid, "Joueur c", "Joueur d")
	ds, rs := s.Directions(), s.Rencontres()
	for slot, m := range map[string]int64{slotA: inSlotA, slotB: inSlotB} {
		if err := ds.AttachSlot(ctx, "", tid, slot, m); err != nil {
			t.Fatalf("AttachSlot: %v", err)
		}
	}
	pair := []storage.PairMember{{Name: "Anna", Club: "Lyon", Rating: 1600}, {Name: "Bruno"}}
	if err := ds.SetPair(ctx, "", tid, "a", pair); err != nil {
		t.Fatalf("SetPair: %v", err)
	}
	ownTables := []domain.TableSetting{{Number: 2, Name: "Seule", AssignedTo: []string{"Bruno"}}}
	if err := rs.SetTournamentTableSettings(ctx, "", tid, ownTables); err != nil {
		t.Fatal(err)
	}
	rid, err := rs.Create(ctx, "", domain.Rencontre{Name: "Festival", StartsOn: "2026-10-03", EndsOn: "2026-10-04", Tables: 14})
	if err != nil {
		t.Fatal(err)
	}
	if err := rs.Attach(ctx, "", tid, rid); err != nil {
		t.Fatal(err)
	}
	if err := rs.SetEventRooms(ctx, "", tid, []string{"A"}); err != nil {
		t.Fatal(err)
	}
	const rCreated, rUpdated = "2026-09-01 08:00:00", "2026-09-02 09:30:00"
	if err := rs.KeepDates(ctx, "", rid, rCreated, rUpdated); err != nil {
		t.Fatalf("Rencontres.KeepDates: %v", err)
	}
	srcRec, err := ds.Get(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	srcEvents, err := ds.LoadEvents(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}

	opts := ingest.WholeTenant(ingest.FormatSQLite)
	opts.Selection = ingest.Selection{TournamentIDs: []int64{tid}, TournamentMatches: true}
	path := filepath.Join(t.TempDir(), "out.db")
	if _, err := ingest.ExportSQLite(ctx, s, "", path, opts); err != nil {
		t.Fatalf("ExportSQLite: %v", err)
	}
	importInto := func(target storage.Storage) ingest.Summary {
		t.Helper()
		sum, err := ingest.DBImporter{S: target}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil)
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		return sum
	}
	// check verifies the Direction the target holds for the tournament named
	// "Open de Lyon" is the source's.
	check := func(label string, target storage.Storage) {
		t.Helper()
		var got int64
		for tr, err := range target.Tournaments().List(ctx, "", storage.ListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			if tr.Name == "Open de Lyon" {
				got = tr.ID
			}
		}
		rec, err := target.Directions().Get(ctx, "", got)
		if err != nil {
			t.Fatalf("%s: no direction read back: %v", label, err)
		}
		if rec.OutputDir != "" || rec.State != srcRec.State || rec.Config != srcRec.Config ||
			!rec.CreatedAt.Equal(srcRec.CreatedAt) || !rec.UpdatedAt.Equal(srcRec.UpdatedAt) {
			t.Errorf("%s: record = %+v, want the source's %+v without its folder", label, rec, srcRec)
		}
		evs, err := target.Directions().LoadEvents(ctx, "", got)
		if err != nil || len(evs) != len(srcEvents) {
			t.Fatalf("%s: journal = %d events, %v; want %d", label, len(evs), err, len(srcEvents))
		}
		pairs, err := target.Directions().Pairs(ctx, "", got)
		if err != nil || !slices.Equal(pairs["a"], pair) {
			t.Errorf("%s: pairs = %+v, %v; want a = %+v", label, pairs, err, pair)
		}
		tables, err := target.Rencontres().TournamentTableSettings(ctx, "", got)
		if err != nil || len(tables) != 1 || tables[0].Name != "Seule" {
			t.Errorf("%s: own tables = %+v, %v", label, tables, err)
		}
		replayed, err := direction.Open(ctx, storage.BindDirection(target.Directions(), ""), got)
		if err != nil {
			t.Fatalf("%s: replaying the imported journal: %v", label, err)
		}
		filled, err := target.Directions().FilledSlots(ctx, "", got)
		if err != nil {
			t.Fatal(err)
		}
		bySlot := map[string]string{}
		for _, f := range filled {
			if _, ok := replayed.State().Matches[tournoi.MatchID(f.SlotID)]; !ok {
				t.Errorf("%s: slot %s fills no pairing", label, f.SlotID)
			}
			bySlot[f.SlotID] = f.Player1
		}
		if bySlot[slotA] != "Joueur a" || bySlot[slotB] != "Joueur c" || len(bySlot) != 2 {
			t.Errorf("%s: filled slots = %v, want %s → Joueur a, %s → Joueur c", label, bySlot, slotA, slotB)
		}
		in, err := target.Rencontres().Of(ctx, "", got)
		if err != nil || in == 0 {
			t.Fatalf("%s: tournament in no rencontre (%v)", label, err)
		}
		r, err := target.Rencontres().Get(ctx, "", in)
		if err != nil || r.Name != "Festival" || r.Tables != 14 || r.OutputDir != "" ||
			!reflect.DeepEqual(r.EventRooms[got], []string{"A"}) {
			t.Errorf("%s: rencontre = %+v, %v", label, r, err)
		}
		if r != nil && (!sameStamp(r.CreatedAt, rCreated) || !sameStamp(r.UpdatedAt, rUpdated)) {
			t.Errorf("%s: rencontre dates = %q / %q, want the source's %q / %q", label, r.CreatedAt, r.UpdatedAt, rCreated, rUpdated)
		}
	}

	// A blank library.
	blank, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "blank.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer blank.Close()
	if sum := importInto(blank); sum.Directions != 1 || sum.Rencontres != 1 {
		t.Errorf("blank: summary = %+v, want one direction and one rencontre", sum)
	}
	check("blank", blank)

	// The library still holding the tournament, directed: nothing changes.
	if sum := importInto(s); sum.Directions != 0 || sum.Rencontres != 0 || sum.Tournaments != 0 || sum.Matches != 0 {
		t.Errorf("already held: summary = %+v, want nothing written", sum)
	}
	if evs, _ := ds.LoadEvents(ctx, "", tid); len(evs) != len(srcEvents) {
		t.Errorf("already held: journal grew to %d events", len(evs))
	}
	if all, _ := rs.List(ctx, ""); len(all) != 1 {
		t.Errorf("already held: %d rencontres, want 1", len(all))
	}

	// The library holding the tournament and its matches, without its
	// Direction nor its Rencontre: both come back around the same matches.
	if err := rs.Delete(ctx, "", rid); err != nil {
		t.Fatal(err)
	}
	if err := ds.Delete(ctx, "", tid); err != nil {
		t.Fatal(err)
	}
	if err := rs.SetTournamentTableSettings(ctx, "", tid, nil); err != nil {
		t.Fatal(err)
	}
	if sum := importInto(s); sum.Directions != 1 || sum.Rencontres != 1 || sum.Matches != 0 || sum.Tournaments != 0 {
		t.Errorf("undirected: summary = %+v, want one direction, one rencontre, no match", sum)
	}
	check("undirected", s)
	if _, slot, _ := ds.SlotOf(ctx, "", inSlotA); slot != slotA {
		t.Errorf("undirected: the held match fills %q, want %s", slot, slotA)
	}
	// And a second import changes nothing.
	if sum := importInto(s); sum.Directions != 0 || sum.Rencontres != 0 {
		t.Errorf("second import: summary = %+v, want nothing written", sum)
	}
}

// fingerprintedTournamentMatch files a Match the import recognises by its
// match_hash, as every imported Match carries one.
func fingerprintedTournamentMatch(t *testing.T, s storage.Storage, tid int64, p1, p2 string) int64 {
	t.Helper()
	ctx := context.Background()
	m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7, MatchHash: "empreinte-" + p1,
		MatchDate: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tid, id); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	return id
}

// sameStamp compares two dates to the second whatever text each backend
// gives them (a space or a "T" between date and time, with or without a zone).
func sameStamp(a, b string) bool {
	digits := func(s string) string {
		var out []rune
		for _, c := range s {
			if c >= '0' && c <= '9' {
				out = append(out, c)
			}
		}
		if len(out) > 14 {
			out = out[:14]
		}
		return string(out)
	}
	return digits(a) == digits(b)
}

// testImportSameNamedDirections: two directed Tournaments of one name merge
// into one target Tournament, which takes the first Direction only.
func testImportSameNamedDirections(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	first, _ := launchedDirection(t, s, "Doublon")
	second, _ := launchedDirection(t, s, "Doublon")
	opts := ingest.WholeTenant(ingest.FormatSQLite)
	opts.Selection = ingest.Selection{TournamentIDs: []int64{first, second}, TournamentMatches: true}
	path := filepath.Join(t.TempDir(), "out.db")
	if _, err := ingest.ExportSQLite(ctx, s, "", path, opts); err != nil {
		t.Fatalf("ExportSQLite: %v", err)
	}
	blank, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "blank.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer blank.Close()
	sum, err := ingest.DBImporter{S: blank}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if sum.Tournaments != 1 || sum.Directions != 1 {
		t.Errorf("summary = %+v, want one tournament and one direction", sum)
	}
}
