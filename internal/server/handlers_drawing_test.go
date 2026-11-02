package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A drawing is set whole, read back in the order drawn, cleared by an empty
// set; a refused mark is a 400 that leaves the drawing as it was.
func TestPositionsDrawing_SetGetClear(t *testing.T) {
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
	pid := jsonInt(id.ID)
	get := func() string {
		rec := serveAcross(t, srv, "1", "", "/v1/positions.drawing", `{"id":`+pid+`}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("drawing: %d %s", rec.Code, rec.Body)
		}
		return rec.Body.String()
	}
	if got := get(); got != `{"marks":[]}`+"\n" && got != `{"marks":[]}` {
		t.Fatalf("fresh drawing = %s, want an empty list (not null)", got)
	}
	set := `{"id":` + pid + `,"marks":[{"kind":"arrow","from":13,"to":7,"color":"red"},{"kind":"square","from":6,"to":6,"color":"blue"}]}`
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDrawing", set); rec.Code != http.StatusOK {
		t.Fatalf("set: %d %s", rec.Code, rec.Body)
	}
	var back struct {
		Marks []struct {
			Kind  string `json:"kind"`
			From  int    `json:"from"`
			To    int    `json:"to"`
			Color string `json:"color"`
		} `json:"marks"`
	}
	if err := json.Unmarshal([]byte(get()), &back); err != nil || len(back.Marks) != 2 || back.Marks[0].Kind != "arrow" || back.Marks[0].To != 7 || back.Marks[1].Color != "blue" {
		t.Fatalf("after set: %+v, %v", back, err)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDrawing", `{"id":`+pid+`,"marks":[{"kind":"arrow","from":3,"to":3,"color":"red"}]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("an arrow to itself: %d %s, want 400", rec.Code, rec.Body)
	}
	if err := json.Unmarshal([]byte(get()), &back); err != nil || len(back.Marks) != 2 {
		t.Errorf("a refused drawing changed the previous one: %+v", back)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDrawing", `{"id":`+pid+`,"marks":[]}`); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body)
	}
	if got := get(); got != `{"marks":[]}`+"\n" && got != `{"marks":[]}` {
		t.Errorf("after clear = %s", got)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDrawing", `{"id":999999,"marks":[]}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown position: %d, want 404", rec.Code)
	}
}
