package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
