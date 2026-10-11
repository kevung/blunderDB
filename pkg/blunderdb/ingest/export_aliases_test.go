package ingest

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// An export carries the aliases whose spelling or canonical name an exported
// match writes, so the recipient reads a merged player as one; an alias that
// names nobody in the file stays home.
func TestExportCarriesAliasesOfExportedMatches(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	kept := &domain.Match{Player1Name: "Kevin Unger", Player2Name: "B.", Event: "Paris Open", MatchLength: 5}
	left := &domain.Match{Player1Name: "Zed", Player2Name: "Yann", Event: "Lyon Cup", MatchLength: 5}
	for _, m := range []*domain.Match{kept, left} {
		if _, err := src.Matches().Save(ctx, "", m); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []struct {
		kind             storage.AliasKind
		alias, canonical string
	}{
		{storage.AliasPlayer, "Kevin U.", "Kevin Unger"}, // canonical written by the match
		{storage.AliasPlayer, "B.", "Bob"},               // spelling written by the match
		{storage.AliasPlayer, "Zed Z.", "Zed"},           // only an unexported match
		{storage.AliasEvent, "Paris", "Paris Open"},
		{storage.AliasEvent, "Lyon", "Lyon Cup"},
	} {
		if err := src.Aliases().Set(ctx, "", a.kind, a.alias, a.canonical); err != nil {
			t.Fatal(err)
		}
	}

	export := func(aliases bool) (players, events []string, report ExportReport) {
		t.Helper()
		path := filepath.Join(t.TempDir(), "export.sqlite")
		report, err := ExportSQLite(ctx, src, "", path, ExportOptions{
			Format:    FormatSQLite,
			Selection: Selection{MatchIDs: []int64{kept.ID}},
			Aliases:   aliases,
		})
		if err != nil {
			t.Fatal(err)
		}
		dst, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer dst.Close()
		for kind, out := range map[storage.AliasKind]*[]string{storage.AliasPlayer: &players, storage.AliasEvent: &events} {
			list, err := dst.Aliases().List(ctx, "", kind)
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range list {
				*out = append(*out, a.Alias+"→"+a.Canonical)
			}
			slices.Sort(*out)
		}
		return players, events, report
	}

	players, events, report := export(true)
	if want := []string{"B.→Bob", "Kevin U.→Kevin Unger"}; !slices.Equal(players, want) {
		t.Errorf("exported player aliases = %v, want %v", players, want)
	}
	if want := []string{"Paris→Paris Open"}; !slices.Equal(events, want) {
		t.Errorf("exported event aliases = %v, want %v", events, want)
	}
	if report.Aliases != 3 {
		t.Errorf("report.Aliases = %d, want 3", report.Aliases)
	}
	if players, events, _ := export(false); len(players)+len(events) != 0 {
		t.Errorf("aliases left out still exported: %v %v", players, events)
	}
}
