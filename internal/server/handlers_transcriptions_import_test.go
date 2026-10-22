package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// transcriptions.import opens a draft on a .mat or a document, refuses one it
// cannot read with a 400, and is a gesture: absent without --transcription.
func TestTranscriptionImport(t *testing.T) {
	ts := newTranscriptionServer(t, 0)
	mat, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gnubg_selfplay_drops_7p.mat"))
	if err != nil {
		t.Fatal(err)
	}
	status, body := gesture(t, ts, testTenant, "/v1/transcriptions.import", 0, map[string]any{"mat": string(mat)})
	st := decodeState(t, status, body)
	if st.SessionID == "" || len(st.Annotated.Document.Actions) == 0 {
		t.Fatalf("import of a .mat: %s", body)
	}

	doc := transcript.Document{Header: transcript.Header{MatchLength: 3}, Actions: st.Annotated.Document.Actions[:4]}
	status, body = gesture(t, ts, testTenant, "/v1/transcriptions.import", 0, map[string]any{"document": doc})
	if again := decodeState(t, status, body); again.ID == st.ID || len(again.Annotated.Document.Actions) != 4 {
		t.Fatalf("import of a document: %s", body)
	}

	for name, req := range map[string]any{
		"neither":      map[string]any{},
		"both":         map[string]any{"mat": string(mat), "document": doc},
		"unknown kind": map[string]any{"document": map[string]any{"header": map[string]any{"match_length": 3}, "actions": []any{map[string]any{"side": 0, "kind": "teleport"}}}},
		"not a .mat":   map[string]any{"mat": "hello"},
	} {
		if status, body := gesture(t, ts, testTenant, "/v1/transcriptions.import", 0, req); status != http.StatusBadRequest {
			t.Errorf("%s: status %d: %s", name, status, body)
		}
	}
	status, body = gesture(t, ts, testTenant, "/v1/transcriptions.list", 0, map[string]any{})
	var drafts []json.RawMessage
	if err := json.Unmarshal(body, &drafts); status != http.StatusOK || err != nil || len(drafts) != 2 {
		t.Fatalf("refused imports must write nothing: %s", body)
	}
}
