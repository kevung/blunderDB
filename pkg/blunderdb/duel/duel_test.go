package duel

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

func newStore(t *testing.T) storage.Storage {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// fixedRand makes the seed a known one, so a test's dice are reproducible.
// It never runs dry: every Duel a Service creates draws the same seed.
type fixedRand byte

func (r fixedRand) Read(p []byte) (int, error) {
	copy(p, bytes.Repeat([]byte{byte(r)}, len(p)))
	return len(p), nil
}

// sideFirst is a delegated Side for the tests: it rolls rather than doubles,
// takes every double, and plays the first legal play.
const sideFirst SideKind = "test-first"

type firstPlay struct{}

func (firstPlay) Decide(_ context.Context, d Decision) (Play, bool, error) {
	return answer(d), true, nil
}

func answer(d Decision) Play {
	switch d.Kind {
	case DecideCube:
		return Play{Side: d.Side, Kind: PlayRoll}
	case DecideAnswer:
		return Play{Side: d.Side, Kind: PlayTake}
	}
	plays := domain.LegalMoves(&d.Position)
	return Play{Side: d.Side, Kind: PlayMove, Steps: plays[0].Steps}
}

func testSides(spec SideSpec) (Side, error) {
	if spec.Kind == sideFirst {
		return firstPlay{}, nil
	}
	return ExternalOnly(spec)
}

func newService(t *testing.T, st storage.Storage, seedByte byte) *Service {
	t.Helper()
	return New(st, Options{Sides: testSides, Rand: fixedRand(seedByte)})
}

func refusalKind(t *testing.T, err error) transcript.RefusalKind {
	t.Helper()
	var r *transcript.Refusal
	if !errors.As(err, &r) {
		t.Fatalf("error %v is not a *transcript.Refusal", err)
	}
	return r.Kind
}

func external(name string) SideSpec { return SideSpec{Kind: SideExternal, Name: name} }

// TestRollIsAFunctionOfSeedAndRank: the same seed and rank give the same roll,
// every die is a die, and the faces come out evenly.
func TestRollIsAFunctionOfSeedAndRank(t *testing.T) {
	seed, err := newSeed(fixedRand(7))
	if err != nil {
		t.Fatal(err)
	}
	var faces [7]int
	for n := range 6000 {
		a, err := Roll(seed, n)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := Roll(seed, n)
		if a != b {
			t.Fatalf("rank %d: %v then %v", n, a, b)
		}
		for _, v := range a {
			if v < 1 || v > 6 {
				t.Fatalf("rank %d: %v is no roll", n, a)
			}
			faces[v]++
		}
	}
	for f := 1; f <= 6; f++ {
		if faces[f] < 1800 || faces[f] > 2200 {
			t.Errorf("face %d came out %d times in 12000 dice", f, faces[f])
		}
	}
	// The stated derivation, pinned: a change to it changes every Duel's dice.
	if got, _ := Roll(seed, 0); got != [2]int{2, 6} {
		t.Errorf("Roll(seed 0x07…, 0) = %v; the derivation changed", got)
	}
	fp, err := Fingerprint(seed)
	if err != nil || len(fp) != 64 || fp == seed {
		t.Errorf("Fingerprint = %q, %v", fp, err)
	}
}

// TestDuelPlayedToTheEnd: two delegated Sides play a 3-point match in the
// call that creates it; the Match is written with its origin, the seed is
// revealed and matches the fingerprint, and the draft is gone.
func TestDuelPlayedToTheEnd(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 3)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 3,
		Sides: [2]SideSpec{{Kind: sideFirst, Name: "Alice"}, {Kind: sideFirst, Name: "Bob"}}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.Awaiting != nil || s.Ended == nil || s.Ended.MatchID == 0 {
		t.Fatalf("two delegated Sides end the match in the call that creates it: %+v", s)
	}
	if s.Score[0] < 3 && s.Score[1] < 3 {
		t.Fatalf("score %v: the match is not won", s.Score)
	}
	if list, err := svc.List(ctx, ""); err != nil || len(list) != 0 {
		t.Errorf("no Duel stays in suspense: %+v, %v", list, err)
	}
}

