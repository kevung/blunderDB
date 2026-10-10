package duel

import (
	"context"
	"crypto/rand"
	"encoding/json"
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

// Settings is what a Duel is created with (ADR-0072 rule 12).
type Settings struct {
	// MatchLength is 1 to 25 points, or 0 for a money session; Jacoby is a
	// money session's only rule to choose.
	MatchLength int  `json:"matchLength"`
	Jacoby      bool `json:"jacoby"`
	// Start is the Position the first game begins at; nil is the opening
	// position at the start of the match.
	Start *domain.Position `json:"start,omitempty"`
	// Away is the score, in Away scores (Crawford sentinels included), the
	// first game is played at: it replaces the Start's, and without a Start
	// the first game begins at the opening position at that score.
	Away *[2]int `json:"away,omitempty"`
	// Reroll drops the Start's roll: the Arbiter draws it.
	Reroll bool `json:"reroll,omitempty"`
	// AfterCube begins after the Start's cube decision: the side on roll
	// rolls at once, and a double the Start shows offered is taken. By
	// default the Duel begins before it, the decision being what is trained.
	AfterCube bool `json:"afterCube,omitempty"`
	// SingleGame ends the Duel with its first game, written alone as the
	// Match (or thrown away): training at one score.
	SingleGame bool `json:"singleGame,omitempty"`
	// Sides are player 1's and player 2's; a Side without a kind is external.
	Sides [2]SideSpec `json:"sides"`
	// DiscardAtEnd throws the draft away when the match is won, instead of
	// writing the Match — which is the default.
	DiscardAtEnd bool `json:"discardAtEnd,omitempty"`
	// Cadence is the Duel's clock; nil, the default, plays without one. The
	// durations of the decisions are measured either way.
	Cadence *Cadence `json:"cadence,omitempty"`
	// CombinedSeed has the Duel wait, its fingerprint published, for a
	// contribution from each external Side before the first roll; the dice
	// then come from CombinedSeed. It needs an external Side.
	CombinedSeed bool `json:"combinedSeed,omitempty"`
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
// row at every call and replayed, whether it is open included: nothing lives
// in memory, so losing the process loses nothing played, and several
// processes on one library see the same Duels open. Any number of a scope's
// Duels may be open at once (ADR-0072 rule 10).
//
// Each Duel has its own lock, so a Bot computing its move holds up its own
// Duel only. Between processes, the draft's revision arbitrates: a gesture
// that read a draft another one has since moved is refused with
// storage.ErrConflict, and nothing of it is written.
type Service struct {
	store storage.Storage
	opts  Options
	locks duelLocks
}

// duelLocks hands out one mutex per Duel, dropped once nobody holds or
// waits for it.
type duelLocks struct {
	mu sync.Mutex
	m  map[duelKey]*duelLock
}

type duelKey struct {
	scope string
	id    int64
}

type duelLock struct {
	sync.Mutex
	refs int
}

// lock takes the Duel's mutex and returns its release.
func (l *duelLocks) lock(scope string, id int64) func() {
	k := duelKey{scope, id}
	l.mu.Lock()
	if l.m == nil {
		l.m = map[duelKey]*duelLock{}
	}
	e := l.m[k]
	if e == nil {
		e = &duelLock{}
		l.m[k] = e
	}
	e.refs++
	l.mu.Unlock()
	e.Lock()
	return func() {
		e.Unlock()
		l.mu.Lock()
		if e.refs--; e.refs == 0 {
			delete(l.m, k)
		}
		l.mu.Unlock()
	}
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
	return &Service{store: store, opts: opts}
}

// Rebind points the Service at store, for a library rewritten under a new
// handle with the same rows, ids and revisions (a vacuum by file swap): a Duel
// in progress stays open, since that is in its row. The caller excludes every
// other call for its duration, since store is read without a lock.
func (s *Service) Rebind(store storage.Storage) {
	s.store = store
}

// State is a Duel as a caller sees it. It never carries the seed: the
// fingerprint stands for it until the Match reveals it.
type State struct {
	ID          int64                 `json:"id"`
	Revision    int64                 `json:"revision"`
	Header      transcript.Header     `json:"header"`
	Start       *domain.Position      `json:"start,omitempty"`
	SingleGame  bool                  `json:"singleGame,omitempty"`
	Sides       [2]SideSpec           `json:"sides"`
	Fingerprint string                `json:"fingerprint"`
	Actions     []transcript.Action   `json:"actions"`
	Games       []transcript.GameInfo `json:"games"`
	Score       [2]int                `json:"score"`
	// Awaiting is the Decision a Side owes; nil once the Duel has ended.
	Awaiting *Decision `json:"awaiting,omitempty"`
	// Clock is the Cadence's clock, nil without one.
	Clock *ClockState `json:"clock,omitempty"`
	// Contributions are the Sides' contributions to a combined seed, in
	// player order, and AwaitingContribution the Sides (0 or 1) still owed:
	// while one is, nothing is rolled and no Decision is awaited.
	CombinedSeed         bool      `json:"combinedSeed,omitempty"`
	Contributions        [2]string `json:"contributions,omitzero"`
	AwaitingContribution []int     `json:"awaitingContribution,omitempty"`
	// Ended is set by the call that ended the Duel.
	Ended *Ending `json:"ended,omitempty"`
	// Sheet is the match sheet the desktop draws with the Transcription's
	// view: the Actions with what each derived. Left out of the wire, where a
	// client derives its own from Actions and Games.
	Sheet transcript.Annotated `json:"-"`
}

// Ending is how a Duel ended: the Match it became, with its seed revealed, or
// nothing written at all.
type Ending struct {
	// MatchID is the Match written, 0 when the draft was thrown away.
	MatchID int64 `json:"matchId"`
	// DiceSeed is revealed with the Match; "" when nothing was written.
	DiceSeed  string `json:"diceSeed,omitempty"`
	Discarded bool   `json:"discarded,omitempty"`
	// Forfeited is the player (1 or 2) who gave the match up, 0 none.
	Forfeited int `json:"forfeited,omitempty"`
	// OverTime is the player (1 or 2) whose reserve ran out first, 0 none.
	OverTime int `json:"overTime,omitempty"`
}

// Summary is a Duel as a list shows it, open or in suspense.
type Summary struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Open      bool   `json:"open"`
}

// Create draws the seed, checks the session and the Start, plays what the
// Arbiter plays alone up to the first Decision, writes the draft and opens it;
// no other Duel is suspended. A Start the rules do not allow is refused with
// its *transcript.Refusal. No lock is needed: nobody else knows the Duel's id
// before it returns.
func (s *Service) Create(ctx context.Context, scope string, set Settings) (*State, error) {
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
		switch sp := &set.Sides[i]; {
		case sp.Declared != nil && sp.Kind != SideExternal:
			return nil, fmt.Errorf("player %d: only an external Side declares a Bot: %w", i+1, storage.ErrInvalid)
		case sp.Kind == SideBot:
			sp.Name = BotName(sp.Level)
		case sp.Declared != nil:
			if err := sp.Declared.check(); err != nil {
				return nil, fmt.Errorf("player %d: %w", i+1, err)
			}
			if sp.Name == "" {
				sp.Name = BotName(sp.Declared.Configuration)
			}
		}
	}
	if set.CombinedSeed && set.Sides[0].Kind != SideExternal && set.Sides[1].Kind != SideExternal {
		return nil, fmt.Errorf("a combined seed takes the contribution of an external Side, and both are delegated: %w", storage.ErrInvalid)
	}
	if set.MatchLength == 0 && !set.SingleGame && set.Sides[0].Kind != SideExternal && set.Sides[1].Kind != SideExternal {
		// Nobody would ever be awaited and a money session never ends: the
		// call would play forever.
		return nil, &transcript.Refusal{Kind: RefusedEndless,
			Detail: "a money session between two delegated Sides never ends"}
	}
	start, doubled, err := resolveStart(set)
	if err != nil {
		return nil, err
	}
	doc := document{
		FormatVersion: FormatVersion,
		Header: transcript.Header{
			MatchLength: set.MatchLength, Jacoby: set.Jacoby,
			Player1: set.Sides[0].Name, Player2: set.Sides[1].Name,
			Date: s.opts.Now(),
		},
		Start:        start,
		SingleGame:   set.SingleGame,
		Sides:        set.Sides,
		DiscardAtEnd: set.DiscardAtEnd,
		Fingerprint:  fp,
		Cadence:      set.Cadence,
		CombinedSeed: set.CombinedSeed,
	}
	g, err := newGame(doc, seed)
	if err != nil {
		return nil, err
	}
	g.now = s.opts.Now
	if err := g.beginStart(doubled, set.AfterCube); err != nil {
		return nil, err
	}
	g.setReserves()
	sides, err := s.sides(g.doc.Sides)
	if err != nil {
		return nil, err
	}
	if err := g.settle(ctx, sides); err != nil {
		return nil, err
	}
	row := &storage.Duel{FormatVersion: strconv.Itoa(FormatVersion), Label: label(g.doc.Header), DiceSeed: seed, Open: true}
	if row.Document, err = encode(g.doc); err != nil {
		return nil, err
	}
	row.ID, err = s.store.Duels().Save(ctx, scope, row)
	if err != nil {
		return nil, err
	}
	if g.finished() || g.timeLost() {
		// Two delegated Sides play the whole match in this call.
		return s.end(ctx, scope, row, g, !g.doc.DiscardAtEnd)
	}
	return state(row, g), nil
}

