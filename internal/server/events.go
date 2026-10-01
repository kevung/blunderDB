package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// GET /v1/events streams, as Server-Sent Events, one short message per committed gesture of
// the caller's tenant (ADR-0057 rule 6): what moved and its new version, never the state —
// the client reads again with If-None-Match. The route is served only when a write family is
// (--direction or --transcription): without one, nothing in this daemon writes what it would
// announce.
//
// The route is registered outside domainRoutes on purpose: it is a GET that never ends, not a
// POST /v1/<family>.<method> call, so `call` cannot dispatch it and the generated contract
// documents it by hand.
const eventsPath = "/v1/events"

const (
	// defaultEventsHeartbeat keeps a proxy from closing a quiet stream (most cut an idle
	// connection between 30 and 60 seconds).
	defaultEventsHeartbeat = 25 * time.Second
	// defaultEventsBuffer bounds what a subscriber may fall behind before it is dropped.
	defaultEventsBuffer = 64
	// eventsWriteTimeout bounds each write: a client that stopped reading is cut off instead of
	// holding a goroutine for ever.
	eventsWriteTimeout = 30 * time.Second
	// eventsRetry is the reconnection delay the stream advises (milliseconds).
	eventsRetry = 3000
)

// unboundedPaths are the routes withDeadlines leaves without a deadline: a stream that lasts
// as long as its client listens. The handler bounds each write itself.
var unboundedPaths = map[string]bool{eventsPath: true}

// eventsEnabled reports whether the daemon serves /v1/events and publishes its gestures.
func (s *Server) eventsEnabled() bool {
	return s.opts.EnableDirection || s.opts.Transcription
}

func (s *Server) eventRoutes() []route {
	if !s.eventsEnabled() {
		return nil
	}
	return []route{{http.MethodGet, eventsPath, s.handleEvents}}
}

// eventFilterParams are the query parameters a subscription accepts, each a list of ids
// (repeated, or comma-separated).
var eventFilterParams = []string{"tournament", "rencontre", "transcription"}

func parseEventFilter(q url.Values) (events.Filter, error) {
	var f events.Filter
	for k := range q {
		known := false
		for _, p := range eventFilterParams {
			known = known || k == p
		}
		if !known {
			return f, fmt.Errorf("%w: unknown parameter %q; a subscription filters on %s",
				storage.ErrInvalid, k, strings.Join(eventFilterParams, ", "))
		}
	}
	lists := []*[]int64{&f.Tournaments, &f.Rencontres, &f.Transcriptions}
	for i, p := range eventFilterParams {
		for _, v := range q[p] {
			for _, part := range strings.Split(v, ",") {
				id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
				if err != nil || id <= 0 {
					return f, fmt.Errorf("%w: %s=%q is not an id", storage.ErrInvalid, p, part)
				}
				*lists[i] = append(*lists[i], id)
			}
		}
	}
	return f, nil
}

// handleEvents serves one subscription until its client leaves, its queue overflows or the
// daemon stops. No history is kept: a client that reconnects (Last-Event-ID) or is dropped
// for falling behind receives a resync event, and reads everything it shows again.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	f, err := parseEventFilter(r.URL.Query())
	if err != nil {
		writeStorageError(w, err)
		return
	}
	sub, err := s.events.Subscribe(scopeOf(r), f, s.opts.eventsBuffer)
	if err != nil {
		writeErrorCode(w, CodeInternal, "the daemon is stopping")
		return
	}
	defer sub.Cancel()

	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	// nginx buffers a proxied response unless told not to.
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	write := func(b []byte) bool {
		_ = rc.SetWriteDeadline(time.Now().Add(eventsWriteTimeout))
		if _, err := w.Write(b); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	head := fmt.Sprintf("retry: %d\n: subscribed\n\n", eventsRetry)
	if r.Header.Get("Last-Event-ID") != "" {
		head += resyncFrame("reconnected")
	}
	if !write([]byte(head)) {
		return
	}

	beat := time.NewTicker(s.opts.eventsHeartbeat)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-beat.C:
			if !write([]byte(": ping\n\n")) {
				return
			}
		case d, ok := <-sub.C:
			if !ok {
				if sub.Overflowed() {
					write([]byte(resyncFrame("overflow")))
				}
				return
			}
			data, err := json.Marshal(d.Event)
			if err != nil {
				return
			}
			frame := fmt.Sprintf("id: %s-%d\nevent: %s\ndata: %s\n\n", s.eventsEpoch, d.Seq, d.Event.Kind, data)
			if !write([]byte(frame)) {
				return
			}
		}
	}
}

// resyncFrame tells the client it may have missed events: it reads everything again.
func resyncFrame(reason string) string {
	data, _ := json.Marshal(map[string]string{"kind": string(events.KindResync), "reason": reason})
	return fmt.Sprintf("event: %s\ndata: %s\n\n", events.KindResync, data)
}
