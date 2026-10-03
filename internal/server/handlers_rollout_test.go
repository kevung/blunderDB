package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// tinyRollout keeps the engine's part of these tests under a second.
const tinyRollout = "fast,games=36,min-games=36,truncation=2,candidates=2"

// A rollout asked with store lands beside the position's analysis and
// rollout.list reads it back; rollout.filter skips what is already rolled
// out with the same settings and streams to "done".
func TestRolloutRoutes_StoreListFilter(t *testing.T) {
	ts := newTestServer(t)

	p := domain.InitializePosition()
	resp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
	var saved idResp
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp = post(t, ts, "/v1/rollout.position", rolloutPositionReq{PositionID: saved.ID, Rollout: tinyRollout, Store: true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rollout.position status %d", resp.StatusCode)
	}
	var got rolloutPositionResp
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !got.Stored || got.Result == nil || len(got.Result.Candidates) != 2 {
		t.Fatalf("rollout.position: %+v", got)
	}

	resp = post(t, ts, "/v1/rollout.list", positionIDReq{PositionID: saved.ID})
	var list rolloutListResp
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(list.Rollouts) != 1 || list.Rollouts[0].Signature != got.Result.Signature || list.Rollouts[0].Candidates[0].CI95 <= 0 {
		t.Fatalf("rollout.list: %+v", list)
	}

	resp = post(t, ts, "/v1/rollout.filter", rolloutFilterReq{Rollout: tinyRollout})
	defer resp.Body.Close()
	var last map[string]any
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		last = nil
		if err := json.Unmarshal(sc.Bytes(), &last); err != nil {
			t.Fatal(err)
		}
	}
	if last["event"] != "done" || last["total"] != float64(0) {
		t.Errorf("rollout.filter over a position already rolled out: %v", last)
	}

	// A bare position is refused: serve exposes no evaluator (ADR-0015).
	bad := post(t, ts, "/v1/rollout.position", map[string]any{"xgid": "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10", "rollout": tinyRollout})
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Errorf("a rollout without positionId: status %d, want 400", bad.StatusCode)
	}
}

// A request cancelled while the sweep gathers its positions ends on the
// "cancelled" event, as one cancelled between two rollouts does.
func TestRolloutFilter_CancelledDuringGather(t *testing.T) {
	_, srv := newTestServerAndHandler(t)
	body, _ := json.Marshal(rolloutFilterReq{Rollout: tinyRollout})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/v1/rollout.filter", bytes.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()
	srv.handleRolloutFilter(rec, req)

	var last map[string]any
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		last = nil
		if err := json.Unmarshal(sc.Bytes(), &last); err != nil {
			t.Fatal(err)
		}
	}
	if last["event"] != "cancelled" {
		t.Errorf("last event %v, want cancelled", last)
	}
}

// A position that is not there is a 404; the engine's refusal is a 400.
func TestRolloutPosition_UnknownPositionIsNotFound(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/rollout.position", rolloutPositionReq{PositionID: 424242, Rollout: tinyRollout})
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, want 404", resp.StatusCode)
	}
}
