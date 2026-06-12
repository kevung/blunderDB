package database

import (
	"testing"
)

// setupAnkiCollectionWithPositions creates a collection with N positions and returns (db, collectionID, positionIDs).
func setupAnkiCollectionWithPositions(t *testing.T, n int) (*Database, int64, []int64) {
	t.Helper()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, n)

	colID, err := db.CreateCollection("AnkiSource", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := db.AddPositionsToCollection(colID, ids); err != nil {
		t.Fatalf("AddPositionsToCollection: %v", err)
	}

	return db, colID, ids
}

func TestCreateAnkiDeck(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	id, err := db.CreateAnkiDeck("MyDeck", "A deck", "collection", colID, "")
	if err != nil {
		t.Fatalf("CreateAnkiDeck: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestGetAllAnkiDecks(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	db.CreateAnkiDeck("D1", "", "collection", colID, "")
	db.CreateAnkiDeck("D2", "", "collection", colID, "")

	decks, err := db.GetAllAnkiDecks()
	if err != nil {
		t.Fatalf("GetAllAnkiDecks: %v", err)
	}
	if len(decks) != 2 {
		t.Fatalf("got %d decks, want 2", len(decks))
	}
}

func TestUpdateAnkiDeck(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	id, _ := db.CreateAnkiDeck("Old", "OldDesc", "collection", colID, "")
	if err := db.UpdateAnkiDeck(id, "New", "NewDesc"); err != nil {
		t.Fatalf("UpdateAnkiDeck: %v", err)
	}

	decks, _ := db.GetAllAnkiDecks()
	for _, d := range decks {
		if d.ID == id {
			if d.Name != "New" || d.Description != "NewDesc" {
				t.Errorf("got name=%q desc=%q", d.Name, d.Description)
			}
			return
		}
	}
	t.Error("deck not found")
}

func TestUpdateAnkiDeckParams(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	id, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	if err := db.UpdateAnkiDeckParams(id, 0.85, 180.0, true); err != nil {
		t.Fatalf("UpdateAnkiDeckParams: %v", err)
	}

	decks, _ := db.GetAllAnkiDecks()
	for _, d := range decks {
		if d.ID == id {
			if d.RequestRetention != 0.85 {
				t.Errorf("retention = %f, want 0.85", d.RequestRetention)
			}
			if d.MaximumInterval != 180.0 {
				t.Errorf("maxInterval = %f, want 180", d.MaximumInterval)
			}
			if !d.EnableFuzz {
				t.Error("enableFuzz should be true")
			}
			return
		}
	}
	t.Error("deck not found")
}

func TestDeleteAnkiDeck(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	id, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(id)

	if err := db.DeleteAnkiDeck(id); err != nil {
		t.Fatalf("DeleteAnkiDeck: %v", err)
	}

	decks, _ := db.GetAllAnkiDecks()
	for _, d := range decks {
		if d.ID == id {
			t.Error("deck still exists after delete")
		}
	}

	// Cards should be cascade-deleted
	var count int
	db.db.QueryRow(`SELECT COUNT(*) FROM anki_card WHERE deck_id = ?`, id).Scan(&count)
	if count != 0 {
		t.Errorf("found %d orphan cards after deck delete", count)
	}
}

func TestSyncAnkiDeck_Collection(t *testing.T) {
	db, colID, ids := setupAnkiCollectionWithPositions(t, 5)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	if err := db.SyncAnkiDeck(deckID); err != nil {
		t.Fatalf("SyncAnkiDeck: %v", err)
	}

	stats, err := db.GetAnkiDeckStats(deckID)
	if err != nil {
		t.Fatalf("GetAnkiDeckStats: %v", err)
	}
	if stats.TotalCount != len(ids) {
		t.Errorf("totalCount = %d, want %d", stats.TotalCount, len(ids))
	}
}

func TestSyncAnkiDeck_Idempotent(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)
	_ = db.SyncAnkiDeck(deckID) // second sync

	stats, _ := db.GetAnkiDeckStats(deckID)
	if stats.TotalCount != 3 {
		t.Errorf("totalCount = %d after double sync, want 3", stats.TotalCount)
	}
}

func TestSyncAnkiDeck_AddNew(t *testing.T) {
	db, colID, ids := setupAnkiCollectionWithPositions(t, 3)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	// Add another position to the collection
	moreIDs := getPositionIDs(t, db, 5)
	var newID int64
	for _, mid := range moreIDs {
		found := false
		for _, eid := range ids {
			if mid == eid {
				found = true
				break
			}
		}
		if !found {
			newID = mid
			break
		}
	}
	if newID == 0 {
		t.Skip("no additional position available")
	}
	_ = db.AddPositionToCollection(colID, newID)
	_ = db.SyncAnkiDeck(deckID)

	stats, _ := db.GetAnkiDeckStats(deckID)
	if stats.TotalCount != 4 {
		t.Errorf("totalCount = %d after adding position, want 4", stats.TotalCount)
	}
}

func TestGetNextAnkiCard(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	card, err := db.GetNextAnkiCard(deckID)
	if err != nil {
		t.Fatalf("GetNextAnkiCard: %v", err)
	}
	if card == nil {
		t.Fatal("expected a card, got nil")
	}
	if card.Card.ID <= 0 {
		t.Error("card ID should be positive")
	}
	if card.Position.ID <= 0 {
		t.Error("position ID should be positive")
	}
}

func TestGetNextAnkiCard_Empty(t *testing.T) {
	db := newTestDB(t)

	colID, _ := db.CreateCollection("Empty", "")
	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")

	card, err := db.GetNextAnkiCard(deckID)
	if err != nil {
		t.Fatalf("GetNextAnkiCard: %v", err)
	}
	if card != nil {
		t.Error("expected nil for empty deck")
	}
}

func TestGetRandomAnkiCard(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	card, err := db.GetRandomAnkiCard(deckID, 0)
	if err != nil {
		t.Fatalf("GetRandomAnkiCard: %v", err)
	}
	if card == nil {
		t.Fatal("expected a card, got nil")
	}
	if card.Card.ID <= 0 || card.Position.ID <= 0 {
		t.Errorf("expected positive ids, got card=%d position=%d", card.Card.ID, card.Position.ID)
	}
}

func TestGetRandomAnkiCard_Empty(t *testing.T) {
	db := newTestDB(t)

	colID, _ := db.CreateCollection("Empty", "")
	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")

	card, err := db.GetRandomAnkiCard(deckID, 0)
	if err != nil {
		t.Fatalf("GetRandomAnkiCard: %v", err)
	}
	if card != nil {
		t.Error("expected nil for empty deck")
	}
}

// TestGetRandomAnkiCard_DoesNotMutateSchedule is the defining property of cram
// mode: drawing cards must never touch the FSRS schedule, unlike ReviewAnkiCard.
func TestGetRandomAnkiCard_DoesNotMutateSchedule(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	before, err := db.GetAnkiDeckStats(deckID)
	if err != nil {
		t.Fatalf("GetAnkiDeckStats: %v", err)
	}

	for i := 0; i < 12; i++ {
		if _, err := db.GetRandomAnkiCard(deckID, 0); err != nil {
			t.Fatalf("GetRandomAnkiCard draw %d: %v", i, err)
		}
	}

	after, err := db.GetAnkiDeckStats(deckID)
	if err != nil {
		t.Fatalf("GetAnkiDeckStats: %v", err)
	}
	if before != after {
		t.Errorf("cram mutated the schedule: before=%+v after=%+v", before, after)
	}
	// All cards must still be New (a real review would graduate them out).
	if after.NewCount != after.TotalCount {
		t.Errorf("expected all %d cards to stay New, got NewCount=%d", after.TotalCount, after.NewCount)
	}
}

// TestGetRandomAnkiCard_ExcludePosition checks that the exclusion avoids
// repeats but still serves a single-card deck.
func TestGetRandomAnkiCard_ExcludePosition(t *testing.T) {
	// Two-card deck: excluding one must return the other (deterministic).
	db, colID, ids := setupAnkiCollectionWithPositions(t, 2)
	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	got, err := db.GetRandomAnkiCard(deckID, ids[0])
	if err != nil {
		t.Fatalf("GetRandomAnkiCard: %v", err)
	}
	if got == nil || got.Position.ID == ids[0] {
		t.Errorf("exclusion failed: expected a position other than %d, got %+v", ids[0], got)
	}

	// Single-card deck: excluding the only card falls back to serving it.
	db1, col1, ids1 := setupAnkiCollectionWithPositions(t, 1)
	deck1, _ := db1.CreateAnkiDeck("D", "", "collection", col1, "")
	_ = db1.SyncAnkiDeck(deck1)

	only, err := db1.GetRandomAnkiCard(deck1, ids1[0])
	if err != nil {
		t.Fatalf("GetRandomAnkiCard single: %v", err)
	}
	if only == nil || only.Position.ID != ids1[0] {
		t.Errorf("single-card fallback failed: expected position %d, got %+v", ids1[0], only)
	}
}

func TestReviewAnkiCard_Again(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 1)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	card, _ := db.GetNextAnkiCard(deckID)
	if card == nil {
		t.Fatal("no card")
	}

	// Rating 1 = Again
	_, err := db.ReviewAnkiCard(card.Card.ID, 1)
	if err != nil {
		t.Fatalf("ReviewAnkiCard: %v", err)
	}

	// Verify card was updated
	var reps, state int
	db.db.QueryRow(`SELECT reps, state FROM anki_card WHERE id = ?`, card.Card.ID).
		Scan(&reps, &state)
	if reps != 1 {
		t.Errorf("reps = %d, want 1", reps)
	}
	// State should be Learning (1) or Relearning (3) after "Again"
	if state != 1 && state != 3 {
		t.Errorf("state = %d, want 1 (Learning) or 3 (Relearning)", state)
	}
}

