// Package events tells a client that another one wrote (ADR-0057 rule 6). A gesture that
// committed publishes one short Event — what it touched and the version or revision it left,
// never the state itself: the client reads what interests it again, with If-None-Match, so
// there is one format of reading and not two.
//
// # Emitters and transports
//
// An emitter (the Direction service, the transcription service) holds a Publisher and calls
// Publish once its transaction has committed; a gesture that rolled back or was refused
// publishes nothing. It never learns where the event goes.
//
// Bus is the in-process transport: it fans an event out to the subscribers of the event's
// scope. A transport across processes — PostgreSQL LISTEN/NOTIFY for several serve daemons on
// one database — is a second Publisher: its Publish sends the event to the database, and its
// listener hands every notification it receives to the local Bus.Publish. The emitters do not
// change; only what the server hands them does.
//
// # Delivery
//
// Publish never blocks on a subscriber. Each subscription has a bounded queue; one that is
// full when an event arrives is closed, flagged Overflowed, and its client reconnects and
// reads everything again. No history is kept: a reconnecting client cannot be told what it
// missed, only that it may have missed something (the serve daemon's resync event).
package events

import (
	"errors"
	"slices"
	"sync"
)

// Kind names what an Event is about.
type Kind string

const (
	// KindDirection: a Direction that plays in no Rencontre moved. TournamentID names it.
	KindDirection Kind = "direction"
	// KindRencontre: a Rencontre or one of its Directions moved — a gesture in one sister moves
	// the version of all. RencontreID names the room, TournamentIDs its members.
	KindRencontre Kind = "rencontre"
	// KindTranscription: a draft moved. TranscriptionID names it, Revision is its new one.
	KindTranscription Kind = "transcription"
	// KindResync tells a client it may have missed events and should read everything again.
	// The bus never publishes it; the serve daemon sends it to a client that reconnects.
	KindResync Kind = "resync"
)

// Event is one committed change: what it touched and the version it left.
type Event struct {
	// Scope is the tenant the change belongs to ("" for a private database). Only the
	// subscribers of that scope receive the event; it is not part of the wire form.
	Scope string `json:"-"`

	Kind            Kind    `json:"kind"`
	TournamentID    int64   `json:"tournamentId,omitempty"`
	TournamentIDs   []int64 `json:"tournamentIds,omitempty"`
	RencontreID     int64   `json:"rencontreId,omitempty"`
	TranscriptionID int64   `json:"transcriptionId,omitempty"`
	// Version is the Direction-Version of what the event names after the change, as a read
	// would carry it; empty when it could not be read back (Removed, or a concurrent writer).
	Version string `json:"version,omitempty"`
	// Revision is a draft's revision after the change.
	Revision int64 `json:"revision,omitempty"`
	// Removed: what the event names no longer exists (a trashed room, an abandoned or
	// finished draft).
	Removed bool `json:"removed,omitempty"`
	// MatchID is the Match a finished draft was saved as.
	MatchID int64 `json:"matchId,omitempty"`
}

// Publisher is what an emitter holds. Publish must not block on any subscriber.
type Publisher interface {
	Publish(Event)
}

// Filter narrows a subscription. An empty Filter receives every event of its scope; otherwise
// an event is delivered when it names any of the listed Directions, Rencontres or drafts.
type Filter struct {
	Tournaments    []int64
	Rencontres     []int64
	Transcriptions []int64
}

// Empty reports whether the filter lets every event through.
func (f Filter) Empty() bool {
	return len(f.Tournaments) == 0 && len(f.Rencontres) == 0 && len(f.Transcriptions) == 0
}

// Match reports whether ev passes the filter.
func (f Filter) Match(ev Event) bool {
	if f.Empty() {
		return true
	}
	if ev.TournamentID != 0 && slices.Contains(f.Tournaments, ev.TournamentID) {
		return true
	}
	for _, id := range ev.TournamentIDs {
		if slices.Contains(f.Tournaments, id) {
			return true
		}
	}
	if ev.RencontreID != 0 && slices.Contains(f.Rencontres, ev.RencontreID) {
		return true
	}
	return ev.TranscriptionID != 0 && slices.Contains(f.Transcriptions, ev.TranscriptionID)
}

// Delivery is an event as a subscriber receives it, with the bus's sequence number.
type Delivery struct {
	Seq   uint64
	Event Event
}

// ErrClosed refuses a subscription on a bus that was closed (the daemon is stopping).
var ErrClosed = errors.New("events: the bus is closed")

// Bus fans events out in memory to the subscribers of their scope.
type Bus struct {
	mu     sync.Mutex
	subs   map[*Subscription]struct{}
	seq    uint64
	closed bool
}

// NewBus returns an open, empty bus.
func NewBus() *Bus {
	return &Bus{subs: map[*Subscription]struct{}{}}
}

// Subscription receives the events of one scope that pass its filter, on C, until it is
// cancelled, its queue overflows, or the bus closes; C is then closed.
type Subscription struct {
	C <-chan Delivery

	bus        *Bus
	c          chan Delivery
	scope      string
	filter     Filter
	overflowed bool
}

// Subscribe opens a subscription to scope's events with a queue of buffer deliveries (at
// least 1).
func (b *Bus) Subscribe(scope string, f Filter, buffer int) (*Subscription, error) {
	if buffer < 1 {
		buffer = 1
	}
	c := make(chan Delivery, buffer)
	s := &Subscription{C: c, bus: b, c: c, scope: scope, filter: f}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrClosed
	}
	b.subs[s] = struct{}{}
	return s, nil
}

// Publish delivers ev to every matching subscriber of its scope without waiting: a subscriber
// whose queue is full is dropped, never waited for. Publishing on a closed bus does nothing.
func (b *Bus) Publish(ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.seq++
	d := Delivery{Seq: b.seq, Event: ev}
	for s := range b.subs {
		if s.scope != ev.Scope || !s.filter.Match(ev) {
			continue
		}
		select {
		case s.c <- d:
		default:
			s.overflowed = true
			b.drop(s)
		}
	}
}

// Close ends every subscription and refuses new ones. Idempotent.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for s := range b.subs {
		b.drop(s)
	}
}

// Subscribers is the number of live subscriptions.
func (b *Bus) Subscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

// drop removes s and closes its channel. Caller holds b.mu.
func (b *Bus) drop(s *Subscription) {
	if _, ok := b.subs[s]; !ok {
		return
	}
	delete(b.subs, s)
	close(s.c)
}

// Cancel ends the subscription. Idempotent, and safe after the bus dropped it.
func (s *Subscription) Cancel() {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	s.bus.drop(s)
}

// Overflowed reports whether the bus dropped the subscription because its queue was full.
// Meaningful once C is closed.
func (s *Subscription) Overflowed() bool {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	return s.overflowed
}

// Discard is a Publisher that drops every event: an emitter with no one to tell.
type Discard struct{}

// Publish does nothing.
func (Discard) Publish(Event) {}
