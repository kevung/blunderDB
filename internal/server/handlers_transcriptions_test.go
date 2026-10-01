package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

func newTranscriptionServer(t *testing.T, ttl time.Duration) *httptest.Server {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, Transcription: true, TranscriptionTTL: ttl})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// gesture POSTs a transcription call with If-Match (rev 0: none) under a
// tenant and returns the status and the raw body.
func gesture(t *testing.T, ts *httptest.Server, tenant, path string, rev int64, body any) (int, []byte) {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(buf))
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	if rev != 0 {
		req.Header.Set("If-Match", `"`+strconv.FormatInt(rev, 10)+`"`)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp.Body)
	return resp.StatusCode, out.Bytes()
}

func decodeState(t *testing.T, status int, body []byte) *transcription.State {
	t.Helper()
	if status != http.StatusOK {
		t.Fatalf("status %d: %s", status, body)
	}
	var st transcription.State
	if err := json.Unmarshal(body, &st); err != nil {
		t.Fatalf("decode state: %v (%s)", err, body)
	}
	return &st
}

// gestureState is gesture decoded as a State, failing on any other status.
func gestureState(t *testing.T, ts *httptest.Server, tenant, path string, rev int64, body any) *transcription.State {
	t.Helper()
	status, raw := gesture(t, ts, tenant, path, rev, body)
	return decodeState(t, status, raw)
}

func createDraft(t *testing.T, ts *httptest.Server, tenant string) *transcription.State {
	t.Helper()
	status, body := gesture(t, ts, tenant, "/v1/transcriptions.create", 0, map[string]any{"header": transcript.Header{MatchLength: 7}})
	return decodeState(t, status, body)
}

// typeAction plays one checker Action, each gesture naming the revision the
// previous one returned.
func typeAction(t *testing.T, ts *httptest.Server, tenant string, st *transcription.State, d1, d2 int) *transcription.State {
	t.Helper()
	for _, g := range []transcript.Gesture{
		{Kind: transcript.GestureEnterDie, Die: d1},
		{Kind: transcript.GestureEnterDie, Die: d2},
		{Kind: transcript.GestureSelectCandidate, Candidate: 0},
		{Kind: transcript.GestureValidate},
	} {
		status, body := gesture(t, ts, tenant, "/v1/transcriptions.apply", st.Revision,
			map[string]any{"id": st.ID, "sessionId": st.SessionID, "gesture": g})
		st = decodeState(t, status, body)
	}
	return st
}

// A gesture without If-Match is refused before it reaches the draft.
func TestTranscriptionGestureWithoutIfMatchIs428(t *testing.T) {
	ts := newTranscriptionServer(t, 0)
	st := createDraft(t, ts, testTenant)
	for _, path := range []string{"/v1/transcriptions.apply", "/v1/transcriptions.undo", "/v1/transcriptions.redo",
		"/v1/transcriptions.finish", "/v1/transcriptions.abandon"} {
		status, body := gesture(t, ts, testTenant, path, 0,
			map[string]any{"id": st.ID, "gesture": transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 6}})
		if status != http.StatusPreconditionRequired {
			t.Errorf("%s without If-Match: status %d, want 428 (%s)", path, status, body)
		}
	}
	got := gestureState(t, ts, testTenant, "/v1/transcriptions.get", 0, map[string]any{"id": st.ID})
	if got.Revision != st.Revision {
		t.Errorf("a refused gesture moved the revision: %d, want %d", got.Revision, st.Revision)
	}
}

// Two gestures that change the document race on the same revision: one
// lands, the other is 409 with the fresh state — revision, document and the
// session's Cursor.
func TestTranscriptionConcurrentApplyOne409(t *testing.T) {
	ts := newTranscriptionServer(t, 0)
	st := createDraft(t, ts, testTenant)

	statuses := make([]int, 2)
	bodies := make([][]byte, 2)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range statuses {
		wg.Go(func() {
			<-start
			statuses[i], bodies[i] = gesture(t, ts, testTenant, "/v1/transcriptions.apply", st.Revision,
				map[string]any{"id": st.ID, "sessionId": st.SessionID,
					"gesture": transcript.Gesture{Kind: transcript.GestureSetLength, MatchLength: 3 + 2*i, HasLength: true}})
		})
	}
	close(start)
	wg.Wait()

	ok, conflict := 0, -1
	for i, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict = i
		default:
			t.Fatalf("status %d: %s", s, bodies[i])
		}
	}
	if ok != 1 || conflict < 0 {
		t.Fatalf("statuses %v, want one 200 and one 409", statuses)
	}
	var env struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bodies[conflict], &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeConflict || env.Error.Details["revision"] != float64(st.Revision+1) {
		t.Errorf("409 envelope = %+v, want code conflict and revision %d", env.Error, st.Revision+1)
	}
	fresh, _ := env.Error.Details["state"].(map[string]any)
	if fresh == nil || fresh["revision"] != float64(st.Revision+1) || fresh["sessionId"] != st.SessionID {
		t.Fatalf("409 must carry the fresh state with its revision and session, got %v", env.Error.Details["state"])
	}
	ann, _ := fresh["annotated"].(map[string]any)
	doc, _ := ann["document"].(map[string]any)
	if _, ok := doc["cursor"]; !ok {
		t.Errorf("the fresh state must carry the document and its Cursor, got %v", fresh["annotated"])
	}
}

