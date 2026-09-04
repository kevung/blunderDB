package server

// handlers_search_query_test.go — /v1/search.parse and /v1/search.query, the
// query-language door onto the search (B.18, #186). Before them, every /v1
// client had to assemble domain.SearchFilters by hand, and twenty-odd filters
// the command bar offers were reachable from nowhere else.

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func parseQuery(t *testing.T, ts *httptest.Server, query string) searchParseResp {
	t.Helper()
	resp := post(t, ts, "/v1/search.parse", searchQueryReq{Query: query})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search.parse(%q) status = %d, want 200", query, resp.StatusCode)
	}
	var out searchParseResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestSearchParseReportsFiltersCanonicalAndDiagnostics(t *testing.T) {
	ts := newTestServer(t)

	got := parseQuery(t, ts, `s cube p>30 E>50 pl"Alice"`)
	if !got.Filters.IncludeCube {
		t.Error("includeCube not set")
	}
	if got.Filters.PipCountFilter != "p>30" {
		t.Errorf("pipCountFilter = %q, want %q", got.Filters.PipCountFilter, "p>30")
	}
	if got.Filters.MoveErrorFilter != "E>50" {
		t.Errorf("moveErrorFilter = %q, want %q", got.Filters.MoveErrorFilter, "E>50")
	}
	if got.Filters.PlayerFilter != `pl"Alice"` {
		t.Errorf("playerFilter = %q", got.Filters.PlayerFilter)
	}
	if got.Canonical == "" {
		t.Error("canonical form is empty")
	}
	// The canonical form must parse back to the same filters — that is what
	// makes it usable as a saved search's identity.
	again := parseQuery(t, ts, got.Canonical)
	if again.Canonical != got.Canonical {
		t.Errorf("canonical form is not stable: %q then %q", got.Canonical, again.Canonical)
	}

	// A token the grammar understands but cannot act on is reported, not hidden.
	withX := parseQuery(t, ts, "s x")
	if len(withX.Diags) == 0 {
		t.Fatal("`x` produced no diagnostic")
	}
	if withX.Diags[0].Kind != "no-effect" {
		t.Errorf("diag kind = %q, want no-effect", withX.Diags[0].Kind)
	}

	// So is a token nothing claimed — search.parse reports where search.query
	// refuses, because reporting is what search.parse is for.
	unknown := parseQuery(t, ts, "s cube nosuchtoken")
	found := false
	for _, d := range unknown.Diags {
		if d.Kind == "unknown" && d.Token == "nosuchtoken" {
			found = true
		}
	}
	if !found {
		t.Errorf("unknown token not reported: %+v", unknown.Diags)
	}
}

func TestSearchQueryStreamsAndRefusesUnreadableQueries(t *testing.T) {
	ts := newTestServer(t)

	// Two distinct positions, so a query has something to stream.
	for i, dt := range []int{domain.CheckerAction, domain.CubeAction} {
		p := domain.InitializePosition()
		p.DecisionType = dt
		p.Board.Points[6].Checkers = 3 + i // distinct boards, so the two do not dedup
		resp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("positions.save(%d) status = %d", dt, resp.StatusCode)
		}
		resp.Body.Close()
	}

	count := func(query string) int {
		resp := post(t, ts, "/v1/search.query", searchQueryReq{Query: query})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("search.query(%q) status = %d, want 200", query, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != ndjsonContentType {
			t.Fatalf("content-type = %q, want %q", ct, ndjsonContentType)
		}
		n := 0
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.TrimSpace(sc.Text()) != "" {
				n++
			}
		}
		return n
	}

	if all := count("s"); all < 2 {
		t.Fatalf("empty query returned %d positions, want at least 2", all)
	}

	// An unreadable query is refused before any row is written, so the client
	// sees an error rather than a confidently wrong result set.
	resp := post(t, ts, "/v1/search.query", searchQueryReq{Query: "s nosuchtoken"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unreadable query status = %d, want 400", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if !strings.Contains(env.Error.Message, "nosuchtoken") {
		t.Errorf("error message %q does not name the offending token", env.Error.Message)
	}

	// A no-effect token still runs the search, and says so in a header rather
	// than corrupting the NDJSON body.
	withX := post(t, ts, "/v1/search.query", searchQueryReq{Query: "s x"})
	defer withX.Body.Close()
	if withX.StatusCode != http.StatusOK {
		t.Fatalf("query with `x` status = %d, want 200", withX.StatusCode)
	}
	if h := withX.Header.Get("X-BlunderDB-Query-Diagnostics"); !strings.Contains(h, "no-effect") {
		t.Errorf("diagnostics header = %q, want a no-effect entry", h)
	}
}
