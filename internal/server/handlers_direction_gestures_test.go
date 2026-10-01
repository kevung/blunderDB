package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// gestureServer is a SQLite daemon started with --direction.
func gestureServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return gestureServerOn(t, st)
}

func gestureServerOn(t *testing.T, st storage.Storage) (*httptest.Server, *Server) {
	t.Helper()
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), EnableDirection: true})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

// gestureResult is what a test reads of a gesture's answer.
type gestureResult struct {
	status   int
	version  string
	replayed bool
	body     []byte
}

// send posts body to path as tenant, with If-Match and Idempotency-Key when given.
func send(t *testing.T, ts *httptest.Server, tenant, path, body, ifMatch, key string) gestureResult {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	if key != "" {
		req.Header.Set(IdempotencyKeyHeader, key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return gestureResult{}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return gestureResult{status: resp.StatusCode, version: resp.Header.Get(versionHeader),
		replayed: resp.Header.Get(idempotencyReplayedHeader) == "true", body: b}
}

// versionOf reads the Direction and returns the version its gestures state.
func (f directedFixture) versionOf(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	r := send(t, ts, f.tenant, "/v1/directions.get", f.body(""), "", "")
	if r.status != http.StatusOK || r.version == "" {
		t.Fatalf("directions.get: status %d, version %q, body %.200s", r.status, r.version, r.body)
	}
	return r.version
}

// body is the JSON naming the fixture's Direction, with extra fields appended.
func (f directedFixture) body(extra string) string {
	b := `{"tournamentId":` + strconv.FormatInt(f.tournamentID, 10)
	if extra != "" {
		b += "," + extra
	}
	return b + "}"
}

// running is the fixture's running matches, as the service sees them.
func (f directedFixture) running(t *testing.T) []*tournoi.Match {
	t.Helper()
	v, err := f.svc.GetDirection(f.ctx, f.tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	return v.Running
}

// resultBody enters m as won by its first player.
func (f directedFixture) resultBody(m *tournoi.Match) string {
	return f.body(`"matchId":"` + string(m.ID) + `","winner":"` + string(m.A) + `"`)
}

// TestDirectionGestures_AbsentWithoutFlag: without --direction a gesture is a route that does
// not exist; the reads stay.
func TestDirectionGestures_AbsentWithoutFlag(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	v := f.versionOf(t, ts)
	for _, path := range []string{"/v1/directions.enterResult", "/v1/directions.create", "/v1/rencontres.setBreaks", "/v1/rencontres.create"} {
		if r := send(t, ts, "1", path, f.body(""), v, ""); r.status != http.StatusNotFound {
			t.Errorf("%s without --direction: status %d; want 404", path, r.status)
		}
	}
	for _, p := range srv.Paths() {
		if strings.HasSuffix(p, ".enterResult") || strings.HasSuffix(p, ".setBreaks") {
			t.Errorf("Paths() lists %s without --direction", p)
		}
	}
}

// TestDirectionGestures_RequireIfMatch: a gesture states its version, or is refused 428 — and
// nothing is written.
func TestDirectionGestures_RequireIfMatch(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	m := f.running(t)[0]
	for _, h := range []string{"", "*", `W/"x"`} {
		r := send(t, ts, "1", "/v1/directions.enterResult", f.resultBody(m), h, "")
		if r.status != http.StatusPreconditionRequired || !strings.Contains(string(r.body), CodePreconditionRequired) {
			t.Errorf("If-Match %q: status %d, body %.200s; want 428", h, r.status, r.body)
		}
	}
	if got := len(f.running(t)); got != 2 {
		t.Errorf("%d matches running after refused gestures; want 2", got)
	}
}

// TestDirectionGestures_RaceOneConflict: two gestures decided on one reading, sent together —
// each valid on its own — and exactly one applies; the other gets 409 with the fresh state.
func TestDirectionGestures_RaceOneConflict(t *testing.T) {
	ts, srv := gestureServer(t)
	checkRaceOneConflict(t, ts, seedDirection(t, srv.opts.Storage, "1", "Open de Lyon"))
}

func checkRaceOneConflict(t *testing.T, ts *httptest.Server, f directedFixture) {
	t.Helper()
	run := f.running(t)
	if len(run) < 2 {
		t.Fatalf("%d running; the race needs two", len(run))
	}
	v := f.versionOf(t, ts)
	var wg sync.WaitGroup
	results := make([]gestureResult, 2)
	start := make(chan struct{})
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i] = send(t, ts, f.tenant, "/v1/directions.enterResult", f.resultBody(run[i]), v, "")
		}()
	}
	close(start)
	wg.Wait()

	var ok, conflict int
	for _, r := range results {
		switch r.status {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
			var env struct {
				Error struct {
					Code    string `json:"code"`
					Details struct {
						Version   string                 `json:"version"`
						Direction *service.DirectionView `json:"direction"`
					} `json:"details"`
				} `json:"error"`
			}
			if err := json.Unmarshal(r.body, &env); err != nil || env.Error.Code != CodeConflict ||
				env.Error.Details.Direction == nil || env.Error.Details.Version == "" {
				t.Errorf("409 without the fresh state: %v, %.300s", err, r.body)
			}
			if r.version != quoteVersion(env.Error.Details.Version) {
				t.Errorf("409 header version %q, details %q", r.version, env.Error.Details.Version)
			}
		default:
			t.Errorf("status %d, body %.200s", r.status, r.body)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("%d applied, %d refused; want exactly one of each", ok, conflict)
	}
	if got := len(f.running(t)); got != 1 {
		t.Errorf("%d running after the race; want 1", got)
	}
}

// TestDirectionGestures_VersionChain: a gesture answers with the version it left, which is the
// next read's, and the next gesture goes through on it; the old one is stale.
func TestDirectionGestures_VersionChain(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	v0 := f.versionOf(t, ts)
	r := send(t, ts, "1", "/v1/directions.addNote", f.body(`"text":"pause café"`), v0, "")
	if r.status != http.StatusOK || r.version == "" || r.version == v0 {
		t.Fatalf("addNote: status %d, version %q (was %q), body %.200s", r.status, r.version, v0, r.body)
	}
	if got := f.versionOf(t, ts); got != r.version {
		t.Errorf("read after the gesture: version %q; the gesture answered %q", got, r.version)
	}
	if r2 := send(t, ts, "1", "/v1/directions.addNote", f.body(`"text":"reprise"`), r.version, ""); r2.status != http.StatusOK {
		t.Errorf("gesture on the answered version: status %d, body %.200s", r2.status, r2.body)
	}
	if r3 := send(t, ts, "1", "/v1/directions.addNote", f.body(`"text":"trop tard"`), v0, ""); r3.status != http.StatusConflict {
		t.Errorf("gesture on the first version: status %d; want 409", r3.status)
	}
	// A room gesture states the Rencontre's version — the same token its members carry.
	rb := `{"id":` + strconv.FormatInt(f.rencontreID, 10) + `,"table":4,"out":true}`
	rv := send(t, ts, "1", "/v1/rencontres.get", `{"id":`+strconv.FormatInt(f.rencontreID, 10)+`}`, "", "").version
	if rv != f.versionOf(t, ts) {
		t.Errorf("the room's version %q differs from its member's", rv)
	}
	if r4 := send(t, ts, "1", "/v1/rencontres.setTableOutOfService", rb, rv, ""); r4.status != http.StatusOK || r4.version == rv {
		t.Errorf("setTableOutOfService: status %d, version %q, body %.200s", r4.status, r4.version, r4.body)
	}
}

// TestDirectionGestures_Idempotent: a gesture resent with its Idempotency-Key — a lost answer,
// a double click — applies once, and the second answer is the first, replayed.
func TestDirectionGestures_Idempotent(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	m := f.running(t)[0]
	v := f.versionOf(t, ts)
	first := send(t, ts, "1", "/v1/directions.enterResult", f.resultBody(m), v, "k-1")
	again := send(t, ts, "1", "/v1/directions.enterResult", f.resultBody(m), v, "k-1")
	if first.status != http.StatusOK || again.status != http.StatusOK || !again.replayed || !bytes.Equal(first.body, again.body) {
		t.Fatalf("first %d, again %d (replayed %v); want the same 200 twice", first.status, again.status, again.replayed)
	}
	hist, err := f.svc.History(f.ctx, f.tournamentID, "", string(m.ID))
	if err != nil {
		t.Fatal(err)
	}
	results := 0
	for _, h := range hist {
		if strings.Contains(h.Kind, "result") {
			results++
		}
	}
	if results != 1 {
		t.Errorf("%d results recorded for %s; want 1 (history %+v)", results, m.ID, hist)
	}
	// The creations take no version, and the key alone keeps them from creating twice.
	c1 := send(t, ts, "1", "/v1/rencontres.create", `{"name":"Annexe","tables":2}`, "", "k-2")
	c2 := send(t, ts, "1", "/v1/rencontres.create", `{"name":"Annexe","tables":2}`, "", "k-2")
	if c1.status != http.StatusOK || !c2.replayed {
		t.Errorf("rencontres.create: %d then replayed=%v", c1.status, c2.replayed)
	}
	if list, _ := f.svc.ListRencontres(f.ctx); len(list) != 2 {
		t.Errorf("%d Rencontres; want 2 (the fixture's and one Annexe)", len(list))
	}
}

// TestDirectionGestures_SingleTenantSQLite: a SQLite daemon has one tenant; no other reaches
// its Directions.
func TestDirectionGestures_SingleTenantSQLite(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, SingleTenant: true, EnableDirection: true})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	f := seedDirection(t, st, "1", "Open de Lyon")
	v := f.versionOf(t, ts)
	if r := send(t, ts, "2", "/v1/directions.addNote", f.body(`"text":"intrus"`), v, ""); r.status != http.StatusBadRequest {
		t.Errorf("tenant 2 on a single-tenant daemon: status %d; want 400", r.status)
	}
}