// List returns the scope's Duels, open or in suspense, most recently played
// first.
func (s *Service) List(ctx context.Context, scope string) ([]Summary, error) {
	var out []Summary
	for row, err := range s.store.Duels().List(ctx, scope) {
		if err != nil {
			return nil, err
		}
		out = append(out, Summary{ID: row.ID, Label: row.Label, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Open: row.Open})
	}
	return out, nil
}

// Open resumes a Duel where it stopped, with the same dice to come. The
// clocks stand still in suspense and start again now (ADR-0073). Other Duels
// stay as they are: a caller that plays one at a time suspends the one it
// leaves. A Duel already open is left as it is.
func (s *Service) Open(ctx context.Context, scope string, id int64) (*State, error) {
	return s.OpenAt(ctx, scope, id, 0)
}

// OpenAt is Open refusing with storage.ErrConflict a draft that is no longer
// at the revision the caller saw (0: none), the check and the opening being
// one step.
func (s *Service) OpenAt(ctx context.Context, scope string, id, revision int64) (*State, error) {
	unlock := s.locks.lock(scope, id)
	defer unlock()
	row, g, opened, err := s.loadOpen(ctx, scope, id, revision)
	if err != nil {
		return nil, err
	}
	if opened {
		if err := s.save(ctx, scope, row, g); err != nil {
			return nil, err
		}
	}
	return state(row, g), nil
}

