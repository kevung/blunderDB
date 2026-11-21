//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/ -run ListFields
package server

import "testing"

// TestPositionsListFieldsDecksIsolationPostgres: under row-level security, the
// list fields of a tenant name only its own study decks — a deck of another
// tenant never shows, even one that asked for the same position ids.
func TestPositionsListFieldsDecksIsolationPostgres(t *testing.T) {
	ts := newPostgresTestServer(t)
	const a, b = "1", "2"
	posA := saveAnalysedPosition(t, ts, a, 1, true)
	posB := saveAnalysedPosition(t, ts, b, 1, true)
	deckA := putInDeck(t, ts, a, "Pile A", posA)
	_ = putInDeck(t, ts, b, "Pile B", posB)
	// B asks its deck for A's position too: whatever B's deck makes of it, it
	// must not reach A's list fields.
	resp := postAsTenant(t, ts, b, "/v1/anki.syncWithPositions", deckSyncPositionsReq{DeckID: deckA, PositionIDs: []int64{posA, posB}})
	resp.Body.Close()

	got := postListFields(t, ts, a, []int64{posA, posB})
	if len(got) != 1 || got[0].PositionID != posA {
		t.Fatalf("tenant %s list fields = %+v, want its own position alone", a, got)
	}
	if d := got[0].Decks; len(d) != 1 || d[0].ID != deckA || d[0].Name != "Pile A" {
		t.Errorf("tenant %s decks = %+v, want Pile A alone", a, d)
	}
	if got := postListFields(t, ts, b, []int64{posA}); len(got) != 0 {
		t.Errorf("tenant %s reads %s's list fields: %+v", b, a, got)
	}
	got = postListFields(t, ts, b, []int64{posB})
	if len(got) != 1 || len(got[0].Decks) != 1 || got[0].Decks[0].Name != "Pile B" {
		t.Errorf("tenant %s list fields = %+v, want Pile B alone", b, got)
	}
}
