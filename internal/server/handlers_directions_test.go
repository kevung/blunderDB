package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// directedFixture is one tenant's directed Tournament, playing in a Rencontre, with matches
// running.
type directedFixture struct {
	tenant       string
	tournamentID int64
	rencontreID  int64
	svc          *service.Service
	ctx          context.Context
}

// seedDirection directs a Tournament of tenant through the service the routes read — the
// gestures are not served yet, so the fixture writes beside the daemon, on its storage.
func seedDirection(t *testing.T, st storage.Storage, tenant, name string) directedFixture {
	t.Helper()
	n, err := storage.ParseTenant(tenant)
	if err != nil {
		t.Fatal(err)
	}
	ctx := storage.WithTenant(context.Background(), n)
	svc := service.New(st, tenant, nil)
	tid, err := st.Tournaments().Create(ctx, tenant, name, "2026-10-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"` + name + `","tables":{"count":4},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatalf("CreateDirection: %v", err)
	}
	rc, err := svc.CreateRencontre(ctx, "Salle "+name, "", "", 4)
	if err != nil {
		t.Fatalf("CreateRencontre: %v", err)
	}
	if _, err := svc.AttachToRencontre(ctx, tid, rc.ID); err != nil {
		t.Fatalf("AttachToRencontre: %v", err)
	}
	if err := svc.EnterParticipants(ctx, tid, `[{"id":"aa","name":"Joueur aa"},{"id":"bb","name":"Joueur bb"},
		{"id":"cc","name":"Joueur cc"},{"id":"dd","name":"Joueur dd"}]`); err != nil {
		t.Fatalf("EnterParticipants: %v", err)
	}
	if v, err := svc.ConfirmAllProposals(ctx, tid); err != nil || len(v.Running) == 0 {
		t.Fatalf("ConfirmAllProposals = %v; want matches running", err)
	}
	return directedFixture{tenant: tenant, tournamentID: tid, rencontreID: rc.ID, svc: svc, ctx: ctx}
}

// enterFirstResult plays a gesture: the first running match is won by its first player.
func (f directedFixture) enterFirstResult(t *testing.T) {
	t.Helper()
	v, err := f.svc.GetDirection(f.ctx, f.tournamentID)
	if err != nil || len(v.Running) == 0 {
		t.Fatalf("GetDirection: %v, %d running", err, len(v.Running))
	}
	m := v.Running[0]
	if _, err := f.svc.EnterResult(f.ctx, f.tournamentID, string(m.ID), string(m.A), 0, 0, ""); err != nil {
		t.Fatalf("EnterResult: %v", err)
	}
}

// readAs POSTs a read under tenant, with an If-None-Match when etag is not empty.
func readAs(t *testing.T, ts *httptest.Server, tenant, path, body, etag string) (readResult, []byte) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return readResult{StatusCode: resp.StatusCode, ETag: resp.Header.Get("ETag")}, b
}

// readResult is what a test reads of an answer, its body already closed.
type readResult struct {
	StatusCode int
	ETag       string
}

// directionReadCalls is every read route with a body naming the fixture's objects.
func directionReadCalls(f directedFixture) map[string]string {
	tid := `{"tournamentId":` + strconv.FormatInt(f.tournamentID, 10) + `}`
	rid := `{"id":` + strconv.FormatInt(f.rencontreID, 10) + `}`
	return map[string]string{
		"/v1/directions.list":             `{}`,
		"/v1/directions.get":              tid,
		"/v1/directions.participants":     tid,
		"/v1/directions.freeParticipants": tid,
		"/v1/directions.tableGrid":        tid,
		"/v1/directions.brackets":         tid,
		"/v1/directions.standings":        tid,
		"/v1/directions.standingsCsv":     tid,
		"/v1/directions.history":          tid,
		"/v1/directions.clock":            tid,
		"/v1/directions.slots":            tid,
		"/v1/directions.lastDecision":     tid,
		"/v1/directions.directory":        `{}`,
		"/v1/directions.pageHtml":         tid,
		"/v1/directions.pairingSheetHtml": `{"tournamentId":` + strconv.FormatInt(f.tournamentID, 10) + `,"round":1}`,
		"/v1/rencontres.list":             `{}`,
		"/v1/rencontres.get":              rid,
		"/v1/rencontres.pageHtml":         rid,
	}
}

