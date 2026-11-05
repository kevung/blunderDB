package gui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/syncremote"
)

type fixedSync struct {
	server string
	auto   bool
}

func (f fixedSync) SyncServerAddress() string { return f.server }
func (f fixedSync) SyncAutoStart() bool       { return f.auto }

func syncApp(t *testing.T, server string) (*App, string) {
	t.Helper()
	keyring.MockInit()
	db := database.NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "lib.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	a := NewApp(db)
	a.sync.source = fixedSync{server: server}
	a.sync.path = filepath.Join(t.TempDir(), "sync-state.json")
	return a, a.sync.path
}

// An offline run says so, keeps the library usable and writes no token anywhere.
func TestSyncOfflineIsSaidAndTheTokenStaysInTheKeyring(t *testing.T) {
	const tok = "bdb_very-secret"
	down := httptest.NewServer(http.NotFoundHandler())
	url := down.URL
	down.Close()
	a, state := syncApp(t, url)
	if err := a.SyncSetToken(url, tok); err != nil {
		t.Fatal(err)
	}
	a.sync.running = true
	a.syncRun(context.Background())
	st := a.SyncStatus()
	if st.Running || st.Error == nil || st.Error.Kind != "unavailable" {
		t.Fatalf("status = %+v, want an unavailable error and no run under way", st)
	}
	if strings.Contains(st.Error.Message, tok) {
		t.Errorf("the error message repeats the token")
	}
	if !st.HasToken || st.LastSuccess != "" {
		t.Errorf("hasToken=%v lastSuccess=%q", st.HasToken, st.LastSuccess)
	}
	if raw, err := os.ReadFile(state); err == nil && strings.Contains(string(raw), tok) {
		t.Errorf("the token is in the state file")
	}
	if _, err := a.db.CreateCollection("still writable", ""); err != nil {
		t.Errorf("the library is blocked after a failed sync: %v", err)
	}
}

func TestSyncWithoutTokenOrWithABadAddressSaysWhy(t *testing.T) {
	a, _ := syncApp(t, "https://sync.example.org/api")
	a.syncRun(context.Background())
	if st := a.SyncStatus(); st.Error == nil || st.Error.Kind != "no_token" {
		t.Errorf("no token: %+v", st.Error)
	}
	if err := a.SyncSetToken("http://remote.example.org", "x"); err == nil {
		t.Errorf("a token for a clear-text remote address was stored")
	}
	a.sync.source = fixedSync{server: "nonsense"}
	a.syncRun(context.Background())
	if st := a.SyncStatus(); st.Error == nil || st.Error.Kind != "address" {
		t.Errorf("bad address: %+v", st.Error)
	}
}

// A successful run records its date and keeps the cursors per library and server.
func TestSyncSuccessIsRecordedAndSurvivesInTheStateFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"cursor":{"seq":0},"more":false,"records":[]}`))
	}))
	defer srv.Close()
	a, state := syncApp(t, srv.URL)
	if err := a.SyncSetToken(srv.URL, "tok"); err != nil {
		t.Fatal(err)
	}
	a.syncRun(context.Background())
	st := a.SyncStatus()
	if st.Error != nil || st.LastSuccess == "" {
		t.Fatalf("status = %+v", st)
	}
	if fi, err := os.Stat(state); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("state file: %v, %v", fi, err)
	}
	// A new App reads the date back.
	b := NewApp(a.db)
	b.sync.source, b.sync.path = a.sync.source, state
	if got := b.SyncStatus().LastSuccess; got != st.LastSuccess {
		t.Errorf("after a restart lastSuccess = %q, want %q", got, st.LastSuccess)
	}
}

// The window starts with an empty in-memory library: a startup run waits for the real one.
func TestSyncAtStartupWaitsForTheRealLibrary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"cursor":{"seq":0},"more":false,"records":[]}`))
	}))
	defer srv.Close()
	keyring.MockInit()
	mem := database.NewDatabase()
	if err := mem.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mem.Close() })
	a := NewApp(mem)
	a.sync.source = fixedSync{server: srv.URL, auto: true}
	a.sync.path = filepath.Join(t.TempDir(), "sync-state.json")
	if err := a.SyncSetToken(srv.URL, "tok"); err != nil {
		t.Fatal(err)
	}
	if a.syncAtStartup() {
		t.Fatal("a startup run began on the in-memory library")
	}
	if err := mem.SetupDatabase(filepath.Join(t.TempDir(), "real.db")); err != nil {
		t.Fatal(err)
	}
	if !a.syncAtStartup() {
		t.Fatal("no startup run on the real library")
	}
	a.sync.wg.Wait()
	if st := a.SyncStatus(); st.Error != nil || st.LastSuccess == "" {
		t.Errorf("status = %+v", st)
	}
	a.sync.source = fixedSync{server: srv.URL, auto: false}
	if a.syncAtStartup() {
		t.Error("a startup run began with the switch off")
	}
}

