package ingest

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A draft travels with the match it produced, whatever else the export
// narrows; one that belongs to no exported match travels only when every
// match and every position do.
func TestExportSendsTheDraftsOfTheExportedMatches(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	matchOf := map[string]int64{}
	for _, player := range []string{"Alice", "Bruno"} {
		id, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: player, Player2Name: "Zoé", MatchLength: 5})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: id, GameNumber: 1}); err != nil {
			t.Fatal(err)
		}
		matchOf[player] = id
	}
	const doc = `{"format_version":3,"header":{"match_length":5},"actions":[],"cursor":0}`
	for _, d := range []storage.Transcription{
		{Label: "Alice", MatchID: matchOf["Alice"]},
		{Label: "Bruno", MatchID: matchOf["Bruno"]},
		{Label: "brouillon"},
	} {
		d.FormatVersion, d.Document = "3", doc
		if _, err := src.Transcriptions().Save(ctx, "", &d); err != nil {
			t.Fatal(err)
		}
	}

	oneMatch := func(transcriptions bool) ExportOptions {
		return ExportOptions{
			Selection:      Selection{MatchIDs: []int64{matchOf["Alice"]}, MatchPositions: true},
			Transcriptions: transcriptions,
		}
	}
	narrowed := WholeTenant(FormatSQLite)
	narrowed.Selection.AllCollections = false
	withoutDrafts := WholeTenant(FormatSQLite)
	withoutDrafts.Transcriptions = false

	cases := []struct {
		name string
		opts ExportOptions
		want []string
	}{
		{"one match", oneMatch(true), []string{"Alice"}},
		{"one match, drafts off", oneMatch(false), nil},
		{"whole library", WholeTenant(FormatSQLite), []string{"Alice", "Bruno", "brouillon"}},
		{"library without collections", narrowed, []string{"Alice", "Bruno", "brouillon"}},
		{"whole library, drafts off", withoutDrafts, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "export.db")
			if _, err := ExportSQLite(ctx, src, "", path, tc.opts); err != nil {
				t.Fatal(err)
			}
			dst, err := sqlite.Open(ctx, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer dst.Close()

			playerOf := map[int64]string{}
			for m, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
				if err != nil {
					t.Fatal(err)
				}
				playerOf[m.ID] = m.Player1Name
			}
			var got []string
			for tr, err := range dst.Transcriptions().List(ctx, "") {
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, tr.Label)
				if linked := playerOf[tr.MatchID]; linked != tr.Label && !(tr.MatchID == 0 && tr.Label == "brouillon") {
					t.Errorf("draft %q is linked to match %d (%q)", tr.Label, tr.MatchID, linked)
				}
			}
			sort.Strings(got)
			if len(got) != len(tc.want) || (len(got) > 0 && !equalStrings(got, tc.want)) {
				t.Errorf("exported drafts %v, want %v", got, tc.want)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
