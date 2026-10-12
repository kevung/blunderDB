package ingest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A search deck's source travels with its rows renumbered: its ids, and its
// command when the command can say in the file what it said at the source;
// a command or a payload that cannot is abandoned, the ids staying.
func TestCarriedSearchSource(t *testing.T) {
	positions := map[int64]int64{10: 1, 11: 2}
	matches := map[int64]int64{40: 3}
	for _, tc := range []struct {
		name, source string
		want         map[string]any // nil: the legacy list below
		legacy       string
	}{
		{
			name:   "ids and an id command renumbered",
			source: `{"command":"s id10,11","position":"{\"a\":1}","ids":[10,11,12]}`,
			want:   map[string]any{"command": "s id1 id2", "position": `{"a":1}`, "ids": []any{1.0, 2.0}},
		},
		{
			name:   "a command naming no row travels as written",
			source: `{"command":"s e>0.1","ids":[10]}`,
			want:   map[string]any{"command": "s e>0.1", "ids": []any{1.0}},
		},
		{
			name:   "a match command renumbered",
			source: `{"command":"s ma40","ids":[]}`,
			want:   map[string]any{"command": "s ma3", "ids": []any{}},
		},
		{
			name:   "a command naming a row left behind is abandoned with its board",
			source: `{"command":"s ma41","position":"{}","ids":[11]}`,
			want:   map[string]any{"ids": []any{2.0}},
		},
		{
			name:   "a payload naming rows is abandoned, library travels",
			source: `{"ids":[10],"payload":{"matchIDsFilter":"40"},"library":true}`,
			want:   map[string]any{"ids": []any{1.0}, "library": true},
		},
		{
			name:   "a payload naming no row travels",
			source: `{"ids":[],"payload":{"likeTargetId":0,"searchText":"x"}}`,
			want:   map[string]any{"ids": []any{}, "payload": map[string]any{"likeTargetId": 0.0, "searchText": "x"}},
		},
		{name: "legacy list", source: "10,12, 11", legacy: "1,2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := carriedSearchSource(tc.source, positions, matches, nil)
			if tc.want == nil {
				if got != tc.legacy {
					t.Errorf("got %q, want %q", got, tc.legacy)
				}
				return
			}
			var doc map[string]any
			if err := json.Unmarshal([]byte(got), &doc); err != nil {
				t.Fatalf("source %q is no JSON: %v", got, err)
			}
			if !reflect.DeepEqual(doc, tc.want) {
				t.Errorf("got %v, want %v", doc, tc.want)
			}
		})
	}
}

// A search deck goes to a file and comes back into another library with its
// source still a document whose ids and command name the positions it held,
// and Sync fills it from those ids at every stop.
func TestSearchDeckRoundTrip(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	var ids []int64
	for _, dice := range [][2]int{{3, 1}, {6, 5}, {4, 2}} {
		p := domain.InitializePosition()
		p.Dice = dice
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	kept := ids[1:]
	source := sourceOf(t, map[string]any{"command": "s id" + itoa(kept[0]) + "," + itoa(kept[1]), "position": "{}", "ids": kept})
	deck, err := src.Anki().CreateDeck(ctx, "", "Recherche", "", domain.AnkiSourceSearch, 0, source)
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Anki().Sync(ctx, "", deck); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(t.TempDir(), "export.db")
	if _, err := ExportSQLite(ctx, src, "", file, ExportOptions{
		Format:    FormatSQLite,
		Selection: Selection{DeckIDs: []int64{deck}, DeckPositions: true},
	}); err != nil {
		t.Fatal(err)
	}
	checkSearchDeck(t, file, 2)

	// A library already holding a position numbers the file's otherwise.
	target := filepath.Join(t.TempDir(), "target.db")
	tgt, err := sqlite.Open(ctx, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	own := domain.InitializePosition()
	own.Dice = [2]int{5, 5}
	if _, err := tgt.Positions().Save(ctx, "", &own); err != nil {
		t.Fatal(err)
	}
	if _, err := (DBImporter{S: tgt}).Import(ctx, "", Source{Format: FormatNativeDB, Path: file}, nil); err != nil {
		t.Fatal(err)
	}
	tgt.Close()
	checkSearchDeck(t, target, 2)
}

// checkSearchDeck opens the library at path and checks that its one search
// deck names, in its ids and in its command, positions it holds, and that
// Sync fills it with want cards from them.
func checkSearchDeck(t *testing.T, path string, want int) {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var deck *domain.AnkiDeck
	for d, err := range s.Anki().ListDecks(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		deck = d
	}
	if deck == nil {
		t.Fatal("no deck in the library")
	}
	var doc struct {
		Command string  `json:"command"`
		IDs     []int64 `json:"ids"`
	}
	if err := json.Unmarshal([]byte(deck.SourceCommand), &doc); err != nil {
		t.Fatalf("deck source %q is no JSON document: %v", deck.SourceCommand, err)
	}
	if len(doc.IDs) != want {
		t.Fatalf("deck source ids = %v, want %d", doc.IDs, want)
	}
	if wantCmd := "s id" + itoa(doc.IDs[0]) + " id" + itoa(doc.IDs[1]); doc.Command != wantCmd {
		t.Errorf("deck command = %q, want %q", doc.Command, wantCmd)
	}
	for _, id := range doc.IDs {
		if _, err := s.Positions().Load(ctx, "", id); err != nil {
			t.Errorf("deck source names position %d, absent: %v", id, err)
		}
	}
	if err := s.Anki().Sync(ctx, "", deck.ID); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, err := range s.Anki().DeckCards(ctx, "", deck.ID) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != want {
		t.Errorf("deck holds %d cards after Sync, want %d", n, want)
	}
}

func sourceOf(t *testing.T, doc map[string]any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func itoa(n int64) string { b, _ := json.Marshal(n); return string(b) }
