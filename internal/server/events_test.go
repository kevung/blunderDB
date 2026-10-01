package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// sseFrame is one frame of the stream, as a client parses it.
type sseFrame struct {
	id, event string
	data      events.Event
	raw       string
	comment   string
}

// sseStream is an open subscription: its frames arrive on frames, closed at the end of the
// stream.
type sseStream struct {
	frames  chan sseFrame
	cancel  context.CancelFunc
	resp    *http.Response
	opening sseFrame
}

// subscribe opens GET /v1/events?query as tenant and waits for the subscription to be
// registered (the ": subscribed" comment), so a gesture sent afterwards is seen. The resync
// every stream opens with is kept in s.opening.
func subscribe(t *testing.T, base, tenant, query string, header http.Header) *sseStream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+eventsPath+query, nil)
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Accept-Encoding", "gzip")
	for k, vs := range header {
		req.Header[k] = vs
	}
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // the reader goroutine closes it at the end of the stream
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		t.Fatalf("subscribe: status %d, %s", resp.StatusCode, b)
	}
	s := &sseStream{frames: make(chan sseFrame, 64), cancel: cancel, resp: resp}
	go func() {
		defer close(s.frames)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		var f sseFrame
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				if f.event != "" || f.comment != "" {
					s.frames <- f
				}
				f = sseFrame{}
			case strings.HasPrefix(line, ": "):
				f.comment = strings.TrimPrefix(line, ": ")
			case strings.HasPrefix(line, "id: "):
				f.id = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				f.event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				f.raw = strings.TrimPrefix(line, "data: ")
				_ = json.Unmarshal([]byte(f.raw), &f.data)
			}
		}
	}()
	t.Cleanup(cancel)
	if f := s.next(t); f.comment != "subscribed" {
		t.Fatalf("first frame %+v; want the subscribed comment", f)
	}
	s.opening = s.next(t)
	if s.opening.event != string(events.KindResync) || s.opening.id == "" {
		t.Fatalf("the stream opens with %+v; want a resync with an id", s.opening)
	}
	return s
}

// next is the next frame that is not a heartbeat.
func (s *sseStream) next(t *testing.T) sseFrame {
	t.Helper()
	for {
		select {
		case f, ok := <-s.frames:
			if !ok {
				t.Fatal("the stream ended")
			}
			if f.comment == "ping" {
				continue
			}
			return f
		case <-time.After(5 * time.Second):
			t.Fatal("no frame within 5s")
		}
	}
}

// ended waits for the stream to close.
func (s *sseStream) ended(t *testing.T) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-s.frames:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("the stream did not end")
		}
	}
}

