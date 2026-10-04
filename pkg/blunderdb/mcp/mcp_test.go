package mcp_test

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/mcp"
)

// demoServer opens a private copy of the demo database (fictional players,
// three analysed matches, collections, a study deck) behind the /v1 engine.
func demoServer(t *testing.T, o internalserver.Options) *internalserver.Server {
	t.Helper()
	src, err := os.Open(filepath.Join("..", "..", "..", "internal", "gui", "demo.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	zr, err := gzip.NewReader(src)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "demo.db")
	dst, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, zr); err != nil {
		t.Fatal(err)
	}
	dst.Close()

	ctx := context.Background()
	st, err := internalserver.OpenStorage(ctx, "sqlite", path, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	o.Storage = st
	o.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := internalserver.New(o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.Close() })
	return srv
}

// connect wires a client to the MCP server over an in-memory transport.
func connect(t *testing.T, engine http.Handler, opts mcp.Options) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	ct, st := sdk.NewInMemoryTransports()
	if _, err := mcp.NewServer(engine, opts).Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

type obj = map[string]any

func call(t *testing.T, cs *sdk.ClientSession, name string, args obj) obj {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("%s: tool error: %s", name, text(res))
	}
	var out obj
	raw, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: structured content: %v", name, err)
	}
	return out
}

func callErr(t *testing.T, cs *sdk.ClientSession, name string, args obj) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err.Error()
	}
	if !res.IsError {
		t.Fatalf("%s: want a tool error, got %s", name, text(res))
	}
	return text(res)
}

func text(res *sdk.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func list(t *testing.T, o obj, key string) []any {
	t.Helper()
	s, ok := o[key].([]any)
	if !ok || len(s) == 0 {
		t.Fatalf("%s: empty or missing in %v", key, o)
	}
	return s
}

func id(t *testing.T, v any, key string) float64 {
	t.Helper()
	n, ok := v.(obj)[key].(float64)
	if !ok || n == 0 {
		t.Fatalf("%s missing in %v", key, v)
	}
	return n
}

func toolNames(t *testing.T, cs *sdk.ClientSession) []string {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		if tool.Description == "" {
			t.Errorf("%s has no description", tool.Name)
		}
	}
	sort.Strings(names)
	return names
}

var writeTools = []string{"add_to_collection", "anki_review", "comment_position", "create_collection", "save_position"}

func TestReadOnlyByDefault(t *testing.T) {
	cs := connect(t, demoServer(t, internalserver.Options{}).Handler(), mcp.Options{Tenant: "1"})
	names := toolNames(t, cs)
	for _, w := range writeTools {
		for _, n := range names {
			if n == w {
				t.Errorf("write tool %s offered without Write", w)
			}
		}
	}
	if len(names) < 18 {
		t.Errorf("only %d read tools: %v", len(names), names)
	}
}

