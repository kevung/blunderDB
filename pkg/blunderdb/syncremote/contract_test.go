package syncremote_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/syncremote"
)

// scripted is a Remote whose answers a test dictates; it merges what it
// accepts into a real library so a run can be checked end to end.
type scripted struct {
	lib      *database.Database
	imports  [][]changeset.Record
	exports  []storage.Cursor
	importFn func(call int, recs []changeset.Record) error
	exportFn func(call int, after storage.Cursor, limit int) (changeset.Set, bool, error)
	pageFn   func(call int, after storage.Cursor, limit int) (syncremote.Page, bool, error)
	limits   []int
}

func (s *scripted) Import(ctx context.Context, recs []changeset.Record) (changeset.Report, error) {
	s.imports = append(s.imports, recs)
	if s.importFn != nil {
		if err := s.importFn(len(s.imports), recs); err != nil {
			return changeset.Report{}, err
		}
	}
	return database.JournalOf(s.lib).Import(ctx, changeset.Set{Records: recs})
}

func (s *scripted) Export(ctx context.Context, after storage.Cursor, limit int) (syncremote.Page, error) {
	s.exports = append(s.exports, after)
	s.limits = append(s.limits, limit)
	if s.pageFn != nil {
		if p, ok, err := s.pageFn(len(s.exports), after, limit); ok || err != nil {
			return p, err
		}
	}
	if s.exportFn != nil {
		if set, ok, err := s.exportFn(len(s.exports), after, limit); ok || err != nil {
			return syncremote.Page{Set: set}, err
		}
	}
	set, err := database.JournalOf(s.lib).Export(ctx, after, limit)
	return syncremote.Page{Set: set}, err
}

func tooLarge(max int) func(int, []changeset.Record) error {
	return func(_ int, recs []changeset.Record) error {
		if len(recs) > max {
			return fmt.Errorf("%w: payload_too_large", syncremote.ErrTooLarge)
		}
		return nil
	}
}

func fill(t *testing.T, d *database.Database, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		create(t, d, fmt.Sprintf("C%d", i))
	}
}

func run(t *testing.T, local, remote *database.Database, r syncremote.Remote, opt syncremote.Options) (syncremote.Cursors, syncremote.Result, error) {
	t.Helper()
	return syncremote.Run(context.Background(), database.JournalOf(local), r, syncremote.Cursors{}, opt)
}

func TestPayloadTooLargeHalvesThePageDownToOneRecord(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 8)
	r := &scripted{lib: b, importFn: tooLarge(2)}
	_, res, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Sent.Unsynchronizable != 0 || len(names(t, b)) != 8 {
		t.Errorf("unsynchronizable=%d, server holds %d, want 0 and 8", res.Sent.Unsynchronizable, len(names(t, b)))
	}
	if got := len(r.imports[0]); got != 8 {
		t.Errorf("first attempt carried %d, want the whole page of 8", got)
	}
	if got := len(r.imports[1]); got != 4 {
		t.Errorf("second attempt carried %d, want half (4)", got)
	}
}

func TestARecordTooLargeAloneIsDeclaredUnsynchronizableAndTheRunGoesOn(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 4)
	bad := ""
	r := &scripted{lib: b, importFn: func(_ int, recs []changeset.Record) error {
		for _, rec := range recs {
			if bad == "" {
				bad = rec.UUID
			}
			if rec.UUID == bad {
				return syncremote.ErrTooLarge
			}
		}
		return nil
	}}
	cur, res, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatalf("a record too large alone must not stop the run: %v", err)
	}
	if res.Sent.Unsynchronizable != 1 || len(names(t, b)) != 3 {
		t.Errorf("unsynchronizable=%d, server holds %d, want 1 and 3", res.Sent.Unsynchronizable, len(names(t, b)))
	}
	if cur.Sent.IsStart() {
		t.Error("the cursor did not move past the page")
	}
}

func TestTheAllowanceIsNeverRetriedAndKeepsTheServersOwnWords(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 3)
	var srvCalls int
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".import") {
			srvCalls++
		}
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"code":"allowance_exceeded","message":"Quota atteint : 100 enregistrements."}}`))
	}))
	t.Cleanup(hs.Close)
	c, err := syncremote.NewClient(hs.URL, token, hs.Client())
	if err != nil {
		t.Fatal(err)
	}
	cur, _, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if !errors.Is(err, syncremote.ErrAllowance) || !strings.Contains(err.Error(), "Quota atteint : 100 enregistrements.") {
		t.Fatalf("err = %v, want the allowance error with the server's message", err)
	}
	if srvCalls != 1 {
		t.Errorf("the import was called %d times, want 1 (no retry)", srvCalls)
	}
	if !cur.Sent.IsStart() || len(names(t, b)) != 0 {
		t.Error("a refused page must leave the cursor where it was, so that the same page is replayed")
	}
}

func TestAToolLockedDuringAMatchIsADistinctReasonToRetryAfterTheGame(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"tool_locked_during_match","message":"match en cours"}}`))
	}))
	t.Cleanup(hs.Close)
	c, _ := syncremote.NewClient(hs.URL, token, hs.Client())
	_, err := c.Import(context.Background(), nil)
	if !errors.Is(err, syncremote.ErrToolLocked) {
		t.Fatalf("err = %v, want ErrToolLocked", err)
	}
	// Another 409 is not that reason.
	hs2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"other","message":"x"}}`))
	}))
	t.Cleanup(hs2.Close)
	c2, _ := syncremote.NewClient(hs2.URL, token, hs2.Client())
	if _, err := c2.Import(context.Background(), nil); errors.Is(err, syncremote.ErrToolLocked) {
		t.Error("a 409 of another code was taken for a locked tool")
	}
}

func TestRetryAfterIsWaitedOutThenTheSamePageIsReplayed(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 3)
	var slept []time.Duration
	r := &scripted{lib: b}
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow down"}}`))
	}))
	t.Cleanup(hs.Close)
	c, _ := syncremote.NewClient(hs.URL, token, hs.Client())
	_, err := c.Import(context.Background(), nil)
	if d, ok := syncremote.RetryAfter(err); !ok || d != 7*time.Second || !errors.Is(err, syncremote.ErrRateLimited) {
		t.Fatalf("err = %v, RetryAfter = %v %v, want 7s", err, d, ok)
	}

	r.importFn = func(call int, _ []changeset.Record) error {
		if call <= 2 {
			return &rateLimitedForTest{after: 3 * time.Second}
		}
		return nil
	}
	_, res, err := run(t, a, b, r, syncremote.Options{Sleep: func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(slept) != 2 || slept[0] != 3*time.Second {
		t.Errorf("slept %v, want two waits of 3s", slept)
	}
	if res.Sent.Records != 3 || len(names(t, b)) != 3 {
		t.Errorf("sent %d, server holds %d, want 3 and 3", res.Sent.Records, len(names(t, b)))
	}
	if len(r.imports[0]) != len(r.imports[2]) {
		t.Error("the page was not replayed whole")
	}
}

func TestARetryAfterTooLongStopsTheRunInsteadOfHoldingIt(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 2)
	r := &scripted{lib: b, importFn: func(int, []changeset.Record) error { return &rateLimitedForTest{after: time.Hour} }}
	cur, _, err := run(t, a, b, r, syncremote.Options{Sleep: func(context.Context, time.Duration) error {
		t.Error("a wait of an hour must not be slept")
		return nil
	}})
	if !errors.Is(err, syncremote.ErrRateLimited) {
		t.Fatalf("err = %v", err)
	}
	if !cur.Sent.IsStart() {
		t.Error("cursor moved")
	}
}

