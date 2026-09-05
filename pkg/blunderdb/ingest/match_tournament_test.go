package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// tournamentNames lists the tournaments in the database, with their match count.
func tournamentNames(t *testing.T, s storage.Storage) map[string]int {
	t.Helper()
	got := map[string]int{}
	for tr, err := range s.Tournaments().List(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		got[tr.Name] = tr.MatchCount
	}
	return got
}

// Every importer reads the file's event name into match.event, but nothing
// used to turn it into a tournament: a library imported entirely from files
// that name their event showed an empty Tournaments panel. WriteMatch now
// files a match it creates under its event.
func TestWriteMatch_FilesTheMatchUnderItsEvent(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	g := sampleGraph()
	g.Match.Event = "  Championnat de Paris 2026  " // padded on purpose
	res := writeGraph(t, s, g)

	if res.Tournament != "Championnat de Paris 2026" {
		t.Errorf("res.Tournament = %q, want the trimmed event name", res.Tournament)
	}
	if got := tournamentNames(t, s); got["Championnat de Paris 2026"] != 1 {
		t.Errorf("tournaments = %v, want the event with one match", got)
	}

	m, err := s.Matches().Get(ctx, "", res.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	if m.TournamentID == nil {
		t.Fatal("the match was not linked to its tournament")
	}
}

// Two matches of the same event share one tournament — the name is looked up
// before it is created.
func TestWriteMatch_TwoMatchesOfOneEventShareTheTournament(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	first := sampleGraph()
	first.Match.Event = "Open de Lyon"
	writeGraph(t, s, first)

	second := sampleGraph()
	second.Match.Event = "Open de Lyon"
	second.Match.MatchHash = "xg-hash-2"
	second.Match.CanonicalHash = "canon-2"
	second.Match.Player1Name = "Carol"
	writeGraph(t, s, second)

	got := tournamentNames(t, s)
	if len(got) != 1 || got["Open de Lyon"] != 2 {
		t.Errorf("tournaments = %v, want a single \"Open de Lyon\" holding both matches", got)
	}
}

// A file that names no event creates no tournament — an empty name would
// otherwise become a tournament called "".
func TestWriteMatch_NoEventNoTournament(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	res := writeGraph(t, s, sampleGraph())
	if res.Tournament != "" {
		t.Errorf("res.Tournament = %q, want empty", res.Tournament)
	}
	if got := tournamentNames(t, s); len(got) != 0 {
		t.Errorf("tournaments = %v, want none", got)
	}
}

// Re-importing a match that is already stored must not move it: the user may
// have filed it by hand, under a name of their own.
func TestWriteMatch_ExistingMatchIsNotRefiled(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	g := sampleGraph()
	g.Match.Event = "Open de Lyon"
	res := writeGraph(t, s, g)

	// The user refiles it by hand.
	if err := s.Tournaments().SetMatchByName(ctx, "", res.MatchID, "Mes parties"); err != nil {
		t.Fatal(err)
	}

	again := sampleGraph()
	again.Match.Event = "Open de Lyon"
	if r := writeGraph(t, s, again); !r.Skipped {
		t.Fatalf("the second write should be skipped as a duplicate, got %+v", r)
	}

	m, err := s.Matches().Get(ctx, "", res.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	got := tournamentNames(t, s)
	if got["Mes parties"] != 1 || got["Open de Lyon"] != 0 {
		t.Errorf("tournaments = %v, want the hand-filed one to keep the match", got)
	}
	if m.TournamentID == nil {
		t.Fatal("the match lost its tournament")
	}
}
