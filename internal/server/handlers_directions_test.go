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

	tournoi "github.com/kevung/nicomaque"

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
		"/v1/directions.freeSlots":        tid,
		"/v1/directions.tableGrid":        tid,
		"/v1/directions.brackets":         tid,
		"/v1/directions.standings":        tid,
		"/v1/directions.statuses":         tid,
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

// frozenServer is a SQLite daemon whose clock stands still, so a tag can only change because
// what the route reads did.
func frozenServer(t *testing.T) (*httptest.Server, *Server, directedFixture) {
	t.Helper()
	ts, srv := newTestServerAndHandler(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	now := time.Date(2026, 10, 1, 14, 0, 10, 0, time.UTC)
	srv.opts.now = func() time.Time { return now }
	return ts, srv, f
}

// tagOf reads a route and returns its tag.
func tagOf(t *testing.T, ts *httptest.Server, f directedFixture, path, body string) string {
	t.Helper()
	resp, b := readAs(t, ts, f.tenant, path, body, "")
	if resp.StatusCode != http.StatusOK || resp.ETag == "" {
		t.Fatalf("%s: status %d, tag %q, body %.200s", path, resp.StatusCode, resp.ETag, b)
	}
	return resp.ETag
}

// TestDirectionReads_RenameChangesTagNow: a Tournament's name is on its pages, so renaming it
// changes the tag within the minute.
func TestDirectionReads_RenameChangesTagNow(t *testing.T) {
	ts, srv, f := frozenServer(t)
	calls := directionReadCalls(f)
	paths := []string{"/v1/directions.get", "/v1/directions.pageHtml", "/v1/rencontres.pageHtml", "/v1/directions.list"}
	before := map[string]string{}
	for _, p := range paths {
		before[p] = tagOf(t, ts, f, p, calls[p])
	}
	if err := srv.opts.Storage.Tournaments().Update(f.ctx, f.tenant, f.tournamentID, "Open de Lyon — finale", "2026-10-01", "Lyon"); err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if resp, _ := readAs(t, ts, f.tenant, p, calls[p], before[p]); resp.StatusCode != http.StatusOK {
			t.Errorf("%s after a rename: status %d, want 200 under a new tag", p, resp.StatusCode)
		}
	}
}

// TestDirectionReads_DraftChangesSlotsTag: a draft started from a Slot shows in slots at once.
func TestDirectionReads_DraftChangesSlotsTag(t *testing.T) {
	ts, srv, f := frozenServer(t)
	body := directionReadCalls(f)["/v1/directions.slots"]
	tag := tagOf(t, ts, f, "/v1/directions.slots", body)
	v, err := f.svc.GetDirection(f.ctx, f.tournamentID)
	if err != nil || len(v.Running) == 0 {
		t.Fatalf("GetDirection: %v", err)
	}
	doc := `{"header":{"round":"Ronde #` + string(v.Running[0].ID) + `","tournament_id":` + strconv.FormatInt(f.tournamentID, 10) + `}}`
	if _, err := srv.opts.Storage.Transcriptions().Save(f.ctx, f.tenant, &storage.Transcription{Document: doc}); err != nil {
		t.Fatal(err)
	}
	if resp, _ := readAs(t, ts, f.tenant, "/v1/directions.slots", body, tag); resp.StatusCode != http.StatusOK {
		t.Errorf("slots after a draft started: status %d, want 200 under a new tag", resp.StatusCode)
	}
}

// TestDirectionReads_WildcardAndValidation: If-None-Match * names no answer, and a request the
// route refuses is refused whatever condition it carries.
func TestDirectionReads_WildcardAndValidation(t *testing.T) {
	ts, _, f := frozenServer(t)
	calls := directionReadCalls(f)
	if resp, _ := readAs(t, ts, f.tenant, "/v1/directions.get", calls["/v1/directions.get"], "*"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("If-None-Match *: status %d, want 400", resp.StatusCode)
	}
	tag := tagOf(t, ts, f, "/v1/directions.pairingSheetHtml", calls["/v1/directions.pairingSheetHtml"])
	bad := `{"tournamentId":` + strconv.FormatInt(f.tournamentID, 10) + `,"round":-1}`
	for _, cond := range []string{"", tag, "*"} {
		if resp, _ := readAs(t, ts, f.tenant, "/v1/directions.pairingSheetHtml", bad, cond); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("round -1 with If-None-Match %q: status %d, want 400", cond, resp.StatusCode)
		}
	}
}

