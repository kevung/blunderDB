//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go.

package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// A draft belongs to its tenant: another tenant neither lists it, reads it,
// types into its session nor into the draft itself.
func TestTranscriptionTenantIsolation(t *testing.T) {
	ts := newPostgresTestServerWith(t, func(o *Options) { o.Transcription = true })
	st := typeAction(t, ts, "1", createDraft(t, ts, "1"), 6, 3)

	status, body := gesture(t, ts, "2", "/v1/transcriptions.list", 0, struct{}{})
	var list []json.RawMessage
	if status != http.StatusOK || json.Unmarshal(body, &list) != nil || len(list) != 0 {
		t.Fatalf("tenant 2 list: status %d, %s; want 200 and no draft", status, body)
	}
	if status, _ := gesture(t, ts, "2", "/v1/transcriptions.get", 0, map[string]any{"id": st.ID}); status != http.StatusNotFound {
		t.Errorf("tenant 2 get: status %d, want 404", status)
	}
	die := transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 4}
	if status, _ := gesture(t, ts, "2", "/v1/transcriptions.apply", st.Revision,
		map[string]any{"id": st.ID, "sessionId": st.SessionID, "gesture": die}); status != http.StatusGone {
		t.Errorf("tenant 2 naming tenant 1's session: status %d, want 410", status)
	}
	if status, _ := gesture(t, ts, "2", "/v1/transcriptions.apply", st.Revision,
		map[string]any{"id": st.ID, "gesture": die}); status != http.StatusNotFound {
		t.Errorf("tenant 2 typing into tenant 1's draft: status %d, want 404", status)
	}
	if status, _ := gesture(t, ts, "2", "/v1/transcriptions.abandon", st.Revision, map[string]any{"id": st.ID}); status != http.StatusNotFound {
		t.Errorf("tenant 2 abandoning tenant 1's draft: status %d, want 404", status)
	}

	got := gestureState(t, ts, "1", "/v1/transcriptions.get", 0, map[string]any{"id": st.ID})
	if got.Revision != st.Revision || len(got.Annotated.Document.Actions) != 1 {
		t.Fatalf("tenant 1's draft moved: revision %d, %d actions; want %d, 1", got.Revision, len(got.Annotated.Document.Actions), st.Revision)
	}
	if status, body := gesture(t, ts, "1", "/v1/transcriptions.finish", st.Revision, map[string]any{"id": st.ID, "sessionId": st.SessionID}); status != http.StatusOK {
		t.Fatalf("tenant 1 finish: status %d (%s)", status, body)
	}
}
