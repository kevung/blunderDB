package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestExportDeckThenImport: a deck exported on its own carries its positions
// and not the sender's others, arrives without its review history, and a
// second import of the same file changes nothing.
func TestExportDeckThenImport(t *testing.T) {
	dir := t.TempDir()
	src := NewDatabase()
	if err := src.SetupDatabase(filepath.Join(dir, "coach.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = src.Close() })
	inDeck := domain.InitializePosition()
	kept := domain.InitializePosition()
	kept.Dice = [2]int{6, 5}
	pid, err := src.SavePosition(&inDeck)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.SavePosition(&kept); err != nil {
		t.Fatal(err)
	}
	deck, err := src.CreateAnkiDeck("Ouvertures", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.SyncAnkiDeckWithPositions(deck, []int64{pid}); err != nil {
		t.Fatal(err)
	}
	card, err := src.GetNextAnkiCard(deck)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.ReviewAnkiCard(card.Card.ID, 3); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "deck.db")
	if err := src.ExportDatabase(ExportOptions{ExportPath: file, DeckIDs: []int64{deck}}); err != nil {
		t.Fatal(err)
	}

	dst := NewDatabase()
	if err := dst.SetupDatabase(filepath.Join(dir, "eleve.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dst.Close() })
	for pass, want := range []int{1, 0} {
		res, err := dst.CommitImportDatabase(file)
		if err != nil {
			t.Fatalf("pass %d: %v", pass+1, err)
		}
		if res["decks"] != want {
			t.Errorf("pass %d: decks = %v, want %d", pass+1, res["decks"], want)
		}
	}
	decks, err := dst.GetAllAnkiDecks()
	if err != nil {
		t.Fatal(err)
	}
	if len(decks) != 1 || decks[0].Name != "Ouvertures" {
		t.Fatalf("decks = %+v, want Ouvertures alone", decks)
	}
	ps, err := dst.GetAnkiDeckPositions(decks[0].ID)
	if err != nil || len(ps) != 1 {
		t.Fatalf("deck positions = %d (%v), want 1", len(ps), err)
	}
	all, err := dst.LoadAllPositions()
	if err != nil || len(all) != 1 {
		t.Errorf("recipient holds %d positions (%v), want the deck's one", len(all), err)
	}
	log, err := dst.GetAnkiReviewLog(0, 0)
	if err != nil || len(log) != 0 {
		t.Errorf("review history travelled: %d entries (%v)", len(log), err)
	}
}
