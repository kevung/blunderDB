package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

type derivationBody struct {
	DerivedFrom *struct {
		FromID int64  `json:"from_id"`
		Move   string `json:"move"`
	} `json:"derived_from"`
}

// A variant is linked to its origin with the move played, read back, listed
// from the origin, replaced and cleared; a refused link is a 400 that leaves
// the previous one, an unknown position a 404.
func TestPositionsDerivation_SetGetClear(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, Logger: slog.New(slog.DiscardHandler), TrustReadTenants: true})
	if err != nil {
		t.Fatal(err)
	}
	save := func(body string) string {
		rec := serveAcross(t, srv, "1", "", "/v1/positions.save", body)
		var id idResp
		if err := json.Unmarshal(rec.Body.Bytes(), &id); err != nil || id.ID == 0 {
			t.Fatalf("positions.save: %d %s", rec.Code, rec.Body)
		}
		return jsonInt(id.ID)
	}
	origin := save(`{"position":` + initialPositionJSON(t) + `}`)
	variant := save(`{"position":` + otherPositionJSON(t) + `}`)

	get := func(id string) derivationBody {
		rec := serveAcross(t, srv, "1", "", "/v1/positions.derivedFrom", `{"id":`+id+`}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("derivedFrom: %d %s", rec.Code, rec.Body)
		}
		var b derivationBody
		if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
			t.Fatal(err)
		}
		return b
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.derivedFrom", `{"id":`+variant+`}`); rec.Body.String() != `{"derived_from":null}`+"\n" && rec.Body.String() != `{"derived_from":null}` {
		t.Fatalf("fresh = %s, want an explicit null", rec.Body)
	}
	set := `{"id":` + variant + `,"derived_from":{"from_id":` + origin + `,"move":"13/7 24/23"}}`
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDerivedFrom", set); rec.Code != http.StatusOK {
		t.Fatalf("set: %d %s", rec.Code, rec.Body)
	}
	if b := get(variant); b.DerivedFrom == nil || jsonInt(b.DerivedFrom.FromID) != origin || b.DerivedFrom.Move != "13/7 24/23" {
		t.Fatalf("after set: %+v", b.DerivedFrom)
	}
	rec := serveAcross(t, srv, "1", "", "/v1/positions.variants", `{"id":`+origin+`}`)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"ids":[`+variant+`]}`+"\n" && rec.Body.String() != `{"ids":[`+variant+`]}` {
		t.Errorf("variants: %d %s", rec.Code, rec.Body)
	}
	for name, body := range map[string]string{
		"itself": `{"id":` + variant + `,"derived_from":{"from_id":` + variant + `}}`,
		"loop":   `{"id":` + origin + `,"derived_from":{"from_id":` + variant + `}}`,
	} {
		if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDerivedFrom", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s, want 400", name, rec.Code, rec.Body)
		}
	}
	if b := get(variant); b.DerivedFrom == nil || b.DerivedFrom.Move != "13/7 24/23" {
		t.Errorf("a refused link changed the previous one: %+v", b.DerivedFrom)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDerivedFrom", `{"id":`+variant+`}`); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body)
	}
	if b := get(variant); b.DerivedFrom != nil {
		t.Errorf("after clear: %+v", b.DerivedFrom)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.setDerivedFrom", `{"id":999999}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown position: %d, want 404", rec.Code)
	}
}

// otherPositionJSON is the initial position after one checker moved: another
// board, hence another position.
func otherPositionJSON(t *testing.T) string {
	t.Helper()
	p := domain.InitializePosition()
	from, to := -1, -1
	for i := 1; i <= 24; i++ {
		if from < 0 && p.Board.Points[i].Checkers >= 2 && p.Board.Points[i].Color == domain.Black {
			from = i
		}
		if to < 0 && p.Board.Points[i].Checkers == 0 {
			to = i
		}
	}
	p.Board.Points[from].Checkers--
	p.Board.Points[to] = domain.Point{Checkers: 1, Color: domain.Black}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
