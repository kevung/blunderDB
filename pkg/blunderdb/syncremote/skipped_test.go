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

func tooLargeCoded() error {
	return fmt.Errorf("%w: payload_too_large", syncremote.ErrTooLarge)
}

// A 413 that is not the server's own answer says nothing about a record: the run stops, nothing is set aside.
func TestA413WithoutTheServersCodeStopsTheRunAndSetsNothingAside(t *testing.T) {
	a := openLibrary(t)
	fill(t, a, 20)
	calls := 0
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte("<html><body>413 Request Entity Too Large</body></html>"))
	}))
	t.Cleanup(hs.Close)
	c, _ := syncremote.NewClient(hs.URL, token, hs.Client())
	cur, res, err := syncremote.Run(context.Background(), database.JournalOf(a), c, syncremote.Cursors{}, syncremote.Options{})
	if !errors.Is(err, syncremote.ErrBlocked) || errors.Is(err, syncremote.ErrTooLarge) {
		t.Fatalf("err = %v, want ErrBlocked", err)
	}
	if calls != 1 || !cur.Sent.IsStart() || len(cur.Skipped) != 0 || res.Sent.Unsynchronizable != 0 {
		t.Errorf("calls=%d cursor=%+v skipped=%v unsync=%d: nothing may move", calls, cur.Sent, cur.Skipped, res.Sent.Unsynchronizable)
	}
}

// Through a real 413 with the server's code the page is halved: the status mapping is what links the two.
func TestARealCodedPayloadTooLargeHalvesThePage(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 6)
	var sizes []int
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Records []changeset.Record `json:"records"`
		}
		_ = jsonDecode(r, &req)
		sizes = append(sizes, len(req.Records))
		if len(req.Records) > 2 {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = w.Write([]byte(`{"error":{"code":"payload_too_large","message":"too big"}}`))
			return
		}
		rep, _ := database.JournalOf(b).Import(r.Context(), changeset.Set{Records: req.Records})
		_ = jsonEncode(w, rep)
	}))
	t.Cleanup(hs.Close)
	c, _ := syncremote.NewClient(hs.URL, token, hs.Client())
	_, _, err := syncremote.Run(context.Background(), database.JournalOf(a), &importOnly{c, b}, syncremote.Cursors{}, syncremote.Options{})
	if err != nil || len(names(t, b)) != 6 {
		t.Fatalf("err=%v, server holds %d, want 6 (sizes %v)", err, len(names(t, b)), sizes)
	}
	if len(sizes) < 2 || sizes[0] <= 2 {
		t.Errorf("sizes %v: the first attempt should have been refused whole", sizes)
	}
}

// importOnly sends to the HTTP client and receives from a local library, so the test needs one handler.
type importOnly struct {
	c   *syncremote.Client
	lib *database.Database
}

func (o *importOnly) Import(ctx context.Context, r []changeset.Record) (changeset.Report, error) {
	return o.c.Import(ctx, r)
}
func (o *importOnly) Export(ctx context.Context, after storage.Cursor, limit int) (syncremote.Page, error) {
	set, err := database.JournalOf(o.lib).Export(ctx, after, limit)
	return syncremote.Page{Set: set}, err
}

func refuseUUID(bad *string) func(int, []changeset.Record) error {
	return func(_ int, recs []changeset.Record) error {
		for _, rec := range recs {
			if rec.UUID == *bad {
				return tooLargeCoded()
			}
		}
		return nil
	}
}

// A record the server refuses is not lost: it is listed with its type and identifier, kept in the cursors,
// tried again at each pass, and leaves the list when it goes through.
func TestARefusedRecordIsKeptListedRetriedAndReleasedWhenItGoesThrough(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 4)
	bad := ""
	r := &scripted{lib: b}
	r.importFn = func(call int, recs []changeset.Record) error {
		if bad == "" {
			bad = recs[0].UUID
		}
		return refuseUUID(&bad)(call, recs)
	}
	cur, res, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cur.Skipped) != 1 || cur.Skipped[0].UUID != bad || cur.Skipped[0].Family == "" || cur.Skipped[0].Direction != syncremote.DirectionSent || cur.Skipped[0].Reason != syncremote.ReasonTooLarge {
		t.Fatalf("skipped = %+v, want the refused record by family and uuid", cur.Skipped)
	}
	if len(res.Skipped) != 1 || len(names(t, b)) != 3 {
		t.Fatalf("result lists %d, server holds %d, want 1 and 3", len(res.Skipped), len(names(t, b)))
	}

	// Next pass: still refused, tried alone, still listed.
	r.imports = nil
	cur, res, err = syncremote.Run(context.Background(), database.JournalOf(a), r, cur, syncremote.Options{})
	if err != nil || len(cur.Skipped) != 1 || len(res.Skipped) != 1 {
		t.Fatalf("second pass: err=%v skipped=%+v", err, cur.Skipped)
	}
	if len(r.imports) == 0 || len(r.imports[0]) != 1 || r.imports[0][0].UUID != bad {
		t.Fatalf("the pass should begin by sending the one record again, sent %d call(s)", len(r.imports))
	}

	// The server now accepts it.
	bad = "none"
	cur, res, err = syncremote.Run(context.Background(), database.JournalOf(a), r, cur, syncremote.Options{})
	if err != nil || len(cur.Skipped) != 0 || len(res.Skipped) != 0 || len(names(t, b)) != 4 {
		t.Fatalf("third pass: err=%v skipped=%+v server holds %d, want released and 4", err, cur.Skipped, len(names(t, b)))
	}
}

