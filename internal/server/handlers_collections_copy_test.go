package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// collections.copy answers the target and the number of positions it gained;
// a malformed target is a 400, an unknown source a 404, and a copy to a new
// name resent under its Idempotency-Key creates one collection.
func TestCollectionsCopy(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv, err := New(Options{Storage: st, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	post := func(path, body, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "1")
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	var ids []int64
	for i := 1; i <= 2; i++ {
		p := domain.InitializePosition()
		p.Score = [2]int{i, 0}
		id, err := st.Positions().Save(ctx, "1", &p)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	src, err := st.Collections().Create(ctx, "1", "src", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Collections().AddPositions(ctx, "1", src, ids); err != nil {
		t.Fatal(err)
	}

	decode := func(rec *httptest.ResponseRecorder) storage.CollectionCopyResult {
		t.Helper()
		if rec.Code != http.StatusOK {
			t.Fatalf("collections.copy: status %d: %s", rec.Code, rec.Body.String())
		}
		var res storage.CollectionCopyResult
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("decode %s: %v", rec.Body.String(), err)
		}
		return res
	}
	first := decode(post("/v1/collections.copy", `{"fromCollectionId":1,"name":"copy"}`, "copy-once"))
	if first.Copied != 2 || first.ID == src {
		t.Errorf("copy to a new name = %+v, want a new id and 2 copied", first)
	}
	if again := decode(post("/v1/collections.copy", `{"fromCollectionId":1,"name":"copy"}`, "copy-once")); again != first {
		t.Errorf("replayed copy = %+v, want %+v", again, first)
	}
	n := 0
	for _, err := range st.Collections().List(ctx, "1") {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 2 {
		t.Errorf("%d collections after a replayed copy, want 2", n)
	}
	into := decode(post("/v1/collections.copy", `{"fromCollectionId":1,"toCollectionId":2}`, ""))
	if into.ID != first.ID || into.Copied != 0 {
		t.Errorf("copy into a collection already holding everything = %+v, want id %d and 0 copied", into, first.ID)
	}

	for body, want := range map[string]int{
		`{"fromCollectionId":1}`:                               http.StatusBadRequest,
		`{"fromCollectionId":1,"toCollectionId":2,"name":"x"}`: http.StatusBadRequest,
		`{"fromCollectionId":99,"name":"x"}`:                   http.StatusNotFound,
		`{"fromCollectionId":1,"toCollectionId":99}`:           http.StatusNotFound,
	} {
		if rec := post("/v1/collections.copy", body, ""); rec.Code != want {
			t.Errorf("collections.copy %s: status %d, want %d: %s", body, rec.Code, want, rec.Body.String())
		}
	}
}