func TestReadTools(t *testing.T) {
	cs := connect(t, demoServer(t, internalserver.Options{}).Handler(), mcp.Options{Tenant: "1"})

	ov := call(t, cs, "database_overview", nil)
	if c := ov["counts"].(obj); c["positions"].(float64) == 0 {
		t.Fatalf("overview counts: %v", c)
	}
	list(t, ov, "topPlayers")

	found := call(t, cs, "search_positions", obj{"query": "s E>100", "limit": 5})
	positions := list(t, found, "positions")
	if len(positions) > 5 {
		t.Errorf("limit ignored: %d rows", len(positions))
	}
	if found["canonical"] == "" {
		t.Error("no canonical form")
	}
	pid := id(t, positions[0], "id")
	if !strings.Contains(positions[0].(obj)["xgid"].(string), ":") {
		t.Errorf("summary xgid: %v", positions[0])
	}
	if msg := callErr(t, cs, "search_positions", obj{"query": "s qqq"}); !strings.Contains(msg, "qqq") {
		t.Errorf("unknown token not named: %s", msg)
	}

	pos := call(t, cs, "get_position", obj{"positionId": pid})
	if pos["analysis"] == nil {
		t.Fatalf("get_position: no analysis for a searched blunder: %v", pos)
	}
	ex := call(t, cs, "explain_error", obj{"positionId": pid})
	if ex["played"] == "" || ex["best"] == nil {
		t.Errorf("explain_error: %v", ex)
	}
	list(t, call(t, cs, "similar_positions", obj{"positionId": pid, "limit": 3}), "similar")

	xgid := positions[0].(obj)["xgid"].(string)
	dec := call(t, cs, "decode_position", obj{"text": "XGID=" + xgid})
	if dec["position"].(obj)["xgid"] != xgid {
		t.Errorf("decode_position round trip: %v, want %s", dec["position"], xgid)
	}
	checker := call(t, cs, "search_positions", obj{"query": "s E>50 d", "limit": 50})
	var checkerID float64
	for _, p := range list(t, checker, "positions") {
		if p.(obj)["decision"] == "checker" {
			checkerID = id(t, p, "id")
			break
		}
	}
	if checkerID == 0 {
		t.Fatal("no checker decision in the demo blunders")
	}
	list(t, call(t, cs, "legal_moves", obj{"positionId": checkerID}), "plays")
	if call(t, cs, "race_epc", obj{"text": "XGID=" + xgid})["epc"] == nil {
		t.Error("race_epc: no epc")
	}

	players := list(t, call(t, cs, "list_players", nil), "players")
	name := players[0].(obj)["Name"].(string)
	if al := call(t, cs, "player_aliases", nil); al["aliases"] == nil || al["suggestions"] == nil {
		t.Errorf("player_aliases: %v", al)
	}
	st := call(t, cs, "player_stats", obj{"player": name})
	if st["Totals"].(obj)["NumDecisions"].(float64) == 0 {
		t.Errorf("player_stats: %v", st["Totals"])
	}
	other := players[1].(obj)["Name"].(string)
	if h := call(t, cs, "head_to_head", obj{"player": name, "playerB": other}); h["player_a"] != name || h["player_b"] != other {
		t.Errorf("head_to_head: %v", h)
	}
	list(t, call(t, cs, "pr_by_window", obj{"player": name, "months": 12}), "windows")
	list(t, call(t, cs, "player_ranking", obj{"minDecisions": 1}), "ranking")
	recurring := call(t, cs, "recurring_errors", obj{"player": name, "limit": 3})
	list(t, recurring, "TopBlunders")
	if groups, _ := recurring["Groups"].([]any); len(groups) > 3 {
		t.Errorf("recurring_errors: %d groups, limit is 3", len(groups))
	} else if len(groups) > 0 {
		g0 := groups[0].(obj)
		if n, _ := g0["Positions"].(float64); int(n) < len(g0["PositionIDs"].([]any)) {
			t.Errorf("recurring_errors: Positions = %v, fewer than the ids returned", g0["Positions"])
		}
		if _, ok := g0["Truncated"].(bool); !ok {
			t.Errorf("recurring_errors: Truncated missing: %v", g0)
		}
		if ids, _ := groups[0].(obj)["PositionIDs"].([]any); len(ids) == 0 || len(ids) > 50 {
			t.Errorf("recurring_errors: group ids = %d, want 1..%d", len(ids), 50)
		}
	}

	training := call(t, cs, "training_stats", obj{"player": name, "window": "month"})
	if training["Window"] != "month" {
		t.Errorf("training_stats: window = %v, want month", training["Window"])
	}

	matches := list(t, call(t, cs, "list_matches", obj{"player": name}), "matches")
	mid := id(t, matches[0], "id")
	if call(t, cs, "get_match", obj{"matchId": mid})["performance"] == nil {
		t.Error("get_match: no performance")
	}
	list(t, call(t, cs, "list_tournaments", nil), "tournaments")

	colls := list(t, call(t, cs, "list_collections", nil), "collections")
	list(t, call(t, cs, "collection_positions", obj{"collectionId": id(t, colls[0], "id")}), "positions")
	list(t, call(t, cs, "study_decks", nil), "decks")
	list(t, call(t, cs, "search_comments", obj{"query": "cube"}), "comments")
	if _, ok := call(t, cs, "saved_searches", nil)["searches"]; !ok {
		t.Error("saved_searches: no searches key")
	}

	drawn := call(t, cs, "quiz_draw", obj{"query": "s E>50 d"})["position"].(obj)
	if _, leaked := drawn["analysis"]; leaked {
		t.Error("quiz_draw leaks the analysis")
	}
	answer := "nd"
	if drawn["decision"] == "checker" {
		moves := call(t, cs, "get_position", obj{"positionId": drawn["id"]})["analysis"].(obj)["checkerMoves"].([]any)
		answer = moves[0].(obj)["move"].(string)
	}
	if v := call(t, cs, "quiz_grade", obj{"positionId": drawn["id"], "answer": answer}); len(v) == 0 {
		t.Error("quiz_grade: empty verdict")
	}
}