// TestDirectionReads_MemberWithoutDirection: a Tournament attached to the room but not directed
// leaves the wall page readable, and tagged.
func TestDirectionReads_MemberWithoutDirection(t *testing.T) {
	ts, srv, f := frozenServer(t)
	st := srv.opts.Storage
	plain, err := st.Tournaments().Create(f.ctx, f.tenant, "Consolante", "2026-10-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Rencontres().Attach(f.ctx, f.tenant, plain, f.rencontreID); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/v1/rencontres.pageHtml", "/v1/rencontres.get", "/v1/directions.get"} {
		tagOf(t, ts, f, p, directionReadCalls(f)[p])
	}
}

// TestDirectionReads_NoServerPath: an output folder is a path of the server's disk, which a
// remote client has no use for and should not learn.
func TestDirectionReads_NoServerPath(t *testing.T) {
	ts, _, f := frozenServer(t)
	if err := f.svc.SetDirectionOutputDir(f.ctx, f.tournamentID, "/srv/secret-dir"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SetRencontreOutputDir(f.ctx, f.rencontreID, "/srv/secret-room"); err != nil {
		t.Fatal(err)
	}
	for p, body := range directionReadCalls(f) {
		if _, b := readAs(t, ts, f.tenant, p, body, ""); strings.Contains(string(b), "/srv/secret") {
			t.Errorf("%s shows a server path: %.300s", p, b)
		}
	}
}

// statusesOf reads directions.statuses with extra request fields, keyed by player.
func statusesOf(t *testing.T, ts *httptest.Server, f directedFixture, extra string) map[string]service.PlayerStatusRow {
	t.Helper()
	resp, b := readAs(t, ts, f.tenant, "/v1/directions.statuses", f.body(extra), "")
	var rows []service.PlayerStatusRow
	if err := json.Unmarshal(b, &rows); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("directions.statuses: status %d, body %s (%v)", resp.StatusCode, b, err)
	}
	out := map[string]service.PlayerStatusRow{}
	for _, r := range rows {
		out[r.Player] = r
	}
	if len(out) != len(rows) {
		t.Fatalf("directions.statuses lists a player twice: %s", b)
	}
	return out
}

// TestDirectionReads_StatusesSwiss: every entry of a Swiss with lives is playing, with its
// lives, its match, its table and its opponent; ids narrows the answer and a stranger is left
// out; a withdrawn player says so.
func TestDirectionReads_StatusesSwiss(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")

	all := statusesOf(t, ts, f, "")
	if len(all) != 4 {
		t.Fatalf("statuses = %+v; want the four entries", all)
	}
	for _, m := range f.running(t) {
		a, b := all[string(m.A)], all[string(m.B)]
		want := service.PlayerStatusRow{Player: string(m.A), Kind: tournoi.StatusPlaying, Lives: 2,
			MatchID: string(m.ID), Table: m.Table, Opponent: string(m.B)}
		if a.Kind != want.Kind || a.Lives != want.Lives || a.MatchID != want.MatchID || a.Table != want.Table || a.Table == 0 ||
			a.Opponent != want.Opponent || b.Opponent != string(m.A) || b.MatchID != want.MatchID {
			t.Errorf("statuses of match %s = %+v and %+v; want both playing it, two lives each", m.ID, a, b)
		}
	}

	if some := statusesOf(t, ts, f, `"ids":["cc","nobody","aa"]`); len(some) != 2 || some["cc"].Player == "" || some["aa"].Player == "" {
		t.Errorf("statuses of cc, nobody, aa = %+v; want cc and aa only", some)
	}
	if none := statusesOf(t, ts, f, `"ids":[]`); len(none) != 0 {
		t.Errorf("statuses of no id = %+v; want none", none)
	}

	f.enterFirstResult(t)
	loser := string(f.lastLoser(t))
	if got := statusesOf(t, ts, f, "")[loser]; got.Kind != tournoi.StatusPlaying || got.Lives != 1 || got.MatchID != "" {
		t.Errorf("status of %s after a loss = %+v; want playing, one life, no running match", loser, got)
	}
	if r := send(t, ts, "1", "/v1/directions.withdraw", f.body(`"id":"`+loser+`"`), f.versionOf(t, ts), ""); r.status != http.StatusOK {
		t.Fatalf("withdraw: status %d, body %.200s", r.status, r.body)
	}
	if got := statusesOf(t, ts, f, "")[loser]; got.Kind != tournoi.StatusWithdrawn {
		t.Errorf("status of %s once withdrawn = %+v; want withdrawn", loser, got)
	}
}

