package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestPositionsArkID(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	srv, err := New(Options{Storage: s})
	if err != nil {
		t.Fatal(err)
	}
	post := func(route, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/"+route, strings.NewReader(body))
		req.Header.Set(middleware.TenantHeader, "1")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	const compact = "[34]65:x2(ySQmLhfA)S3Qm3L3hf3"
	rec := post("positions.fromArkID", `{"arkid":"`+compact+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("fromArkID: %d (%s)", rec.Code, rec.Body)
	}
	var pos domain.Position
	if err := json.Unmarshal(rec.Body.Bytes(), &pos); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"position": pos, "expanded": true})
	rec = post("positions.arkId", string(body))
	var out arkIDResp
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.ArkID != "[34]65:x2(ymhf)m3hf3(SQLA)S3QL3" {
		t.Fatalf("arkId expanded: %d %s", rec.Code, rec.Body)
	}
	if rec = post("positions.fromArkID", `{"arkid":"[34]65:f3"}`); rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad ArkID: got %d", rec.Code)
	}
}
