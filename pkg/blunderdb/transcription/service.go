package transcription

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// ErrSessionGone answers a gesture naming a session this service no longer
// holds: it expired, the process restarted, or another instance served the
// draft. Nothing typed is lost (the row is written after every gesture); the
// caller reopens the draft and continues at its end (ADR-0057 rule 3).
var ErrSessionGone = errors.New("transcription: session gone, reopen the draft")

// StaleError answers a gesture naming a revision the draft no longer has:
// another writer went first. Revision is the draft's current one and State
// the draft as it now is — document, session and Cursor, the undo stack
// emptied when the session had to be reloaded — which the caller redraws
// before replaying its gesture, if it still holds.
type StaleError struct {
	Revision int64
	State    *State
}

func (e *StaleError) Error() string {
	return fmt.Sprintf("transcription: stale revision, the draft is at revision %d", e.Revision)
}

// Unwrap makes a StaleError a storage.ErrConflict for every caller that maps
// storage errors.
func (e *StaleError) Unwrap() error { return storage.ErrConflict }

// ErrorDetails is what an API error envelope carries besides the message.
func (e *StaleError) ErrorDetails() map[string]any {
	return map[string]any{"revision": e.Revision, "state": e.State}
}

// Options configures a Service.
type Options struct {
	// TTL closes a session idle for longer; 0 keeps sessions until Close,
	// which is what the desktop wants (one user, one process).
	TTL time.Duration
	// Now is the clock, for tests; time.Now when nil.
	Now func() time.Time
}

// Expect is what a gesture claims to have seen. Session "" uses the draft's
// live session, opening one when there is none (the desktop, and `call`,
// where every invocation is its own process). Revision 0 claims nothing and
// the session's own revision is the expectation; the write still fails if
// another writer moved the row. The API refuses a gesture without a revision
// before it reaches the service (ADR-0057 rule 4).
type Expect struct {
	Session  string
	Revision int64
}

// State is what every gesture returns: the draft replayed, the revision the
// next gesture must name and the session it was typed in.
type State struct {
	ID        int64                `json:"id"`
	Revision  int64                `json:"revision"`
	SessionID string               `json:"sessionId,omitempty"`
	Annotated transcript.Annotated `json:"annotated"`
	CanUndo   bool                 `json:"canUndo"`
	CanRedo   bool                 `json:"canRedo"`
}

// Service holds the transcription logic over the storage contract and the
// sessions of the drafts being typed. One Service serves the desktop, the CLI
// and the daemon alike; every method takes the tenant as scope.
type Service struct {
	store storage.Storage
	ttl   time.Duration
	now   func() time.Time

	mu       sync.Mutex
	sessions map[key]*session

	// editMu keeps two "edit this Match" from opening two drafts on it.
	editMu sync.Mutex
}

type key struct {
	scope string
	id    int64
}

// session is the non-durable part of a draft being typed: the Entry and the
// undo stack live in the Editor (ADR-0045 rule 1), rev is the row's revision
// the Editor agrees with.
type session struct {
	mu     sync.Mutex
	id     string
	ed     *transcript.Editor
	rev    int64
	used   time.Time
	closed bool
}

// New returns a Service over store.
func New(store storage.Storage, o Options) *Service {
	now := o.Now
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, ttl: o.TTL, now: now, sessions: make(map[key]*session)}
}

// Forget drops every session, as when the library behind the service is
// replaced: a row id of the previous library must not answer for the next.
func (s *Service) Forget() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, ss := range s.sessions {
		ss.closed = true
		delete(s.sessions, k)
	}
}

// Close releases a draft's session only; the row stays and the draft resumes
// from the list. A named session that is not the live one is ErrSessionGone
// and the live one is left alone; "" closes whichever is live (the desktop).
func (s *Service) Close(scope string, id int64, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key{scope, id}
	ss := s.sessions[k]
	if sessionID != "" && (ss == nil || ss.id != sessionID) {
		return ErrSessionGone
	}
	if ss != nil {
		ss.closed = true
		delete(s.sessions, k)
	}
	return nil
}

