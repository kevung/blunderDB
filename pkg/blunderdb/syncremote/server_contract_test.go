package syncremote_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/syncremote"
)

// collectionWith makes a collection of n distinct positions and returns its name.
func collectionWith(t *testing.T, d *database.Database, n int) {
	t.Helper()
	cid := create(t, d, "Study")
	for i := 0; i < n; i++ {
		p := domain.InitializePosition()
		p.Board.Points[10+i] = domain.Point{Checkers: 1, Color: domain.White}
		id, err := d.SavePosition(&p)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.AddPositionToCollection(cid, id); err != nil {
			t.Fatal(err)
		}
	}
}

func members(t *testing.T, d *database.Database) int {
	t.Helper()
	cs, err := d.GetAllCollections()
	if err != nil || len(cs) != 1 {
		t.Fatalf("collections: %v %v", cs, err)
	}
	n, err := d.CountCollectionPositions(cs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// A reference to a record set aside is not lost: the collection that holds it
// went through without the position, and is sent again once the position does.
func TestAReferenceToASetAsideRecordIsRetriedWhenTheRecordGoesThrough(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	collectionWith(t, a, 2)
	bad := ""
	r := &scripted{lib: b}
	r.importFn = func(call int, recs []changeset.Record) error {
		if bad == "" {
			for _, rec := range recs {
				if rec.Family == "position" {
					bad = rec.UUID
					break
				}
			}
		}
		return refuseUUID(&bad)(call, recs)
	}
	cur, res, err := run(t, a, b, r, syncremote.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if members(t, b) != 1 {
		t.Fatalf("server holds %d members, want 1 while one position is set aside", members(t, b))
	}
	listsReference := func(sk []syncremote.Skip) bool {
		for _, k := range sk {
			if k.Family == "collection" && k.Reason == syncremote.ReasonReference {
				return true
			}
		}
		return false
	}
	if !listsReference(cur.Skipped) || !listsReference(res.Skipped) {
		t.Fatalf("the collection with a reference to the set-aside position must be listed: %+v", cur.Skipped)
	}
	for pass := 2; pass <= 3; pass++ {
		if cur, _, err = syncremote.Run(context.Background(), database.JournalOf(a), r, cur, syncremote.Options{}); err != nil {
			t.Fatal(err)
		}
		if members(t, b) != 1 || !listsReference(cur.Skipped) {
			t.Fatalf("pass %d: members=%d skipped=%+v, want the divergence still listed", pass, members(t, b), cur.Skipped)
		}
	}
	bad = "none"
	cur, _, err = syncremote.Run(context.Background(), database.JournalOf(a), r, cur, syncremote.Options{})
	if err != nil || len(cur.Skipped) != 0 || members(t, b) != 2 {
		t.Fatalf("after the position goes through: err=%v skipped=%+v members=%d, want released and 2", err, cur.Skipped, members(t, b))
	}
}

// The same on the way in: the server skipped a position, the collection came without it.
func TestAReferenceToARecordTheServerSkippedIsRetriedWhenItServesIt(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	collectionWith(t, b, 2)
	skip := true
	r := &scripted{lib: b}
	r.pageFn = func(_ int, after storage.Cursor, _ int) (syncremote.Page, bool, error) {
		if !skip {
			return syncremote.Page{}, false, nil
		}
		set, err := database.JournalOf(b).Export(context.Background(), after, 100)
		if err != nil {
			return syncremote.Page{}, false, err
		}
		var kept []changeset.Record
		var sk []syncremote.Skipped
		for _, rec := range set.Records {
			if rec.Family == "position" && len(sk) == 0 {
				sk = append(sk, syncremote.Skipped{Family: rec.Family, UUID: rec.UUID, Size: 300 << 20})
				continue
			}
			kept = append(kept, rec)
		}
		set.Records = kept
		return syncremote.Page{Set: set, Skipped: sk}, true, nil
	}
	cur, _, err := run(t, a, b, r, syncremote.Options{})
	if err != nil || members(t, a) != 1 {
		t.Fatalf("err=%v members=%d, want 1", err, members(t, a))
	}
	for pass := 2; pass <= 3; pass++ {
		if cur, _, err = syncremote.Run(context.Background(), database.JournalOf(a), r, cur, syncremote.Options{}); err != nil || members(t, a) != 1 || len(cur.Skipped) < 2 {
			t.Fatalf("pass %d: err=%v members=%d skipped=%+v, want the divergence listed", pass, err, members(t, a), cur.Skipped)
		}
	}
	skip = false
	cur, _, err = syncremote.Run(context.Background(), database.JournalOf(a), r, cur, syncremote.Options{})
	if err != nil || len(cur.Skipped) != 0 || members(t, a) != 2 {
		t.Fatalf("after the server serves it: err=%v skipped=%+v members=%d, want released and 2", err, cur.Skipped, members(t, a))
	}
}

// Matches go first and alone, then the rest: an account with a cap refuses the positions of a match that is not there yet.
func TestMatchesAreSentBeforeThePositionsAndNeverInTheSameRequest(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	if _, err := a.ImportXGMatch("../../../testdata/test.xg"); err != nil {
		t.Fatal(err)
	}
	collectionWith(t, a, 2)
	r := &scripted{lib: b}
	if _, _, err := run(t, a, b, r, syncremote.Options{}); err != nil {
		t.Fatal(err)
	}
	matches := 0
	for i, call := range r.imports {
		m, o := 0, 0
		for _, rec := range call {
			if rec.Family == "match" {
				m++
			} else {
				o++
			}
		}
		t.Logf("request %d: %d matches, %d others", i+1, m, o)
		if m > 0 && o > 0 {
			t.Fatalf("request %d mixes %d matches with %d other records", i+1, m, o)
		}
		matches += m
		if matches == 0 {
			t.Fatalf("request %d carries positions before any match was sent", i+1)
		}
	}
	if len(r.imports) < 2 || matches == 0 {
		t.Fatalf("requests=%d matches=%d: the fixture should hold both", len(r.imports), matches)
	}
}

func fakeServer(t *testing.T, h http.HandlerFunc) *syncremote.Client {
	t.Helper()
	hs := httptest.NewServer(h)
	t.Cleanup(hs.Close)
	c, err := syncremote.NewClient(hs.URL, token, hs.Client())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func answer(w http.ResponseWriter, status int, body string, hdr ...string) {
	for i := 0; i+1 < len(hdr); i += 2 {
		w.Header().Set(hdr[i], hdr[i+1])
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// A connection cut leaves the import running on the server for up to 15 s: its 429 import_in_progress is waited out, well past three tries.
func TestImportInProgressIsWaitedOutAsLongAsTheServerAsks(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	create(t, a, "Study")
	var n atomic.Int32
	c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) <= 6 {
			answer(w, 429, `{"error":{"code":"import_in_progress","message":"busy"}}`, "Retry-After", "2")
			return
		}
		var req struct {
			Records []changeset.Record `json:"records"`
		}
		_ = jsonDecode(r, &req)
		rep, _ := database.JournalOf(b).Import(r.Context(), changeset.Set{Records: req.Records})
		_ = jsonEncode(w, rep)
	})
	var waited time.Duration
	opt := syncremote.Options{Sleep: func(_ context.Context, d time.Duration) error { waited += d; return nil }}
	_, _, err := syncremote.Run(context.Background(), database.JournalOf(a), &importOnly{c, b}, syncremote.Cursors{}, opt)
	if err != nil || len(names(t, b)) != 1 || waited != 12*time.Second {
		t.Fatalf("err=%v server holds %d waited %v, want it through after 12s", err, len(names(t, b)), waited)
	}
}

// A 429 import_in_progress without Retry-After still carries a wait.
func TestImportInProgressWithoutRetryAfterStillCarriesAWait(t *testing.T) {
	c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		answer(w, 429, `{"error":{"code":"import_in_progress"}}`)
	})
	_, err := c.Import(context.Background(), nil)
	if d, ok := syncremote.RetryAfter(err); !ok || d <= 0 || !errors.Is(err, syncremote.ErrRateLimited) {
		t.Fatalf("err=%v wait=%v ok=%v, want a rate limit with a wait", err, d, ok)
	}
}

// A 413 payload_too_large on an export is the server out of time: the page is asked again smaller, down to one record.
func TestPayloadTooLargeOnAnExportDividesThePage(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 6)
	r := &scripted{lib: b}
	r.exportFn = func(_ int, _ storage.Cursor, limit int) (changeset.Set, bool, error) {
		if limit > 2 {
			return changeset.Set{}, true, fmt.Errorf("%w: payload_too_large", syncremote.ErrTooLarge)
		}
		return changeset.Set{}, false, nil
	}
	cur, _, err := run(t, a, b, r, syncremote.Options{})
	if err != nil || len(names(t, a)) != 6 || len(cur.Skipped) != 0 {
		t.Fatalf("err=%v holds %d skipped=%v, want 6", err, len(names(t, a)), cur.Skipped)
	}
	for _, l := range r.limits[len(r.limits)-3:] {
		if l > 2 {
			t.Fatalf("limits %v: the size that worked should be kept", r.limits)
		}
	}
}