func eventServerOn(t *testing.T, st storage.Storage, mod func(*Options)) (*httptest.Server, *Server) {
	t.Helper()
	o := Options{Storage: st, Metrics: metrics.New(), EnableDirection: true, Transcription: true}
	if mod != nil {
		mod(&o)
	}
	srv, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

func memStorage(t *testing.T) storage.Storage {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestEvents_SQLite(t *testing.T) {
	checkEvents(t, memStorage(t))
}

// checkEvents runs the event checks over one backend.
func checkEvents(t *testing.T, st storage.Storage) {
	ts, _ := eventServerOn(t, st, nil)
	mine := seedDirection(t, st, "1", "Open de Lyon")
	theirs := seedDirection(t, st, "2", "Open de Nice")

	t.Run("a gesture is one message with its version", func(t *testing.T) {
		s := subscribe(t, ts.URL, "1", "", nil)
		r := send(t, ts, "1", "/v1/directions.addNote", mine.body(`"text":"un"`), mine.versionOf(t, ts), "")
		if r.status != http.StatusOK {
			t.Fatalf("addNote: %d %s", r.status, r.body)
		}
		f := s.next(t)
		ev := f.data
		if f.event != string(events.KindRencontre) || ev.RencontreID != mine.rencontreID ||
			!slices.Contains(ev.TournamentIDs, mine.tournamentID) || f.id == "" {
			t.Fatalf("frame %+v", f)
		}
		if quoteVersion(ev.Version) != r.version {
			t.Errorf("event version %q; the gesture answered %s", ev.Version, r.version)
		}
	})

	t.Run("a refused or stale gesture is no message", func(t *testing.T) {
		s := subscribe(t, ts.URL, "1", "", nil)
		v := mine.versionOf(t, ts)
		if r := send(t, ts, "1", "/v1/directions.addNote", mine.body(`"text":"x"`), `"stale"`, ""); r.status != http.StatusConflict {
			t.Fatalf("stale addNote: %d", r.status)
		}
		if r := send(t, ts, "1", "/v1/directions.enterResult", mine.body(`"matchId":"nope","winner":"aa"`), v, ""); r.status != http.StatusBadRequest {
			t.Fatalf("refused enterResult: %d %s", r.status, r.body)
		}
		if r := send(t, ts, "1", "/v1/directions.addNote", mine.body(`"text":"deux"`), v, ""); r.status != http.StatusOK {
			t.Fatalf("addNote: %d %s", r.status, r.body)
		}
		// The first message is the gesture that applied: the two before it published nothing.
		if f := s.next(t); quoteVersion(f.data.Version) == v || f.data.RencontreID != mine.rencontreID {
			t.Fatalf("frame %+v; want the applied gesture's", f)
		}
	})

	t.Run("a subscriber hears its own tenant only", func(t *testing.T) {
		s := subscribe(t, ts.URL, "2", "", nil)
		send(t, ts, "1", "/v1/directions.addNote", mine.body(`"text":"trois"`), mine.versionOf(t, ts), "")
		if r := send(t, ts, "2", "/v1/directions.addNote", theirs.body(`"text":"nice"`), theirs.versionOf(t, ts), ""); r.status != http.StatusOK {
			t.Fatalf("tenant 2 addNote: %d", r.status)
		}
		if f := s.next(t); f.data.RencontreID != theirs.rencontreID {
			t.Fatalf("tenant 2 heard %+v", f)
		}
	})

	t.Run("a filter narrows the subscription", func(t *testing.T) {
		draft := createDraft(t, ts, "1")
		s := subscribe(t, ts.URL, "1", "?transcription="+strconv.FormatInt(draft.ID, 10), nil)
		send(t, ts, "1", "/v1/directions.addNote", mine.body(`"text":"quatre"`), mine.versionOf(t, ts), "")
		st := typeAction(t, ts, "1", draft, 3, 1)
		f := s.next(t)
		if f.event != string(events.KindTranscription) || f.data.TranscriptionID != draft.ID || f.data.Revision == 0 || f.data.Revision > st.Revision {
			t.Fatalf("frame %+v; want the draft's", f)
		}
		for f.data.Revision != st.Revision {
			f = s.next(t)
		}
		// A gesture on a stale revision is refused, and silent.
		if code, body := gesture(t, ts, "1", "/v1/transcriptions.apply", st.Revision-1,
			map[string]any{"id": st.ID, "sessionId": st.SessionID, "gesture": map[string]any{"kind": "validate"}}); code != http.StatusConflict {
			t.Fatalf("stale apply: %d %s", code, body)
		}
		gesture(t, ts, "1", "/v1/transcriptions.abandon", st.Revision, map[string]any{"id": draft.ID})
		if f = s.next(t); !f.data.Removed {
			t.Fatalf("frame %+v; want the removal", f)
		}
		if f.data.TranscriptionID != draft.ID {
			t.Fatalf("removal %+v", f)
		}
	})
}

// TestEvents_Reconnect: no history is kept, so a client that reconnects is told to read again.
func TestEvents_Reconnect(t *testing.T) {
	ts, _ := eventServerOn(t, memStorage(t), nil)
	s := subscribe(t, ts.URL, "1", "", http.Header{"Last-Event-ID": {"x-1"}})
	if !strings.Contains(s.opening.raw, `"reconnected"`) {
		t.Fatalf("opening %+v; want a resync for a reconnection", s.opening)
	}
}

// TestEvents_EveryOpeningIsResync: a client that reconnects before it heard a single event has
// no Last-Event-ID to send, and may still have missed gestures; every stream opens with a
// resync, with an id the client can send back.
func TestEvents_EveryOpeningIsResync(t *testing.T) {
	ts, _ := eventServerOn(t, memStorage(t), nil)
	s := subscribe(t, ts.URL, "1", "", nil)
	if !strings.Contains(s.opening.raw, `"subscribed"`) {
		t.Fatalf("opening %+v", s.opening)
	}
}

func getEvents(t *testing.T, base, tenant string, header http.Header) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, base+eventsPath, nil)
	req.Header.Set(middleware.TenantHeader, tenant)
	for k, vs := range header {
		req.Header[k] = vs
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// TestEvents_StoppingIs503: a subscription refused because the daemon stops is unavailable,
// not a failure of the daemon.
func TestEvents_StoppingIs503(t *testing.T) {
	ts, srv := eventServerOn(t, memStorage(t), nil)
	srv.events.Close()
	resp := getEvents(t, ts.URL, "1", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d; want 503", resp.StatusCode)
	}
}

// TestEvents_StreamsPerTenantAreCapped: beyond the cap a tenant gets 429; another tenant is
// not affected.
func TestEvents_StreamsPerTenantAreCapped(t *testing.T) {
	ts, _ := eventServerOn(t, memStorage(t), func(o *Options) { o.eventsMaxPerTenant = 2 })
	subscribe(t, ts.URL, "1", "", nil)
	subscribe(t, ts.URL, "1", "", nil)
	resp := getEvents(t, ts.URL, "1", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("third stream: %d; want 429", resp.StatusCode)
	}
	subscribe(t, ts.URL, "2", "", nil)
}

// TestEvents_CORSLetsLastEventIDThrough: a browser reconnecting sends Last-Event-ID; the
// preflight must allow it.
func TestEvents_CORSLetsLastEventIDThrough(t *testing.T) {
	ts, _ := eventServerOn(t, memStorage(t), func(o *Options) { o.CORSAllowOrigin = "*" })
	req, _ := http.NewRequest(http.MethodOptions, ts.URL+eventsPath, nil)
	req.Header.Set("Origin", "http://example.org")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !strings.Contains(resp.Header.Get("Access-Control-Allow-Headers"), "Last-Event-ID") {
		t.Fatalf("Allow-Headers %q", resp.Header.Get("Access-Control-Allow-Headers"))
	}
}

// TestEvents_StreamIsNotTimed: a stream lasts as long as its client; its duration would drown
// the latency histogram. It is counted, not timed.
func TestEvents_StreamIsNotTimed(t *testing.T) {
	ts, srv := eventServerOn(t, memStorage(t), func(o *Options) { o.EnableMetrics = true })
	s := subscribe(t, ts.URL, "1", "", nil)
	s.cancel()
	s.ended(t)
	var out strings.Builder
	waitFor(t, func() bool {
		out.Reset()
		srv.opts.Metrics.WritePrometheus(&out)
		return strings.Contains(out.String(), `blunderdb_http_requests_total{method="GET",path="/v1/events"`)
	}, "the stream is not counted")
	if strings.Contains(out.String(), `blunderdb_http_request_duration_seconds_count{method="GET",path="/v1/events"`) {
		t.Fatal("the stream's duration is in the latency histogram")
	}
}

// TestEvents_StreamIsNotCompressed: the stream asked for gzip and gets raw frames, flushed one
// by one; and it outlives the request timeout an ordinary call gets.
func TestEvents_StreamIsNotCompressed(t *testing.T) {
	ts, _ := eventServerOn(t, memStorage(t), func(o *Options) {
		o.RequestTimeout = 200 * time.Millisecond
		o.eventsHeartbeat = 100 * time.Millisecond
	})
	s := subscribe(t, ts.URL, "1", "", nil)
	if enc := s.resp.Header.Get("Content-Encoding"); enc != "" {
		t.Fatalf("Content-Encoding %q", enc)
	}
	deadline := time.Now().Add(600 * time.Millisecond)
	beats := 0
	for time.Now().Before(deadline) {
		select {
		case f, ok := <-s.frames:
			if !ok {
				t.Fatal("the stream ended within the request timeout's reach")
			}
			if f.comment == "ping" {
				beats++
			}
		case <-time.After(time.Until(deadline)):
		}
	}
	if beats < 3 {
		t.Fatalf("%d heartbeats in 600ms at 100ms", beats)
	}
}

func TestEvents_BadRequests(t *testing.T) {
	ts, _ := eventServerOn(t, memStorage(t), nil)
	for _, q := range []string{"?tournament=abc", "?tournament=0", "?table=1"} {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+eventsPath+q, nil)
		req.Header.Set(middleware.TenantHeader, "1")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: %d; want 400", q, resp.StatusCode)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, ts.URL+eventsPath, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("no tenant: %d; want 400", resp.StatusCode)
	}
}

// TestEvents_AbsentWithoutWriteFamily: with nothing that writes, there is nothing to announce.
func TestEvents_AbsentWithoutWriteFamily(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+eventsPath, nil)
	req.Header.Set(middleware.TenantHeader, "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status %d; want 404", resp.StatusCode)
	}
	if srv.eventsEnabled() {
		t.Error("events enabled without a write family")
	}
}