// WithEditor runs fn on a draft's live Editor under its session lock and
// reports whether there was one. For inspection (tests, diagnostics): what
// fn changes is not written.
func (s *Service) WithEditor(scope string, id int64, fn func(*transcript.Editor)) bool {
	s.mu.Lock()
	ss := s.sessions[key{scope, id}]
	s.mu.Unlock()
	if ss == nil {
		return false
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.closed {
		return false
	}
	fn(ss.ed)
	return true
}

// sweep drops the sessions idle past the TTL. Caller holds s.mu.
func (s *Service) sweep() {
	if s.ttl <= 0 {
		return
	}
	limit := s.now().Add(-s.ttl)
	for k, ss := range s.sessions {
		if ss.used.Before(limit) {
			ss.closed = true
			delete(s.sessions, k)
		}
	}
}

// acquire returns the draft's session locked. A named session must be the
// live one; an unnamed one is the live session or a new one loaded from the
// row. The caller unlocks ss.mu.
func (s *Service) acquire(ctx context.Context, scope string, id int64, sessionID string) (*session, error) {
	for {
		s.mu.Lock()
		s.sweep()
		k := key{scope, id}
		ss := s.sessions[k]
		if sessionID != "" && (ss == nil || ss.id != sessionID) {
			s.mu.Unlock()
			return nil, ErrSessionGone
		}
		fresh := ss == nil
		if fresh {
			ss = &session{id: newSessionID()}
			s.sessions[k] = ss
		}
		ss.used = s.now()
		ss.mu.Lock()
		s.mu.Unlock()

		if ss.closed {
			// Dropped between the lookup and the lock: start over, or report
			// the named session gone.
			ss.mu.Unlock()
			if sessionID != "" {
				return nil, ErrSessionGone
			}
			continue
		}
		if fresh {
			if err := s.load(ctx, scope, id, ss); err != nil {
				s.drop(scope, id, ss)
				ss.mu.Unlock()
				return nil, err
			}
		}
		if err := s.forgetDeletedMatch(ctx, scope, id, ss); err != nil {
			s.drop(scope, id, ss)
			ss.mu.Unlock()
			return nil, err
		}
		return ss, nil
	}
}

// install registers a session for a draft just written, replacing any.
func (s *Service) install(scope string, id int64, doc transcript.Document, rev int64) *session {
	ss := &session{id: newSessionID(), ed: transcript.NewEditor(doc), rev: rev, used: s.now()}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key{scope, id}
	if old := s.sessions[k]; old != nil {
		old.closed = true
	}
	s.sessions[k] = ss
	return ss
}

// drop removes ss if it is still the draft's session. Caller holds ss.mu.
func (s *Service) drop(scope string, id int64, ss *session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key{scope, id}
	if s.sessions[k] == ss {
		delete(s.sessions, k)
	}
	ss.closed = true
}

// load reads the row into ss. Caller holds ss.mu.
func (s *Service) load(ctx context.Context, scope string, id int64, ss *session) error {
	doc, rev, err := s.read(ctx, scope, id)
	if err != nil {
		return err
	}
	ss.ed = transcript.NewEditor(doc)
	ss.rev = rev
	return nil
}

// forgetDeletedMatch takes a draft whose Match was deleted from the library
// back to "never saved": the column went NULL with the Match, but the
// document still names it, so every write would break the foreign key and
// the next finish would replace a Match that is gone. The row is rewritten at
// once so that it agrees with its column. Caller holds ss.mu.
func (s *Service) forgetDeletedMatch(ctx context.Context, scope string, id int64, ss *session) error {
	m := ss.ed.Doc.Header.MatchID
	if m == nil {
		return nil
	}
	_, err := s.store.Matches().Get(ctx, scope, *m)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, storage.ErrNotFound):
		return err
	}
	ss.ed.SetMatchID(nil)
	return s.write(ctx, scope, id, ss, nil)
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("transcription: session id: %v", err))
	}
	return hex.EncodeToString(b[:])
}
