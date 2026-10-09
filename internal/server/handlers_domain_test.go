package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

const testTenant = "1"

// post issues a POST to the daemon with the tenant header and a JSON body.
func post(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestPositionsRoundtrip(t *testing.T) {
	ts := newTestServer(t)

	// Save a position.
	p := domain.InitializePosition()
	resp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save status = %d, want 200", resp.StatusCode)
	}
	var saved idResp
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID == 0 {
		t.Fatal("save returned id 0")
	}

	// Load it back.
	resp2 := post(t, ts, "/v1/positions.load", idReq(saved))
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("load status = %d, want 200", resp2.StatusCode)
	}
	var got domain.Position
	if err := json.NewDecoder(resp2.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != saved.ID {
		t.Fatalf("loaded id = %d, want %d", got.ID, saved.ID)
	}
}

func TestPositionsListNDJSON(t *testing.T) {
	ts := newTestServer(t)
	p := domain.InitializePosition()
	post(t, ts, "/v1/positions.save", positionReq{Position: &p}).Body.Close()

	resp := post(t, ts, "/v1/positions.list", listReq{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != ndjsonContentType {
		t.Fatalf("content-type = %q, want %q", ct, ndjsonContentType)
	}
	n := 0
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("ndjson line not JSON: %q (%v)", line, err)
		}
		n++
	}
	if n != 1 {
		t.Fatalf("ndjson rows = %d, want 1", n)
	}
}

func TestLoadMissingReturnsNotFound(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/positions.load", idReq{ID: 999999})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeNotFound {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeNotFound)
	}
}

// TestUnknownRouteHasItsOwnCode: a method the daemon does not serve answers
// unknown_route, so a client never reads a version mismatch as missing data.
func TestUnknownRouteHasItsOwnCode(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/positions.noSuchMethod", struct{}{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeUnknownRoute {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeUnknownRoute)
	}
}

// TestPositionsSaveReportsCreated: the first save of a position says created,
// the second does not, and both name the same row.
func TestPositionsSaveReportsCreated(t *testing.T) {
	ts := newTestServer(t)
	save := func() positionSaveResp {
		t.Helper()
		p := domain.InitializePosition()
		resp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("save status = %d, want 200", resp.StatusCode)
		}
		var got positionSaveResp
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	first, second := save(), save()
	if !first.Created || second.Created {
		t.Fatalf("created = %v then %v, want true then false", first.Created, second.Created)
	}
	if first.ID != second.ID {
		t.Fatalf("ids %d then %d, want the same row", first.ID, second.ID)
	}
}

func TestVoidHandlerReturnsOK(t *testing.T) {
	ts := newTestServer(t)
	// Deleting a non-existent position is a no-op that succeeds.
	resp := post(t, ts, "/v1/positions.delete", idReq{ID: 12345})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var ok okResp
	if err := json.NewDecoder(resp.Body).Decode(&ok); err != nil {
		t.Fatal(err)
	}
	if !ok.OK {
		t.Fatal("expected ok=true")
	}
}

func TestMetadataCounts(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/metadata.counts", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["positions"]; !ok {
		t.Fatalf("counts missing 'positions' field: %v", body)
	}
}

func TestInvalidJSONBody(t *testing.T) {
	ts := newTestServer(t)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/positions.load", strings.NewReader("{not json"))
	req.Header.Set(middleware.TenantHeader, testTenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeInvalid {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeInvalid)
	}
}

// TestLivingCollectionRoutesResolveTheQuery: a living collection holds no
// membership rows, its positions are those its saved query finds. The daemon
// answers as the GUI and the CLI do, on every route that reads a collection.
func TestLivingCollectionRoutesResolveTheQuery(t *testing.T) {
	ts := newTestServer(t)
	saveID := func(p domain.Position) int64 {
		resp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
		defer resp.Body.Close()
		var saved idResp
		if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
			t.Fatal(err)
		}
		return saved.ID
	}
	checker := domain.InitializePosition()
	checker.DecisionType = domain.CheckerAction
	cube := domain.InitializePosition()
	cube.DecisionType = domain.CubeAction
	cube.Board.Points[1] = domain.Point{Checkers: 1, Color: domain.White}
	cube.Board.Points[3] = domain.Point{Checkers: 1, Color: domain.White}
	saveID(checker)
	cubeID := saveID(cube)

	resp := post(t, ts, "/v1/collections.create", collectionCreateReq{Name: "living"})
	var col idResp
	if err := json.NewDecoder(resp.Body).Decode(&col); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	post(t, ts, "/v1/collections.setFilter", collectionFilterReq{ID: col.ID, Query: "s n<1"}).Body.Close()

	req := collectionPositionsReq{CollectionID: col.ID}
	var ids []int64
	resp = post(t, ts, "/v1/collections.positionIds", req)
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(ids) != 2 {
		t.Fatalf("positionIds = %v, want both positions (none was ever played)", ids)
	}
	var n int
	resp = post(t, ts, "/v1/collections.countPositions", req)
	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if n != 2 {
		t.Errorf("countPositions = %d, want 2", n)
	}
	var idx int
	resp = post(t, ts, "/v1/collections.indexOfPosition", collPositionReq{CollectionID: col.ID, PositionID: cubeID})
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if idx < 0 {
		t.Errorf("indexOfPosition = %d, want the rank of a held position", idx)
	}
	resp = post(t, ts, "/v1/collections.positions", req)
	defer resp.Body.Close()
	lines := 0
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("positions streamed %d rows, want 2", lines)
	}

	// Figer : l'appartenance devient les lignes, la requête s'efface.
	resp = post(t, ts, "/v1/collections.freeze", idReq(col))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("freeze status = %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = post(t, ts, "/v1/collections.countPositions", req)
	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if n != 2 {
		t.Errorf("frozen count = %d, want 2", n)
	}
	resp = post(t, ts, "/v1/collections.freeze", idReq(col))
	if resp.StatusCode == http.StatusOK {
		t.Error("freezing a hand-made collection must be refused")
	}
	resp.Body.Close()
}