// A single record that cannot get through is kept too, without a refusal; the run goes on to receive.
func TestARecordInterruptedAloneIsSetAsideAndTheReceptionStillRuns(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 4)
	create(t, b, "FromServer")
	bad := ""
	r := &scripted{lib: b, importFn: func(_ int, recs []changeset.Record) error {
		if bad == "" {
			bad = recs[len(recs)-1].UUID
		}
		for _, rec := range recs {
			if rec.UUID == bad {
				return fmt.Errorf("%w: %w: timeout", syncremote.ErrUnavailable, syncremote.ErrInterrupted)
			}
		}
		return nil
	}}
	cur, _, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cur.Skipped) != 1 || cur.Skipped[0].Reason != syncremote.ReasonInterrupted {
		t.Fatalf("skipped = %+v", cur.Skipped)
	}
	has(t, a, "FromServer")
	if cur.Received.IsStart() {
		t.Error("the reception did not run")
	}
}

// Whatever fails without being a size or a break is not a reason to set a record aside.
func TestAnyOtherFailureOfAHalfStopsTheRunWithTheCursorWhereItWas(t *testing.T) {
	for name, failing := range map[string]int{"first half": 0, "second half": 2} {
		t.Run(name, func(t *testing.T) {
			a, b := openLibrary(t), openLibrary(t)
			fill(t, a, 4)
			var target string
			r := &scripted{lib: b, importFn: func(_ int, recs []changeset.Record) error {
				if len(recs) > 2 {
					return tooLargeCoded()
				}
				if target == "" && len(recs) == 2 {
					target = recs[0].UUID
				}
				if len(recs) == 2 && recs[0].UUID == target && failing == 0 {
					return fmt.Errorf("%w (HTTP 503)", syncremote.ErrUnavailable)
				}
				if len(recs) == 2 && recs[0].UUID != target && failing == 2 {
					return fmt.Errorf("%w (HTTP 503)", syncremote.ErrUnavailable)
				}
				return nil
			}}
			cur, _, err := run(t, a, b, r, syncremote.Options{})
			if !errors.Is(err, syncremote.ErrUnavailable) || !cur.Sent.IsStart() || len(cur.Skipped) != 0 {
				t.Fatalf("err=%v cursor=%+v skipped=%v: a failed half must stop the run before the cursor moves", err, cur.Sent, cur.Skipped)
			}
		})
	}
}

// What the server skipped on the way in is listed and retried like what it refused on the way out.
func TestWhatTheServerSkippedOnReceptionIsListedAndAskedAgain(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 3)
	skipFirst := true
	r := &scripted{lib: b}
	r.pageFn = func(_ int, after storage.Cursor, limit int) (syncremote.Page, bool, error) {
		if !skipFirst {
			return syncremote.Page{}, false, nil
		}
		set, err := database.JournalOf(b).Export(context.Background(), after, 100)
		if err != nil || len(set.Records) < 2 {
			return syncremote.Page{}, false, err
		}
		gone := set.Records[1]
		set.Records = append(set.Records[:1:1], set.Records[2:]...)
		return syncremote.Page{Set: set, Skipped: []syncremote.Skipped{{Family: gone.Family, UUID: gone.UUID, Size: 300 << 20}}}, true, nil
	}
	cur, res, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cur.Skipped) != 1 || cur.Skipped[0].Direction != syncremote.DirectionReceived || cur.Skipped[0].Size != 300<<20 || len(res.Skipped) != 1 {
		t.Fatalf("skipped = %+v", cur.Skipped)
	}
	if len(names(t, a)) != 2 {
		t.Fatalf("library holds %d, want 2 (one skipped)", len(names(t, a)))
	}
	skipFirst = false
	cur, _, err = syncremote.Run(context.Background(), database.JournalOf(a), r, cur, syncremote.Options{})
	if err != nil || len(cur.Skipped) != 0 || len(names(t, a)) != 3 {
		t.Fatalf("after the server serves it: err=%v skipped=%+v holds %d, want released and 3", err, cur.Skipped, len(names(t, a)))
	}
}