// TestDuelEndsIntoAMatch: the last Play of an external Side wins the match;
// the Match carries the games, the winner and its origin.
func TestDuelEndsIntoAMatch(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 3)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 1,
		Sides: [2]SideSpec{external("Alice"), {Kind: sideFirst, Name: "Bot"}}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fingerprint := s.Fingerprint
	for i := 0; s.Ended == nil; i++ {
		if i > 500 {
			t.Fatal("the match never ended")
		}
		if s.Awaiting == nil || s.Awaiting.Side != domain.Black {
			t.Fatalf("an external Side's decision should be awaited: %+v", s.Awaiting)
		}
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	if s.Ended.MatchID == 0 || s.Ended.Discarded {
		t.Fatalf("Ended = %+v, want a Match", s.Ended)
	}
	if fp, _ := Fingerprint(s.Ended.DiceSeed); fp != fingerprint {
		t.Errorf("the revealed seed does not match the fingerprint published at creation")
	}
	if _, err := st.Duels().Get(ctx, "", s.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("the draft must be gone once the Match is written: %v", err)
	}
	o, err := st.Duels().Origin(ctx, "", s.Ended.MatchID)
	if err != nil || o.DiceSeed != s.Ended.DiceSeed || o.StoppedEarly || o.Start != "" {
		t.Errorf("origin = %+v, %v", o, err)
	}
	m, err := st.Matches().Get(ctx, "", s.Ended.MatchID)
	if err != nil || m.Player1Name != "Alice" || m.Player2Name != "Bot" || m.MatchLength != 1 || m.MatchHash == "" {
		t.Fatalf("match = %+v, %v", m, err)
	}
	games, err := gamesOf(st, s.Ended.MatchID)
	if err != nil || len(games) != 1 || games[0].Winner == domain.WinnerUnfinished || games[0].PointsWon == 0 {
		t.Errorf("games = %+v, %v", games, err)
	}
}