// lastLoser is the first entry of the fixture to have lost a match.
func (f directedFixture) lastLoser(t *testing.T) tournoi.PlayerID {
	t.Helper()
	rows, err := f.svc.Participants(f.ctx, f.tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Losses > 0 {
			return tournoi.PlayerID(r.ID)
		}
	}
	t.Fatal("no player has lost yet")
	return ""
}

// TestDirectionReads_StatusesBracket: a bracket of three exempts one player, eliminates the
// loser of each match and names the winner once the final is played.
func TestDirectionReads_StatusesBracket(t *testing.T) {
	ts, srv := gestureServer(t)
	st := srv.opts.Storage
	ctx := storage.WithTenant(context.Background(), 1)
	svc := service.New(st, "1", nil)
	tid, err := st.Tournaments().Create(ctx, "1", "Coupe", "2026-10-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CreateDirection(ctx, tid, `{"name":"Coupe","tables":{"count":4},"phases":[{"kind":"bracket","length":5}]}`, 7); err != nil {
		t.Fatalf("CreateDirection: %v", err)
	}
	if err := svc.EnterParticipants(ctx, tid, `[{"id":"aa","name":"A"},{"id":"bb","name":"B"},{"id":"cc","name":"C"}]`); err != nil {
		t.Fatalf("EnterParticipants: %v", err)
	}
	f := directedFixture{tenant: "1", tournamentID: tid, svc: svc, ctx: ctx}
	play := func() (winner, loser string) {
		t.Helper()
		v, err := svc.ConfirmAllProposals(ctx, tid)
		if err != nil || len(v.Running) != 1 {
			t.Fatalf("ConfirmAllProposals: %v, %d running; want one match", err, len(v.Running))
		}
		m := v.Running[0]
		if _, err := svc.EnterResult(ctx, tid, string(m.ID), string(m.A), 0, 0, ""); err != nil {
			t.Fatalf("EnterResult: %v", err)
		}
		return string(m.A), string(m.B)
	}

	if _, err := svc.ConfirmAllProposals(ctx, tid); err != nil {
		t.Fatal(err)
	}
	first := statusesOf(t, ts, f, "")
	kinds := map[tournoi.StatusKind]int{}
	for _, r := range first {
		kinds[r.Kind]++
		if r.Kind == tournoi.StatusBye && (r.MatchID != "" || r.Round == 0 || r.Label.Empty()) {
			t.Errorf("the exempt player = %+v; want no running match, and the round and label of the match they enter at", r)
		}
	}
	for _, r := range first {
		if r.Lives != 0 {
			t.Errorf("lives of %s in a bracket = %d; want 0, a bracket counts no lives", r.Player, r.Lives)
		}
	}
	if kinds[tournoi.StatusPlaying] != 2 || kinds[tournoi.StatusBye] != 1 {
		t.Fatalf("statuses of a bracket of three = %+v; want two playing and one bye", first)
	}

	_, out1 := play()
	if got := statusesOf(t, ts, f, "")[out1]; got.Kind != tournoi.StatusEliminated {
		t.Errorf("status of the semi-final's loser = %+v; want eliminated", got)
	}
	champion, out2 := play()
	last := statusesOf(t, ts, f, "")
	if last[champion].Kind != tournoi.StatusWinner || last[out2].Kind != tournoi.StatusEliminated || last[out1].Kind != tournoi.StatusEliminated {
		t.Errorf("statuses once the final is played = %+v; want %s winner, the others eliminated", last, champion)
	}
}

