package server

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"
)

func (p recordingPositions) Exists(ctx context.Context, scope string, zobrist uint64) (int64, bool, error) {
	p.log.read(scope)
	return p.PositionStore.Exists(ctx, scope, zobrist)
}

// The writing tenant is "here": across.positionElsewhere reads every other
// read tenant, in the read set's order, and writes nothing.
func TestAcrossElsewhere_ReadsTheOtherTenantsInOrder(t *testing.T) {
	srv, log := newRecordingServer(t, false)
	rec := serveAcross(t, srv, "1", "3,2", "/v1/across.positionElsewhere", `{"zobrist":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	reads, writes := log.snapshot()
	if !slices.Equal(reads, []string{"3", "2"}) {
		t.Errorf("store read %v, want [3 2]", reads)
	}
	if len(writes) != 0 {
		t.Errorf("an elsewhere read wrote under %v", writes)
	}
	var resp acrossElsewhereResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v: %s", err, rec.Body)
	}
	if len(resp.Results) != 2 || resp.Results[0].Tenant != "3" || resp.Results[1].Tenant != "2" || resp.Zobrist != 1 {
		t.Errorf("response = %+v, want tenants 3 then 2 on hash 1", resp)
	}
}

func TestAcrossElsewhere_NeedsABoard(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.positionElsewhere", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("no board: status %d, want 400", rec.Code)
	}
}
