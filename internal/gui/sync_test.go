package gui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
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