type failingMerge struct{ syncremote.Local }

func (failingMerge) Import(context.Context, changeset.Set) (changeset.Report, error) {
	return changeset.Report{}, errors.New("disk full")
}

func TestTheReceivedCursorStaysWhenTheMergeFails(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 2)
	r := &scripted{lib: b}
	cur, _, err := syncremote.Run(context.Background(), failingMerge{database.JournalOf(a)}, r, syncremote.Cursors{}, syncremote.Options{})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("err = %v", err)
	}
	if !cur.Received.IsStart() {
		t.Error("the received cursor moved past records that were not merged")
	}
}

func TestAtMostThreeWaitsPerCall(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 2)
	r := &scripted{lib: b, importFn: func(int, []changeset.Record) error { return &rateLimitedForTest{after: time.Second} }}
	waits := 0
	_, _, err := run(t, a, b, r, syncremote.Options{Sleep: func(context.Context, time.Duration) error { waits++; return nil }})
	if !errors.Is(err, syncremote.ErrRateLimited) || waits != 3 || len(r.imports) != 4 {
		t.Fatalf("err=%v waits=%d calls=%d, want a rate limit after 3 waits and 4 calls", err, waits, len(r.imports))
	}
}

func TestARateLimitedReceptionIsWaitedOutAndReplayed(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 2)
	r := &scripted{lib: b}
	r.pageFn = func(call int, _ storage.Cursor, _ int) (syncremote.Page, bool, error) {
		if call == 1 {
			return syncremote.Page{}, true, &rateLimitedForTest{after: time.Second}
		}
		return syncremote.Page{}, false, nil
	}
	waits := 0
	_, _, err := run(t, a, b, r, syncremote.Options{Sleep: func(context.Context, time.Duration) error { waits++; return nil }})
	if err != nil || waits != 1 || len(names(t, a)) != 2 {
		t.Fatalf("err=%v waits=%d holds %d", err, waits, len(names(t, a)))
	}
}

// A stale cursor restarts the reception once; a second one in the same run is an error, not a loop.
func TestAStaleReceivedCursorRestartsOnlyOncePerRun(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 6)
	r := &scripted{lib: b}
	first, _, err := run(t, a, b, r, syncremote.Options{Page: 2})
	if err != nil || first.Received.IsStart() {
		t.Fatal(err)
	}
	r.exports = nil
	r.pageFn = func(call int, after storage.Cursor, _ int) (syncremote.Page, bool, error) {
		if call > 8 {
			return syncremote.Page{}, true, errors.New("looping")
		}
		if !after.IsStart() {
			return syncremote.Page{}, true, storage.ErrCursorStale
		}
		set, err := database.JournalOf(b).Export(context.Background(), after, 2)
		return syncremote.Page{Set: set}, true, err
	}
	_, _, err = syncremote.Run(context.Background(), database.JournalOf(a), r, first, syncremote.Options{Page: 2})
	if !errors.Is(err, storage.ErrCursorStale) {
		t.Fatalf("err = %v, want the stale cursor reported after one restart", err)
	}
}

// A break after the connection was made is a reason to send less; no connection at all is not.
func TestOnlyABreakAfterTheConnectionIsAnInterruption(t *testing.T) {
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) }))
	t.Cleanup(hang.Close)
	c, _ := syncremote.NewClient(hang.URL, token, &http.Client{Timeout: 100 * time.Millisecond})
	if _, err := c.Import(context.Background(), nil); !errors.Is(err, syncremote.ErrInterrupted) || !errors.Is(err, syncremote.ErrUnavailable) {
		t.Errorf("timeout while sending: %v, want an interruption", err)
	}
	cut := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_body","message":"unreadable request body"}}`))
	}))
	t.Cleanup(cut.Close)
	c, _ = syncremote.NewClient(cut.URL, token, cut.Client())
	if _, err := c.Import(context.Background(), nil); !errors.Is(err, syncremote.ErrInterrupted) {
		t.Errorf("400 invalid_body: %v, want an interruption", err)
	}
	gone := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := gone.URL
	gone.Close()
	c, _ = syncremote.NewClient(url, token, nil)
	if _, err := c.Import(context.Background(), nil); errors.Is(err, syncremote.ErrInterrupted) || !errors.Is(err, syncremote.ErrUnavailable) {
		t.Errorf("no connection: %v, want unavailable and not an interruption", err)
	}
}
