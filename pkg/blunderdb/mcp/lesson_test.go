package mcp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/mcp"
)

// A Lesson written through /v1 reads back through the two read tools, steps
// in order (ADR-0066).
func TestLessonTools(t *testing.T) {
	engine := demoServer(t, internalserver.Options{}).Handler()
	post := func(path string, body obj) obj {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "1")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code/100 != 2 {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body)
		}
		var out obj
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}
	lid := post("/v1/lessons.create", obj{"name": "Primes"})["id"]
	post("/v1/lessons.addStep", obj{"lessonId": lid, "title": "Un", "text": "Lisez."})
	post("/v1/lessons.addStep", obj{"lessonId": lid, "title": "Deux"})

	cs := connect(t, engine, mcp.Options{Tenant: "1"})
	lessons := list(t, call(t, cs, "list_lessons", nil), "lessons")
	var found obj
	for _, l := range lessons {
		if l.(obj)["name"] == "Primes" {
			found = l.(obj)
		}
	}
	if found == nil || found["stepCount"] != float64(2) {
		t.Fatalf("list_lessons = %v", lessons)
	}
	steps := list(t, call(t, cs, "lesson", obj{"lessonId": lid}), "steps")
	if len(steps) != 2 || steps[0].(obj)["title"] != "Un" || steps[1].(obj)["title"] != "Deux" {
		t.Fatalf("lesson steps = %v", steps)
	}
}
