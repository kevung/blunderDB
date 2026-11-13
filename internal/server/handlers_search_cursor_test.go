package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// checkForgedCursorIsBadRequest: a page cursor whose anchor the client rewrote — a
// key of another type than the sort column's — is a 400 invalid, never a 500; a key
// outside its column's range is never a 500 either.
func checkForgedCursorIsBadRequest(t *testing.T, ts *httptest.Server, st storage.Storage, tenant string) {
	t.Helper()
	for n := range 3 {
		p := domain.InitializePosition()
		p.Score = [2]int{n, 0}
		if _, err := st.Positions().Save(context.Background(), tenant, &p); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	page := func(cursor string) (*http.Response, []byte) {
		body, _ := json.Marshal(searchPageReq{Filters: domain.SearchFilters{Sort: "pip_asc"}, Limit: 1, Cursor: cursor})
		resp := doPost(t, ts.URL+"/v1/search.idsPage", tenant, string(body))
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp, raw
	}
	resp, raw := page("")
	var first searchIDsPageResp
	if err := json.Unmarshal(raw, &first); resp.StatusCode != http.StatusOK || err != nil || first.NextCursor == "" {
		t.Fatalf("first page: %d %s", resp.StatusCode, raw)
	}
	blob, err := base64.RawURLEncoding.DecodeString(first.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	var w map[string]json.RawMessage
	if err := json.Unmarshal(blob, &w); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"70000", "2147483648", "-9223372036854775808"} {
		w["i"] = json.RawMessage(n)
		forged, _ := json.Marshal(w)
		resp, raw := page(base64.RawURLEncoding.EncodeToString(forged))
		if resp.StatusCode >= http.StatusInternalServerError {
			t.Errorf("key %s outside the smallint pip_1: %d %s; want a page or 400", n, resp.StatusCode, strings.TrimSpace(string(raw)))
		}
	}
	delete(w, "i")
	for kind, v := range map[string]string{"b": "true", "t": "1600000000000000000"} {
		w[kind] = json.RawMessage(v)
		forged, _ := json.Marshal(w)
		delete(w, kind)
		resp, raw := page(base64.RawURLEncoding.EncodeToString(forged))
		if resp.StatusCode != http.StatusBadRequest || decodeEnvelope(t, raw).Error.Code != CodeInvalid {
			t.Errorf("key of kind %s on an integer sort: %d %s; want 400 invalid", kind, resp.StatusCode, strings.TrimSpace(string(raw)))
		}
	}
}

func TestSearchPageForgedCursorIsBadRequest(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	checkForgedCursorIsBadRequest(t, ts, srv.opts.Storage, testTenant)
}