func TestReviewAnkiCard_Good(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 1)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	card, _ := db.GetNextAnkiCard(deckID)
	if card == nil {
		t.Fatal("no card")
	}

	// Rating 3 = Good
	_, err := db.ReviewAnkiCard(card.Card.ID, 3)
	if err != nil {
		t.Fatalf("ReviewAnkiCard: %v", err)
	}

	var reps int
	var stability float64
	db.db.QueryRow(`SELECT reps, stability FROM anki_card WHERE id = ?`, card.Card.ID).
		Scan(&reps, &stability)
	if reps != 1 {
		t.Errorf("reps = %d, want 1", reps)
	}
	if stability <= 0 {
		t.Errorf("stability = %f, want > 0", stability)
	}
}

func TestReviewAnkiCard_Easy(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 1)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	card, _ := db.GetNextAnkiCard(deckID)
	if card == nil {
		t.Fatal("no card")
	}

	// Rating 4 = Easy
	_, err := db.ReviewAnkiCard(card.Card.ID, 4)
	if err != nil {
		t.Fatalf("ReviewAnkiCard: %v", err)
	}

	var stability float64
	db.db.QueryRow(`SELECT stability FROM anki_card WHERE id = ?`, card.Card.ID).
		Scan(&stability)
	if stability <= 0 {
		t.Errorf("stability = %f, want > 0", stability)
	}
}

