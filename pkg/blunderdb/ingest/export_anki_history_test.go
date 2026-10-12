package ingest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A deck leaves as a fresh study copy unless its history is asked for; then
// each card keeps its schedule and the file holds its review events, on the
// file's own card, deck and position numbers.
func TestExportCarriesAnkiHistoryOnlyWhenAsked(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	// A position saved first and left out shifts the file's numbering away
	// from the source's.
	var ids []int64
	for _, dice := range [][2]int{{6, 6}, {3, 1}, {4, 2}} {
		p := domain.InitializePosition()
		p.Dice = dice
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	deck, err := src.Anki().CreateDeck(ctx, "", "Ouvertures", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Anki().SyncWithPositions(ctx, "", deck, ids); err != nil {
		t.Fatal(err)
	}
	// Review every card once: the one whose position stays behind too.
	for range ids {
		next, err := src.Anki().NextCard(ctx, "", deck)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := src.Anki().ReviewCard(ctx, "", next.Card.ID, 3); err != nil {
			t.Fatal(err)
		}
	}

	// A suspended card stays suspended in the history.
	// Read before writing: the in-memory source holds one connection.
	var toSuspend int64
	for c, err := range src.Anki().DeckCards(ctx, "", deck) {
		if err != nil {
			t.Fatal(err)
		}
		if toSuspend == 0 && c.PositionID != ids[0] {
			toSuspend = c.ID
		}
	}
	if err := src.Anki().SetCardSuspended(ctx, "", toSuspend, true); err != nil {
		t.Fatal(err)
	}

	for _, history := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "export.db")
		opts := ExportOptions{Selection: Selection{PositionIDs: ids[1:], DeckIDs: []int64{deck}}, AnkiHistory: history}
		if _, err := ExportSQLite(ctx, src, "", path, opts); err != nil {
			t.Fatal(err)
		}
		dst, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		var dstDeck int64
		for d, err := range dst.Anki().ListDecks(ctx, "") {
			if err != nil {
				t.Fatal(err)
			}
			dstDeck = d.ID
		}
		reviewed, held := 0, 0
		cards := map[int64]bool{}
		for c, err := range dst.Anki().DeckCards(ctx, "", dstDeck) {
			if err != nil {
				t.Fatal(err)
			}
			cards[c.ID] = true
			if c.Reps > 0 {
				reviewed++
			}
			if c.Suspended {
				held++
			}
		}
		var logs int
		for l, err := range dst.Anki().ReviewLog(ctx, "", 0, 0) {
			if err != nil {
				t.Fatal(err)
			}
			logs++
			if l.DeckID != dstDeck || !cards[l.CardID] {
				t.Errorf("history %v: review event on deck %d card %d, not on the file's deck %d", history, l.DeckID, l.CardID, dstDeck)
			}
		}
		dst.Close()

		want := 0
		if history {
			want = 2
		}
		if wantHeld := want / 2; held != wantHeld {
			t.Errorf("history %v: %d suspended cards, want %d", history, held, wantHeld)
		}
		if len(cards) != 2 || reviewed != want || logs != want {
			t.Errorf("history %v: %d cards, %d reviewed, %d review events; want 2, %d, %d", history, len(cards), reviewed, logs, want, want)
		}
	}
}
