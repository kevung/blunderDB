package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	tournoi "github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/render"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// seedSwiss directs a continuous Swiss of n players in the room rencontreID, nothing launched.
func seedSwiss(t *testing.T, st storage.Storage, tenant, name string, rencontreID int64, n int) directedFixture {
	t.Helper()
	tn, err := storage.ParseTenant(tenant)
	if err != nil {
		t.Fatal(err)
	}
	ctx := storage.WithTenant(t.Context(), tn)
	svc := service.New(st, tenant, nil)
	tid, err := st.Tournaments().Create(ctx, tenant, name, "2026-10-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf(`{"name":%q,"tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`, name)
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	if rencontreID == 0 {
		rc, err := svc.CreateRencontre(ctx, "Salle "+name, "", "", 4)
		if err != nil {
			t.Fatal(err)
		}
		rencontreID = rc.ID
	}
	if _, err := svc.AttachToRencontre(ctx, tid, rencontreID); err != nil {
		t.Fatal(err)
	}
	for i := range n {
		if _, err := svc.AddParticipant(ctx, tid, fmt.Sprintf("%s %d", name, i+1), "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	return directedFixture{tenant: tenant, tournamentID: tid, rencontreID: rencontreID, svc: svc, ctx: ctx}
}

// shownList reads the Direction as a client does: its version, and the launchable proposals
// as the JSON to send back.
func (f directedFixture) shownList(t *testing.T, ts *httptest.Server) (version, actions string, n int) {
	t.Helper()
	r := send(t, ts, f.tenant, "/v1/directions.get", f.body(""), "", "")
	var view struct {
		Proposals []json.RawMessage `json:"proposals"`
	}
	if err := json.Unmarshal(r.body, &view); r.status != http.StatusOK || err != nil {
		t.Fatalf("directions.get: status %d, %v", r.status, err)
	}
	var list []json.RawMessage
	for _, raw := range view.Proposals {
		var a tournoi.Action
		if err := json.Unmarshal(raw, &a); err != nil {
			t.Fatal(err)
		}
		if a.Kind != tournoi.ActWait && !service.Held(a) {
			list = append(list, raw)
		}
	}
	blob, _ := json.Marshal(list)
	return r.version, `"actions":` + string(blob), len(list)
}

func (f directedFixture) batches(t *testing.T, st storage.Storage) (batches, matches int) {
	t.Helper()
	dir, err := direction.Open(f.ctx, storage.BindDirection(st.Directions(), f.tenant), f.tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	lots := render.Batches(dir.State())
	for _, l := range lots {
		matches += len(l)
	}
	return len(lots), matches
}

func reasonOf(t *testing.T, r gestureResult) string {
	t.Helper()
	var env struct {
		Error struct {
			Details struct {
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(r.body, &env); err != nil {
		t.Fatalf("error envelope: %v in %.200s", err, r.body)
	}
	return env.Error.Details.Reason
}

// checkConfirmProposals: the list a client read is confirmed as one batch, once under its
// Idempotency-Key; decided on a version that moved it is a 409; naming tables a sister event
// took it is a 400 whose details.reason says so — and neither writes anything.
func checkConfirmProposals(t *testing.T, ts *httptest.Server, st storage.Storage, tenant string) {
	t.Helper()
	const path = "/v1/directions.confirmProposals"
	a := seedSwiss(t, st, tenant, "A", 0, 8)
	b := seedSwiss(t, st, tenant, "B", a.rencontreID, 4)

	seenB, listB, n := b.shownList(t, ts)
	if n != 2 {
		t.Fatalf("B shows %d launchable proposals, want 2", n)
	}
	v, list, n := a.shownList(t, ts)
	if n != 4 {
		t.Fatalf("A shows %d launchable proposals, want 4", n)
	}
	first := send(t, ts, tenant, path, a.body(list), v, "list-1")
	again := send(t, ts, tenant, path, a.body(list), v, "list-1")
	if first.status != http.StatusOK || !again.replayed || !bytes.Equal(first.body, again.body) {
		t.Fatalf("first %d (%.200s), again %d replayed=%v; want the same 200 twice", first.status, first.body, again.status, again.replayed)
	}
	if lots, matches := a.batches(t, st); lots != 1 || matches != 4 {
		t.Errorf("A plays %d matches in %d batches, want 4 in one", matches, lots)
	}

	// A's gesture moved the room's version: B's reading is stale.
	if r := send(t, ts, tenant, path, b.body(listB), seenB, ""); r.status != http.StatusConflict {
		t.Errorf("B on a stale version: status %d, body %.200s; want 409", r.status, r.body)
	}
	// With the fresh version, the list it read still names A's tables.
	r := send(t, ts, tenant, path, b.body(listB), b.versionOf(t, ts), "")
	if r.status != http.StatusBadRequest || reasonOf(t, r) != string(direction.ReasonTableTaken) {
		t.Errorf("B on A's tables: status %d, body %.300s; want 400 with reason table_taken", r.status, r.body)
	}
	if got := len(b.running(t)); got != 0 {
		t.Errorf("%d matches of B launched by refused lists", got)
	}
	r = send(t, ts, tenant, path, b.body(`"actions":[]`), b.versionOf(t, ts), "")
	if r.status != http.StatusBadRequest {
		t.Errorf("an empty list: status %d; want 400", r.status)
	}
}

func TestDirectionGestures_ConfirmProposals(t *testing.T) {
	ts, srv := gestureServer(t)
	checkConfirmProposals(t, ts, srv.opts.Storage, "1")
}

// seedTiedPool plays a pool of four to a 2-2-1-1 finish and withdraws its winner: two tied
// repechages are proposed for one place.
func seedTiedPool(t *testing.T, st storage.Storage, tenant string) directedFixture {
	t.Helper()
	tn, err := storage.ParseTenant(tenant)
	if err != nil {
		t.Fatal(err)
	}
	ctx := storage.WithTenant(t.Context(), tn)
	svc := service.New(st, tenant, nil)
	tid, err := st.Tournaments().Create(ctx, tenant, "Poule", "2026-10-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Poule","tables":{"count":4},"phases":[` +
		`{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2,"entry":"survivors"},` +
		`{"kind":"bracket","length":5,"entry":"survivors"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	ids := []string{"p1", "p2", "p3", "p4"}
	for i, id := range ids {
		if _, err := svc.AddParticipantAs(ctx, tid, id, "Joueur "+id, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	beats := map[[2]string]bool{
		{"p1", "p2"}: true, {"p1", "p3"}: true, {"p2", "p3"}: true,
		{"p2", "p4"}: true, {"p3", "p4"}: true, {"p4", "p1"}: true,
	}
	for played, step := 0, 0; played < 6; step++ {
		v, err := svc.ConfirmAllProposals(ctx, tid)
		if err != nil || step > 20 {
			t.Fatalf("the pool stalls after %d matches: %v", played, err)
		}
		for _, m := range v.Running {
			winner := string(m.B)
			if beats[[2]string{string(m.A), string(m.B)}] {
				winner = string(m.A)
			}
			if _, err := svc.EnterResult(ctx, tid, string(m.ID), winner, m.Length, 2, ""); err != nil {
				t.Fatal(err)
			}
			played++
		}
	}
	if _, err := svc.WithdrawParticipant(ctx, tid, "p1", false); err != nil {
		t.Fatal(err)
	}
	return directedFixture{tenant: tenant, tournamentID: tid, svc: svc, ctx: ctx}
}

// checkConfirmProposalsRollsBack: a list refused after its first proposal was recorded — two
// repechages for one place — leaves the journal and the version as they were; so does a
// passage listed over the undecided repechage.
func checkConfirmProposalsRollsBack(t *testing.T, ts *httptest.Server, st storage.Storage, tenant string) {
	t.Helper()
	const path = "/v1/directions.confirmProposals"
	f := seedTiedPool(t, st, tenant)
	v, err := f.svc.GetDirection(f.ctx, f.tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	var reps, passage []tournoi.Action
	for _, a := range v.Proposals {
		switch a.Kind {
		case tournoi.ActRepechage:
			reps = append(reps, a)
		case tournoi.ActNextPhase, tournoi.ActDraw:
			passage = append(passage, a)
		}
	}
	if len(reps) != 2 || len(passage) == 0 {
		t.Fatalf("the queue holds %+v, want two repechages and a passage", v.Proposals)
	}
	events := func() int {
		h, err := f.svc.History(f.ctx, f.tournamentID, "", "")
		if err != nil {
			t.Fatal(err)
		}
		return len(h)
	}
	before, version := events(), f.versionOf(t, ts)
	for want, list := range map[direction.Reason][]tournoi.Action{
		direction.ReasonProposalChanged:  reps,
		direction.ReasonRepechagePending: passage,
	} {
		blob, _ := json.Marshal(list)
		r := send(t, ts, tenant, path, f.body(`"actions":`+string(blob)), version, "")
		if r.status != http.StatusBadRequest || reasonOf(t, r) != string(want) {
			t.Errorf("%s: status %d, body %.300s; want 400 with that reason", want, r.status, r.body)
		}
		if got := f.versionOf(t, ts); got != version || events() != before {
			t.Errorf("%s: %d events (were %d), version moved: %v", want, events(), before, got != version)
		}
	}
}

// checkConfirmProposalsRace: two confirmations of one list on one reading launch it once.
func checkConfirmProposalsRace(t *testing.T, ts *httptest.Server, st storage.Storage, tenant string) {
	t.Helper()
	f := seedSwiss(t, st, tenant, "Course", 0, 8)
	v, list, n := f.shownList(t, ts)
	if n != 4 {
		t.Fatalf("%d launchable proposals, want 4", n)
	}
	results := make([]gestureResult, 2)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i] = send(t, ts, tenant, "/v1/directions.confirmProposals", f.body(list), v, "")
		}()
	}
	close(start)
	wg.Wait()
	got := []int{results[0].status, results[1].status}
	slices.Sort(got)
	if !slices.Equal(got, []int{http.StatusOK, http.StatusConflict}) {
		t.Fatalf("statuses %v; want one 200 and one 409", got)
	}
	if lots, matches := f.batches(t, st); lots != 1 || matches != 4 {
		t.Errorf("%d matches in %d batches after the race, want 4 in one", matches, lots)
	}
}

func TestDirectionGestures_ConfirmProposalsRollsBack(t *testing.T) {
	ts, srv := gestureServer(t)
	checkConfirmProposalsRollsBack(t, ts, srv.opts.Storage, "1")
}

func TestDirectionGestures_ConfirmProposalsRace(t *testing.T) {
	ts, srv := gestureServer(t)
	checkConfirmProposalsRace(t, ts, srv.opts.Storage, "1")
}
