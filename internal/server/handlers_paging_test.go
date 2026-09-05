package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// The page cap (#232) is enforced generically, on every request type that
// implements pagedReq. That is what makes it hold for a route added later —
// but only if the route's request type actually implements it, which nothing
// but a test can check. G.9 (#237) added three such routes, and a request
// type that quietly forgot the method would be capped by nothing at all.
func TestPageLimit_AppliesToEveryListingRoute(t *testing.T) {
	ts := newTestServer(t)

	for _, tc := range []struct {
		route string
		body  any
	}{
		{"/v1/positions.list", map[string]any{"limit": 1001}},
		{"/v1/positions.listIds", map[string]any{"limit": 1001}},
		{"/v1/matches.list", map[string]any{"limit": 1001}},
		{"/v1/comments.listAll", map[string]any{"limit": 1001}},
		{"/v1/tournaments.list", map[string]any{"limit": 1001}},
		{"/v1/collections.positions", map[string]any{"collectionId": 1, "limit": 1001}},
	} {
		t.Run(tc.route, func(t *testing.T) {
			resp := post(t, ts, tc.route, tc.body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status %d, want 400 — a page of 1001 rows must be refused", resp.StatusCode)
			}
			raw, _ := io.ReadAll(resp.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			msg, _ := body["message"].(string)
			if msg == "" {
				msg = string(raw)
			}
			if !strings.Contains(msg, "maximum page size") {
				t.Errorf("the refusal does not say why: %s", msg)
			}
		})
	}

	// And a page inside the cap is honoured, not refused: the cap must not
	// have been implemented by refusing every limit.
	resp := post(t, ts, "/v1/comments.listAll", map[string]any{"limit": 10})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("a page of ten was refused: status %d, %s", resp.StatusCode, raw)
	}
}