// TestDirectionGestures_WritePages: the display page is the service's business — a gesture
// sent to the daemon rewrites the page in the folder the database names.
func TestDirectionGestures_WritePages(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	dir := t.TempDir()
	if _, err := f.svc.SetRencontreOutputDir(f.ctx, f.rencontreID, dir); err != nil {
		t.Fatal(err)
	}
	// What setting the folder wrote goes: only the gesture's own writing may remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
	m := f.running(t)[0]
	if r := send(t, ts, "1", "/v1/directions.enterResult", f.resultBody(m), f.versionOf(t, ts), ""); r.status != http.StatusOK {
		t.Fatalf("enterResult: %d %.200s", r.status, r.body)
	}
	if pages := htmlUnder(t, dir); len(pages) < 2 {
		t.Errorf("after a gesture, %v under %s; want the event's page and the room's", pages, dir)
	}
	// The gesture's answer never names a path of the server's disk.
	r := send(t, ts, "1", "/v1/directions.addNote", f.body(`"text":"x"`), f.versionOf(t, ts), "")
	if strings.Contains(string(r.body), dir) {
		t.Errorf("a gesture's answer names the server's folder: %.300s", r.body)
	}
}

// TestRunCallDirectionGesture: `call` serves the gestures, with --if-match, as the daemon does.
func TestRunCallDirectionGesture(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "call.db")
	st, err := sqlite.Open(context.Background(), dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := seedDirection(t, st, "1", "Open de Lyon")
	v, err := f.svc.DirectionVersion(f.ctx, f.tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	note := f.body(`"text":"par call"`)
	if _, err := captureStdout(t, func() error {
		return RunCall([]string{"directions.addNote", "--db", dbPath, "--json", note})
	}); err == nil {
		t.Error("call without --if-match succeeded; want 428")
	}
	out, err := captureStdout(t, func() error {
		return RunCall([]string{"directions.addNote", "--db", dbPath, "--if-match", v, "--json", note})
	})
	if err != nil || !strings.Contains(out, `"tournamentId"`) {
		t.Fatalf("call directions.addNote: %v (out=%.300s)", err, out)
	}
}

// htmlUnder lists the HTML files under dir.
func htmlUnder(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".html") {
			out = append(out, p)
		}
		return nil
	})
	return out
}