// checkDirectionReads is the behaviour both backends must show: every read answers with a
// tag, a matching If-None-Match is a 304 until a gesture lands, and an unknown id is a 404.
func checkDirectionReads(t *testing.T, ts *httptest.Server, f directedFixture) {
	t.Helper()
	tags := map[string]string{}
	for path, body := range directionReadCalls(f) {
		resp, b := readAs(t, ts, f.tenant, path, body, "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d, body %s", path, resp.StatusCode, b)
			continue
		}
		tag := resp.ETag
		if !strings.HasPrefix(tag, `W/"`) {
			t.Errorf("%s: ETag %q, want a weak tag", path, tag)
			continue
		}
		tags[path] = tag
		if resp, b := readAs(t, ts, f.tenant, path, body, tag); resp.StatusCode != http.StatusNotModified || len(b) != 0 {
			t.Errorf("%s with its own tag: status %d, %d bytes; want 304 and no body", path, resp.StatusCode, len(b))
		}
	}

	var page struct {
		HTML string `json:"html"`
	}
	_, b := readAs(t, ts, f.tenant, "/v1/rencontres.pageHtml", directionReadCalls(f)["/v1/rencontres.pageHtml"], "")
	if err := json.Unmarshal(b, &page); err != nil || !strings.Contains(page.HTML, "Joueur") {
		t.Errorf("rencontres.pageHtml = %.200s (%v); want the wall page with the players", b, err)
	}

	f.enterFirstResult(t)
	for _, path := range []string{"/v1/directions.get", "/v1/directions.history", "/v1/rencontres.pageHtml", "/v1/directions.list"} {
		resp, _ := readAs(t, ts, f.tenant, path, directionReadCalls(f)[path], tags[path])
		if resp.StatusCode != http.StatusOK || resp.ETag == tags[path] {
			t.Errorf("%s after a gesture: status %d, tag %q (was %q); want 200 under a new tag",
				path, resp.StatusCode, resp.ETag, tags[path])
		}
	}

	tid := strconv.FormatInt(f.tournamentID, 10)
	for _, body := range []string{`{"tournamentId":` + tid + `,"round":-1}`, `{"tournamentId":` + tid + `,"round":99}`} {
		if resp, b := readAs(t, ts, f.tenant, "/v1/directions.pairingSheetHtml", body, ""); resp.StatusCode >= 500 {
			t.Errorf("pairingSheetHtml %s: status %d, body %s; a bad round is the caller's error", body, resp.StatusCode, b)
		}
	}

	for path, body := range map[string]string{
		"/v1/directions.get":      `{"tournamentId":987654}`,
		"/v1/directions.pageHtml": `{"tournamentId":987654}`,
		"/v1/rencontres.get":      `{"id":987654}`,
		"/v1/rencontres.pageHtml": `{"id":987654}`,
	} {
		resp, b := readAs(t, ts, f.tenant, path, body, "")
		if resp.StatusCode != http.StatusNotFound || resp.ETag != "" {
			t.Errorf("%s on an unknown id: status %d, tag %q, body %s; want 404, no tag", path, resp.StatusCode, resp.ETag, b)
		}
	}
}

func TestDirectionReads_SQLite(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	checkDirectionReads(t, ts, f)
}

// TestDirectionReads_TagLivesOneMinute: proposals and pages depend on the time of the reading,
// so a tag does not outlive its minute even when no gesture is made.
func TestDirectionReads_TagLivesOneMinute(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	now := time.Date(2026, 10, 1, 14, 0, 10, 0, time.UTC)
	srv.opts.now = func() time.Time { return now }
	body := directionReadCalls(f)["/v1/directions.get"]
	resp, _ := readAs(t, ts, "1", "/v1/directions.get", body, "")
	tag := resp.ETag
	now = now.Add(30 * time.Second)
	if resp, _ := readAs(t, ts, "1", "/v1/directions.get", body, tag); resp.StatusCode != http.StatusNotModified {
		t.Errorf("same minute: status %d, want 304", resp.StatusCode)
	}
	now = now.Add(30 * time.Second)
	if resp, _ := readAs(t, ts, "1", "/v1/directions.get", body, tag); resp.StatusCode != http.StatusOK {
		t.Errorf("next minute: status %d, want 200", resp.StatusCode)
	}
}

// TestDirectionReads_SingleTenantSQLite: a SQLite daemon has one tenant, so the Direction of
// that file is out of reach of any other X-Tenant-ID.
func TestDirectionReads_SingleTenantSQLite(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, SingleTenant: true})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	f := seedDirection(t, st, "1", "Open de Lyon")
	for path, body := range directionReadCalls(f) {
		if resp, b := readAs(t, ts, "2", path, body, ""); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s as tenant 2: status %d, body %s; want 400", path, resp.StatusCode, b)
		}
	}
}

// TestRunCallDirectionReads: `call` reaches the same routes as the daemon (parity).
func TestRunCallDirectionReads(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "call.db")
	st, err := sqlite.Open(context.Background(), dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := seedDirection(t, st, "1", "Open de Lyon")
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error {
		return RunCall([]string{"directions.get", "--db", dbPath, "--json", `{"tournamentId":` + strconv.FormatInt(f.tournamentID, 10) + `}`})
	})
	if err != nil {
		t.Fatalf("directions.get: %v (out=%s)", err, out)
	}
	var v service.DirectionView
	if err := json.NewDecoder(bytes.NewReader([]byte(out))).Decode(&v); err != nil || v.TournamentID != f.tournamentID || len(v.Running) == 0 {
		t.Fatalf("directions.get = %.300s (%v)", out, err)
	}
	out, err = captureStdout(t, func() error {
		return RunCall([]string{"rencontres.pageHtml", "--db", dbPath, "--json", `{"id":` + strconv.FormatInt(f.rencontreID, 10) + `}`})
	})
	if err != nil || !strings.Contains(out, "Joueur") {
		t.Fatalf("rencontres.pageHtml: %v (out=%.300s)", err, out)
	}
}
