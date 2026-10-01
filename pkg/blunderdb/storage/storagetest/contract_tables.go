package storagetest

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// twoRooms is a Rencontre's properties out of order, one table without
// anyone assigned: the read gives them back by number, AssignedTo never nil.
func twoRooms() []domain.TableSetting {
	return []domain.TableSetting{
		{Number: 21, Name: "Stream", Room: "B", Reserved: true},
		{Number: 3, Room: "A", AssignedTo: []string{"Ada Lovecraft", "Bruno Diderot"}},
		{Number: 22, Room: "B"},
	}
}

func wantSorted() []domain.TableSetting {
	return []domain.TableSetting{
		{Number: 3, Room: "A", AssignedTo: []string{"Ada Lovecraft", "Bruno Diderot"}},
		{Number: 21, Name: "Stream", Room: "B", Reserved: true, AssignedTo: []string{}},
		{Number: 22, Room: "B", AssignedTo: []string{}},
	}
}

// testTableSettingsRencontre: a Rencontre's table properties round-trip by
// number, are replaced as a whole, refuse a bad list without writing it, and
// go with the Rencontre (ADR-0058).
func testTableSettingsRencontre(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	rs := s.Rencontres()
	id, err := rs.Create(ctx, "", domain.Rencontre{Name: "Open", Tables: 32})
	if err != nil {
		t.Fatal(err)
	}
	r, err := rs.Get(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	if r.TableSettings == nil || len(r.TableSettings) != 0 || r.EventRooms == nil {
		t.Fatalf("a Rencontre without properties reads %#v / %#v, want empty and non-nil", r.TableSettings, r.EventRooms)
	}

	if err := rs.SetTableSettings(ctx, "", id, twoRooms()); err != nil {
		t.Fatalf("SetTableSettings: %v", err)
	}
	if r, err = rs.Get(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.TableSettings, wantSorted()) {
		t.Fatalf("Get.TableSettings = %#v, want %#v", r.TableSettings, wantSorted())
	}
	list, err := rs.List(ctx, "")
	if err != nil || len(list) != 1 || !reflect.DeepEqual(list[0].TableSettings, wantSorted()) {
		t.Fatalf("List = %v, %v; want the same settings as Get", list, err)
	}

	for name, bad := range map[string][]domain.TableSetting{
		"zero":      {{Number: 0}},
		"negative":  {{Number: -2}},
		"duplicate": {{Number: 4}, {Number: 4, Room: "B"}},
	} {
		if err := rs.SetTableSettings(ctx, "", id, bad); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("SetTableSettings(%s) = %v, want ErrInvalid", name, err)
		}
	}
	if r, _ = rs.Get(ctx, "", id); !reflect.DeepEqual(r.TableSettings, wantSorted()) {
		t.Fatalf("a refused list changed the settings: %#v", r.TableSettings)
	}

	if err := rs.SetTableSettings(ctx, "", id, []domain.TableSetting{{Number: 7, Name: "Vedette"}}); err != nil {
		t.Fatal(err)
	}
	if r, _ = rs.Get(ctx, "", id); len(r.TableSettings) != 1 || r.TableSettings[0].Number != 7 || r.TableSettings[0].Name != "Vedette" {
		t.Fatalf("after replace: %#v, want table 7 only", r.TableSettings)
	}
	if err := rs.SetTableSettings(ctx, "", id, nil); err != nil {
		t.Fatal(err)
	}
	if r, _ = rs.Get(ctx, "", id); len(r.TableSettings) != 0 {
		t.Fatalf("after clearing: %#v", r.TableSettings)
	}

	if err := rs.SetTableSettings(ctx, "", 9999, twoRooms()); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetTableSettings on a missing Rencontre = %v, want ErrNotFound", err)
	}

	_ = rs.SetTableSettings(ctx, "", id, twoRooms())
	if err := rs.Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete a Rencontre with table settings: %v", err)
	}
	// A new Rencontre reads none of the deleted one's rows.
	again, _ := rs.Create(ctx, "", domain.Rencontre{Name: "Suivant"})
	if r, _ = rs.Get(ctx, "", again); len(r.TableSettings) != 0 {
		t.Fatalf("a new Rencontre inherits %#v", r.TableSettings)
	}
}

