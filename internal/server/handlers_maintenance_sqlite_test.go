package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestMaintenanceVacuumSQLite: the SQLite-backed server compacts its database
// and answers with the two sizes (both 0 on the :memory: test store — the
// shrink itself is covered by the backend's own test), and the data is still
// there afterwards. The PostgreSQL refusal is TestMaintenanceVacuumPostgresNotSupported
// (handlers_maintenance_test.go, tagged postgres).
func TestMaintenanceVacuumSQLite(t *testing.T) {
	ts := newTestServer(t)

	p := domain.InitializePosition()
	saveResp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
	defer saveResp.Body.Close()
	var saved idResp
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}

	resp := post(t, ts, "/ops/maintenance.vacuum", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var res storage.VacuumResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.SizeBefore != 0 || res.SizeAfter != 0 {
		t.Errorf("sizes on :memory: = %+v, want zeros", res)
	}

	loadResp := post(t, ts, "/v1/positions.load", idReq(saved))
	defer loadResp.Body.Close()
	if loadResp.StatusCode != http.StatusOK {
		t.Fatalf("post-vacuum load status = %d, want 200", loadResp.StatusCode)
	}
}