// TestDuelResumesWithTheSameDice: a fresh Service over the same store opens
// the draft at the same point, the same roll on the board.
func TestDuelResumesWithTheSameDice(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 9)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 7, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for range 6 {
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	before := *s.Awaiting

	again := newService(t, st, 0)
	r, err := again.Open(ctx, "", s.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if r.Awaiting == nil || r.Awaiting.Side != before.Side || r.Awaiting.Kind != before.Kind ||
		r.Awaiting.Position.Dice != before.Position.Dice || r.Awaiting.Position.Board != before.Position.Board {
		t.Fatalf("resumed at %+v, want %+v", r.Awaiting, before)
	}
	// Resuming restarts the clocks, which is a write.
	if r.Revision != s.Revision+1 || len(r.Actions) != len(s.Actions) {
		t.Errorf("resumed revision %d with %d actions, want %d with %d", r.Revision, len(r.Actions), s.Revision+1, len(s.Actions))
	}
	// Both services play on identically: the dice to come are the seed's.
	a, err := svc.Play(ctx, "", s.ID, r.Revision, answer(before))
	if err != nil {
		t.Fatalf("Play: %v", err)
	}
	if _, err := again.Play(ctx, "", s.ID, r.Revision, answer(before)); !errors.Is(err, storage.ErrConflict) {
		t.Errorf("a Play under a revision moved on: got %v, want ErrConflict", err)
	}
	b, err := again.Open(ctx, "", s.ID)
	if err != nil || b.Awaiting == nil || b.Awaiting.Position.Dice != a.Awaiting.Position.Dice {
		t.Errorf("reopened after the Play: %+v, %v; want %+v", b.Awaiting, err, a.Awaiting)
	}
}

// TestDuelRefusesPlays: a Play from the side not awaited, a Play of the wrong
// kind, an illegal move, and a Play on a Duel that is not the open one.
func TestDuelRefusesPlays(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 5)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	d := *s.Awaiting
	if d.Kind != DecideMove {
		t.Fatalf("the opening roll is the Arbiter's: a move should be awaited, got %+v", d)
	}
	if k := refusalKind(t, play(svc, s, Play{Side: 1 - d.Side, Kind: PlayMove})); k != RefusedNotAwaited {
		t.Errorf("Play from the other side: %q", k)
	}
	if k := refusalKind(t, play(svc, s, Play{Side: d.Side, Kind: PlayDouble})); k != RefusedNotAwaited {
		t.Errorf("a double when a move is awaited: %q", k)
	}
	bad := Play{Side: d.Side, Kind: PlayMove, Steps: []domain.CheckerStep{{From: 1, To: 2}}}
	if k := refusalKind(t, play(svc, s, bad)); k != transcript.RefusedIllegalMove && k != transcript.RefusedInconsistentDice {
		t.Errorf("an illegal move: %q", k)
	}
	if got, _ := st.Duels().Get(ctx, "", s.ID); got.Revision != s.Revision {
		t.Errorf("a refused Play wrote the draft: revision %d, want %d", got.Revision, s.Revision)
	}

	other, err := svc.Create(ctx, "", Settings{MatchLength: 3, Sides: [2]SideSpec{external("C"), external("D")}})
	if err != nil {
		t.Fatalf("Create other: %v", err)
	}
	if err := play(svc, s, answer(d)); !errors.Is(err, ErrNotOpen) {
		t.Errorf("Play on a Duel in suspense: got %v, want ErrNotOpen", err)
	}
	list, err := svc.List(ctx, "")
	if err != nil || len(list) != 2 || !list[0].Open || list[0].ID != other.ID || list[1].Open {
		t.Errorf("List = %+v, %v; one open, the newest", list, err)
	}
	reopened, err := svc.Open(ctx, "", s.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := play(svc, reopened, answer(d)); err != nil {
		t.Errorf("Play once reopened: %v", err)
	}
}

func gamesOf(st storage.Storage, matchID int64) ([]*domain.Game, error) {
	var out []*domain.Game
	for g, err := range st.Matches().Games(context.Background(), "", matchID) {
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func play(svc *Service, s *State, p Play) error {
	_, err := svc.Play(context.Background(), "", s.ID, s.Revision, p)
	return err
}

// TestDuelStart: a Start is refused by name when it is a double waiting for
// its answer or a cube decision the rules do not offer; a Start's own roll is
// played without drawing; a turn whose cube is not available begins with the
// Arbiter's roll.
func TestDuelStart(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 1)
	board := transcript.InitialBoard()
	at := func(away [2]int, cube domain.Cube, decision int, dice [2]int) *domain.Position {
		return &domain.Position{Board: board, Cube: cube, Score: away, PlayerOnRoll: domain.Black, DecisionType: decision, Dice: dice}
	}
	create := func(p *domain.Position) (*State, error) {
		return svc.Create(ctx, "", Settings{MatchLength: 7, Start: p, Sides: [2]SideSpec{external("A"), external("B")}})
	}

	refused := []struct {
		name  string
		start *domain.Position
		want  transcript.RefusalKind
	}{
		{"double in the middle", at([2]int{5, 5}, domain.Cube{Owner: domain.None, Value: 1}, domain.CubeAction, [2]int{}), RefusedStartPendingDouble},
		{"opponent's cube", at([2]int{5, 5}, domain.Cube{Owner: domain.White, Value: 1}, domain.CubeAction, [2]int{}), RefusedStartPendingDouble},
		{"cube decision with a roll", at([2]int{5, 5}, domain.Cube{Owner: domain.Black, Value: 1}, domain.CubeAction, [2]int{3, 1}), RefusedStartDecision},
		{"cube decision in the Crawford game", at([2]int{1, 5}, domain.Cube{Owner: domain.None}, domain.CubeAction, [2]int{}), RefusedStartDecision},
	}
	for _, c := range refused {
		if _, err := create(c.start); err == nil {
			t.Errorf("%s: accepted", c.name)
		} else if k := refusalKind(t, err); k != c.want {
			t.Errorf("%s: refused %q, want %q", c.name, k, c.want)
		}
	}

	s, err := create(at([2]int{5, 5}, domain.Cube{Owner: domain.Black, Value: 1}, domain.CubeAction, [2]int{}))
	if err != nil || s.Awaiting == nil || s.Awaiting.Kind != DecideCube {
		t.Fatalf("a cube decision Start: %+v, %v", s, err)
	}

	s, err = create(at([2]int{5, 5}, domain.Cube{Owner: domain.Black, Value: 1}, domain.CheckerAction, [2]int{6, 5}))
	if err != nil || s.Awaiting == nil || s.Awaiting.Kind != DecideMove || s.Awaiting.Position.Dice != [2]int{6, 5} {
		t.Fatalf("a Start with its roll: %+v, %v", s.Awaiting, err)
	}

	// The Crawford game has no cube: the Arbiter rolls for the side on roll.
	s, err = create(at([2]int{1, 5}, domain.Cube{Owner: domain.Black, Value: 0}, domain.CheckerAction, [2]int{}))
	if err == nil {
		t.Fatalf("an owned cube at 1 is no cube: %+v", s)
	}
	crawford := at([2]int{1, 5}, domain.Cube{Owner: domain.None}, domain.CheckerAction, [2]int{})
	crawford.Board.Points[24].Checkers, crawford.Board.Points[23].Checkers = 1, 1
	crawford.Board.Points[23].Color = domain.Black
	s, err = create(crawford)
	if err != nil || s.Awaiting == nil || s.Awaiting.Kind != DecideMove || s.Awaiting.Side != domain.Black {
		t.Fatalf("a Crawford Start: %+v, %v; the Arbiter rolls when the cube is not available", s.Awaiting, err)
	}
}

// TestDuelStop: stopping and keeping writes the games as they stand — no
// winner for the game in progress, the match marked stopped early; stopping
// and throwing away writes nothing.
func TestDuelStop(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 4)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Stop(ctx, "", s.ID, s.Revision, true); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("keeping a Duel nothing was played in: got %v, want ErrInvalid", err)
	}
	for range 4 {
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	kept, err := svc.Stop(ctx, "", s.ID, s.Revision, true)
	if err != nil || kept.Ended == nil || kept.Ended.MatchID == 0 || !kept.Ended.StoppedEarly {
		t.Fatalf("Stop and keep: %+v, %v", kept, err)
	}
	games, err := gamesOf(st, kept.Ended.MatchID)
	if err != nil || len(games) != 1 || games[0].Winner != domain.WinnerUnfinished || games[0].PointsWon != 0 {
		t.Errorf("games of a stopped Duel = %+v, %v; the game in progress has no winner", games, err)
	}
	if o, err := st.Duels().Origin(ctx, "", kept.Ended.MatchID); err != nil || !o.StoppedEarly {
		t.Errorf("origin = %+v, %v", o, err)
	}

	s, err = svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	thrown, err := svc.Stop(ctx, "", s.ID, s.Revision, false)
	if err != nil || thrown.Ended == nil || !thrown.Ended.Discarded || thrown.Ended.DiceSeed != "" {
		t.Fatalf("Stop and throw away: %+v, %v", thrown, err)
	}
	if _, err := st.Duels().Get(ctx, "", s.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("a thrown-away draft must be gone: %v", err)
	}
}

// TestDuelMatchStaysClosed: a Match played from a Start is not written as a
// .mat, and no Match played here reopens as a Transcription.
func TestDuelMatchStaysClosed(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 2)
	start := &domain.Position{Board: transcript.InitialBoard(), Cube: domain.Cube{Owner: domain.Black, Value: 1},
		Score: [2]int{3, 3}, PlayerOnRoll: domain.Black}
	s, err := svc.Create(ctx, "", Settings{MatchLength: 3, Start: start,
		Sides: [2]SideSpec{{Kind: sideFirst, Name: "A"}, {Kind: sideFirst, Name: "B"}}})
	if err != nil || s.Ended == nil {
		t.Fatalf("Create: %+v, %v", s, err)
	}
	o, err := st.Duels().Origin(ctx, "", s.Ended.MatchID)
	if err != nil || o.Start == "" {
		t.Fatalf("origin = %+v, %v; the Start is part of it", o, err)
	}
	if _, _, _, err := ingest.ReadMatchForMAT(ctx, st, "", s.Ended.MatchID); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf(".mat of a Match with a Start: got %v, want ErrInvalid", err)
	}
	tr := transcription.New(st, transcription.Options{})
	if _, _, err := tr.EditMatch(ctx, "", s.Ended.MatchID); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("reopening a Duel's Match as a Transcription: got %v, want ErrInvalid", err)
	}
}

