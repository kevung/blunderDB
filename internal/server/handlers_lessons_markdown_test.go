package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A Lesson leaves as Markdown and a Markdown file comes back as Lessons; an
// unreadable file is a 400 that writes nothing.
func TestLessonsMarkdown_RoundTripAndRefusal(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	post := func(path, body string) (int, string) {
		rec := serveAcross(t, srv, "1", "", path, body)
		return rec.Code, rec.Body.String()
	}
	var created idResp
	code, out := post("/v1/lessons.create", `{"name":"Bear-off","description":"Basics"}`)
	if code != http.StatusOK || json.Unmarshal([]byte(out), &created) != nil {
		t.Fatalf("create: %d %s", code, out)
	}
	if code, out = post("/v1/lessons.addStep", `{"lessonId":`+jsonInt(created.ID)+`,"title":"Look","text":"first"}`); code != http.StatusOK {
		t.Fatalf("addStep: %d %s", code, out)
	}

	var md struct {
		Markdown string `json:"markdown"`
	}
	code, out = post("/v1/lessons.markdown", `{"ids":[`+jsonInt(created.ID)+`]}`)
	if code != http.StatusOK || json.Unmarshal([]byte(out), &md) != nil || !strings.Contains(md.Markdown, "# Bear-off") || !strings.Contains(md.Markdown, "## Look") {
		t.Fatalf("markdown: %d %s", code, out)
	}

	body, _ := json.Marshal(map[string]string{"markdown": md.Markdown + "\n# An article\n\nText.\n"})
	var imp struct {
		IDs []int64 `json:"ids"`
	}
	code, out = post("/v1/lessons.importMarkdown", string(body))
	if code != http.StatusOK || json.Unmarshal([]byte(out), &imp) != nil || len(imp.IDs) != 2 {
		t.Fatalf("importMarkdown: %d %s", code, out)
	}

	if code, out = post("/v1/lessons.importMarkdown", `{"markdown":"no title"}`); code != http.StatusBadRequest {
		t.Errorf("a file with no title: %d %s; want 400", code, out)
	}
	if code, out = post("/v1/lessons.markdown", `{"ids":[999999]}`); code != http.StatusNotFound {
		t.Errorf("unknown lesson: %d %s; want 404", code, out)
	}
	code, out = post("/v1/lessons.list", `{}`)
	var list []map[string]any
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatalf("lessons.list: %v in %s", err, out)
	}
	if code != http.StatusOK || len(list) != 3 {
		t.Errorf("lessons after the refusals: %d %s; want 3", code, out)
	}
}
