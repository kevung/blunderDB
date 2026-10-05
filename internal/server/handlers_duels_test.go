package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func duelServerOn(t *testing.T, st storage.Storage, on bool) *httptest.Server {
	t.Helper()
	srv, err := New(Options{Storage: st, Duel: on})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func duelState(t *testing.T, status int, body []byte) *duel.State {
	t.Helper()
	if status != http.StatusOK {
		t.Fatalf("status %d: %s", status, body)
	}
	var st duel.State
	if err := json.Unmarshal(body, &st); err != nil {
		t.Fatal(err)
	}
	return &st
}

// A Duel is created, read, played under If-Match, and still answers after the
// daemon restarted: nothing but the draft is needed to act on it.
func TestDuelGesturesSurviveARestart(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := duelServerOn(t, st, true)

	status, body := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 3})
	created := duelState(t, status, body)
	if created.Fingerprint == "" || bytes.Contains(body, []byte("diceSeed")) {
		t.Fatalf("a created Duel shows its fingerprint and no seed: %s", body)
	}
	if created.Awaiting == nil || created.Awaiting.Kind != duel.DecideMove {
		t.Fatalf("awaiting %+v, want the opening move", created.Awaiting)
	}
	resign := func(d *duel.Decision) map[string]any {
		plays := domain.LegalMoves(&d.Position)
		return map[string]any{"id": created.ID, "play": map[string]any{"side": d.Side, "kind": "move", "steps": plays[0].Steps}}
	}

	// No If-Match: refused.
	if status, _ := gesture(t, ts, testTenant, "/v1/duels.act", 0, resign(created.Awaiting)); status != http.StatusPreconditionRequired {
		t.Errorf("act without If-Match: status %d, want 428", status)
	}

	// A new daemon over the same storage knows no open Duel.
	ts2 := duelServerOn(t, st, true)
	if status, _ := gesture(t, ts2, testTenant, "/v1/duels.act", created.Revision+7, resign(created.Awaiting)); status != http.StatusConflict {
		t.Errorf("act on a stale revision: status %d, want 409", status)
	}
	status, body = gesture(t, ts2, testTenant, "/v1/duels.act", created.Revision, resign(created.Awaiting))
	played := duelState(t, status, body)
	if played.Awaiting == nil || played.Awaiting.Side == created.Awaiting.Side {
		t.Fatalf("after the move: awaiting %+v, want the other Side", played.Awaiting)
	}

	// The wrong Side is refused as the client's request.
	if status, _ := gesture(t, ts2, testTenant, "/v1/duels.act", played.Revision, map[string]any{"id": created.ID, "play": map[string]any{"side": created.Awaiting.Side, "kind": "roll"}}); status != http.StatusBadRequest {
		t.Errorf("a Play by the wrong Side: status %d, want 400", status)
	}

	// Throw it away.
	status, body = gesture(t, ts2, testTenant, "/v1/duels.discard", played.Revision, map[string]any{"id": created.ID})
	if ended := duelState(t, status, body); ended.Ended == nil || !ended.Ended.Discarded {
		t.Errorf("discard: ended %+v", ended.Ended)
	}
	if status, _ := gesture(t, ts2, testTenant, "/v1/duels.get", 0, map[string]any{"id": created.ID}); status != http.StatusNotFound {
		t.Errorf("get after discard: status %d, want 404", status)
	}
}

// Without --duel the gestures answer as absent routes; the reads are served.
func TestDuelGesturesAbsentWithoutTheFlag(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := duelServerOn(t, st, false)
	for _, p := range []string{"create", "open", "act", "flag", "suspend", "stop", "discard"} {
		if status, _ := gesture(t, ts, testTenant, "/v1/duels."+p, 1, map[string]any{"id": 1}); status != http.StatusNotFound {
			t.Errorf("duels.%s without the flag: status %d, want 404", p, status)
		}
	}
	if status, body := gesture(t, ts, testTenant, "/v1/duels.list", 0, struct{}{}); status != http.StatusOK {
		t.Errorf("duels.list without the flag: status %d (%s)", status, body)
	}
}

// A side the build does not offer is the client's request, not a failure.
func TestDuelCreateUnknownSideIsInvalid(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := duelServerOn(t, st, true)
	status, body := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 1, "sides": []any{map[string]any{"kind": "bot:3"}, map[string]any{}}})
	if status != http.StatusBadRequest {
		t.Errorf("create with a bot side nothing resolves: status %d, want 400 (%s)", status, body)
	}
}