// TestDuelResignationBeforeAnyMove: at double match point, player 1 resigns
// before any checker is moved; the match is won, and its Match is written
// with the one game and its winner, though the game holds no Move.
func TestDuelResignationBeforeAnyMove(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 6)
	start := &domain.Position{Board: transcript.InitialBoard(), Cube: domain.Cube{Owner: domain.None},
		Score: [2]int{0, 0}, PlayerOnRoll: domain.Black}
	start.Board.Points[24].Checkers, start.Board.Points[23].Checkers = 1, 1
	start.Board.Points[23].Color = domain.Black
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Start: start, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil || s.Awaiting == nil {
		t.Fatalf("Create at DMP: %+v, %v", s, err)
	}
	if s, err = svc.Play(ctx, "", s.ID, s.Revision, Play{Side: s.Awaiting.Side, Kind: PlayResign, Level: 1}); err != nil {
		t.Fatalf("resign: %v", err)
	}
	if s.Ended == nil || s.Ended.MatchID == 0 || s.Ended.StoppedEarly {
		t.Fatalf("a resigned match point ends into a Match: %+v", s.Ended)
	}
	games, err := gamesOf(st, s.Ended.MatchID)
	if err != nil || len(games) != 1 || games[0].Winner != domain.WinnerPlayer2 || games[0].PointsWon != 1 {
		t.Errorf("games = %+v, %v; player 2 wins the resigned game", games, err)
	}
}
