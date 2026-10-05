package duel

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// ErrNotOpen: the Duel is in suspense, and another one is open. Only the open
// Duel is played (ADR-0072 rule 10); Open switches.
var ErrNotOpen = errors.New("duel is not the open one")

// Settings is what a Duel is created with (ADR-0072 rule 12).
type Settings struct {
	// MatchLength is 1 to 25 points, or 0 for a money session; Jacoby is a
	// money session's only rule to choose.
	MatchLength int  `json:"matchLength"`
	Jacoby      bool `json:"jacoby"`
	// Start is the Position the first game begins at; nil is the opening
	// position at the start of the match.
	Start *domain.Position `json:"start,omitempty"`
	// Sides are player 1's and player 2's; a Side without a kind is external.
	Sides [2]SideSpec `json:"sides"`
	// DiscardAtEnd throws the draft away when the match is won, instead of
	// writing the Match — which is the default.
	DiscardAtEnd bool `json:"discardAtEnd,omitempty"`
	// Cadence is the Duel's clock; nil, the default, plays without one. The
	// durations of the decisions are measured either way.
	Cadence *Cadence `json:"cadence,omitempty"`
}

// Options configures a Service.
type Options struct {
	// Sides resolves a draft's SideSpec into the Side the Arbiter asks;
	// Resolve when nil.
	Sides SideResolver
	// Rand is where seeds are drawn from; crypto/rand when nil.
	Rand io.Reader
	// Now dates a new Duel's Match; time.Now when nil.
	Now func() time.Time
}

// Service is the Arbiter over storage.DuelStore: it creates, resumes and
// ends Duels, and writes the draft after every Play. A Duel is read from its
// row at every call and replayed: nothing but which Duel is open lives in
// memory, so losing the process loses nothing played.
type Service struct {
	store storage.Storage
	opts  Options

	mu   sync.Mutex
	open map[string]int64
}

// New returns a Service over store.
func New(store storage.Storage, opts Options) *Service {
	if opts.Sides == nil {
		opts.Sides = Resolve
	}
	if opts.Rand == nil {
		opts.Rand = rand.Reader
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Service{store: store, opts: opts, open: map[string]int64{}}
}

// State is a Duel as a caller sees it. It never carries the seed: the
// fingerprint stands for it until the Match reveals it.
type State struct {
	ID          int64                 `json:"id"`
	Revision    int64                 `json:"revision"`
	Header      transcript.Header     `json:"header"`
	Start       *domain.Position      `json:"start,omitempty"`
	Sides       [2]SideSpec           `json:"sides"`
	Fingerprint string                `json:"fingerprint"`
	Actions     []transcript.Action   `json:"actions"`
	Games       []transcript.GameInfo `json:"games"`
	Score       [2]int                `json:"score"`
	// Awaiting is the Decision a Side owes; nil once the Duel has ended.
	Awaiting *Decision `json:"awaiting,omitempty"`
	// Clock is the Cadence's clock, nil without one.
	Clock *ClockState `json:"clock,omitempty"`
	// Ended is set by the call that ended the Duel.
	Ended *Ending `json:"ended,omitempty"`
}

// Ending is how a Duel ended: the Match it became, with its seed revealed, or
// nothing written at all.
type Ending struct {
	// MatchID is the Match written, 0 when the draft was thrown away.
	MatchID int64 `json:"matchId"`
	// DiceSeed is revealed with the Match; "" when nothing was written.
	DiceSeed     string `json:"diceSeed,omitempty"`
	StoppedEarly bool   `json:"stoppedEarly,omitempty"`
	Discarded    bool   `json:"discarded,omitempty"`
	// OverTime is the player (1 or 2) whose reserve ran out first, 0 none.
	OverTime int `json:"overTime,omitempty"`
}

// Summary is a Duel in suspense, as a list shows it.
type Summary struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Open      bool   `json:"open"`
}

