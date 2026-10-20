package gui

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// recordingEngine stands for the engine: it records what reached it.
type recordingEngine struct {
	calls   int
	tenant  string
	readSet string
	body    string
}

func (e *recordingEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.calls++
	e.tenant = r.Header.Get("X-Tenant-ID")
	e.readSet = r.Header.Get("X-Read-Tenants")
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)
	e.body = buf.String()
	w.WriteHeader(http.StatusOK)
}

func serveDisplay(h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestDisplayHandler_ServesOnlyTheNamedTournamentPage(t *testing.T) {
	eng := &recordingEngine{}
	h := displayHandler(eng, 42)

	rec := serveDisplay(h, http.MethodPost, displayRoute, `{"tournamentId":42}`,
		map[string]string{"X-Tenant-ID": "7", "X-Read-Tenants": "1,7"})
	if rec.Code != http.StatusOK || eng.calls != 1 {
		t.Fatalf("display route: code=%d calls=%d", rec.Code, eng.calls)
	}
	if eng.tenant != "1" || eng.readSet != "" {
		t.Errorf("tenant must be set by the host, not the visitor: tenant=%q readSet=%q", eng.tenant, eng.readSet)
	}
	if eng.body != `{"tournamentId":42}` {
		t.Errorf("engine must receive the request body unchanged, got %q", eng.body)
	}

	refused := []struct{ method, path, body string }{
		{http.MethodPost, displayRoute, `{"tournamentId":43}`},
		{http.MethodPost, displayRoute, `not json`},
		{http.MethodPost, displayRoute, `{"tournamentId":42,"pad":"` + strings.Repeat("x", displayBodyLimit) + `"}`},
		{http.MethodPost, "/v1/directions.list", `{}`},
		{http.MethodPost, "/v1/positions.delete", `{"id":1}`},
		{http.MethodPost, "/v1/directions.recordResult", `{"tournamentId":42}`},
		{http.MethodGet, "/ops/vacuum", ``},
		{http.MethodPost, "/mcp", `{}`},
	}
	for _, c := range refused {
		before := eng.calls
		rec := serveDisplay(h, c.method, c.path, c.body, nil)
		if rec.Code < 400 || eng.calls != before {
			t.Errorf("%s %s (%.30s): code=%d, engine reached=%v", c.method, c.path, c.body, rec.Code, eng.calls != before)
		}
	}
}

func TestDisplayHandler_ServesThePageAndRedirectsTheRoot(t *testing.T) {
	h := displayHandler(&recordingEngine{}, 42)
	if rec := serveDisplay(h, http.MethodGet, "/app/", "", nil); rec.Code != http.StatusOK ||
		!strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("GET /app/: code=%d type=%q", rec.Code, rec.Header().Get("Content-Type"))
	}
	rec := serveDisplay(h, http.MethodGet, "/", "", nil)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/app/?tournoi=42" {
		t.Errorf("GET /: code=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestQRDataURI_IsAPNG(t *testing.T) {
	uri, err := qrDataURI("http://192.168.1.10:8770/app/?tournoi=42")
	if err != nil || !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatalf("qrDataURI: %v %.40q", err, uri)
	}
}

func TestStartTournamentDisplay_OffUntilAskedAndNeedsATournament(t *testing.T) {
	a := &App{}
	if st := a.GetTournamentDisplayStatus(); st.Running {
		t.Fatal("the display must be off by default")
	}
	if st := a.StartTournamentDisplay(TournamentDisplayConfig{}); st.Running || st.Error == "" {
		t.Errorf("no tournament named: %+v", st)
	}
}

