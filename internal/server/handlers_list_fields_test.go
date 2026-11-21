package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func postListFields(t *testing.T, ts *httptest.Server, tenant string, ids []int64) []storage.PositionListFields {
	t.Helper()
	resp := postAsTenant(t, ts, tenant, "/v1/positions.listFields", idsReq{IDs: ids})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("positions.listFields status = %d, want 200", resp.StatusCode)
	}
	var got []storage.PositionListFields
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("positions.listFields answered null, want a JSON array")
	}
	return got
}

func TestPositionsListFields(t *testing.T) {
	ts := newTestServer(t)
	analysed := saveAnalysedPosition(t, ts, testTenant, 1, true)
	bare := saveAnalysedPosition(t, ts, testTenant, 2, false)

	got := postListFields(t, ts, testTenant, []int64{bare, 987654321, analysed})
	if len(got) != 2 || got[0].PositionID != bare || got[1].PositionID != analysed {
		t.Fatalf("positions.listFields = %+v, want bare then analysed, the unknown id skipped", got)
	}
	if got[0].Analysed || got[0].PlayedErrorMP != nil || !got[1].Analysed {
		t.Errorf("analysed flags = %v / %v, want false / true, no error without analysis", got[0].Analysed, got[1].Analysed)
	}
	if got[0].Date == "" || got[0].MatchDated || got[0].Collections == nil {
		t.Errorf("bare position = %+v, want its writing day and an empty collection list", got[0])
	}

	resp := postAsTenant(t, ts, testTenant, "/v1/positions.listFields", idsReq{IDs: make([]int64, maxPageSize+1)})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("positions.listFields over %d ids: status %d, want 400", maxPageSize, resp.StatusCode)
	}
}

func TestSearchPageListFields(t *testing.T) {
	ts := newTestServer(t)
	a := saveAnalysedPosition(t, ts, testTenant, 1, true)
	b := saveAnalysedPosition(t, ts, testTenant, 2, false)

	page := func(listFields bool) (searchPageResp, map[string]json.RawMessage) {
		t.Helper()
		resp := postAsTenant(t, ts, testTenant, "/v1/search.page", searchPageReq{Limit: 10, ListFields: listFields})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("search.page status = %d, want 200", resp.StatusCode)
		}
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatal(err)
		}
		var r searchPageResp
		buf, _ := json.Marshal(raw)
		if err := json.Unmarshal(buf, &r); err != nil {
			t.Fatal(err)
		}
		return r, raw
	}

	if _, raw := page(false); raw["listFields"] != nil {
		t.Errorf("search.page without listFields answered %s, want the field absent", raw["listFields"])
	}
	r, _ := page(true)
	if len(r.Positions) != 2 || len(r.ListFields) != len(r.Positions) {
		t.Fatalf("search.page: %d positions, %d list rows; want 2 and 2", len(r.Positions), len(r.ListFields))
	}
	analysed := map[int64]bool{a: true, b: false}
	for i, p := range r.Positions {
		f := r.ListFields[i]
		if f.PositionID != p.ID || f.Analysed != analysed[p.ID] {
			t.Errorf("row %d: position %d, list fields %+v; want the same position, analysed %v", i, p.ID, f, analysed[p.ID])
		}
	}
}

// putInDeck creates a study deck of the tenant and gives it a card of each
// position, through the API.
func putInDeck(t *testing.T, ts *httptest.Server, tenant, name string, ids ...int64) int64 {
	t.Helper()
	resp := postAsTenant(t, ts, tenant, "/v1/anki.createDeck", deckCreateReq{Name: name, SourceType: domain.AnkiSourceSearch})
	var deck idResp
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&deck) != nil {
		resp.Body.Close()
		t.Fatalf("anki.createDeck status = %d, want 200 and an id", resp.StatusCode)
	}
	resp.Body.Close()
	resp = postAsTenant(t, ts, tenant, "/v1/anki.syncWithPositions", deckSyncPositionsReq{DeckID: deck.ID, PositionIDs: ids})
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		t.Fatalf("anki.syncWithPositions status = %d, want success", resp.StatusCode)
	}
	return deck.ID
}

func TestPositionsListFieldsDecks(t *testing.T) {
	ts := newTestServer(t)
	inDecks := saveAnalysedPosition(t, ts, testTenant, 1, true)
	bare := saveAnalysedPosition(t, ts, testTenant, 2, false)
	second := putInDeck(t, ts, testTenant, "Second", inDecks)
	first := putInDeck(t, ts, testTenant, "First", inDecks)

	resp := postAsTenant(t, ts, testTenant, "/v1/positions.listFields", idsReq{IDs: []int64{inDecks, bare}})
	var raw []map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw) != 2 {
		resp.Body.Close()
		t.Fatalf("positions.listFields = %v, %v; want two rows", raw, err)
	}
	resp.Body.Close()
	if string(raw[1]["decks"]) != "[]" {
		t.Errorf("a position in no deck: decks = %s, want []", raw[1]["decks"])
	}
	var decks []storage.DeckRef
	if err := json.Unmarshal(raw[0]["decks"], &decks); err != nil || len(decks) != 2 ||
		decks[0] != (storage.DeckRef{ID: second, Name: "Second"}) || decks[1] != (storage.DeckRef{ID: first, Name: "First"}) {
		t.Errorf("decks = %s, want Second then First, in the order they were created", raw[0]["decks"])
	}
	for _, key := range []string{"positionId", "analysed", "playedErrorMp", "date", "matchDated", "collections"} {
		if _, ok := raw[0][key]; !ok {
			t.Errorf("positions.listFields row lost its %q field", key)
		}
	}

	resp = postAsTenant(t, ts, testTenant, "/v1/search.page", searchPageReq{Limit: 10, ListFields: true})
	var page searchPageResp
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		resp.Body.Close()
		t.Fatal(err)
	}
	resp.Body.Close()
	for i, p := range page.Positions {
		want := 0
		if p.ID == inDecks {
			want = 2
		}
		if f := page.ListFields[i]; f.Decks == nil || len(f.Decks) != want {
			t.Errorf("search.page row of position %d: decks %+v, want %d", p.ID, f.Decks, want)
		}
	}
}