// TestLivingCollectionEvaluateAndDeck: a collection created living in one
// call is read under its declared ceiling with the true total, a refused
// query is a client error, and a deck it feeds reports the evaluation next to
// ok, which a client reading only it relies on.
func TestLivingCollectionEvaluateAndDeck(t *testing.T) {
	ts := newTestServer(t)
	for i := range 3 {
		p := domain.InitializePosition()
		p.Board.Points[6].Checkers = i + 2
		post(t, ts, "/v1/positions.save", positionReq{Position: &p}).Body.Close()
	}

	resp := post(t, ts, "/v1/collections.create", collectionCreateReq{Name: "bad", FilterQuery: "s quux"})
	resp.Body.Close()
	if resp.StatusCode < 400 || resp.StatusCode >= 500 {
		t.Errorf("an unreadable query: status %d, want a client error", resp.StatusCode)
	}

	resp = post(t, ts, "/v1/collections.create", collectionCreateReq{Name: "living", FilterQuery: "s n<1"})
	var col idResp
	if err := json.NewDecoder(resp.Body).Decode(&col); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	var ev storage.CollectionEvaluation
	resp = post(t, ts, "/v1/collections.evaluate", collectionEvaluateReq{ID: col.ID, Limit: 2})
	if err := json.NewDecoder(resp.Body).Decode(&ev); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !ev.Living || len(ev.PositionIDs) != 2 || ev.Total != 3 || !ev.Truncated || ev.Cap != 2 {
		t.Errorf("evaluate = %+v; want living, 2 ids of 3, truncated at 2", ev)
	}

	resp = post(t, ts, "/v1/anki.createDeck", deckCreateReq{Name: "living", SourceType: domain.AnkiSourceCollection, SourceID: col.ID})
	var deck idResp
	if err := json.NewDecoder(resp.Body).Decode(&deck); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	var sync deckSyncResp
	resp = post(t, ts, "/v1/anki.sync", deckIDReq{DeckID: deck.ID})
	if err := json.NewDecoder(resp.Body).Decode(&sync); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !sync.OK || sync.Source == nil || sync.Source.Total != 3 || sync.Source.Truncated {
		t.Errorf("anki.sync = %+v; want ok with a source of 3, not truncated", sync)
	}

	// A counter opens the positions it counts: three new cards, three ids,
	// each at its rank; an unknown counter is the client's mistake.
	var n int
	resp = post(t, ts, "/v1/anki.filteredPositionCount", deckFilterReq{DeckID: deck.ID, Filter: "new"})
	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	var ids []int64
	resp = post(t, ts, "/v1/anki.filteredPositionIds", deckFilterReq{DeckID: deck.ID, Filter: "new"})
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if n != 3 || len(ids) != 3 {
		t.Fatalf("anki.filteredPosition* new = %d, %v; want 3 and 3 ids", n, ids)
	}
	var rank int
	resp = post(t, ts, "/v1/anki.indexOfFilteredPosition", deckFilterReq{DeckID: deck.ID, Filter: "new", PositionID: ids[2]})
	if err := json.NewDecoder(resp.Body).Decode(&rank); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if rank != 2 {
		t.Errorf("anki.indexOfFilteredPosition = %d; want 2", rank)
	}
	resp = post(t, ts, "/v1/anki.filteredPositionCount", deckFilterReq{DeckID: deck.ID, Filter: "overdue"})
	resp.Body.Close()
	if resp.StatusCode < 400 || resp.StatusCode >= 500 {
		t.Errorf("an unknown counter: status %d, want a client error", resp.StatusCode)
	}
}