func TestEveryReplayableFailureLeavesTheCursorSoTheSamePageGoesAgain(t *testing.T) {
	cases := map[string]error{
		"402":     fmt.Errorf("%w", syncremote.ErrAllowance),
		"timeout": fmt.Errorf("%w: timeout", syncremote.ErrUnavailable),
		"5xx":     fmt.Errorf("%w (HTTP 503)", syncremote.ErrUnavailable),
		"429":     fmt.Errorf("%w", syncremote.ErrRateLimited),
	}
	for name, failure := range cases {
		t.Run(name, func(t *testing.T) {
			a, b := openLibrary(t), openLibrary(t)
			fill(t, a, 3)
			r := &scripted{lib: b, importFn: func(call int, _ []changeset.Record) error {
				if call == 1 {
					return failure
				}
				return nil
			}}
			cur, _, err := run(t, a, b, r, syncremote.Options{})
			if err == nil || !cur.Sent.IsStart() {
				t.Fatalf("err=%v cursor=%+v: the failed page must not be settled", err, cur)
			}
			cur, _, err = run(t, a, b, r, syncremote.Options{})
			if err != nil || len(names(t, b)) != 3 {
				t.Fatalf("replay: err=%v, server holds %d", err, len(names(t, b)))
			}
			if len(r.imports[0]) != len(r.imports[1]) {
				t.Errorf("replay carried %d, first attempt %d", len(r.imports[1]), len(r.imports[0]))
			}
			_ = cur
		})
	}
}

func TestNoImportCarriesMoreThanFiveHundredRecords(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 3)
	r := &scripted{lib: b}
	if _, _, err := run(t, a, b, r, syncremote.Options{Page: 5000}); err != nil {
		t.Fatal(err)
	}
	for _, l := range r.limits {
		if l > 500 {
			t.Errorf("asked for %d records, want at most 500", l)
		}
	}
	for _, im := range r.imports {
		if len(im) > 500 {
			t.Errorf("imported %d records in one call", len(im))
		}
	}
}

func TestAShortPageWithMoreAlwaysResumesFromTheCursorTheServerGave(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 5)
	r := &scripted{lib: b, exportFn: func(_ int, after storage.Cursor, limit int) (changeset.Set, bool, error) {
		set, err := database.JournalOf(b).Export(context.Background(), after, 2) // fewer than limit, more:true
		return set, true, err
	}}
	cur, _, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(names(t, a)) != 5 {
		t.Fatalf("library holds %d, want 5", len(names(t, a)))
	}
	for i := 1; i < len(r.exports); i++ {
		if r.exports[i] == r.exports[i-1] {
			t.Errorf("export %d repeated the cursor of the previous one", i)
		}
	}
	if cur.Received.IsStart() {
		t.Error("received cursor did not advance")
	}
}

func TestAStaleReceivedCursorRestartsFromAnEmptyOne(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 2)
	r := &scripted{lib: b}
	first, _, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	r.exports = nil
	r.exportFn = func(call int, after storage.Cursor, limit int) (changeset.Set, bool, error) {
		if !after.IsStart() {
			return changeset.Set{}, true, storage.ErrCursorStale
		}
		return changeset.Set{}, false, nil
	}
	if _, _, err := syncremote.Run(context.Background(), database.JournalOf(a), r, first, syncremote.Options{}); err != nil {
		t.Fatal(err)
	}
	last := r.exports[len(r.exports)-1]
	if !last.IsStart() {
		t.Error("the run did not restart from the empty cursor after a 410")
	}
}

// rateLimitedForTest stands for a 429 answer without going through HTTP.
type rateLimitedForTest struct{ after time.Duration }

func (e *rateLimitedForTest) Error() string             { return "limited" }
func (e *rateLimitedForTest) RetryAfter() time.Duration { return e.after }
func (e *rateLimitedForTest) Unwrap() error             { return syncremote.ErrRateLimited }