func TestReviewAnkiCard_Progression(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 1)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	card, _ := db.GetNextAnkiCard(deckID)
	if card == nil {
		t.Fatal("no card")
	}
	cardID := card.Card.ID

	// Again → Good → Good → Easy
	ratings := []int{1, 3, 3, 4}
	for _, r := range ratings {
		if _, err := db.ReviewAnkiCard(cardID, r); err != nil {
			t.Fatalf("ReviewAnkiCard(rating=%d): %v", r, err)
		}
	}

	var reps int
	db.db.QueryRow(`SELECT reps FROM anki_card WHERE id = ?`, cardID).Scan(&reps)
	if reps != 4 {
		t.Errorf("reps = %d after 4 reviews, want 4", reps)
	}
}

func TestResetAnkiDeck(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 3)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	// Review a card
	card, _ := db.GetNextAnkiCard(deckID)
	if card != nil {
		db.ReviewAnkiCard(card.Card.ID, 3)
	}

	if err := db.ResetAnkiDeck(deckID); err != nil {
		t.Fatalf("ResetAnkiDeck: %v", err)
	}

	// All cards should be back to new
	var maxReps int
	db.db.QueryRow(`SELECT COALESCE(MAX(reps), 0) FROM anki_card WHERE deck_id = ?`, deckID).Scan(&maxReps)
	if maxReps != 0 {
		t.Errorf("max reps = %d after reset, want 0", maxReps)
	}
}

func TestGetAnkiDeckStats(t *testing.T) {
	db, colID, _ := setupAnkiCollectionWithPositions(t, 5)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	stats, err := db.GetAnkiDeckStats(deckID)
	if err != nil {
		t.Fatalf("GetAnkiDeckStats: %v", err)
	}
	if stats.TotalCount != 5 {
		t.Errorf("totalCount = %d, want 5", stats.TotalCount)
	}
	// All should be new initially
	if stats.NewCount != 5 {
		t.Errorf("newCount = %d, want 5", stats.NewCount)
	}
}

func TestGetAnkiDeckPositions(t *testing.T) {
	db, colID, ids := setupAnkiCollectionWithPositions(t, 3)

	deckID, _ := db.CreateAnkiDeck("D", "", "collection", colID, "")
	_ = db.SyncAnkiDeck(deckID)

	positions, err := db.GetAnkiDeckPositions(deckID)
	if err != nil {
		t.Fatalf("GetAnkiDeckPositions: %v", err)
	}
	if len(positions) != len(ids) {
		t.Errorf("got %d positions, want %d", len(positions), len(ids))
	}
}