func TestPayloadTooLargeOnAnExportOfOneRecordIsTooLargeNotUnreachable(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, b, 2)
	r := &scripted{lib: b}
	r.exportFn = func(int, storage.Cursor, int) (changeset.Set, bool, error) {
		return changeset.Set{}, true, fmt.Errorf("%w: payload_too_large", syncremote.ErrTooLarge)
	}
	_, _, err := run(t, a, b, r, syncremote.Options{})
	if !errors.Is(err, syncremote.ErrTooLarge) || errors.Is(err, syncremote.ErrUnavailable) {
		t.Fatalf("err=%v, want too large and not unavailable", err)
	}
}

func TestPayloadTooLargeOverHTTPOnExportIsDivided(t *testing.T) {
	var limits []int
	c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Limit int `json:"limit"`
		}
		_ = jsonDecode(r, &req)
		limits = append(limits, req.Limit)
		if req.Limit > 100 {
			answer(w, 413, `{"error":{"code":"payload_too_large","message":"deadline"}}`)
			return
		}
		_ = jsonEncode(w, map[string]any{"records": []any{}, "more": false})
	})
	a := openLibrary(t)
	_, _, err := syncremote.Run(context.Background(), database.JournalOf(a), &exportOnly{c}, syncremote.Cursors{}, syncremote.Options{})
	if err != nil || len(limits) < 2 || limits[0] != 500 || limits[len(limits)-1] > 100 {
		t.Fatalf("err=%v limits=%v, want the page divided below 100", err, limits)
	}
}