// statusFixture is a Direction of cfg with the entries aa, bb, cc and dd, and a way to play the
// running matches, the entry whose id sorts first winning each.
func statusFixture(t *testing.T, srv *Server, cfg string) (directedFixture, func()) {
	t.Helper()
	st := srv.opts.Storage
	ctx := storage.WithTenant(context.Background(), 1)
	svc := service.New(st, "1", nil)
	tid, err := st.Tournaments().Create(ctx, "1", "Coupe", "2026-10-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatalf("CreateDirection: %v", err)
	}
	if err := svc.EnterParticipants(ctx, tid, `[{"id":"aa","name":"A"},{"id":"bb","name":"B"},{"id":"cc","name":"C"},{"id":"dd","name":"D"}]`); err != nil {
		t.Fatalf("EnterParticipants: %v", err)
	}
	play := func() {
		t.Helper()
		v, err := svc.ConfirmAllProposals(ctx, tid)
		if err != nil || len(v.Running) == 0 {
			t.Fatalf("ConfirmAllProposals: %v, %d running; want a match to play", err, len(v.Running))
		}
		for _, m := range v.Running {
			if _, err := svc.EnterResult(ctx, tid, string(m.ID), string(min(m.A, m.B)), 0, 0, ""); err != nil {
				t.Fatalf("EnterResult: %v", err)
			}
		}
	}
	return directedFixture{tenant: "1", tournamentID: tid, svc: svc, ctx: ctx}, play
}

// TestDirectionReads_StatusesQualified: once a phase is over, those who pass are qualified for
// the next and the others eliminated.
func TestDirectionReads_StatusesQualified(t *testing.T) {
	ts, srv := gestureServer(t)
	f, play := statusFixture(t, srv, `{"name":"Coupe","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":5,"lives":1,"mode":"rounds"},{"kind":"bracket","length":5,"entry":"top:2"}]}`)
	play()
	play()
	got := statusesOf(t, ts, f, "")
	kinds := map[tournoi.StatusKind]int{}
	for _, r := range got {
		kinds[r.Kind]++
		if r.Kind == tournoi.StatusQualified && (r.Phase != 1 || r.MatchID != "") {
			t.Errorf("a qualified player = %+v; want phase 1 and no match", r)
		}
	}
	if kinds[tournoi.StatusQualified] != 2 || kinds[tournoi.StatusEliminated] != 2 {
		t.Fatalf("statuses once the Swiss is over = %+v; want its first two qualified and the others eliminated", got)
	}
}

// TestDirectionReads_StatusesNotEntered: an entry added once a full bracket is drawn is engaged
// in no phase.
func TestDirectionReads_StatusesNotEntered(t *testing.T) {
	ts, srv := gestureServer(t)
	f, _ := statusFixture(t, srv, `{"name":"Coupe","tables":{"count":4},"phases":[{"kind":"bracket","length":5}]}`)
	if _, err := f.svc.ConfirmAllProposals(f.ctx, f.tournamentID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AddParticipant(f.ctx, f.tournamentID, "Retardataire", "", 0); err != nil {
		t.Fatalf("AddParticipant: %v", err)
	}
	late := statusesOf(t, ts, f, "")
	for _, id := range []string{"aa", "bb", "cc", "dd"} {
		delete(late, id)
	}
	if len(late) != 1 {
		t.Fatalf("statuses beyond the four first entries = %+v; want the latecomer alone", late)
	}
	for _, r := range late {
		if r.Kind != tournoi.StatusNotEntered || r.Lives != 0 || r.MatchID != "" {
			t.Errorf("status of an entry added once the bracket is full = %+v; want not_entered, in no match", r)
		}
	}
}

// TestDirectionReads_StatusesUndecided: a player out of a phase whose best N pass, known at
// its end only, is undecided while the phase goes on.
func TestDirectionReads_StatusesUndecided(t *testing.T) {
	ts, srv := gestureServer(t)
	f, play := statusFixture(t, srv, `{"name":"Coupe","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":5,"lives":1,"mode":"rounds"},{"kind":"bracket","length":5,"entry":"top:2"}]}`)
	play()
	got := statusesOf(t, ts, f, "")
	undecided := 0
	for _, r := range got {
		if r.Kind == tournoi.StatusUndecided {
			undecided++
			if r.Phase != 1 || r.Lives != 0 || r.MatchID != "" {
				t.Errorf("an undecided player = %+v; want phase 1, no life left and no match", r)
			}
		}
	}
	if undecided != 2 {
		t.Errorf("statuses after the first round = %+v; want its two losers undecided", got)
	}
}