// Over HTTP a session gesture names its session: none is 400, an unknown one
// 410, so a client whose session expired learns it instead of being handed a
// new one silently, and nobody closes a session that is not theirs.
func TestTranscriptionGesturesRequireTheirSession(t *testing.T) {
	ts := newTranscriptionServer(t, 0)
	// One Action, so that a finish naming no session would have something to write.
	st := typeAction(t, ts, testTenant, createDraft(t, ts, testTenant), 6, 3)
	die := transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 6}
	for _, path := range []string{"/v1/transcriptions.apply", "/v1/transcriptions.undo",
		"/v1/transcriptions.redo", "/v1/transcriptions.close", "/v1/transcriptions.finish"} {
		if status, body := gesture(t, ts, testTenant, path, st.Revision,
			map[string]any{"id": st.ID, "gesture": die}); status != http.StatusBadRequest {
			t.Errorf("%s without sessionId: status %d, want 400 (%s)", path, status, body)
		}
		if status, body := gesture(t, ts, testTenant, path, st.Revision,
			map[string]any{"id": st.ID, "sessionId": "not-a-session", "gesture": die}); status != http.StatusGone {
			t.Errorf("%s with an unknown sessionId: status %d, want 410 (%s)", path, status, body)
		}
	}
	// The live session survived all of the above.
	gestureState(t, ts, testTenant, "/v1/transcriptions.apply", st.Revision,
		map[string]any{"id": st.ID, "sessionId": st.SessionID, "gesture": die})
}

// A session past its TTL answers 410; reopening gives a session on the
// whole document and typing resumes: nothing typed is lost.
func TestTranscriptionExpiredSession410ThenReopen(t *testing.T) {
	const ttl = 300 * time.Millisecond
	ts := newTranscriptionServer(t, ttl)
	st := typeAction(t, ts, testTenant, createDraft(t, ts, testTenant), 6, 3)

	time.Sleep(ttl + 100*time.Millisecond)
	status, body := gesture(t, ts, testTenant, "/v1/transcriptions.apply", st.Revision,
		map[string]any{"id": st.ID, "sessionId": st.SessionID, "gesture": transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 5}})
	if status != http.StatusGone {
		t.Fatalf("gesture on an expired session: status %d, want 410 (%s)", status, body)
	}

	re := gestureState(t, ts, testTenant, "/v1/transcriptions.open", 0, map[string]any{"id": st.ID})
	if n := len(re.Annotated.Document.Actions); n != 1 {
		t.Fatalf("reopened with %d actions, want 1", n)
	}
	if re.Revision != st.Revision || re.SessionID == st.SessionID {
		t.Fatalf("reopened at revision %d, session %q; want %d and a new session", re.Revision, re.SessionID, st.Revision)
	}
	re = typeAction(t, ts, testTenant, re, 5, 2)
	if n := len(re.Annotated.Document.Actions); n != 2 {
		t.Fatalf("after the reopen: %d actions, want 2", n)
	}
}

// Without --transcription the gestures answer as absent routes; the reads
// are still served.
func TestTranscriptionWritesAbsentWithoutTheFlag(t *testing.T) {
	ts := newTestServer(t)
	for _, path := range []string{"/v1/transcriptions.create", "/v1/transcriptions.open", "/v1/transcriptions.apply",
		"/v1/transcriptions.undo", "/v1/transcriptions.redo", "/v1/transcriptions.close",
		"/v1/transcriptions.finish", "/v1/transcriptions.abandon", "/v1/transcriptions.editMatch"} {
		if status, _ := gesture(t, ts, testTenant, path, 1, map[string]any{"id": 1}); status != http.StatusNotFound {
			t.Errorf("%s without the flag: status %d, want 404", path, status)
		}
	}
	if status, body := gesture(t, ts, testTenant, "/v1/transcriptions.list", 0, struct{}{}); status != http.StatusOK {
		t.Errorf("transcriptions.list without the flag: status %d, want 200 (%s)", status, body)
	}
}

// get carries the revision as ETag and answers 304 to it; finish writes the
// Match and the draft is gone.
func TestTranscriptionGetETagAndFinish(t *testing.T) {
	ts := newTranscriptionServer(t, 0)
	st := typeAction(t, ts, testTenant, createDraft(t, ts, testTenant), 6, 3)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/transcriptions.get", bytes.NewReader([]byte(`{"id":`+strconv.FormatInt(st.ID, 10)+`}`)))
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("If-None-Match", revisionETag(st.Revision))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified || resp.Header.Get("ETag") != revisionETag(st.Revision) {
		t.Fatalf("get with the current ETag: status %d, ETag %q; want 304, %s", resp.StatusCode, resp.Header.Get("ETag"), revisionETag(st.Revision))
	}

	status, body := gesture(t, ts, testTenant, "/v1/transcriptions.finish", st.Revision, map[string]any{"id": st.ID, "sessionId": st.SessionID})
	if status != http.StatusOK {
		t.Fatalf("finish: status %d (%s)", status, body)
	}
	var res transcription.SaveResult
	if err := json.Unmarshal(body, &res); err != nil || res.MatchID == 0 || res.Games != 1 {
		t.Fatalf("finish result = %+v, %v", res, err)
	}
	if status, _ := gesture(t, ts, testTenant, "/v1/transcriptions.get", 0, map[string]any{"id": st.ID}); status != http.StatusNotFound {
		t.Errorf("get after finish: status %d, want 404", status)
	}
}

// Under `call` every invocation is its own process: a gesture naming no
// session uses the draft's live one or opens one.
func TestTranscriptionSessionPerCallNeedsNoSession(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, Transcription: true, SessionPerCall: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	d := createDraft(t, ts, testTenant)
	gestureState(t, ts, testTenant, "/v1/transcriptions.apply", d.Revision,
		map[string]any{"id": d.ID, "gesture": transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 6}})
	if status, body := gesture(t, ts, testTenant, "/v1/transcriptions.close", 0, map[string]any{"id": d.ID}); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("close without sessionId under call: status %d (%s)", status, body)
	}
}
