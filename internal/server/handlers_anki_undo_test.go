package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestUndoReviewRoute grades a card then undoes the grade over HTTP: the
// card comes back as it was, and a second undo is the named 409.
func TestUndoReviewRoute(t *testing.T) {
	ts := newTestServer(t)
	decode := func(path string, req, v any) {
		t.Helper()
		resp := post(t, ts, path, req)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, body)
		}
		if v != nil {
			if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
				t.Fatal(err)
			}
		}
	}
	p := domain.InitializePosition()
	var pos idResp
	decode("/v1/positions.save", positionReq{Position: &p}, &pos)
	var deck idResp
	decode("/v1/anki.createDeck", deckCreateReq{Name: "d", SourceType: domain.AnkiSourceSearch}, &deck)
	decode("/v1/anki.syncWithPositions", deckSyncPositionsReq{DeckID: deck.ID, PositionIDs: []int64{pos.ID}}, nil)
	var card domain.AnkiReviewCard
	decode("/v1/anki.nextCard", deckIDReq{DeckID: deck.ID}, &card)

	decode("/v1/anki.reviewCard", reviewCardReq{CardID: card.Card.ID, Rating: 3}, nil)
	var restored domain.AnkiReviewCard
	decode("/v1/anki.undoReview", cardIDReq{CardID: card.Card.ID}, &restored)
	if restored.Card != card.Card {
		t.Errorf("restored card = %+v, want %+v", restored.Card, card.Card)
	}

	resp := post(t, ts, "/v1/anki.undoReview", cardIDReq{CardID: card.Card.ID})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict || !strings.Contains(string(body), "the card has no review to undo") {
		t.Errorf("second undo = %d %s, want 409 naming the refusal", resp.StatusCode, body)
	}
}
