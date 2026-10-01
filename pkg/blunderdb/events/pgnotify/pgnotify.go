// Package pgnotify carries the events of one serve daemon to the others that share its
// PostgreSQL database, with LISTEN/NOTIFY. A Transport is the events.Publisher such a daemon
// hands its emitters: Publish delivers the event to the local Bus at once, then sends it on one
// channel; a dedicated LISTEN connection hands every notification of another instance to the
// local Bus, whose subscribers see it as they see a local one.
//
// # NOTIFY after the commit
//
// An emitter publishes once its transaction has committed, and an event carries what a read
// after the commit sees — the Direction-Version, the draft's revision. A pg_notify inside the
// gesture's transaction would be delivered only at its COMMIT, which is stronger, but the
// emitter would have to hold the transaction and know the version before it commits; it holds
// neither, and the in-memory bus already relies on the publish-after-commit contract. So the
// Transport sends after the commit, on its own connection. What that costs: a daemon that dies
// between its commit and its NOTIFY never tells the other instances. Their clients learn of the
// gesture at their next resync — a reconnection, an overflow, another instance's restart.
//
// # One channel, the scope in the payload
//
// Every instance listens on Channel and the payload names the scope (the tenant): a channel
// name is an identifier of at most 63 bytes, a tenant id is not. The receiving Bus delivers an
// event only to the subscribers of its scope, so the isolation by tenant holds on reception
// exactly as it does locally. A payload is at most MaxPayload bytes, under PostgreSQL's limit;
// an event that would not fit is sent as a resync of its scope instead.
//
// # Own events, sequence, losses
//
// Each Transport draws a random instance id and puts it in its payloads: it ignores its own
// notifications, which its Bus already delivered. Each instance numbers its deliveries per
// scope on its own, so an SSE id means nothing to another instance: a client that a load
// balancer moves to another instance reconnects, and the resync that opens every stream tells
// it to read everything again.
//
// When the LISTEN connection drops, the notifications sent meanwhile are lost — PostgreSQL
// keeps none for a session that is gone. The Transport reconnects with an exponential backoff
// and, once it listens again, sends a resync to every local subscriber. When a NOTIFY cannot be
// sent (the database unreachable, the queue full), the scope is remembered and a resync of it is
// sent to the other instances as soon as one can be.
package pgnotify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/events"
)

// Channel is the one PostgreSQL channel every instance notifies and listens on.
const Channel = "blunderdb_events"

// MaxPayload bounds a notification's payload, under PostgreSQL's 8000-byte limit.
const MaxPayload = 7900

const (
	// queueSize bounds the notifications waiting to be sent; beyond it Publish does not wait,
	// it remembers the scope to resync.
	queueSize = 1024
	// sendTimeout bounds one NOTIFY.
	sendTimeout = 5 * time.Second
	// closeTimeout bounds what Close spends on UNLISTEN and on the notifications still queued.
	closeTimeout = 2 * time.Second
	// ReasonMissed is the reason of the resync a Transport sends when it may have lost events.
	ReasonMissed = "missed"
)

// Options tunes a Transport. The zero value is usable.
type Options struct {
	Logger *slog.Logger
	// MinBackoff and MaxBackoff bound the wait between two attempts to listen again, or to send
	// a pending resync (defaults 250 ms and 30 s).
	MinBackoff, MaxBackoff time.Duration
}

// wire is a notification's payload. Event is nil for a resync of Scope.
type wire struct {
	Instance string        `json:"i"`
	Scope    string        `json:"s"`
	Event    *events.Event `json:"e,omitempty"`
}

// Transport is the events.Publisher of an instance among several on one PostgreSQL database.
type Transport struct {
	bus      *events.Bus
	instance string
	log      *slog.Logger
	listenCf *pgx.ConnConfig
	pool     *pgxpool.Pool
	min, max time.Duration

	queue  chan wire
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once

	mu      sync.Mutex
	closed  bool
	pending map[string]bool // scopes whose events another instance may have missed
}

