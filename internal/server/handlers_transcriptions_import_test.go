package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
)

// transcriptions.import opens a draft on a .mat or on a document's
// {header, actions}, refuses one it cannot read with a 400, and is a gesture:
// absent without --transcription.
func TestTranscriptionImport(t *testing.T) {
	ts := newTranscriptionServer(t, 0)
	mat, err := os.ReadFile(filepath.Join("..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat"))
	if err != nil {
		t.Fatal(err)
	}
	status, body := gesture(t, ts, testTenant, "/v1/transcriptions.import", 0, map[string]any{"mat": string(mat)})
	st := decodeState(t, status, body)
	if st.SessionID == "" || len(st.Annotated.Document.Actions) == 0 {
		t.Fatalf("import of a .mat: %s", body)
	}
	_, wantMAT := gesture(t, ts, testTenant, "/v1/transcriptions.exportMat", 0, map[string]any{"id": st.ID})

	// The whole match, as the stored draft holds it, posted as {header, actions}.
	doc := st.Annotated.Document
	status, body = gesture(t, ts, testTenant, "/v1/transcriptions.import", 0, map[string]any{
		"header": doc.Header, "actions": doc.Actions, "next_score": doc.NextScore,
	})
	again := decodeState(t, status, body)
	if again.ID == st.ID || len(again.Annotated.Document.Actions) != len(doc.Actions) {
		t.Fatalf("import of a document: %s", body)
	}
	_, gotMAT := gesture(t, ts, testTenant, "/v1/transcriptions.exportMat", 0, map[string]any{"id": again.ID})
	var w, g transcriptionMATResp
	if json.Unmarshal(wantMAT, &w) != nil || json.Unmarshal(gotMAT, &g) != nil || w.Text == "" || w.Text != g.Text {
		t.Fatalf("the document's draft renders another .mat:\n%s\n---\n%s", g.Text, w.Text)
	}

	for name, req := range map[string]any{
		"neither":      map[string]any{},
		"both":         map[string]any{"mat": string(mat), "actions": doc.Actions},
		"header alone": map[string]any{"header": doc.Header},
		"unknown kind": map[string]any{"header": map[string]any{"match_length": 3}, "actions": []any{map[string]any{"side": 0, "kind": "teleport"}}},
		"not a .mat":   map[string]any{"mat": "hello"},
	} {
		if status, body := gesture(t, ts, testTenant, "/v1/transcriptions.import", 0, req); status != http.StatusBadRequest {
			t.Errorf("%s: status %d: %s", name, status, body)
		}
	}
	if n := countDrafts(t, ts); n != 2 {
		t.Fatalf("refused imports must write nothing: %d drafts", n)
	}
}

// A request resent under the same Idempotency-Key opens one draft, and the
// second answer is the first one's.
func TestTranscriptionImportIdempotent(t *testing.T) {
	ts := newTranscriptionServer(t, 0)
	mat, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gnubg_selfplay_drops_7p.mat"))
	if err != nil {
		t.Fatal(err)
	}
	buf, _ := json.Marshal(map[string]any{"mat": string(mat)})
	var ids []int64
	for range 2 {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/transcriptions.import", bytes.NewReader(buf))
		req.Header.Set(middleware.TenantHeader, testTenant)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "import-once")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		_, _ = out.ReadFrom(resp.Body)
		resp.Body.Close()
		ids = append(ids, decodeState(t, resp.StatusCode, out.Bytes()).ID)
	}
	if ids[0] != ids[1] {
		t.Fatalf("the replayed request answered draft %d, the first %d", ids[1], ids[0])
	}
	if n := countDrafts(t, ts); n != 1 {
		t.Fatalf("%d drafts under one Idempotency-Key", n)
	}
}

func countDrafts(t *testing.T, ts *httptest.Server) int {
	t.Helper()
	status, body := gesture(t, ts, testTenant, "/v1/transcriptions.list", 0, map[string]any{})
	var drafts []json.RawMessage
	if err := json.Unmarshal(body, &drafts); status != http.StatusOK || err != nil {
		t.Fatalf("transcriptions.list: %d %s", status, body)
	}
	return len(drafts)
}