// Over the real engine and the demo file: the page of a directed tournament comes back with an
// ETag, and the same ETag sent back costs a 304 — what the phones poll on.
func TestDisplayHandler_ServesTheDemoTournamentConditionally(t *testing.T) {
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(a.stopDisplay)
	path, err := a.PrepareDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	if err := db.OpenDatabase(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	list, err := db.ListDirections()
	if err != nil || len(list) == 0 {
		t.Fatalf("the demo file must hold a Direction: %v %d", err, len(list))
	}
	id := list[0].TournamentID
	h := displayHandler(&a.display.engine, id)
	body := fmt.Sprintf(`{"tournamentId":%d}`, id)

	first := serveDisplay(h, http.MethodPost, displayRoute, body, map[string]string{"Content-Type": "application/json"})
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"html"`) {
		t.Fatalf("page: code=%d body=%.200s", first.Code, first.Body.String())
	}
	tag := first.Header().Get("ETag")
	if tag == "" {
		t.Fatal("the page must carry an ETag")
	}
	again := serveDisplay(h, http.MethodPost, displayRoute, body, map[string]string{"Content-Type": "application/json", "If-None-Match": tag})
	if again.Code != http.StatusNotModified {
		t.Fatalf("unchanged page: code=%d, want 304", again.Code)
	}

	// A Direction write — naming a table, which the page shows — moves the tag: the phones
	// get the new page on their next poll.
	tables := []domain.TableSetting{{Number: 1, Name: "Terrasse", AssignedTo: []string{}}}
	view, err := db.GetDirection(id)
	if err != nil {
		t.Fatal(err)
	}
	if rid := view.RencontreID; rid != 0 {
		_, err = db.SetRencontreTables(rid, tables)
	} else {
		_, err = db.SetDirectionTables(id, tables)
	}
	if err != nil {
		t.Fatal(err)
	}
	changed := serveDisplay(h, http.MethodPost, displayRoute, body, map[string]string{"Content-Type": "application/json", "If-None-Match": tag})
	if changed.Code != http.StatusOK || changed.Header().Get("ETag") == tag {
		t.Errorf("after a write: code=%d etag changed=%v, want 200 and a new tag", changed.Code, changed.Header().Get("ETag") != tag)
	}
}

// openDemo opens a fresh copy of the demo file in db.
func openDemo(t *testing.T, a *App, db *database.Database) {
	t.Helper()
	path, err := a.PrepareDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	if err := db.OpenDatabase(path); err != nil {
		t.Fatal(err)
	}
}

// The display shows one file's tournament: opening another file stops it, and the port is
// released, rather than serving the same id from the new file.
func TestTournamentDisplay_StopsWhenTheOpenFileChanges(t *testing.T) {
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(a.stopDisplay)
	t.Cleanup(func() { _ = db.Close() })
	openDemo(t, a, db)
	list, err := db.ListDirections()
	if err != nil || len(list) == 0 {
		t.Fatalf("the demo file must hold a Direction: %v", err)
	}
	port := freePort(t)
	st := a.StartTournamentDisplay(TournamentDisplayConfig{TournamentID: list[0].TournamentID, Port: port})
	if !st.Running {
		t.Fatalf("display did not start: %s", st.Error)
	}
	openDemo(t, a, db)
	if st := a.GetTournamentDisplayStatus(); st.Running {
		t.Fatal("the display must stop when another file is opened")
	}
	if conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port))); err == nil {
		_ = conn.Close()
		t.Error("the port must be released")
	}
}

func TestChooseAddress_PrefersTheAskedThenTheFirst(t *testing.T) {
	addrs := []string{"192.168.1.10", "172.17.0.1"}
	if got := chooseAddress(addrs, "172.17.0.1"); got != "172.17.0.1" {
		t.Errorf("asked address: %q", got)
	}
	if got := chooseAddress(addrs, "10.0.0.9"); got != "192.168.1.10" {
		t.Errorf("unknown address falls back to the first: %q", got)
	}
	if got := chooseAddress(nil, ""); got != "" {
		t.Errorf("no network: %q", got)
	}
	for name, want := range map[string]bool{"docker0": true, "br-1a2b": true, "virbr0": true, "tun0": true, "wg0": true, "eth0": false, "wlp3s0": false, "enp0s31f6": false} {
		if isVirtual(name) != want {
			t.Errorf("isVirtual(%q) = %v", name, !want)
		}
	}
}

// A vacuum by file replacement closes the connection the engines' Storage borrowed: the
// assistant and the display must rebuild on the new one rather than serve its errors.
func TestLiveEngines_AnswerAfterAVacuumBySwap(t *testing.T) {
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(a.stopDisplay)
	t.Cleanup(a.stopMCP)
	t.Cleanup(func() { _ = db.Close() })
	openDemo(t, a, db)
	list, err := db.ListDirections()
	if err != nil || len(list) == 0 {
		t.Fatalf("the demo file must hold a Direction: %v", err)
	}
	body := fmt.Sprintf(`{"tournamentId":%d}`, list[0].TournamentID)
	// The assistant reaches the engine as the MCP server does, tenant 1; the display through
	// its own handler, which sets the tenant itself.
	headers := map[string]string{"Content-Type": "application/json", "X-Tenant-ID": "1"}
	handlers := map[string]http.Handler{
		"assistant": &a.mcp.engine,
		"display":   displayHandler(&a.display.engine, list[0].TournamentID),
	}
	ask := func(when string) {
		t.Helper()
		for name, h := range handlers {
			if r := serveDisplay(h, http.MethodPost, displayRoute, body, headers); r.Code != http.StatusOK {
				t.Fatalf("%s %s: code=%d body=%.200s", name, when, r.Code, r.Body.String())
			}
		}
	}
	ask("before the vacuum")
	first := a.mcp.engine.srv
	if _, err := db.Vacuum(); err != nil {
		t.Fatal(err)
	}
	ask("after the vacuum")
	if a.mcp.engine.srv == first {
		t.Error("the swap must rebuild the engine on the new connection")
	}
}
