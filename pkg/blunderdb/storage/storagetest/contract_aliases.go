// Contract cases for the aliases of players and events: a flat table, read
// as one person by the players list, the stats and the search, and what a
// merge of players records.
package storagetest

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testAliasesStayFlat(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	as := s.Aliases()
	for _, bad := range [][2]string{{"", "x"}, {"x", ""}, {"x", " x "}} {
		if err := as.Set(ctx, "", storage.AliasPlayer, bad[0], bad[1]); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("Set(%q, %q) = %v, want ErrInvalid", bad[0], bad[1], err)
		}
	}
	if err := as.Set(ctx, "", "nobody", "a", "b"); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Set with an unknown kind = %v, want ErrInvalid", err)
	}
	must := func(alias, canonical string) {
		t.Helper()
		if err := as.Set(ctx, "", storage.AliasPlayer, alias, canonical); err != nil {
			t.Fatalf("Set(%q, %q): %v", alias, canonical, err)
		}
	}
	must("B", "A")
	must("C", "B") // follows B to A
	must("A", "Z") // A's aliases move to Z
	got, err := as.List(ctx, "", storage.AliasPlayer)
	if err != nil {
		t.Fatal(err)
	}
	want := []storage.Alias{{Alias: "A", Canonical: "Z"}, {Alias: "B", Canonical: "Z"}, {Alias: "C", Canonical: "Z"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %+v, want %+v", got, want)
	}
	must("Z", "A") // the pair turns round
	got, _ = as.List(ctx, "", storage.AliasPlayer)
	want = []storage.Alias{{Alias: "B", Canonical: "A"}, {Alias: "C", Canonical: "A"}, {Alias: "Z", Canonical: "A"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List after turning round = %+v, want %+v", got, want)
	}
	if ev, _ := as.List(ctx, "", storage.AliasEvent); len(ev) != 0 {
		t.Fatalf("player aliases leaked into events: %+v", ev)
	}
	if ok, err := as.Remove(ctx, "", storage.AliasPlayer, "C"); err != nil || !ok {
		t.Fatalf("Remove(C) = %v, %v", ok, err)
	}
	if ok, err := as.Remove(ctx, "", storage.AliasPlayer, "C"); err != nil || ok {
		t.Fatalf("Remove(C) twice = %v, %v; want false", ok, err)
	}
}

func testAliasesReadAsOnePerson(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	for _, p := range [][2]string{{"Alice", "Bob"}, {"alice", "Carol"}, {"A. Lice", "Bob"}} {
		m := domain.Match{Player1Name: p[0], Player2Name: p[1], MatchLength: 5}
		if _, err := ms.Save(ctx, "", &m); err != nil {
			t.Fatal(err)
		}
	}
	sugg, err := s.Aliases().Suggest(ctx, "", storage.AliasPlayer)
	if err != nil {
		t.Fatal(err)
	}
	if len(sugg) != 1 || sugg[0].Canonical != "Alice" || !reflect.DeepEqual(sugg[0].Aliases, []string{"alice"}) {
		t.Fatalf("Suggest = %+v, want alice → Alice", sugg)
	}

	if err := ms.MergePlayers(ctx, "", []string{"alice", "A. Lice", "Alice"}, "Alice"); err != nil {
		t.Fatal(err)
	}
	for m, err := range ms.List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if m.Player1Name == "Alice" && m.Player2Name == "Carol" {
			t.Fatalf("a merge rewrote a stored name: %+v", m)
		}
	}
	names, err := s.Stats().PlayerNames(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 || names[0] != (storage.PlayerFrequency{Name: "Alice", Count: 3}) {
		t.Fatalf("PlayerNames = %+v, want Alice ×3 first", names)
	}
	if sugg, _ := s.Aliases().Suggest(ctx, "", storage.AliasPlayer); len(sugg) != 0 {
		t.Fatalf("Suggest after the merge = %+v, want nothing left", sugg)
	}
	rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Name == "alice" || r.Name == "A. Lice" {
			t.Fatalf("players table keeps the alias %q as a row: %+v", r.Name, rows)
		}
		if r.Name == "Alice" && r.Matches != 3 {
			t.Fatalf("Alice plays %d matches in the players table, want 3", r.Matches)
		}
	}
}
