package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// Two instances over one database — two serve daemons on one PostgreSQL, the desktop and a
// `call` on one SQLite file — share no process lock: only the backend's guard stands between
// their gestures. Each check runs one gesture on each instance, on one reading.

// raceOnTwo sends one gesture to each instance at once and returns their answers.
func raceOnTwo(t *testing.T, tss [2]*httptest.Server, tenant string, paths, bodies [2]string, version string) [2]gestureResult {
	t.Helper()
	var out [2]gestureResult
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			out[i] = send(t, tss[i], tenant, paths[i], bodies[i], version, "")
		}()
	}
	close(start)
	wg.Wait()
	return out
}

// oneApplied requires exactly one 200 and one 409, the 409 carrying a version.
func oneApplied(t *testing.T, what string, rs [2]gestureResult) {
	t.Helper()
	var ok, conflict int
	for _, r := range rs {
		switch r.status {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
			if r.version == "" {
				t.Errorf("%s: 409 without Direction-Version, body %.200s", what, r.body)
			}
		default:
			t.Errorf("%s: status %d, body %.300s", what, r.status, r.body)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Errorf("%s: %d applied, %d refused; want exactly one of each", what, ok, conflict)
	}
}

// seedSister directs another Tournament of f's room, with four players waiting.
func seedSister(t *testing.T, st storage.Storage, f directedFixture, name, prefix string) directedFixture {
	t.Helper()
	svc := service.New(st, f.tenant, nil)
	tid, err := st.Tournaments().Create(f.ctx, f.tenant, name, "2026-10-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"` + name + `","tables":{"count":4},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := svc.CreateDirection(f.ctx, tid, cfg, 7); err != nil {
		t.Fatalf("CreateDirection: %v", err)
	}
	if _, err := svc.AttachToRencontre(f.ctx, tid, f.rencontreID); err != nil {
		t.Fatalf("AttachToRencontre: %v", err)
	}
	players := ""
	for i, id := range []string{"a", "b", "c", "d"} {
		if i > 0 {
			players += ","
		}
		players += `{"id":"` + prefix + id + `","name":"Joueur ` + prefix + id + `"}`
	}
	if err := svc.EnterParticipants(f.ctx, tid, "["+players+"]"); err != nil {
		t.Fatalf("EnterParticipants: %v", err)
	}
	return directedFixture{tenant: f.tenant, tournamentID: tid, rencontreID: f.rencontreID, svc: svc, ctx: f.ctx}
}

// checkTwoInstances runs the three races of two instances over stA and stB, one database.
func checkTwoInstances(t *testing.T, stA, stB storage.Storage) {
	tsA, _ := gestureServerOn(t, stA)
	tsB, _ := gestureServerOn(t, stB)
	tss := [2]*httptest.Server{tsA, tsB}

	t.Run("one log", func(t *testing.T) {
		f := seedDirection(t, stA, "1", "Open de Lyon")
		run := f.running(t)
		rs := raceOnTwo(t, tss, f.tenant, [2]string{"/v1/directions.enterResult", "/v1/directions.enterResult"},
			[2]string{f.resultBody(run[0]), f.resultBody(run[1])}, f.versionOf(t, tsA))
		oneApplied(t, "two results", rs)
		if got := len(f.running(t)); got != 1 {
			t.Errorf("%d running after the race; want 1", got)
		}
	})

	t.Run("sister logs", func(t *testing.T) {
		f := seedDirection(t, stA, "1", "Open de Nice")
		b := seedSister(t, stA, f, "Speed", "s")
		c := seedSister(t, stA, f, "Consolante", "k")
		rs := raceOnTwo(t, tss, f.tenant, [2]string{"/v1/directions.confirmAllProposals", "/v1/directions.confirmAllProposals"},
			[2]string{b.body(""), c.body("")}, f.versionOf(t, tsA))
		oneApplied(t, "two sisters seating", rs)
		seated := map[int]string{}
		for _, x := range []directedFixture{f, b, c} {
			for _, m := range x.running(t) {
				if prev, taken := seated[m.Table]; taken {
					t.Errorf("table %d holds %s and %s", m.Table, prev, m.ID)
				}
				seated[m.Table] = string(m.ID)
			}
		}
	})

	t.Run("rencontres.update", func(t *testing.T) {
		f := seedDirection(t, stA, "1", "Open de Pau")
		r := send(t, tsA, f.tenant, "/v1/rencontres.get", `{"id":`+strconv.FormatInt(f.rencontreID, 10)+`}`, "", "")
		if r.status != http.StatusOK || r.version == "" {
			t.Fatalf("rencontres.get: status %d, version %q", r.status, r.version)
		}
		body := func(name string) string {
			return `{"id":` + strconv.FormatInt(f.rencontreID, 10) + `,"name":"` + name + `","tables":4}`
		}
		rs := raceOnTwo(t, tss, f.tenant, [2]string{"/v1/rencontres.update", "/v1/rencontres.update"},
			[2]string{body("Salle A"), body("Salle B")}, r.version)
		oneApplied(t, "two renames", rs)
	})
}

// TestDirectionGestures_TwoInstancesSQLite: two handles on one file, as the desktop and a
// `call` are — two pools, two processes' worth of locks.
func TestDirectionGestures_TwoInstancesSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "salle.db")
	open := func() storage.Storage {
		st, err := sqlite.Open(context.Background(), path, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		return st
	}
	stA := open()
	checkTwoInstances(t, stA, open())
}
