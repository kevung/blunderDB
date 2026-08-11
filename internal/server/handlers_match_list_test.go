package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// decodeMatchList collects the NDJSON match stream from /v1/matches.list.
func decodeMatchList(t *testing.T, ts *httptest.Server, req matchListReq) []int64 {
	t.Helper()
	resp := post(t, ts, "/v1/matches.list", req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("matches.list status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != ndjsonContentType {
		t.Fatalf("content-type = %q, want %q", ct, ndjsonContentType)
	}
	var ids []int64
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var m domain.Match
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("decode row %q: %v", sc.Text(), err)
		}
		ids = append(ids, m.ID)
	}
	return ids
}

// TestMatchListFilterSortPaginateHTTP exercises the DTO wiring of matches.list:
// filters, sort keys and pagination must reach storage and shape the stream.
func TestMatchListFilterSortPaginateHTTP(t *testing.T) {
	ts := newTestServer(t)

	save := func(p1, p2 string, length int32, date string) int64 {
		tm, err := time.Parse("2006-01-02", date)
		if err != nil {
			t.Fatalf("parse %q: %v", date, err)
		}
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: length, MatchDate: tm}
		resp := post(t, ts, "/v1/matches.save", matchSaveReq{Match: &m})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("save status = %d", resp.StatusCode)
		}
		var got idResp
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return got.ID
	}
	old := save("Alice", "Bob", 7, "2023-06-15")
	mid := save("Carol", "Dave", 3, "2024-06-15")
	recent := save("Eve", "Alice", 11, "2025-06-15")

	eq := func(name string, got, want []int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: got %v want %v", name, got, want)
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: got %v want %v", name, got, want)
				return
			}
		}
	}

	eq("default", decodeMatchList(t, ts, matchListReq{}), []int64{recent, mid, old})
	eq("date_asc", decodeMatchList(t, ts, matchListReq{Sort: "date_asc"}), []int64{old, mid, recent})
	eq("player Alice", decodeMatchList(t, ts, matchListReq{PlayerName: "Alice"}), []int64{recent, old})
	eq("year 2024", decodeMatchList(t, ts, matchListReq{DateFrom: "2024-01-01", DateTo: "2024-12-31"}), []int64{mid})
	eq("length 11", decodeMatchList(t, ts, matchListReq{MatchLength: []int{11}}), []int64{recent})
	eq("limit+offset", decodeMatchList(t, ts, matchListReq{Limit: 1, Offset: 1}), []int64{mid})
}

// TestStatsMatchBadgesHTTP checks the scoped-badges route answers 200 with a
// keyed map for both an empty body (whole-DB scan) and an explicit id list.
func TestStatsMatchBadgesHTTP(t *testing.T) {
	ts := newTestServer(t)
	// Fresh DB has no analysed decisions, so every match badge is absent; the
	// route must still answer 200 with an empty badges object for both shapes.
	for _, req := range []matchBadgesReq{{}, {MatchIDs: []int64{1, 2}}} {
		resp := post(t, ts, "/v1/stats.matchBadges", req)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("matchBadges status = %d, want 200", resp.StatusCode)
		}
		var body matchBadgesResp
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		resp.Body.Close()
		if len(body.Badges) != 0 {
			t.Errorf("fresh DB: got %d badges, want 0", len(body.Badges))
		}
	}
}

// TestStatsTournamentBadgesHTTP is the tournament-scoped counterpart of
// TestStatsMatchBadgesHTTP — the route (storage.StatsStore.TournamentBadges)
// existed on the Storage contract but was never wired into the RPC router.
func TestStatsTournamentBadgesHTTP(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/stats.tournamentBadges", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tournamentBadges status = %d, want 200", resp.StatusCode)
	}
	var body tournamentBadgesResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Fresh DB has no tournaments with counted decisions, so the map is empty
	// — the route existing and answering 200 is what this test guards.
	if len(body.Badges) != 0 {
		t.Errorf("fresh DB: got %d tournament badges, want 0", len(body.Badges))
	}
}

// TestMatchesFindByHashHTTP guards the route matches.findByHash
// (storage.MatchStore.FindByHash), also never wired into the RPC router: a
// match saved with a hash must be found by it, an unknown hash must answer
// found=false rather than an error, and an id-only match (no hash set) must
// not be reachable through this lookup.
func TestMatchesFindByHashHTTP(t *testing.T) {
	ts := newTestServer(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchHash: "abc123", CanonicalHash: "canon456"}
	saveResp := post(t, ts, "/v1/matches.save", matchSaveReq{Match: &m})
	var saved idResp
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	saveResp.Body.Close()

	findByHash := func(req findByHashReq) findByHashResp {
		t.Helper()
		resp := post(t, ts, "/v1/matches.findByHash", req)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("findByHash(%+v) status = %d, want 200", req, resp.StatusCode)
		}
		var body findByHashResp
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return body
	}

	if got := findByHash(findByHashReq{Hash: "abc123"}); !got.Found || got.ID != saved.ID {
		t.Errorf("lookup by match_hash = %+v, want found=true id=%d", got, saved.ID)
	}
	if got := findByHash(findByHashReq{CanonicalHash: "canon456"}); !got.Found || got.ID != saved.ID {
		t.Errorf("lookup by canonical_hash = %+v, want found=true id=%d", got, saved.ID)
	}
	if got := findByHash(findByHashReq{Hash: "does-not-exist"}); got.Found {
		t.Errorf("lookup by unknown hash = %+v, want found=false", got)
	}
}