func TestSyncErrorKindsFromTheServersAnswers(t *testing.T) {
	answer := func(status int, body string) error {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		defer srv.Close()
		c, err := syncremote.NewClient(srv.URL, "bdb_x", srv.Client())
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Import(context.Background(), nil)
		return fmt.Errorf("sending: %w", err)
	}
	if e := syncError(answer(402, `{"error":{"code":"allowance_exceeded","message":"Quota atteint."}}`)); e.Kind != "allowance" || e.Message != "Quota atteint." {
		t.Errorf("allowance = %+v, want the server's words as they are", e)
	}
	if e := syncError(answer(402, `{"error":{"code":"allowance_exceeded","message":"Plan limit","axis":"positions","allowed":150,"held":140,"new":20}}`)); e.Kind != "allowance" || e.Allowance == nil || *e.Allowance != (syncremote.AllowanceFigures{Axis: "positions", Allowed: 150, Held: 140, New: 20}) {
		t.Errorf("allowance with figures = %+v, want them carried for the window to word", e)
	}
	if e := syncError(answer(422, `{"error":{"code":"invalid_records","message":"x"}}`)); e.Kind != "invalid_records" {
		t.Errorf("422 = %+v, want a refusal of content, not an unreachable server", e)
	}
	if e := syncError(answer(403, `{"error":{"code":"no_journal","message":"no journal here"}}`)); e.Kind != "server_refusal" {
		t.Errorf("403 with the server's own code = %+v, want its refusal and not a token problem", e)
	}
	if e := syncError(answer(403, `{"error":{"code":"forbidden","message":"x"}}`)); e.Kind != "refused" {
		t.Errorf("403 forbidden = %+v, want the token refused", e)
	}
	if e := syncError(answer(413, `{"error":{"code":"payload_too_large","message":"x"}}`)); e.Kind != "too_large" {
		t.Errorf("413 with the server's code = %+v", e)
	}
	if e := syncError(answer(413, "<html>413</html>")); e.Kind != "blocked" {
		t.Errorf("413 from something else = %+v, want blocked: no record is to blame", e)
	}
	if e := syncError(answer(409, `{"error":{"code":"tool_locked_during_match","message":"x"}}`)); e.Kind != "tool_locked" {
		t.Errorf("locked tool = %+v", e)
	}
}

// The records set aside survive in the state file and reach the window with their type and identifier.
func TestSyncSkippedRecordsAreKeptAndShownAfterARestart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"cursor":{"seq":0},"more":false,"records":[]}`))
	}))
	defer srv.Close()
	a, state := syncApp(t, srv.URL)
	addr, _ := syncremote.NormalizeAddress(srv.URL)
	entries := map[string]syncEntry{syncKey(a.db, addr): {Cursors: syncremote.Cursors{Skipped: []syncremote.Skip{
		{Direction: syncremote.DirectionSent, Family: "match", UUID: "u-1", Reason: syncremote.ReasonTooLarge},
	}}}}
	if err := a.sync.store(entries); err != nil {
		t.Fatal(err)
	}
	b := NewApp(a.db)
	b.sync.source, b.sync.path = a.sync.source, state
	got := b.SyncStatus().Skipped
	if len(got) != 1 || got[0].Family != "match" || got[0].UUID != "u-1" {
		t.Fatalf("skipped = %+v", got)
	}
}

// A revoked token is said, the library keeps its cursors, and removing the token
// leaves nothing in the keyring; the token is in no log line, status or state file.
func TestSyncARevokedTokenIsSaidAndTheTokenIsInNoLog(t *testing.T) {
	const tok = "bdb_revoked-secret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()
	a, state := syncApp(t, srv.URL)
	if err := a.SyncSetToken(srv.URL, tok); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	a.sync.running = true
	a.syncRun(context.Background())
	st := a.SyncStatus()
	if st.Error == nil || st.Error.Kind != "refused" || !st.HasToken || st.LastSuccess != "" {
		t.Fatalf("status = %+v, want the token refused and kept until removed", st)
	}
	if !strings.Contains(logs.String(), "sync failed") {
		t.Fatalf("the failure is not logged, so the log check proves nothing: %q", logs.String())
	}
	shown, _ := json.Marshal(st)
	raw, _ := os.ReadFile(state)
	for where, text := range map[string]string{"log": logs.String(), "status": string(shown), "state file": string(raw)} {
		if strings.Contains(text, tok) {
			t.Errorf("the token is in the %s", where)
		}
	}

	if err := a.SyncSetToken(srv.URL, ""); err != nil {
		t.Fatal(err)
	}
	addr, _ := syncremote.NormalizeAddress(srv.URL)
	if _, err := keyring.Get(syncKeyringService, addr); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("after removal the keyring answers %v", err)
	}
	if st := a.SyncStatus(); st.HasToken {
		t.Fatalf("status still says a token is stored")
	}
	a.sync.running = true
	a.syncRun(context.Background())
	if st := a.SyncStatus(); st.Error == nil || st.Error.Kind != "no_token" {
		t.Fatalf("a run without the token = %+v, want no_token and no call", st.Error)
	}
}