// Create draws the seed, checks the session and the Start, plays what the
// Arbiter plays alone up to the first Decision, writes the draft and opens it.
// A Start the rules do not allow is refused with its *transcript.Refusal.
func (s *Service) Create(ctx context.Context, scope string, set Settings) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	seed, err := newSeed(s.opts.Rand)
	if err != nil {
		return nil, err
	}
	fp, err := Fingerprint(seed)
	if err != nil {
		return nil, err
	}
	if set.Cadence != nil {
		if err := set.Cadence.check(set.MatchLength); err != nil {
			return nil, err
		}
		cad := *set.Cadence
		if cad.TimeOut == "" {
			cad.TimeOut = TimeContinue
		}
		set.Cadence = &cad
	}
	for i := range set.Sides {
		if set.Sides[i].Kind == "" {
			set.Sides[i].Kind = SideExternal
		}
		if set.Sides[i].Kind == SideBot {
			set.Sides[i].Name = BotName(set.Sides[i].Level)
		}
	}
	if set.MatchLength == 0 && set.Sides[0].Kind != SideExternal && set.Sides[1].Kind != SideExternal {
		// Nobody would ever be awaited and a money session never ends: the
		// call would play forever.
		return nil, &transcript.Refusal{Kind: RefusedEndless,
			Detail: "a money session between two delegated Sides never ends"}
	}
	doc := document{
		FormatVersion: FormatVersion,
		Header: transcript.Header{
			MatchLength: set.MatchLength, Jacoby: set.Jacoby,
			Player1: set.Sides[0].Name, Player2: set.Sides[1].Name,
			Date: s.opts.Now(),
		},
		Start:        set.Start,
		Sides:        set.Sides,
		DiscardAtEnd: set.DiscardAtEnd,
		Fingerprint:  fp,
		Cadence:      set.Cadence,
	}
	g, err := newGame(doc, seed)
	if err != nil {
		return nil, err
	}
	g.now = s.opts.Now
	g.setReserves()
	sides, err := s.sides(g.doc.Sides)
	if err != nil {
		return nil, err
	}
	if err := g.settle(ctx, sides); err != nil {
		return nil, err
	}
	row := &storage.Duel{FormatVersion: strconv.Itoa(FormatVersion), Label: label(g.doc.Header), DiceSeed: seed}
	if row.Document, err = encode(g.doc); err != nil {
		return nil, err
	}
	if err := s.suspendOpen(ctx, scope); err != nil {
		return nil, err
	}
	row.ID, err = s.store.Duels().Save(ctx, scope, row)
	if err != nil {
		return nil, err
	}
	s.open[scope] = row.ID
	if g.finished() || g.timeLost() {
		// Two delegated Sides play the whole match in this call.
		return s.end(ctx, scope, row, g, !g.doc.DiscardAtEnd)
	}
	return state(row, g), nil
}

// List returns the scope's Duels in suspense, most recently played first.
func (s *Service) List(ctx context.Context, scope string) ([]Summary, error) {
	s.mu.Lock()
	open := s.open[scope]
	s.mu.Unlock()
	var out []Summary
	for row, err := range s.store.Duels().List(ctx, scope) {
		if err != nil {
			return nil, err
		}
		out = append(out, Summary{ID: row.ID, Label: row.Label, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Open: row.ID == open})
	}
	return out, nil
}

// Open resumes a Duel where it stopped, with the same dice to come, and makes
// it the open one: the Duel open before it goes into suspense. The clocks
// stand still in suspense and start again now (ADR-0073).
func (s *Service) Open(ctx context.Context, scope string, id int64) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.openLocked(ctx, scope, id, 0)
}