// Start connects to dsn, listens on Channel and returns the running Transport, which feeds bus.
// The first LISTEN must succeed: a daemon that cannot listen at start refuses to start rather
// than serve streams that would miss the other instances' gestures.
func Start(ctx context.Context, dsn string, bus *events.Bus, o Options) (*Transport, error) {
	if o.Logger == nil {
		o.Logger = slog.New(slog.DiscardHandler)
	}
	if o.MinBackoff <= 0 {
		o.MinBackoff = 250 * time.Millisecond
	}
	if o.MaxBackoff < o.MinBackoff {
		o.MaxBackoff = max(30*time.Second, o.MinBackoff)
	}
	var id [8]byte
	_, _ = rand.Read(id[:])
	t := &Transport{
		bus: bus, instance: hex.EncodeToString(id[:]), log: o.Logger, min: o.MinBackoff, max: o.MaxBackoff,
		queue: make(chan wire, queueSize), pending: map[string]bool{},
	}

	cf, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgnotify: %w", err)
	}
	// The name lets an operator tell the listener among the backends (pg_stat_activity).
	cf.RuntimeParams["application_name"] = "blunderdb-events-" + t.instance
	t.listenCf = cf
	pcf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgnotify: %w", err)
	}
	pcf.MaxConns = 1 // one sender goroutine
	pcf.ConnConfig.RuntimeParams["application_name"] = "blunderdb-notify-" + t.instance
	if t.pool, err = pgxpool.NewWithConfig(ctx, pcf); err != nil {
		return nil, fmt.Errorf("pgnotify: %w", err)
	}
	conn, err := t.connect(ctx)
	if err != nil {
		t.pool.Close()
		return nil, fmt.Errorf("pgnotify: listen: %w", err)
	}

	run, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.wg.Add(2)
	go t.listen(run, conn)
	go t.send(run)
	return t, nil
}

// Instance is the id this Transport puts in its notifications.
func (t *Transport) Instance() string { return t.instance }

// Wants is true until Close: another instance may have a subscriber of any scope, and this one
// cannot know.
func (t *Transport) Wants(string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return !t.closed
}

// Publish delivers ev to the local subscribers and queues it for the other instances, never
// waiting on either.
func (t *Transport) Publish(ev events.Event) {
	t.bus.Publish(ev)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	e := ev
	select {
	case t.queue <- wire{Instance: t.instance, Scope: ev.Scope, Event: &e}:
	default:
		t.log.Warn("events: notification queue full; the other instances will resync", "scope", ev.Scope)
		t.pending[ev.Scope] = true
	}
}

// Close stops listening (UNLISTEN, then the connection closes), sends what is still queued
// within a short delay, and waits for its goroutines. Idempotent.
func (t *Transport) Close() {
	t.once.Do(func() {
		t.mu.Lock()
		t.closed = true
		t.mu.Unlock()
		t.cancel()
		t.wg.Wait()
		t.pool.Close()
	})
}

func (t *Transport) connect(ctx context.Context) (*pgx.Conn, error) {
	conn, err := pgx.ConnectConfig(ctx, t.listenCf)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(ctx, "LISTEN "+Channel); err != nil {
		_ = conn.Close(context.Background())
		return nil, err
	}
	return conn, nil
}

// listen hands the other instances' notifications to the bus until ctx ends, reconnecting with
// a backoff when the connection drops; a reconnection resyncs every local subscriber.
func (t *Transport) listen(ctx context.Context, conn *pgx.Conn) {
	defer t.wg.Done()
	for {
		err := t.receive(ctx, conn)
		t.hangUp(conn)
		if ctx.Err() != nil {
			return
		}
		t.log.Warn("events: lost the LISTEN connection; reconnecting", "err", err)
		for wait := t.min; ; wait = min(2*wait, t.max) {
			if !sleep(ctx, wait) {
				return
			}
			if conn, err = t.connect(ctx); err == nil {
				break
			}
			t.log.Warn("events: LISTEN reconnection failed", "err", err, "retry", min(2*wait, t.max))
		}
		t.log.Info("events: listening again; local subscribers resync")
		t.bus.ResyncAll(ReasonMissed)
	}
}

