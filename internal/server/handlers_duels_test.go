package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// Two clients that read the same revision: the first plays, the second is
// refused though the Duel was already open when its request came — the check
// sits with the Action, not with the opening.
func TestDuelSecondClientOnTheSameRevisionIsRefused(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := duelServerOn(t, st, true)
	status, body := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 3})
	c := duelState(t, status, body)
	plays := domain.LegalMoves(&c.Awaiting.Position)
	act := map[string]any{"id": c.ID, "play": map[string]any{"side": c.Awaiting.Side, "kind": "move", "steps": plays[0].Steps}}
	if status, body := gesture(t, ts, testTenant, "/v1/duels.act", c.Revision, act); status != http.StatusOK {
		t.Fatalf("first client: %d %s", status, body)
	}
	for _, path := range []string{"/v1/duels.act", "/v1/duels.open", "/v1/duels.discard", "/v1/duels.stop"} {
		if status, _ := gesture(t, ts, testTenant, path, c.Revision, act); status != http.StatusConflict {
			t.Errorf("%s on a stale revision: status %d, want 409", path, status)
		}
	}
}

// An Action names its Side: one left out is not player 1.
func TestDuelActWithoutASideIsInvalid(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := duelServerOn(t, st, true)
	status, body := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 3})
	c := duelState(t, status, body)
	plays := domain.LegalMoves(&c.Awaiting.Position)
	if status, _ := gesture(t, ts, testTenant, "/v1/duels.act", c.Revision, map[string]any{"id": c.ID, "play": map[string]any{"kind": "move", "steps": plays[0].Steps}}); status != http.StatusBadRequest {
		t.Errorf("act without a side: status %d, want 400", status)
	}
}

// A Duel of two Bots is played to its end by the call that creates it.
func TestDuelCreateOfTwoBotsEnds(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := duelServerOn(t, st, true)
	bot := map[string]any{"kind": "bot", "level": "instant"}
	status, body := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 1, "sides": []any{bot, bot}})
	if s := duelState(t, status, body); s.Ended == nil || s.Ended.MatchID == 0 {
		t.Errorf("ended %+v", s.Ended)
	}
	if status, _ := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 0, "sides": []any{bot, bot}}); status != http.StatusBadRequest {
		t.Errorf("a money session of two bots: status %d, want 400", status)
	}
}

// matches.origin reads back the origin of a Match a Duel became, its revealed
// seed giving the fingerprint published at creation; null for any other Match.
func TestMatchOriginIsServed(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ts := duelServerOn(t, st, true)
	bot := map[string]any{"kind": "bot", "level": "instant"}
	status, body := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 1, "sides": []any{bot, bot}})
	s := duelState(t, status, body)
	if s.Ended == nil || s.Ended.MatchID == 0 {
		t.Fatalf("ended %+v", s.Ended)
	}

	status, body = gesture(t, ts, testTenant, "/v1/matches.origin", 0, map[string]any{"matchId": s.Ended.MatchID})
	var o duel.Origin
	if status != http.StatusOK || json.Unmarshal(body, &o) != nil {
		t.Fatalf("matches.origin: status %d (%s)", status, body)
	}
	if fp, _ := duel.Fingerprint(o.DiceSeed); fp != s.Fingerprint || o.Fingerprint != s.Fingerprint {
		t.Errorf("seed read back gives %q, origin says %q, published %q", fp, o.Fingerprint, s.Fingerprint)
	}
	if o.BotLevel != "instant" {
		t.Errorf("bot level %q, want instant", o.BotLevel)
	}

	status, body = gesture(t, ts, testTenant, "/v1/matches.origin", 0, map[string]any{"matchId": s.Ended.MatchID + 100})
	if status != http.StatusOK || strings.TrimSpace(string(body)) != "null" {
		t.Errorf("origin of a match not played here: status %d (%s), want null", status, body)
	}
}
