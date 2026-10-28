//go:build postgres

package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestGenerateLibraryIsScopedByTenant: a position saved by tenant A is not
// readable by tenant B through the library source (the SQLite backend holds
// a single tenant, so only PostgreSQL can show the isolation).
func TestGenerateLibraryIsScopedByTenant(t *testing.T) {
	ts := newPostgresTestServer(t)
	pos := domain.InitializePosition()
	resp := postTenant(t, ts, "11", "/v1/positions.save", positionReq{Position: &pos})
	var saved positionSaveResp
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	body := trainingGenerateReq{Source: "library", PositionIDs: []int64{saved.ID}}
	for _, c := range []struct {
		tenant string
		status int
	}{{"11", http.StatusOK}, {"12", http.StatusNotFound}} {
		r := postTenant(t, ts, c.tenant, "/v1/training.generateBearoff", body)
		r.Body.Close()
		if r.StatusCode != c.status {
			t.Errorf("tenant %s: status %d, want %d", c.tenant, r.StatusCode, c.status)
		}
	}
}