func TestWriteTools(t *testing.T) {
	cs := connect(t, demoServer(t, internalserver.Options{}).Handler(), mcp.Options{Tenant: "1", Write: true})
	names := toolNames(t, cs)
	for _, w := range writeTools {
		if !slices.Contains(names, w) {
			t.Errorf("write tool %s missing with Write", w)
		}
	}

	saved := call(t, cs, "save_position", obj{"text": "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10"})
	pid := saved["positionId"]
	if pid == nil {
		t.Fatalf("save_position: no id in %v", saved)
	}
	again := call(t, cs, "save_position", obj{"text": "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10"})
	if again["positionId"] != pid || again["created"] != false {
		t.Errorf("second save duplicated: %v then %v", saved, again)
	}
	if got := call(t, cs, "search_positions", obj{"query": "s i"}); len(list(t, got, "positions")) == 0 {
		t.Error("the saved position is not flagged individually imported")
	}

	call(t, cs, "comment_position", obj{"positionId": pid, "text": "un coup à revoir #mcp"})
	if c := call(t, cs, "get_position", obj{"positionId": pid})["comment"]; !strings.Contains(c.(string), "#mcp") {
		t.Errorf("comment not stored: %v", c)
	}
	call(t, cs, "comment_position", obj{"positionId": pid, "text": "deuxième avis", "author": "Alice"})
	thread := list(t, call(t, cs, "position_comments", obj{"positionId": pid}), "comments")
	signed := false
	for _, c := range thread {
		if c.(obj)["author"] == "Alice" {
			signed = true
		}
	}
	if len(thread) != 2 || !signed {
		t.Errorf("thread = %v, want two comments, one signed Alice", thread)
	}
	coll := call(t, cs, "create_collection", obj{"name": "Depuis MCP"})
	cid := coll["id"]
	call(t, cs, "add_to_collection", obj{"collectionId": cid, "positionIds": []any{pid}})
	got := list(t, call(t, cs, "collection_positions", obj{"collectionId": cid}), "positions")
	if len(got) != 1 || got[0].(obj)["id"] != pid {
		t.Errorf("collection holds %v, want [%v]", got, pid)
	}
}

// tenantSpy records the X-Tenant-ID of every /v1 call the tools make.
type tenantSpy struct {
	mu      sync.Mutex
	next    http.Handler
	tenants map[string]bool
}

func (s *tenantSpy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.tenants[r.Header.Get(mcp.TenantHeader)] = true
	s.mu.Unlock()
	s.next.ServeHTTP(w, r)
}

