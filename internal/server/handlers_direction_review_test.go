package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"
)

// TestDirectionGestures_SetConfigRewritesTheRoom: a configuration rewrites the event's page and
// its sisters', once the gesture is committed — and warns of nothing when every page is written.
func TestDirectionGestures_SetConfigRewritesTheRoom(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	seedSister(t, srv.opts.Storage, f, "Speed", "s")
	dir := t.TempDir()
	if _, err := f.svc.SetRencontreOutputDir(f.ctx, f.rencontreID, dir); err != nil {
		t.Fatal(err)
	}
	for _, e := range htmlUnder(t, dir) {
		if err := os.Remove(e); err != nil {
			t.Fatal(err)
		}
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	r := send(t, ts, "1", "/v1/directions.setConfig", f.body(`"config":`+cfg), f.versionOf(t, ts), "")
	if r.status != http.StatusOK {
		t.Fatalf("setConfig: %d %.200s", r.status, r.body)
	}
	if w := r.header.Values(pageWarningHeader); len(w) != 0 {
		t.Errorf("setConfig warned %v although every page could be written", w)
	}
	if pages := htmlUnder(t, dir); len(pages) < 3 {
		t.Errorf("after setConfig, %v under %s; want the event's page, the sister's and the wall", pages, dir)
	}
}

// TestDirectionGestures_EveryGestureAnswersItsVersion: a gesture's answer carries the version of
// the state it returns — the If-Match of the next gesture, as the next read gives it.
func TestDirectionGestures_EveryGestureAnswersItsVersion(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	b := seedSister(t, srv.opts.Storage, f, "Speed", "s")
	room := `{"id":` + strconv.FormatInt(f.rencontreID, 10)
	proposal := func() string {
		v, err := b.svc.GetDirection(b.ctx, b.tournamentID)
		if err != nil || len(v.Proposals) == 0 {
			t.Fatalf("no proposal to confirm: %v", err)
		}
		raw, _ := json.Marshal(v.Proposals[0])
		return b.body(`"action":` + string(raw))
	}
	calls := []struct {
		path string
		body func() string
	}{
		{"/v1/directions.confirmProposal", proposal},
		{"/v1/directions.addNote", func() string { return f.body(`"text":"note"`) }},
		{"/v1/directions.addParticipant", func() string { return f.body(`"name":"Retardataire"`) }},
		{"/v1/directions.makeAbsent", func() string { return f.body(`"id":"aa","until":"2099-10-01T18:00:00Z"`) }},
		{"/v1/directions.enterResult", func() string { return f.resultBody(f.running(t)[0]) }},
		{"/v1/directions.confirmAllProposals", func() string { return b.body("") }},
		{"/v1/rencontres.setBreaks", func() string { return room + `,"breaks":[]}` }},
		{"/v1/rencontres.setTableOutOfService", func() string { return room + `,"table":4,"out":false}` }},
		{"/v1/rencontres.update", func() string { return room + `,"name":"Salle","tables":4}` }},
	}
	for _, c := range calls {
		r := send(t, ts, "1", c.path, c.body(), f.versionOf(t, ts), "")
		if r.status != http.StatusOK {
			t.Errorf("%s: status %d, body %.200s", c.path, r.status, r.body)
			continue
		}
		if r.version == "" {
			t.Errorf("%s: no Direction-Version in the answer", c.path)
		} else if now := f.versionOf(t, ts); r.version != now {
			t.Errorf("%s: answered version %s, the next read gives %s", c.path, r.version, now)
		}
	}
}
