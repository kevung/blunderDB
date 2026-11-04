package syncremote_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/syncremote"
)

const token = "bdb_secret-token-value"

func openLibrary(t *testing.T) *database.Database {
	t.Helper()
	d := database.NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "lib.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func names(t *testing.T, d *database.Database) []string {
	t.Helper()
	cs, err := d.GetAllCollections()
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func has(t *testing.T, d *database.Database, want ...string) {
	t.Helper()
	got := strings.Join(names(t, d), ",")
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("collections %q lack %q", got, w)
		}
	}
}

// server is a remote library behind the two /v1 changeset routes, recording
// what each call carried.
type server struct {
	*httptest.Server
	lib *database.Database

	mu           sync.Mutex
	imported     []int            // records per import call
	exported     []storage.Cursor // cursor of each export call
	failImportAt int              // 1-based import call that answers 503, 0 never
	stale        func(storage.Cursor) bool
	calls        int
}

func newServer(t *testing.T, lib *database.Database) *server {
	t.Helper()
	s := &server{lib: lib}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, `{"error":{"code":"unauthorized","message":"bad token"}}`, http.StatusUnauthorized)
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.calls++
		switch r.URL.Path {
		case "/api/v1/changeset.export":
			var req struct {
				Cursor storage.Cursor `json:"cursor"`
				Limit  int            `json:"limit"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			s.exported = append(s.exported, req.Cursor)
			if s.stale != nil && s.stale(req.Cursor) {
				http.Error(w, `{"error":{"code":"gone","message":"journal restored"}}`, http.StatusGone)
				return
			}
			set, err := database.JournalOf(lib).Export(r.Context(), req.Cursor, req.Limit)
			if err != nil {
				http.Error(w, `{"error":{"code":"internal","message":"x"}}`, http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(set)
		case "/api/v1/changeset.import":
			var req struct {
				Records []changeset.Record `json:"records"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			s.imported = append(s.imported, len(req.Records))
			if s.failImportAt == len(s.imported) {
				http.Error(w, `{"error":{"code":"unavailable","message":"busy"}}`, http.StatusServiceUnavailable)
				return
			}
			rep, err := database.JournalOf(lib).Import(r.Context(), changeset.Set{Records: req.Records})
			if err != nil {
				http.Error(w, `{"error":{"code":"internal","message":"x"}}`, http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(rep)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func client(t *testing.T, s *server, tok string) *syncremote.Client {
	t.Helper()
	c, err := syncremote.NewClient(s.URL+"/api", tok, s.Client())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func create(t *testing.T, d *database.Database, name string) int64 {
	t.Helper()
	id, err := d.CreateCollection(name, "")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestTwoPassesLevelBothLibrariesThenSendOnlyWhatChanged(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	create(t, a, "Openings")
	create(t, b, "Endgames")
	srv := newServer(t, b)
	c := client(t, srv, token)

	cur, res, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	has(t, a, "Openings", "Endgames")
	has(t, b, "Openings", "Endgames")
	if res.Sent.Records == 0 || res.Received.Records == 0 {
		t.Errorf("first pass moved %+v", res)
	}

	// Second pass, one rename on the desktop: only that travels up, and what the
	// first pass brought down is not applied twice.
	srv.mu.Lock()
	srv.imported = nil
	srv.mu.Unlock()
	cs, _ := a.GetAllCollections()
	for _, col := range cs {
		if col.Name == "Openings" {
			if err := a.UpdateCollection(col.ID, "Opening book", ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	cur2, res2, err := syncremote.Run(context.Background(), database.JournalOf(a), c, cur, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	has(t, b, "Opening book")
	if res2.Sent.Modified != 1 {
		t.Errorf("second pass: server modified %d on receipt, want 1 (%+v)", res2.Sent.Modified, res2)
	}
	if res2.Received.Added+res2.Received.Modified != 0 {
		t.Errorf("second pass changed this library on receipt: %+v", res2.Received)
	}
	if cur2 == cur {
		t.Errorf("cursors did not move after a change")
	}
}

func TestReplayOfTheSamePassChangesNothing(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	create(t, a, "One")
	srv := newServer(t, b)
	c := client(t, srv, token)
	if _, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{}); err != nil {
		t.Fatal(err)
	}
	// Lost cursors: everything is sent and received again.
	_, res, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Sent.Added + res.Sent.Modified + res.Received.Added + res.Received.Modified; got != 0 {
		t.Errorf("a replay changed %d objects: %+v", got, res)
	}
	if n := len(names(t, a)); n != 1 {
		t.Errorf("replay left %d collections, want 1", n)
	}
}

func TestPagesAreSettledOneByOneAndAResumeSkipsThem(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	for _, n := range []string{"A1", "A2", "A3"} {
		create(t, a, n)
	}
	srv := newServer(t, b)
	srv.failImportAt = 2
	c := client(t, srv, token)

	var saved []syncremote.Cursors
	opt := syncremote.Options{Page: 1, Save: func(c syncremote.Cursors) error { saved = append(saved, c); return nil }}
	cur, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, opt)
	if err == nil || !strings.Contains(err.Error(), "does not answer") {
		t.Fatalf("a failing page: err = %v, want ErrUnavailable", err)
	}
	if len(saved) != 1 {
		t.Fatalf("%d pages settled before the failure, want 1", len(saved))
	}
	if cur != saved[0] {
		t.Errorf("returned cursors differ from the last saved ones")
	}

	srv.mu.Lock()
	srv.failImportAt, srv.imported = 0, nil
	srv.mu.Unlock()
	if _, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, cur, opt); err != nil {
		t.Fatal(err)
	}
	has(t, b, "A1", "A2", "A3")
	total := 0
	for _, n := range srv.imported {
		total += n
	}
	if total >= 3 {
		t.Errorf("the resume sent %d records, the first page should not be sent again", total)
	}
}

func TestStaleRemoteCursorRestartsFromTheBeginning(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	create(t, b, "Server side")
	srv := newServer(t, b)
	c := client(t, srv, token)
	cur, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	create(t, b, "Later")
	srv.stale = func(after storage.Cursor) bool { return !after.IsStart() }
	if _, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, cur, syncremote.Options{}); err != nil {
		t.Fatalf("a refused cursor must restart from the start: %v", err)
	}
	has(t, a, "Server side", "Later")
}

func TestStaleLocalCursorRestartsFromTheBeginning(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	create(t, a, "Mine")
	srv := newServer(t, b)
	c := client(t, srv, token)
	cur := syncremote.Cursors{Sent: storage.Cursor{Seq: 5, Carried: "another journal"}}
	if _, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, cur, syncremote.Options{}); err != nil {
		t.Fatalf("a stale local cursor must restart: %v", err)
	}
	has(t, b, "Mine")
}

func TestARefusedTokenIsSaidAndNeverRepeated(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	create(t, a, "Mine")
	srv := newServer(t, b)
	const wrong = "bdb_wrong-token"
	c := client(t, srv, wrong)
	cur, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if err == nil || !strings.Contains(err.Error(), "refuses the token") {
		t.Fatalf("err = %v, want a refused token", err)
	}
	if strings.Contains(err.Error(), wrong) {
		t.Errorf("the error repeats the token: %v", err)
	}
	if cur != (syncremote.Cursors{}) {
		t.Errorf("cursors moved on a refused call: %+v", cur)
	}
	if n := len(names(t, b)); n != 0 {
		t.Errorf("the server received %d collections without a valid token", n)
	}
}

func TestAnUnreachableServerIsSaidAndTheLibraryStaysUsable(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	create(t, a, "Mine")
	srv := newServer(t, b)
	c := client(t, srv, token)
	srv.Close()
	_, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if err == nil || !strings.Contains(err.Error(), "does not answer") {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("the error repeats the token")
	}
	create(t, a, "Still writable")
	has(t, a, "Mine", "Still writable")
}

func TestAddress(t *testing.T) {
	for addr, ok := range map[string]bool{
		"https://example.org/api":     true,
		"https://example.org/api/ ":   true,
		"http://localhost:8080":       true,
		"http://127.0.0.1:9":          true,
		"http://example.org":          false,
		"ftp://example.org":           false,
		"https://user:pw@example.org": false,
		"https://example.org/?t=x":    false,
		"example.org":                 false,
		"":                            false,
	} {
		_, err := syncremote.NormalizeAddress(addr)
		if (err == nil) != ok {
			t.Errorf("NormalizeAddress(%q) = %v, want ok=%v", addr, err, ok)
		}
	}
	got, _ := syncremote.NormalizeAddress(" https://example.org/api/ ")
	if got != "https://example.org/api" {
		t.Errorf("normalized to %q", got)
	}
}

func TestARedirectIsNeverFollowedSoTheTokenGoesNowhereElse(t *testing.T) {
	a := openLibrary(t)
	create(t, a, "Mine")
	var leaked bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer other.Close()
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer redirecting.Close()
	c, err := syncremote.NewClient(redirecting.URL, token, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if err == nil || !strings.Contains(err.Error(), "redirects") {
		t.Fatalf("err = %v, want the redirect reported", err)
	}
	if leaked {
		t.Errorf("the redirect was followed")
	}
}

// A record of a family this side does not carry is reported, not passed as done.
func TestWhatTheOtherSideCouldNotApplyIsCounted(t *testing.T) {
	a := openLibrary(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"cursor":{"seq":1},"more":false,"records":[{"seq":1,"uuid":"u1","family":"hologram","op":"create","changed_at":"2026-01-01T00:00:00.000Z","payload":{}}]}`))
	}))
	defer srv.Close()
	c, _ := syncremote.NewClient(srv.URL, token, nil)
	_, res, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Received.Records != 1 || res.Received.Unsupported != 1 {
		t.Errorf("received = %+v, want the record counted as unsupported", res.Received)
	}
}
