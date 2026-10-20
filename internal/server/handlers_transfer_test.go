package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// across.transferPull pulls from a tenant of the read set into the writer's
// own, and from no other tenant.
func TestTransferPull_ReadSetOnly(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	for name, c := range map[string]struct {
		readTenants, body string
		want              int
	}{
		"listed":           {"2", `{"tenant":"2"}`, http.StatusOK},
		"not listed":       {"2", `{"tenant":"3"}`, http.StatusBadRequest},
		"no tenant":        {"2", `{}`, http.StatusBadRequest},
		"into itself":      {"", `{"tenant":"1"}`, http.StatusBadRequest},
		"watermark":        {"2", `{"tenant":"2","export":{"watermark":"Cours"}}`, http.StatusBadRequest},
		"incremental":      {"2", `{"tenant":"2","export":{"incremental":true}}`, http.StatusBadRequest},
		"unknown selector": {"2", `{"tenant":"2","export":{"collectionIds":[1]}}`, http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			rec := serveAcross(t, srv, "1", c.readTenants, "/v1/across.transferPull", c.body)
			if rec.Code != c.want {
				t.Fatalf("status %d, want %d: %s", rec.Code, c.want, rec.Body)
			}
			if c.want == http.StatusOK {
				var rep ingest.TransferReport
				if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil {
					t.Errorf("answer %s: %v", rec.Body, err)
				}
			}
		})
	}
}