// openLocked is Open under s.mu, refusing with storage.ErrConflict a draft
// that is no longer at the revision the caller saw (0: none).
func (s *Service) openLocked(ctx context.Context, scope string, id, revision int64) (*State, error) {
	row, g, err := s.load(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	if revision != 0 && revision != row.Revision {
		return nil, fmt.Errorf("duel %d at revision %d, not %d: %w", id, row.Revision, revision, storage.ErrConflict)
	}
	if s.open[scope] == id {
		return state(row, g), nil
	}
	if err := s.suspendOpen(ctx, scope); err != nil {
		return nil, err
	}
	g.resume(s.opts.Now())
	if err := s.save(ctx, scope, row, g); err != nil {
		return nil, err
	}
	s.open[scope] = id
	return state(row, g), nil
}

// Suspend puts the open Duel in suspense, its clocks stopped, and leaves no
// Duel open; Open resumes it.
func (s *Service) Suspend(ctx context.Context, scope string, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open[scope] != id {
		return fmt.Errorf("duel %d: %w", id, ErrNotOpen)
	}
	return s.suspendOpen(ctx, scope)
}

// suspendOpen stops the clocks of the scope's open Duel, if any, and leaves
// none open. One gone meanwhile has no clock left to stop.
func (s *Service) suspendOpen(ctx context.Context, scope string) error {
	id := s.open[scope]
	if id == 0 {
		return nil
	}
	row, g, err := s.load(ctx, scope, id)
	if errors.Is(err, storage.ErrNotFound) {
		delete(s.open, scope)
		return nil
	}
	if err != nil {
		return err
	}
	g.suspend(s.opts.Now())
	if err := s.save(ctx, scope, row, g); err != nil {
		return err
	}
	delete(s.open, scope)
	return nil
}

// Flag has the Arbiter look at the clock of the Side the open Duel awaits:
// run out, it is noted, and under TimeLoseMatch the Duel ends there (the
// tournament rules count the time as gone when it is noticed). Without a
// Cadence, or with time left, nothing changes.
func (s *Service) Flag(ctx context.Context, scope string, id int64) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open[scope] != id {
		return nil, fmt.Errorf("duel %d: %w", id, ErrNotOpen)
	}
	row, g, err := s.load(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	if !g.flag(s.opts.Now()) {
		return state(row, g), nil
	}
	if g.timeLost() {
		return s.end(ctx, scope, row, g, !g.doc.DiscardAtEnd)
	}
	if err := s.save(ctx, scope, row, g); err != nil {
		return nil, err
	}
	return state(row, g), nil
}

// Play applies a Side's Play to the open Duel under the revision the caller
// saw (0: none), lets the Arbiter play on to the next Decision of an external
// Side, and writes the draft. A Play the rules do not allow is refused with
// its *transcript.Refusal and nothing is written. When the match is won, the
// Duel ends: its Match is written, or the draft thrown away if it was created
// so (State.Ended).
func (s *Service) Play(ctx context.Context, scope string, id, revision int64, p Play) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open[scope] != id {
		return nil, fmt.Errorf("duel %d: %w", id, ErrNotOpen)
	}
	row, g, err := s.load(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	if revision != 0 && revision != row.Revision {
		return nil, fmt.Errorf("duel %d at revision %d, not %d: %w", id, row.Revision, revision, storage.ErrConflict)
	}
	p.At = s.opts.Now()
	if err := g.receive(p); err != nil {
		return nil, err
	}
	sides, err := s.sides(g.doc.Sides)
	if err != nil {
		return nil, err
	}
	if err := g.settle(ctx, sides); err != nil {
		return nil, err
	}
	if g.finished() || g.timeLost() {
		return s.end(ctx, scope, row, g, !g.doc.DiscardAtEnd)
	}
	if err := s.save(ctx, scope, row, g); err != nil {
		return nil, err
	}
	return state(row, g), nil
}

// save rewrites the draft under the row's revision.
func (s *Service) save(ctx context.Context, scope string, row *storage.Duel, g *game) error {
	var err error
	if row.Document, err = encode(g.doc); err != nil {
		return err
	}
	_, err = s.store.Duels().Save(ctx, scope, row)
	return err
}

// Stop ends the open Duel before its end: keep writes the Match as it stands
// — an unfinished game keeps no winner, and a match short of its length is
// marked stopped early — and otherwise the draft is thrown away and nothing of
// the Duel is written. Stopping is never resigning: that is a Play.
func (s *Service) Stop(ctx context.Context, scope string, id, revision int64, keep bool) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open[scope] != id {
		return nil, fmt.Errorf("duel %d: %w", id, ErrNotOpen)
	}
	row, g, err := s.load(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	if revision != 0 && revision != row.Revision {
		return nil, fmt.Errorf("duel %d at revision %d, not %d: %w", id, row.Revision, revision, storage.ErrConflict)
	}
	return s.end(ctx, scope, row, g, keep)
}

// end writes the Match and its origin and deletes the draft in one
// transaction whose first write checks the revision, or deletes the draft
// alone.
func (s *Service) end(ctx context.Context, scope string, row *storage.Duel, g *game, keep bool) (*State, error) {
	parts := g.parts()
	// Over time before anything was played leaves no Match to write.
	if keep && len(parts.Games) == 0 && g.timeLost() {
		keep = false
	}
	if !keep {
		if err := s.store.Duels().Delete(ctx, scope, row.ID); err != nil {
			return nil, err
		}
		delete(s.open, scope)
		st := state(row, g)
		st.Ended, st.Awaiting = &Ending{Discarded: true, OverTime: g.doc.Clock.OverTime}, nil
		return st, nil
	}

	// A game counts once an Action was played in it, a resignation included,
	// which writes no Move.
	if len(parts.Games) == 0 {
		return nil, fmt.Errorf("duel %d: nothing played to keep: %w", row.ID, storage.ErrInvalid)
	}
	header := *parts.Match
	header.MatchHash, header.CanonicalHash = transcription.MatchHashes(parts)
	origin := storage.MatchOrigin{
		DiceSeed:     g.seed,
		StoppedEarly: g.doc.Header.MatchLength > 0 && !g.finished(),
		OverTime:     g.doc.Clock.OverTime,
	}
	if g.doc.Cadence != nil {
		origin.Cadence = g.doc.Cadence.String()
	}
	if g.doc.Start != nil {
		origin.Start = domain.EncodeXGID(g.doc.Start)
	}

	doc, err := encode(g.doc)
	if err != nil {
		return nil, err
	}
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*State, error) {
		_ = tx.Rollback()
		return nil, err
	}
	// The draft is rewritten under the caller's revision first, so a writer
	// that moved it on makes this whole transaction a conflict.
	row.Document = doc
	if _, err := tx.Duels().Save(ctx, scope, row); err != nil {
		return fail(err)
	}
	res, err := ingest.WriteMatch(ctx, tx, scope, transcription.MatchGraph(parts), nil)
	if err != nil {
		return fail(err)
	}
	header.ID = res.MatchID
	if err := tx.Matches().ReplaceHeader(ctx, scope, res.MatchID, &header); err != nil {
		return fail(err)
	}
	origin.MatchID = res.MatchID
	if err := tx.Duels().SetOrigin(ctx, scope, &origin); err != nil {
		return fail(err)
	}
	if err := tx.Duels().Delete(ctx, scope, row.ID); err != nil {
		return fail(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	delete(s.open, scope)
	st := state(row, g)
	st.Awaiting = nil
	st.Ended = &Ending{MatchID: res.MatchID, DiceSeed: g.seed, StoppedEarly: origin.StoppedEarly, OverTime: origin.OverTime}
	return st, nil
}

// load reads a draft and replays it.
func (s *Service) load(ctx context.Context, scope string, id int64) (*storage.Duel, *game, error) {
	row, err := s.store.Duels().Get(ctx, scope, id)
	if err != nil {
		return nil, nil, err
	}
	var doc document
	if err := json.Unmarshal([]byte(row.Document), &doc); err != nil {
		return nil, nil, fmt.Errorf("duel %d: %w", id, err)
	}
	if doc.FormatVersion < 1 || doc.FormatVersion > FormatVersion {
		return nil, nil, fmt.Errorf("duel %d: document format %d, this build reads %d: %w",
			id, doc.FormatVersion, FormatVersion, storage.ErrInvalid)
	}
	// A version 1 draft is a version 2 one without Cadence or clock.
	doc.FormatVersion = FormatVersion
	g, err := loadGame(doc, row.DiceSeed)
	if err != nil {
		return nil, nil, fmt.Errorf("duel %d: %w", id, err)
	}
	g.now = s.opts.Now
	return row, g, nil
}

func (s *Service) sides(specs [2]SideSpec) ([2]Side, error) {
	var out [2]Side
	for i, spec := range specs {
		side, err := s.opts.Sides(spec)
		if err != nil {
			return out, err
		}
		out[i] = side
	}
	return out, nil
}

func encode(doc document) (string, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("encode duel draft: %w", err)
	}
	return string(b), nil
}

func label(h transcript.Header) string {
	switch {
	case h.Player1 == "" && h.Player2 == "":
		return ""
	case h.MatchLength == 0:
		return h.Player1 + " — " + h.Player2 + " (money)"
	}
	return fmt.Sprintf("%s — %s (%d)", h.Player1, h.Player2, h.MatchLength)
}

func state(row *storage.Duel, g *game) *State {
	awaiting := g.awaiting()
	if awaiting != nil {
		awaiting.Since = g.doc.Clock.Since
	}
	var clk *ClockState
	if g.doc.Cadence != nil {
		c := g.doc.Clock
		clk = &ClockState{Cadence: *g.doc.Cadence, Reserve: c.Reserve, Turn: c.Turn, Spent: c.Spent, OverTime: c.OverTime}
	}
	return &State{
		ID:          row.ID,
		Revision:    row.Revision,
		Header:      g.doc.Header,
		Start:       g.doc.Start,
		Sides:       g.doc.Sides,
		Fingerprint: g.doc.Fingerprint,
		Actions:     append([]transcript.Action(nil), g.doc.Actions...),
		Games:       g.m.Games(),
		Score:       g.m.Score(),
		Awaiting:    awaiting,
		Clock:       clk,
	}
}