// receive waits for notifications until the connection fails or ctx ends.
func (t *Transport) receive(ctx context.Context, conn *pgx.Conn) error {
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var w wire
		if err := json.Unmarshal([]byte(n.Payload), &w); err != nil {
			t.log.Warn("events: unreadable notification ignored", "err", err)
			continue
		}
		switch {
		case w.Instance == t.instance:
			// Already delivered by Publish.
		case w.Event == nil:
			t.bus.Resync(w.Scope, ReasonMissed)
		case w.Event.Kind == events.KindResync:
			t.bus.Resync(w.Scope, ReasonMissed)
		default:
			ev := *w.Event
			ev.Scope = w.Scope
			t.bus.Publish(ev)
		}
	}
}

// hangUp unlistens and closes conn. A connection whose wait was cancelled is still open (the
// cancellation is a read deadline), so the UNLISTEN reaches the server; one that failed is not,
// and closing it is all there is to do.
func (t *Transport) hangUp(conn *pgx.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()
	if !conn.IsClosed() {
		_, _ = conn.Exec(ctx, "UNLISTEN "+Channel)
	}
	_ = conn.Close(ctx)
}

// send sends the queued notifications, and the resyncs of the scopes whose events could not be,
// until ctx ends; then it sends what is still queued within closeTimeout.
func (t *Transport) send(ctx context.Context) {
	defer t.wg.Done()
	retry := time.NewTimer(t.min)
	retry.Stop()
	wait := t.min
	for {
		select {
		case w := <-t.queue:
			t.notify(ctx, w)
		case <-retry.C:
		case <-ctx.Done():
			t.drain()
			return
		}
		if t.flushPending(ctx) {
			wait = t.min
			continue
		}
		retry.Reset(wait)
		wait = min(2*wait, t.max)
	}
}

// drain sends what Publish queued before Close, within closeTimeout.
func (t *Transport) drain() {
	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()
	for {
		select {
		case w := <-t.queue:
			t.notify(ctx, w)
		default:
			return
		}
	}
}

// notify sends w, as a resync of its scope when it does not fit; a failure remembers the scope.
func (t *Transport) notify(ctx context.Context, w wire) {
	payload, err := json.Marshal(w)
	if err == nil && len(payload) > MaxPayload {
		payload, err = json.Marshal(wire{Instance: w.Instance, Scope: w.Scope})
	}
	if err == nil && len(payload) > MaxPayload {
		err = errors.New("the scope alone exceeds the payload limit")
	}
	if err == nil {
		err = t.exec(ctx, payload)
	}
	if err != nil {
		t.log.Warn("events: NOTIFY failed; the other instances will resync", "scope", w.Scope, "err", err)
		t.mu.Lock()
		t.pending[w.Scope] = true
		t.mu.Unlock()
	}
}

// flushPending sends a resync of every remembered scope, and reports whether none is left.
func (t *Transport) flushPending(ctx context.Context) bool {
	t.mu.Lock()
	scopes := make([]string, 0, len(t.pending))
	for s := range t.pending {
		scopes = append(scopes, s)
	}
	t.mu.Unlock()
	for _, s := range scopes {
		payload, err := json.Marshal(wire{Instance: t.instance, Scope: s})
		if err != nil || len(payload) > MaxPayload {
			t.mu.Lock()
			delete(t.pending, s)
			t.mu.Unlock()
			continue
		}
		if t.exec(ctx, payload) != nil {
			return false
		}
		t.mu.Lock()
		delete(t.pending, s)
		t.mu.Unlock()
	}
	return true
}

func (t *Transport) exec(ctx context.Context, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	_, err := t.pool.Exec(ctx, "SELECT pg_notify($1, $2)", Channel, string(payload))
	return err
}

// sleep waits d, and reports false when ctx ended first.
func sleep(ctx context.Context, d time.Duration) bool {
	tm := time.NewTimer(d)
	defer tm.Stop()
	select {
	case <-tm.C:
		return true
	case <-ctx.Done():
		return false
	}
}