type exportOnly struct{ c *syncremote.Client }

func (o *exportOnly) Import(ctx context.Context, r []changeset.Record) (changeset.Report, error) {
	return o.c.Import(ctx, r)
}
func (o *exportOnly) Export(ctx context.Context, after storage.Cursor, limit int) (syncremote.Page, error) {
	return o.c.Export(ctx, after, limit)
}

// 402 allowance_exceeded gives the figures; the window words them.
func TestAnAllowanceCarriesItsFiguresWhereverTheServerPutsThem(t *testing.T) {
	for name, body := range map[string]string{
		"in the error":  `{"error":{"code":"allowance_exceeded","message":"Plan limit","axis":"positions","allowed":150,"held":140,"new":20}}`,
		"under details": `{"error":{"code":"allowance_exceeded","message":"Plan limit","details":{"axis":"positions","allowed":150,"held":140,"new":20}}}`,
	} {
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) { answer(w, 402, body) })
		_, err := c.Import(context.Background(), nil)
		f, ok := syncremote.AllowanceOf(err)
		if !errors.Is(err, syncremote.ErrAllowance) || !ok || f != (syncremote.AllowanceFigures{Axis: "positions", Allowed: 150, Held: 140, New: 20}) {
			t.Errorf("%s: err=%v figures=%+v ok=%v", name, err, f, ok)
		}
	}
}

// 422 invalid_records is a refusal of the content: told apart from an unreachable server, and a record alone is set aside.
func TestInvalidRecordsIsARefusalOfContentAndTheRecordIsSetAside(t *testing.T) {
	a, b := openLibrary(t), openLibrary(t)
	fill(t, a, 4)
	bad := ""
	c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Records []changeset.Record `json:"records"`
		}
		_ = jsonDecode(r, &req)
		if bad == "" {
			bad = req.Records[0].UUID
		}
		for _, rec := range req.Records {
			if rec.UUID == bad {
				answer(w, 422, `{"error":{"code":"invalid_records","message":"bad"}}`)
				return
			}
		}
		rep, _ := database.JournalOf(b).Import(r.Context(), changeset.Set{Records: req.Records})
		_ = jsonEncode(w, rep)
	})
	if _, err := c.Import(context.Background(), []changeset.Record{{}}); !errors.Is(err, syncremote.ErrInvalidRecords) && bad != "" {
		t.Fatalf("err=%v, want ErrInvalidRecords", err)
	}
	bad = ""
	cur, _, err := syncremote.Run(context.Background(), database.JournalOf(a), &importOnly{c, b}, syncremote.Cursors{}, syncremote.Options{})
	if err != nil || len(cur.Skipped) != 1 || cur.Skipped[0].Reason != syncremote.ReasonInvalid || len(names(t, b)) != 3 {
		t.Fatalf("err=%v skipped=%+v holds %d, want one set aside as invalid and 3 sent", err, cur.Skipped, len(names(t, b)))
	}
}

// A tenant that keeps no journal refuses the export: said as the server's refusal, asked once, not a token problem.
func TestAServerRefusalOfTheExportStopsTheRunOnceAndIsNotATokenProblem(t *testing.T) {
	a := openLibrary(t)
	var calls atomic.Int32
	c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		answer(w, 403, `{"error":{"code":"no_journal","message":"this account keeps no journal"}}`)
	})
	_, _, err := syncremote.Run(context.Background(), database.JournalOf(a), &exportOnly{c}, syncremote.Cursors{}, syncremote.Options{})
	if !errors.Is(err, syncremote.ErrServerRefusal) || errors.Is(err, syncremote.ErrRefused) || errors.Is(err, syncremote.ErrUnavailable) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d, want one call and the server's refusal", err, calls.Load())
	}
}