// TestEvents_DisconnectLeavesNoSubscriber: a client that leaves takes its subscription and its
// goroutine with it.
func TestEvents_DisconnectLeavesNoSubscriber(t *testing.T) {
	ts, srv := eventServerOn(t, memStorage(t), nil)
	var streams []*sseStream
	for range 5 {
		streams = append(streams, subscribe(t, ts.URL, "1", "", nil))
	}
	if n := srv.events.Subscribers(); n != 5 {
		t.Fatalf("%d subscribers; want 5", n)
	}
	for _, s := range streams {
		s.cancel()
		s.ended(t)
	}
	waitFor(t, func() bool { return srv.events.Subscribers() == 0 }, "subscribers left after disconnect")
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestEvents_ShutdownClosesStreams: stopping the daemon ends every stream at once, instead of
// waiting the shutdown timeout out.
func TestEvents_ShutdownClosesStreams(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	srv, err := New(Options{Storage: memStorage(t), Addr: addr, EnableDirection: true,
		ShutdownTimeout: 20 * time.Second, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	base := "http://" + addr
	waitFor(t, func() bool {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
		}
		return err == nil
	}, "the daemon did not listen")
	s := subscribe(t, base, "1", "", nil)
	start := time.Now()
	stop()
	s.ended(t)
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run waited on the open stream")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("shutdown took the timeout")
	}
}