// loadOpen reads a draft under the Duel's lock, refuses it with
// storage.ErrConflict when it is no longer at the revision the caller saw
// (0: none), and opens it in memory when it was in suspense: the gesture that
// follows writes the opening with its own, in one write. opened reports that
// the draft changed by being opened.
func (s *Service) loadOpen(ctx context.Context, scope string, id, revision int64) (row *storage.Duel, g *game, opened bool, err error) {
	row, g, err = s.load(ctx, scope, id)
	if err != nil {
		return nil, nil, false, err
	}
	if revision != 0 && revision != row.Revision {
		return nil, nil, false, fmt.Errorf("duel %d at revision %d, not %d: %w", id, row.Revision, revision, storage.ErrConflict)
	}
	if !row.Open {
		g.resume(s.opts.Now())
		row.Open, opened = true, true
	}
	return row, g, opened, nil
}

// Suspend puts the Duel in suspense, its clocks stopped, under the revision
// the caller saw (0: none); Open resumes it. A Duel already in suspense is
// left as it is.
func (s *Service) Suspend(ctx context.Context, scope string, id, revision int64) error {
	unlock := s.locks.lock(scope, id)
	defer unlock()
	row, g, err := s.load(ctx, scope, id)
	if err != nil {
		return err
	}
	if revision != 0 && revision != row.Revision {
		return fmt.Errorf("duel %d at revision %d, not %d: %w", id, row.Revision, revision, storage.ErrConflict)
	}
	if !row.Open {
		return nil
	}
	g.suspend(s.opts.Now())
	row.Open = false
	return s.save(ctx, scope, row, g)
}