// testTableSettingsTournament: a Tournament run on its own owns properties of
// the same shape; they are kept while it plays in a Rencontre whose own
// settings use the same numbers, and go with the Tournament.
func testTableSettingsTournament(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	rs := s.Rencontres()
	tid, err := s.Tournaments().Create(ctx, "", "Seul", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := rs.TournamentTableSettings(ctx, "", tid)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("TournamentTableSettings on a fresh Tournament = %#v, %v; want empty", got, err)
	}
	if err := rs.SetTournamentTableSettings(ctx, "", tid, twoRooms()); err != nil {
		t.Fatalf("SetTournamentTableSettings: %v", err)
	}
	rid, _ := rs.Create(ctx, "", domain.Rencontre{Name: "Open"})
	_ = rs.Attach(ctx, "", tid, rid)
	if err := rs.SetTableSettings(ctx, "", rid, twoRooms()); err != nil {
		t.Fatalf("the Rencontre's numbers collide with its member's: %v", err)
	}
	if got, _ = rs.TournamentTableSettings(ctx, "", tid); !reflect.DeepEqual(got, wantSorted()) {
		t.Fatalf("TournamentTableSettings = %#v, want %#v", got, wantSorted())
	}
	if err := rs.SetTournamentTableSettings(ctx, "", tid, []domain.TableSetting{{Number: 1}, {Number: 1}}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("duplicate = %v, want ErrInvalid", err)
	}
	if _, err := rs.TournamentTableSettings(ctx, "", 9999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("TournamentTableSettings on a missing Tournament = %v, want ErrNotFound", err)
	}
	if err := rs.SetTournamentTableSettings(ctx, "", 9999, nil); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetTournamentTableSettings on a missing Tournament = %v, want ErrNotFound", err)
	}

	if err := s.Tournaments().Delete(ctx, "", tid); err != nil {
		t.Fatalf("Delete a Tournament with table settings: %v", err)
	}
	if _, err := rs.TournamentTableSettings(ctx, "", tid); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("settings of a deleted Tournament = %v, want ErrNotFound", err)
	}
	if r, _ := rs.Get(ctx, "", rid); !reflect.DeepEqual(r.TableSettings, wantSorted()) {
		t.Fatalf("deleting a member touched the Rencontre's settings: %#v", r.TableSettings)
	}
}

// testEventRooms: the rooms of an event are read with its Rencontre, an
// empty list is no restriction, and leaving the Rencontre clears them.
func testEventRooms(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	rs := s.Rencontres()
	rid, _ := rs.Create(ctx, "", domain.Rencontre{Name: "Open"})
	main, _ := s.Tournaments().Create(ctx, "", "Principal", "", "")
	dmp, _ := s.Tournaments().Create(ctx, "", "DMP", "", "")
	for _, tid := range []int64{main, dmp} {
		if err := rs.Attach(ctx, "", tid, rid); err != nil {
			t.Fatal(err)
		}
	}
	if err := rs.SetEventRooms(ctx, "", dmp, []string{"B"}); err != nil {
		t.Fatalf("SetEventRooms: %v", err)
	}
	if err := rs.SetEventRooms(ctx, "", main, []string{"A", "B"}); err != nil {
		t.Fatal(err)
	}
	r, _ := rs.Get(ctx, "", rid)
	want := map[int64][]string{main: {"A", "B"}, dmp: {"B"}}
	if !reflect.DeepEqual(r.EventRooms, want) {
		t.Fatalf("EventRooms = %v, want %v", r.EventRooms, want)
	}
	if err := rs.SetEventRooms(ctx, "", main, []string{}); err != nil {
		t.Fatal(err)
	}
	// Attaching to the Rencontre it already plays in keeps the rooms.
	if err := rs.Attach(ctx, "", dmp, rid); err != nil {
		t.Fatal(err)
	}
	if r, _ = rs.Get(ctx, "", rid); !reflect.DeepEqual(r.EventRooms, map[int64][]string{dmp: {"B"}}) {
		t.Fatalf("EventRooms after clearing main and re-attaching dmp = %v", r.EventRooms)
	}

	if err := rs.Attach(ctx, "", dmp, 0); err != nil {
		t.Fatal(err)
	}
	if err := rs.Attach(ctx, "", dmp, rid); err != nil {
		t.Fatal(err)
	}
	if r, _ = rs.Get(ctx, "", rid); len(r.EventRooms) != 0 {
		t.Fatalf("detaching must clear the rooms: %v", r.EventRooms)
	}

	if err := rs.SetEventRooms(ctx, "", 9999, []string{"A"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetEventRooms on a missing Tournament = %v, want ErrNotFound", err)
	}
}

// checkTableSettingIsolation: another tenant neither reads nor writes the
// table properties or the rooms of this tenant's owners.
func checkTableSettingIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	rs := s.Rencontres()
	rid, err := rs.Create(ctx, a, domain.Rencontre{Name: "Open"})
	if err != nil {
		t.Fatal(err)
	}
	tid, err := s.Tournaments().Create(ctx, a, "Seul", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := rs.SetTableSettings(ctx, a, rid, twoRooms()); err != nil {
		t.Fatal(err)
	}
	if err := rs.SetTournamentTableSettings(ctx, a, tid, twoRooms()); err != nil {
		t.Fatal(err)
	}
	if err := rs.SetTableSettings(ctx, b, rid, nil); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetTableSettings(%s, %s's Rencontre) = %v, want ErrNotFound", b, a, err)
	}
	if err := rs.SetTournamentTableSettings(ctx, b, tid, nil); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetTournamentTableSettings(%s) = %v, want ErrNotFound", b, err)
	}
	if _, err := rs.TournamentTableSettings(ctx, b, tid); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("TournamentTableSettings(%s) = %v, want ErrNotFound", b, err)
	}
	if err := rs.SetEventRooms(ctx, b, tid, []string{"A"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetEventRooms(%s) = %v, want ErrNotFound", b, err)
	}
	r, err := rs.Get(ctx, a, rid)
	if err != nil || !reflect.DeepEqual(r.TableSettings, wantSorted()) {
		t.Fatalf("tenant %s's settings after %s's attempts = %#v, %v", a, b, r.TableSettings, err)
	}
	if got, _ := rs.TournamentTableSettings(ctx, a, tid); !reflect.DeepEqual(got, wantSorted()) {
		t.Fatalf("tenant %s's Tournament settings = %#v", a, got)
	}
}