type headerTransport struct{ tenant string }

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if h.tenant != "" {
		r.Header.Set(mcp.TenantHeader, h.tenant)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func httpClient(t *testing.T, url, tenant string) (*sdk.ClientSession, error) {
	t.Helper()
	tr := &sdk.StreamableClientTransport{Endpoint: url, HTTPClient: &http.Client{Transport: headerTransport{tenant}}}
	return sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(context.Background(), tr, nil)
}

// Over HTTP each tool call runs as the tenant its request names, never a
// default: the header reaches every /v1 call the tool makes.
func TestHTTPToolsRunAsTheRequestTenant(t *testing.T) {
	srv := demoServer(t, internalserver.Options{})
	spy := &tenantSpy{next: srv.Handler(), tenants: map[string]bool{}}
	ts := httptest.NewServer(mcp.NewHTTPHandler(spy, mcp.Options{AllowRemoteHost: true}))
	defer ts.Close()

	cs, err := httpClient(t, ts.URL, "7")
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	call(t, cs, "database_overview", nil)
	call(t, cs, "search_positions", obj{"query": "s E>100", "limit": 2})
	if len(spy.tenants) != 1 || !spy.tenants["7"] {
		t.Errorf("tools called /v1 as %v, want only tenant 7", spy.tenants)
	}

	anon, err := httpClient(t, ts.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	defer anon.Close()
	if msg := callErr(t, anon, "database_overview", nil); !strings.Contains(msg, "tenant") {
		t.Errorf("a call without tenant: %s", msg)
	}
}

// The daemon mounts /mcp behind its tenant gate, like /v1.
func TestDaemonMountsMCPBehindTheTenantGate(t *testing.T) {
	ts := httptest.NewServer(demoServer(t, internalserver.Options{}).Handler())
	defer ts.Close()

	if _, err := httpClient(t, ts.URL+"/mcp", ""); err == nil {
		t.Error("/mcp answered a client with no X-Tenant-ID")
	}
	cs, err := httpClient(t, ts.URL+"/mcp", "1")
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	for _, n := range toolNames(t, cs) {
		for _, w := range writeTools {
			if n == w {
				t.Errorf("daemon offers write tool %s without MCPWrite", w)
			}
		}
	}
	list(t, call(t, cs, "search_positions", obj{"query": "s E>100", "limit": 2}), "positions")

	wts := httptest.NewServer(demoServer(t, internalserver.Options{MCPWrite: true}).Handler())
	defer wts.Close()
	wcs, err := httpClient(t, wts.URL+"/mcp", "1")
	if err != nil {
		t.Fatal(err)
	}
	defer wcs.Close()
	if names := toolNames(t, wcs); !slices.Contains(names, "save_position") {
		t.Error("MCPWrite does not offer the write tools")
	}
}

// Without AllowRemoteHost a loopback listener refuses a foreign Host: the guard
// a GUI-hosted server relies on against DNS rebinding.
func TestRebindingGuardIsOnByDefault(t *testing.T) {
	srv := demoServer(t, internalserver.Options{})
	for _, allow := range []bool{false, true} {
		ts := httptest.NewServer(mcp.NewHTTPHandler(srv.Handler(), mcp.Options{Tenant: "1", AllowRemoteHost: allow}))
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
		req, _ := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader(body))
		req.Host = "evil.example"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		ts.Close()
		if forbidden := resp.StatusCode == http.StatusForbidden; forbidden == allow {
			t.Errorf("AllowRemoteHost=%v: status %d", allow, resp.StatusCode)
		}
	}
}

// rollout reads by default — no store argument offered — and writes beside
// the analysis only on a server that writes.
func TestRolloutTool(t *testing.T) {
	const tiny = "fast,games=36,min-games=36,truncation=2,candidates=2"
	const pos = "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10"

	rw := connect(t, demoServer(t, internalserver.Options{}).Handler(), mcp.Options{Tenant: "1", Write: true})
	pid := call(t, rw, "save_position", obj{"text": pos})["positionId"]
	if got := call(t, rw, "rollout", obj{"positionId": pid, "rollout": tiny, "store": true}); got["stored"] != true {
		t.Errorf("rollout with store: %v", got)
	}

	ro := connect(t, demoServer(t, internalserver.Options{}).Handler(), mcp.Options{Tenant: "1"})
	checker := list(t, call(t, ro, "search_positions", obj{"query": "s E>50 d", "limit": 1}), "positions")
	res := call(t, ro, "rollout", obj{"positionId": id(t, checker[0], "id"), "rollout": tiny})
	if res["stored"] != false || res["result"] == nil {
		t.Errorf("read-only rollout: %v", res)
	}
	if msg := callErr(t, ro, "rollout", obj{"positionId": id(t, checker[0], "id"), "rollout": tiny, "store": true}); msg == "" {
		t.Error("a read-only server accepted store")
	}
}

// TestClubTools: the evaluation, review, transcription and direction tools
// answer on the demo database through their /v1 routes.
func TestClubTools(t *testing.T) {
	cs := connect(t, demoServer(t, internalserver.Options{}).Handler(), mcp.Options{Tenant: "1"})

	ev := call(t, cs, "evaluate", obj{"text": "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10", "ply": 0, "candidates": 3})
	if ev["decision"] != "checker" || len(list(t, ev, "moves")) == 0 || len(list(t, ev, "moves")) > 3 {
		t.Errorf("evaluate: %v", ev)
	}
	// An absent ply is the documented default, 2, not a 0-ply search.
	def := call(t, cs, "evaluate", obj{"text": "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10", "candidates": 1})
	if def["depth"] != "2-ply" {
		t.Errorf("evaluate without ply searches at %v; want 2-ply", def["depth"])
	}
	if msg := callErr(t, cs, "evaluate", obj{"text": ""}); !strings.Contains(msg, "XGID") {
		t.Errorf("evaluate without text: %s", msg)
	}
	for tool, key := range map[string]string{"transcribe_list": "transcriptions", "direction_list": "directions", "direction_season": "rows"} {
		if _, ok := call(t, cs, tool, nil)[key]; !ok {
			t.Errorf("%s answers no %s", tool, key)
		}
	}
	decks := list(t, call(t, cs, "study_decks", nil), "decks")
	if len(decks) > 0 {
		call(t, cs, "anki_next", obj{"deckId": id(t, decks[0], "id")})
	}
	// Nothing is due in a deck that does not exist: the tool says so with a null card, as its
	// description promises, rather than failing.
	if c, ok := call(t, cs, "anki_next", obj{"deckId": 987654})["card"]; !ok || c != nil {
		t.Errorf("anki_next with nothing due: card = %v", c)
	}
}

// readTenantsSpy records, per /v1 path, the X-Read-Tenants the tools sent.
type readTenantsSpy struct {
	mu   sync.Mutex
	next http.Handler
	sent map[string][]string
}

func (s *readTenantsSpy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.sent[r.URL.Path] = append(s.sent[r.URL.Path], r.Header.Values(mcp.ReadTenantsHeader)...)
	s.mu.Unlock()
	s.next.ServeHTTP(w, r)
}

type readTenantsTransport struct{ tenant, read string }

func (h readTenantsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set(mcp.TenantHeader, h.tenant)
	r.Header.Set(mcp.ReadTenantsHeader, h.read)
	return http.DefaultTransport.RoundTrip(r)
}

// TestAcrossTools: the club tools answer on the demo database, and the read
// set the proxy put on /mcp reaches their across.* calls and no other route.
func TestAcrossTools(t *testing.T) {
	srv := demoServer(t, internalserver.Options{TrustReadTenants: true})
	spy := &readTenantsSpy{next: srv.Handler(), sent: map[string][]string{}}
	ts := httptest.NewServer(mcp.NewHTTPHandler(spy, mcp.Options{AllowRemoteHost: true}))
	defer ts.Close()
	tr := &sdk.StreamableClientTransport{Endpoint: ts.URL, HTTPClient: &http.Client{Transport: readTenantsTransport{"1", "2"}}}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(context.Background(), tr, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	matches := list(t, call(t, cs, "club_matches", obj{"limit": 2}), "matches")
	tenants := map[any]bool{}
	for _, m := range matches {
		tenants[m.(obj)["tenant"]] = true
	}
	if !tenants["1"] || !tenants["2"] {
		t.Errorf("club_matches read tenants %v, want 1 and 2", tenants)
	}
	first := matches[0].(obj)
	mid := first["match"].(obj)["id"]
	pos := list(t, call(t, cs, "club_match_positions", obj{"tenant": "2", "matchId": mid, "limit": 3}), "positions")
	if len(pos) > 3 {
		t.Errorf("club_match_positions limit 3 gave %d positions", len(pos))
	}
	z, ok := pos[0].(obj)["zobrist"].(string)
	if !ok || z == "" {
		t.Fatalf("club_match_positions gives no zobrist string: %v", pos[0])
	}
	if cm := call(t, cs, "club_comments", obj{"zobrists": []string{z}}); cm["comments"] == nil || cm["truncated"] != false {
		t.Errorf("club_comments = %v, want comments and truncated false", cm)
	}
	if msg := callErr(t, cs, "club_comments", obj{"zobrists": []string{"nope"}}); !strings.Contains(msg, "zobrist") {
		t.Errorf("club_comments with a bad hash: %s", msg)
	}
	if _, ok := call(t, cs, "club_library", nil)["collections"]; !ok {
		t.Error("club_library answers no collections")
	}
	if rk := call(t, cs, "club_ranking", obj{"minDecisions": 1}); rk["rows"] == nil || rk["total"] == nil || rk["truncated"] == nil {
		t.Errorf("club_ranking = %v, want rows, total and truncated", rk)
	}
	call(t, cs, "database_overview", nil)

	spy.mu.Lock()
	defer spy.mu.Unlock()
	for path, sent := range spy.sent {
		across := strings.HasPrefix(path, "/v1/across.")
		if across && !slices.Equal(sent, []string{"2"}) {
			t.Errorf("%s received X-Read-Tenants %v, want [2]", path, sent)
		}
		if !across && len(sent) != 0 {
			t.Errorf("%s received X-Read-Tenants %v; only across.* calls carry it", path, sent)
		}
	}
}

// A missed Decision question keeps its position: training_missed hands it
// back for another drill, and a correct answer stays out.
func TestTrainingMissedTool(t *testing.T) {
	srv := demoServer(t, internalserver.Options{})
	engine := srv.Handler()
	cs := connect(t, engine, mcp.Options{Tenant: "1"})
	pid := int64(id(t, list(t, call(t, cs, "search_positions", obj{"query": "s E>100", "limit": 2}), "positions")[0], "id"))
	other := pid + 1
	body := fmt.Sprintf(`{"exercise":"decision","numbersAsked":2,"faults":1,"items":[
		{"numberType":"decision.checker","wrong":true,"positionId":%d,"answer":"13/7 8/7","errorMp":120},
		{"numberType":"decision.checker","wrong":false,"positionId":%d,"answer":"24/18","errorMp":0}]}`, pid, other)
	req := httptest.NewRequest(http.MethodPost, "/v1/training.save", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "1")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("training.save: %d %s", rec.Code, rec.Body.String())
	}
	got := list(t, call(t, cs, "training_missed", nil), "PositionIDs")
	if len(got) != 1 || got[0].(float64) != float64(pid) {
		t.Fatalf("training_missed = %v, want [%d]", got, pid)
	}
}
