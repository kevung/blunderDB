package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/openingbook"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestOpeningBookRoutes(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	book, err := openingbook.Open("../../pkg/blunderdb/openingbook/testdata/book.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	post := func(srv *Server, path, body string, out any) int {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set(middleware.TenantHeader, "1")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if out != nil && rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
				t.Fatalf("%s: %v", path, err)
			}
		}
		return rec.Code
	}

	empty, err := New(Options{Storage: st, OpeningBook: &openingbook.Book{}})
	if err != nil {
		t.Fatal(err)
	}
	var q openingbook.Question
	if code := post(empty, "/v1/openingbook.question", `{}`, &q); code != http.StatusOK || !q.Empty {
		t.Fatalf("empty book: %d %+v", code, q)
	}

	srv, err := New(Options{Storage: st, OpeningBook: book})
	if err != nil {
		t.Fatal(err)
	}
	var s openingbook.Status
	if code := post(srv, "/v1/openingbook.status", `{}`, &s); code != http.StatusOK || s.Lines != 7 {
		t.Fatalf("status: %d %+v", code, s)
	}
	if code := post(srv, "/v1/openingbook.question", `{"scores":["2-4"],"seed":3}`, &q); code != http.StatusOK || q.Score != "2-4" {
		t.Fatalf("question: %d %+v", code, q)
	}
	if code := post(srv, "/v1/openingbook.question", `{"roll":"second","scores":["2-4"],"seed":3}`, &q); code != http.StatusOK || q.Reply != "21" || q.OpeningPlay != "8/5 6/5" {
		t.Fatalf("second-roll question: %d %+v", code, q)
	}
	var v openingbook.Verdict
	if code := post(srv, "/v1/openingbook.grade", `{"opening":"31","score":"2-4","notation":"8/5 6/5"}`, &v); code != http.StatusOK || !v.Right || len(v.Ranked) != 2 {
		t.Fatalf("grade: %d %+v", code, v)
	}
	if code := post(srv, "/v1/openingbook.grade", `{"opening":"31","reply":"21","score":"9-9","notation":"8/7"}`, nil); code != http.StatusBadRequest {
		t.Fatalf("a line the book lacks: %d, want 400", code)
	}
}
