package storagetest

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func saveMatchAt(t *testing.T, ctx context.Context, s storage.Storage, scope, event, location string) {
	t.Helper()
	m := domain.Match{Player1Name: "A", Player2Name: "B", Event: event, Location: location, MatchLength: 5}
	if _, err := s.Matches().Save(ctx, scope, &m); err != nil {
		t.Fatal(err)
	}
}

// testStatsMatchTextValues: the distinct events and locations come back most
// frequent first, ties alphabetical, empty values left out; event aliases fold
// into their canonical name, location ones do not exist; an unknown field is
// refused.
func testStatsMatchTextValues(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ss := s.Stats()
	for _, c := range [][2]string{
		{"Open", "Paris"}, {"Open", "Lyon"}, {"Masters", "Paris"}, {"Cup", ""}, {"", "Paris"}, {"Masters", "Lille"}, {"Open", ""},
	} {
		saveMatchAt(t, ctx, s, "", c[0], c[1])
	}
	events, err := ss.MatchTextValues(ctx, "", storage.MatchEvent)
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := []storage.PlayerFrequency{{Name: "Open", Count: 3}, {Name: "Masters", Count: 2}, {Name: "Cup", Count: 1}}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %+v, want %+v", events, wantEvents)
	}
	locs, err := ss.MatchTextValues(ctx, "", storage.MatchLocation)
	if err != nil {
		t.Fatal(err)
	}
	wantLocs := []storage.PlayerFrequency{{Name: "Paris", Count: 3}, {Name: "Lille", Count: 1}, {Name: "Lyon", Count: 1}}
	if !reflect.DeepEqual(locs, wantLocs) {
		t.Fatalf("locations = %+v, want %+v", locs, wantLocs)
	}

	if err := s.Aliases().Set(ctx, "", storage.AliasEvent, "Cup", "Masters"); err != nil {
		t.Fatal(err)
	}
	events, err = ss.MatchTextValues(ctx, "", storage.MatchEvent)
	if err != nil {
		t.Fatal(err)
	}
	wantEvents = []storage.PlayerFrequency{{Name: "Masters", Count: 3}, {Name: "Open", Count: 3}}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events after the alias = %+v, want %+v", events, wantEvents)
	}

	if _, err := ss.MatchTextValues(ctx, "", storage.MatchTextField("player1_name")); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("unknown field: err = %v, want ErrInvalid", err)
	}
}

// checkMatchTextValuesIsolation: another tenant's events and locations are
// not offered.
func checkMatchTextValuesIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	saveMatchAt(t, ctx, s, a, "Secret Open", "Secret City")
	for _, f := range []storage.MatchTextField{storage.MatchEvent, storage.MatchLocation} {
		got, err := s.Stats().MatchTextValues(ctx, b, f)
		if err != nil || len(got) != 0 {
			t.Errorf("MatchTextValues(%s, %s) = %v, %v; want none", b, f, got, err)
		}
	}
}
