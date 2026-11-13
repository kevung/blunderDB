package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
