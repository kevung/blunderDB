package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// public_id is set, read back as given and cleared; an unknown position is a 404.
func TestPositionsPublicID_SetGetClear(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, Logger: slog.New(slog.DiscardHandler), TrustReadTenants: true})
	if err != nil {
		t.Fatal(err)
	}
	save := serveAcross(t, srv, "1", "", "/v1/positions.save", `{"position":`+initialPositionJSON(t)+`}`)
	var id idResp
	if err := json.Unmarshal(save.Body.Bytes(), &id); err != nil || id.ID == 0 {
		t.Fatalf("positions.save: %d %s", save.Code, save.Body)
	}
	get := func() (int, string) {
		rec := serveAcross(t, srv, "1", "", "/v1/positions.publicId", `{"id":`+jsonInt(id.ID)+`}`)
		var r struct {
			PublicID string `json:"publicId"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &r)
		return rec.Code, r.PublicID
	}
	if code, v := get(); code != http.StatusOK || v != "" {
		t.Fatalf("fresh: %d %q", code, v)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setPublicId", `{"id":`+jsonInt(id.ID)+`,"publicId":"Pos 7/é"}`); rec.Code != http.StatusOK {
		t.Fatalf("set: %d %s", rec.Code, rec.Body)
	}
	if code, v := get(); code != http.StatusOK || v != "Pos 7/é" {
		t.Errorf("after set: %d %q", code, v)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setPublicId", `{"id":`+jsonInt(id.ID)+`,"publicId":""}`); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body)
	}
	if _, v := get(); v != "" {
		t.Errorf("after clear: %q", v)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setPublicId", `{"id":999999,"publicId":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown position: %d, want 404", rec.Code)
	}
}
