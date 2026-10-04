package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
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

// TestRolloutOnPoolMatchesItsOwnWorkers: played game by game on the shared
// workers, a rollout gives the numbers it gives on its own goroutines, bit
// for bit, at the same seed.
func TestRolloutOnPoolMatchesItsOwnWorkers(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{})
	srv.analysis = newAnalysisPool(3, nil)
	pos := domain.InitializePosition()
	pos.Dice = [2]int{3, 1}
	id, err := srv.opts.Storage.Positions().Save(context.Background(), testTenant, &pos)
	if err != nil {
		t.Fatal(err)
	}

	var got rolloutPositionResp
	postDecode(t, ts, "/v1/rollout.position", rolloutPositionReq{PositionID: id, Rollout: tinyRollout}, &got)

	settings, err := rollout.ParseSpec(tinyRollout)
	if err != nil {
		t.Fatal(err)
	}
	want, err := rollout.Run(context.Background(), pos, settings, rollout.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Result == nil || !reflect.DeepEqual(got.Result.Candidates, want.Candidates) || got.Result.Games != want.Games {
		t.Fatalf("rollout on the pool differs from its own workers:\n%+v\n%+v", got.Result, want)
	}
}

// TestLongRolloutLetsAnotherTenantThrough: with one engine worker, a tenant's
// long rollout yields it between games, so another tenant's evaluation is
// answered while the rollout still runs, not after it.
func TestLongRolloutLetsAnotherTenantThrough(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{})
	srv.analysis = newAnalysisPool(1, nil)
	pos := domain.InitializePosition()
	pos.Dice = [2]int{3, 1}
	id, err := srv.opts.Storage.Positions().Save(context.Background(), testTenant, &pos)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(rolloutPositionReq{PositionID: id, Rollout: "fast,games=746496,min-games=746496,truncation=2,candidates=2"}); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/v1/rollout.position", &body)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", "application/json")
	long := make(chan struct{})
	go func() {
		defer close(long)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}()

	// The rollout holds the worker once its job is queued.
	deadline := time.Now().Add(10 * time.Second)
	for {
		srv.analysis.mu.Lock()
		queued := len(srv.analysis.queues[testTenant]) > 0
		srv.analysis.mu.Unlock()
		if queued {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the rollout never reached the pool")
		}
		runtime.Gosched()
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(gammonnetEvaluateReq{XGID: evalXGID, Ply: new(int)}); err != nil {
		t.Fatal(err)
	}
	evalReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/gammonnet.evaluate", &buf)
	evalReq.Header.Set(middleware.TenantHeader, "2")
	evalReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(evalReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("evaluate beside a long rollout: %d", resp.StatusCode)
	}
	select {
	case <-long:
		t.Fatal("the evaluation was answered only once the rollout had ended")
	default:
	}
	cancel()
	<-long
}