// Flag has the Arbiter look at the clock of the Side the Duel awaits: run
// out, it is noted, and under TimeLoseMatch the Duel ends there (the
// tournament rules count the time as gone when it is noticed). Without a
// Cadence, or with time left, nothing changes but the opening of a Duel in
// suspense, as every gesture opens it.
func (s *Service) Flag(ctx context.Context, scope string, id int64) (*State, error) {
	unlock := s.locks.lock(scope, id)
	defer unlock()
	row, g, opened, err := s.loadOpen(ctx, scope, id, 0)
	if err != nil {
		return nil, err
	}
	if !g.flag(s.opts.Now()) && !opened {
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

// Play applies a Side's Play to the Duel under the revision the caller saw
// (0: none), opening it first if it was in suspense, lets the Arbiter play on to the next Decision of an external
// Side, and writes the draft. A Play the rules do not allow is refused with
// its *transcript.Refusal and nothing is written. When the match is won, the
// Duel ends: its Match is written, or the draft thrown away if it was created
// so (State.Ended).
func (s *Service) Play(ctx context.Context, scope string, id, revision int64, p Play) (*State, error) {
	unlock := s.locks.lock(scope, id)
	defer unlock()
	row, g, _, err := s.loadOpen(ctx, scope, id, revision)
	if err != nil {
		return nil, err
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

// Contribute records side's contribution to the Duel's combined seed
// under the revision the caller saw (0: none), and, the last one in, lets the
// Arbiter roll and play on to the next Decision of an external Side. A
// contribution the Duel does not await is refused with its
// *transcript.Refusal (RefusedContribution) and nothing is written.
func (s *Service) Contribute(ctx context.Context, scope string, id, revision int64, side int, contribution string) (*State, error) {
	unlock := s.locks.lock(scope, id)
	defer unlock()
	row, g, _, err := s.loadOpen(ctx, scope, id, revision)
	if err != nil {
		return nil, err
	}
	if err := g.contribute(side, contribution); err != nil {
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

// Stop ends the Duel before its end: thrown away, nothing of the Duel is
// written. Kept, a money session's Match is written as it stands — an
// unfinished game keeps no winner — since a session has no end of its own; a
// match in points is refused (ErrInvalid), because a Match is written whole or
// not at all (ADR-0072 rule 10): it is suspended, forfeited or thrown away.
// Stopping is never resigning: that is a Play.
func (s *Service) Stop(ctx context.Context, scope string, id, revision int64, keep bool) (*State, error) {
	unlock := s.locks.lock(scope, id)
	defer unlock()
	row, g, _, err := s.loadOpen(ctx, scope, id, revision)
	if err != nil {
		return nil, err
	}
	if keep && g.doc.Header.MatchLength > 0 {
		return nil, fmt.Errorf("duel %d: a match in points is kept only once won; suspend it, forfeit it or discard it: %w", id, storage.ErrInvalid)
	}
	return s.end(ctx, scope, row, g, keep)
}

// Forfeit has side give the Duel's match up: the game in progress ends
// won by the other side, for the points that bring them to the length — at
// money play, a single at the cube's value (transcript.KindForfeit). The
// Duel then ends as a match won does: its Match is written, or the draft
// thrown away if it was created so. Unlike Stop, the Match has a winner.
func (s *Service) Forfeit(ctx context.Context, scope string, id, revision int64, side int) (*State, error) {
	unlock := s.locks.lock(scope, id)
	defer unlock()
	row, g, _, err := s.loadOpen(ctx, scope, id, revision)
	if err != nil {
		return nil, err
	}
	if err := g.forfeit(side); err != nil {
		return nil, err
	}
	return s.end(ctx, scope, row, g, !g.doc.DiscardAtEnd)
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
		st := state(row, g)
		st.Ended, st.Awaiting = &Ending{Discarded: true, OverTime: g.doc.Clock.OverTime, Forfeited: g.forfeitedBy()}, nil
		return st, nil
	}

	// A game counts once an Action was played in it, a resignation included,
	// which writes no Move.
	if len(parts.Games) == 0 {
		return nil, fmt.Errorf("duel %d: nothing played to keep: %w", row.ID, storage.ErrInvalid)
	}
	forfeited := g.forfeitedBy()
	// Losing on time loses the match: the player out of time gives it up, so
	// the Match is written won with its final score, never short of its
	// length. A money session has no match to give and ends as it stands.
	if g.timeLost() && g.doc.Header.MatchLength > 0 && !g.finished() {
		if err := g.forfeit(g.doc.Clock.OverTime - 1); err != nil {
			return nil, err
		}
		parts = g.parts()
	}
	header := *parts.Match
	header.MatchHash, header.CanonicalHash = transcription.MatchHashes(parts)
	origin := storage.MatchOrigin{DiceSeed: g.seed, OverTime: g.doc.Clock.OverTime}
	origin.BotLevel, origin.BotEngine = botOrigin(g.doc.Sides)
	origin.DeclaredBots = declaredOrigin(g.doc.Sides)
	if g.doc.CombinedSeed {
		origin.Contributions = g.doc.Contributions[:]
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
	st := state(row, g)
	st.Awaiting = nil
	st.Ended = &Ending{MatchID: res.MatchID, DiceSeed: g.seed, OverTime: origin.OverTime, Forfeited: forfeited}
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
	// An older draft is a current one without what later versions added.
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
		SingleGame:  g.doc.SingleGame,
		Sides:       g.doc.Sides,
		Fingerprint: g.doc.Fingerprint,
		Actions:     append([]transcript.Action(nil), g.doc.Actions...),
		Games:       g.m.Games(),
		Score:       g.m.Score(),
		Awaiting:    awaiting,
		Clock:       clk,
		Sheet:       g.sheet(),

		CombinedSeed:         g.doc.CombinedSeed,
		Contributions:        g.doc.Contributions,
		AwaitingContribution: g.pendingContributions(),
	}
}

// sheet is the draft as a transcript.Annotated, the Cursor past its last
// Action: nothing is being typed in a Duel.
func (g *game) sheet() transcript.Annotated {
	over, winner := g.m.Finished()
	n := len(g.doc.Actions)
	return transcript.Annotated{
		Document: transcript.Document{FormatVersion: transcript.FormatVersion, Header: g.m.Header(),
			Actions: append([]transcript.Action(nil), g.doc.Actions...), Cursor: n},
		Actions:  append([]transcript.ActionInfo(nil), g.infos...),
		Games:    g.m.Games(),
		Next:     g.m.Next(),
		Finished: over,
		Winner:   winner,
		Score:    g.m.Score(),
		Cursor:   n,
	}
}
