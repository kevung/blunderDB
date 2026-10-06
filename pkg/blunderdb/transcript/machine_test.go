package transcript

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// refusalKind returns the kind of a refusal, or "" when err is none.
func refusalKind(t *testing.T, err error) RefusalKind {
	t.Helper()
	if err == nil {
		return ""
	}
	var r *Refusal
	if !errors.As(err, &r) {
		t.Fatalf("error %v is not a *Refusal", err)
	}
	if r.Detail == "" {
		t.Fatalf("refusal %q carries no reason", r.Kind)
	}
	return r.Kind
}

// legalAction builds the first legal checker play of side under dice from the
// position the Machine awaits.
func legalAction(t *testing.T, m Machine, side int, dice [2]int) Action {
	t.Helper()
	pos := m.Next().Position
	pos.PlayerOnRoll = side
	pos.Dice = [2]int{max(dice[0], dice[1]), min(dice[0], dice[1])}
	pos.DecisionType = domain.CheckerAction
	plays := domain.LegalMoves(&pos)
	if len(plays) == 0 {
		return Action{Side: side, Kind: KindDance, Dice: dice}
	}
	return Action{Side: side, Kind: KindChecker, Dice: dice, Steps: plays[0].Steps}
}

func mustApply(t *testing.T, m Machine, a Action) Machine {
	t.Helper()
	next, _, err := m.Apply(a)
	if err != nil {
		t.Fatalf("Apply(%+v): %v", a, err)
	}
	return next
}

func mustMachine(t *testing.T, h Header, start *domain.Position) Machine {
	t.Helper()
	m, err := NewMachine(h, start)
	if err != nil {
		t.Fatalf("NewMachine: %v", err)
	}
	return m
}

func emptyBoard() domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: domain.None}
	}
	return b
}

// bearOffBoard: player 1 has one checker left on its 1-point, player 2 has borne
// nothing off and stands outside player 1's home board — a gammon, not a backgammon.
func bearOffBoard() domain.Board {
	b := emptyBoard()
	b.Points[1] = domain.Point{Checkers: 1, Color: domain.Black}
	b.Bearoff[domain.Black] = 14
	b.Points[12] = domain.Point{Checkers: 15, Color: domain.White}
	return b
}

func moneyStart(board domain.Board, cube domain.Cube, dice [2]int) *domain.Position {
	return &domain.Position{
		Board:        board,
		Cube:         cube,
		Dice:         dice,
		Score:        [2]int{domain.Unlimited, domain.Unlimited},
		PlayerOnRoll: domain.Black,
	}
}

func matchStart(away [2]int, cube domain.Cube) *domain.Position {
	return &domain.Position{Board: InitialBoard(), Cube: cube, Score: away, PlayerOnRoll: domain.Black}
}

func TestMachineSession(t *testing.T) {
	cases := []struct {
		name string
		h    Header
		want RefusalKind
	}{
		{"money", Header{}, ""},
		{"money capped at 64", Header{MaxCube: 6}, ""},
		{"money with Jacoby", Header{Jacoby: true}, ""},
		{"25 points", Header{MatchLength: 25}, ""},
		{"26 points", Header{MatchLength: 26}, RefusedSession},
		{"negative length", Header{MatchLength: -1}, RefusedSession},
		{"beaver at money", Header{Beaver: true}, RefusedSession},
		{"Jacoby in a match", Header{MatchLength: 5, Jacoby: true}, RefusedSession},
		{"ceiling in a match", Header{MatchLength: 5, MaxCube: 3}, RefusedSession},
		{"money capped at 8", Header{MaxCube: 3}, RefusedSession},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewMachine(c.h, nil)
			if got := refusalKind(t, err); got != c.want {
				t.Fatalf("refusal %q, want %q", got, c.want)
			}
		})
	}
	if got := mustMachine(t, Header{}, nil).Header().MaxCube; got != moneyMaxCube {
		t.Fatalf("money ceiling %d, want %d", got, moneyMaxCube)
	}
}

func TestMachineStartRefused(t *testing.T) {
	money := Header{}
	match := Header{MatchLength: 5}
	centred := centredCube()
	cases := []struct {
		name  string
		h     Header
		start func() *domain.Position
		want  RefusalKind
	}{
		{"default opening at a score", match, func() *domain.Position { return matchStart([2]int{3, 4}, centred) }, ""},
		{"side on roll", money, func() *domain.Position {
			p := moneyStart(InitialBoard(), centred, [2]int{})
			p.PlayerOnRoll = 2
			return p
		}, RefusedStartTurn},
		{"fourteen checkers", money, func() *domain.Position {
			b := InitialBoard()
			b.Points[6].Checkers = 4
			return moneyStart(b, centred, [2]int{})
		}, RefusedStartBoard},
		{"checkers of nobody", money, func() *domain.Position {
			b := InitialBoard()
			b.Points[6].Color = domain.None
			return moneyStart(b, centred, [2]int{})
		}, RefusedStartBoard},
		{"wrong bar", money, func() *domain.Position {
			b := InitialBoard()
			b.Points[6].Checkers = 4
			b.Points[domain.WhiteBar] = domain.Point{Checkers: 1, Color: domain.Black}
			return moneyStart(b, centred, [2]int{})
		}, RefusedStartBoard},
		{"game already over", money, func() *domain.Position {
			b := emptyBoard()
			b.Bearoff[domain.Black] = 15
			b.Points[12] = domain.Point{Checkers: 15, Color: domain.White}
			return moneyStart(b, centred, [2]int{})
		}, RefusedStartBoard},
		{"owned cube at 1", money, func() *domain.Position {
			return moneyStart(InitialBoard(), domain.Cube{Owner: domain.Black}, [2]int{})
		}, RefusedStartCube},
		{"cube above 64 at money", money, func() *domain.Position {
			return moneyStart(InitialBoard(), domain.Cube{Owner: domain.Black, Value: 7}, [2]int{})
		}, RefusedStartCube},
		{"turned cube in the Crawford game", match, func() *domain.Position {
			return matchStart([2]int{1, 3}, domain.Cube{Owner: domain.White, Value: 1})
		}, RefusedStartCube},
		{"score at money", money, func() *domain.Position {
			p := moneyStart(InitialBoard(), centred, [2]int{})
			p.Score = [2]int{3, 3}
			return p
		}, RefusedStartScore},
		{"away beyond the length", match, func() *domain.Position { return matchStart([2]int{6, 5}, centred) }, RefusedStartScore},
		{"negative away", match, func() *domain.Position { return matchStart([2]int{-1, 5}, centred) }, RefusedStartScore},
		{"two Crawford games", match, func() *domain.Position { return matchStart([2]int{1, 1}, centred) }, RefusedStartScore},
		{"Crawford beside post-Crawford", match, func() *domain.Position { return matchStart([2]int{1, 0}, centred) }, RefusedStartScore},
		{"no die", money, func() *domain.Position { return moneyStart(bearOffBoard(), centred, [2]int{7, 1}) }, RefusedStartDice},
		{"half a roll", money, func() *domain.Position { return moneyStart(bearOffBoard(), centred, [2]int{3, 0}) }, RefusedStartDice},
		{"opening double", money, func() *domain.Position { return moneyStart(InitialBoard(), centred, [2]int{4, 4}) }, RefusedStartDice},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewMachine(c.h, c.start())
			if got := refusalKind(t, err); got != c.want {
				t.Fatalf("refusal %q (%v), want %q", got, err, c.want)
			}
		})
	}
}

func TestMachineJacoby(t *testing.T) {
	turned := domain.Cube{Owner: domain.Black, Value: 1}
	cases := []struct {
		name   string
		jacoby bool
		cube   domain.Cube
		want   int
	}{
		{"gammon without Jacoby", false, centredCube(), 2},
		{"gammon under Jacoby, cube centred", true, centredCube(), 1},
		{"gammon under Jacoby, cube turned", true, turned, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := mustMachine(t, Header{Jacoby: c.jacoby}, moneyStart(bearOffBoard(), c.cube, [2]int{6, 5}))
			m = mustApply(t, m, legalAction(t, m, domain.Black, [2]int{6, 5}))
			if got := m.Score()[domain.Black]; got != c.want {
				t.Fatalf("player 1 scores %d, want %d", got, c.want)
			}
			if finished, _ := m.Finished(); finished {
				t.Fatal("a money session never finishes")
			}
		})
	}

	t.Run("resigned gammon under Jacoby", func(t *testing.T) {
		m := mustMachine(t, Header{Jacoby: true}, moneyStart(bearOffBoard(), centredCube(), [2]int{}))
		m = mustApply(t, m, Action{Side: domain.White, Kind: KindResign, Level: 2})
		if got := m.Score()[domain.Black]; got != 1 {
			t.Fatalf("player 1 scores %d, want 1", got)
		}
	})
}

func TestMachineMoneyCeiling(t *testing.T) {
	start := func(value int) *domain.Position {
		p := moneyStart(InitialBoard(), domain.Cube{Owner: domain.Black, Value: value}, [2]int{})
		p.Board.Points[6].Checkers = 4
		p.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.Black}
		return p
	}
	double := Action{Side: domain.Black, Kind: KindDouble}

	m := mustMachine(t, Header{}, start(5))
	m = mustApply(t, m, double)
	mustApply(t, m, Action{Side: domain.White, Kind: KindTake})

	m = mustMachine(t, Header{}, start(6))
	if _, _, err := m.Apply(double); refusalKind(t, err) != RefusedImpossibleCube {
		t.Fatalf("a redouble past 64 is %v, want refused", err)
	}
}

func TestMachineNoBeaver(t *testing.T) {
	m := mustMachine(t, Header{}, nil)
	m = mustApply(t, m, legalAction(t, m, domain.Black, [2]int{3, 1}))
	m = mustApply(t, m, legalAction(t, m, domain.White, [2]int{5, 2}))
	m = mustApply(t, m, Action{Side: domain.Black, Kind: KindDouble})
	_, _, err := m.Apply(Action{Side: domain.White, Kind: KindDouble})
	if refusalKind(t, err) != RefusedOutOfTurn {
		t.Fatalf("a beaver is %v, want refused", err)
	}
	if n := m.Next(); n.Expects != KindTake || n.Side != domain.White {
		t.Fatalf("after the refusal the Machine awaits %+v, want player 2's answer", n)
	}
}

func TestMachineCrawford(t *testing.T) {
	h := Header{MatchLength: 3}

	t.Run("derived after a game", func(t *testing.T) {
		m := mustMachine(t, h, matchStart([2]int{2, 3}, centredCube()))
		if m.Next().Crawford {
			t.Fatal("at 1-0 to 3 the game is not the Crawford game")
		}
		m = mustApply(t, m, legalAction(t, m, domain.Black, [2]int{3, 1}))
		m = mustApply(t, m, Action{Side: domain.White, Kind: KindResign, Level: 1})
		if m.Score() != [2]int{2, 0} || !m.Next().Crawford || !m.Next().GameStart {
			t.Fatalf("after the game: score %v, next %+v", m.Score(), m.Next())
		}
		m = mustApply(t, m, legalAction(t, m, domain.White, [2]int{2, 6}))
		if _, _, err := m.Apply(Action{Side: domain.Black, Kind: KindDouble}); refusalKind(t, err) != RefusedImpossibleCube {
			t.Fatalf("a double in the Crawford game is %v, want refused", err)
		}
	})

	t.Run("from a Crawford Start", func(t *testing.T) {
		start := matchStart([2]int{1, 3}, centredCube())
		start.Board.Points[6].Checkers = 4
		start.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.Black}
		m := mustMachine(t, h, start)
		if !m.Next().Crawford {
			t.Fatal("a Start at 1-away is the Crawford game")
		}
		if _, _, err := m.Apply(Action{Side: domain.Black, Kind: KindDouble}); refusalKind(t, err) != RefusedImpossibleCube {
			t.Fatalf("a double in the Crawford game is %v, want refused", err)
		}
	})

	t.Run("post-Crawford", func(t *testing.T) {
		start := matchStart([2]int{0, 3}, centredCube())
		start.Board.Points[6].Checkers = 4
		start.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.Black}
		start.PlayerOnRoll = domain.White
		m := mustMachine(t, h, start)
		if m.Next().Crawford {
			t.Fatal("a Start at 0-away is after the Crawford game")
		}
		mustApply(t, m, Action{Side: domain.White, Kind: KindDouble})
	})

	t.Run("a 1-point match", func(t *testing.T) {
		m := mustMachine(t, Header{MatchLength: 1}, nil)
		m = mustApply(t, m, legalAction(t, m, domain.Black, [2]int{6, 1}))
		m = mustApply(t, m, Action{Side: domain.White, Kind: KindResign, Level: 1})
		finished, winner := m.Finished()
		if !finished || winner != domain.Black || m.Score() != [2]int{1, 0} {
			t.Fatalf("finished %v winner %d score %v", finished, winner, m.Score())
		}
		if _, _, err := m.Apply(legalAction(t, m, domain.Black, [2]int{6, 1})); refusalKind(t, err) != RefusedPastEnd {
			t.Fatalf("a play after the match is %v, want refused", err)
		}
	})
}

func TestMachineRefusesActions(t *testing.T) {
	opening := mustMachine(t, Header{MatchLength: 5}, nil)
	played := mustApply(t, opening, legalAction(t, opening, domain.Black, [2]int{3, 1}))
	rolled := mustMachine(t, Header{}, moneyStart(bearOffBoard(), centredCube(), [2]int{6, 5}))

	illegal := legalAction(t, played, domain.White, [2]int{4, 2})
	illegal.Steps = []domain.CheckerStep{{From: 12, To: 16}}

	cases := []struct {
		name string
		m    Machine
		a    Action
		want RefusalKind
	}{
		{"opening roll won by the other side", opening, Action{Side: domain.White, Kind: KindChecker, Dice: [2]int{3, 1}}, RefusedInconsistentDice},
		{"opening double", opening, Action{Side: domain.Black, Kind: KindChecker, Dice: [2]int{3, 3}}, RefusedInconsistentDice},
		{"double before the first play", opening, Action{Side: domain.Black, Kind: KindDouble}, RefusedImpossibleCube},
		{"resign before any game", opening, Action{Side: domain.Black, Kind: KindResign, Level: 1}, RefusedOutOfTurn},
		{"same side twice", played, legalAction(t, played, domain.Black, [2]int{4, 2}), RefusedOutOfTurn},
		{"double out of turn", played, Action{Side: domain.Black, Kind: KindDouble}, RefusedOutOfTurn},
		{"take with no double", played, Action{Side: domain.White, Kind: KindTake}, RefusedImpossibleCube},
		{"illegal play", played, illegal, RefusedIllegalMove},
		{"no die", played, Action{Side: domain.White, Kind: KindChecker, Dice: [2]int{0, 2}}, RefusedRoll},
		{"unrecorded play", played, Action{Side: domain.White, Kind: KindUnrecorded, Dice: [2]int{4, 2}}, RefusedRecord},
		{"declared score", played, Action{Side: domain.White, Kind: KindChecker, Dice: [2]int{4, 2}, Score: &[2]int{1, 0}}, RefusedRecord},
		{"unknown side", played, Action{Side: 2, Kind: KindResign}, RefusedOutOfTurn},
		{"not the Start's roll", rolled, legalAction(t, rolled, domain.Black, [2]int{2, 1}), RefusedRoll},
		{"double over a rolled Start", rolled, Action{Side: domain.Black, Kind: KindDouble}, RefusedImpossibleCube},
		{"the Start's roll either way round", rolled, legalAction(t, rolled, domain.Black, [2]int{5, 6}), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := c.m.Next()
			after, _, err := c.m.Apply(c.a)
			if got := refusalKind(t, err); got != c.want {
				t.Fatalf("refusal %q (%v), want %q", got, err, c.want)
			}
			if err != nil && (after.applied != c.m.applied || after.Next() != before) {
				t.Fatal("a refusal moved the Machine")
			}
		})
	}
	if got := rolled.Next().Position.Dice; got != [2]int{6, 5} {
		t.Fatalf("a rolled Start awaits dice %v, want 65", got)
	}
}

// The games after the first begin at the opening position, whatever the Start was.
func TestMachineNextGameAtOpening(t *testing.T) {
	m := mustMachine(t, Header{}, moneyStart(bearOffBoard(), centredCube(), [2]int{6, 5}))
	m = mustApply(t, m, legalAction(t, m, domain.Black, [2]int{6, 5}))
	n := m.Next()
	if !n.GameStart || n.GameNumber != 2 || n.Position.Board != InitialBoard() || n.Position.Cube != centredCube() {
		t.Fatalf("next game %+v, want game 2 at the opening position", n)
	}
	if len(m.Games()) != 1 || !m.Games()[0].Finished || m.Games()[0].PointsWon != 2 {
		t.Fatalf("games %+v", m.Games())
	}
}

// TestMachineCubeAvailable pins when a turn begins with a decision: the cube is
// offered to the side on roll unless no roll is awaited, the game is the Crawford
// game, the opponent owns it, the ceiling is reached or the cube is dead at the score.
func TestMachineCubeAvailable(t *testing.T) {
	at := func(away [2]int, cube domain.Cube) *domain.Position {
		p := matchStart(away, cube)
		p.Board = bearOffBoard()
		return p
	}
	cases := []struct {
		name  string
		h     Header
		start *domain.Position
		want  bool
	}{
		{"no game yet", Header{MatchLength: 7}, nil, false},
		{"centred cube", Header{MatchLength: 7}, at([2]int{5, 5}, domain.Cube{Owner: domain.None}), true},
		{"own cube", Header{MatchLength: 7}, at([2]int{5, 5}, domain.Cube{Owner: domain.Black, Value: 1}), true},
		{"opponent's cube", Header{MatchLength: 7}, at([2]int{5, 5}, domain.Cube{Owner: domain.White, Value: 1}), false},
		{"Crawford game", Header{MatchLength: 7}, at([2]int{1, 5}, domain.Cube{Owner: domain.None}), false},
		{"post-Crawford", Header{MatchLength: 7}, at([2]int{3, 0}, domain.Cube{Owner: domain.None}), true},
		{"dead at the score", Header{MatchLength: 7}, at([2]int{2, 5}, domain.Cube{Owner: domain.Black, Value: 1}), false},
		{"alive below it", Header{MatchLength: 7}, at([2]int{3, 5}, domain.Cube{Owner: domain.Black, Value: 1}), true},
		{"money ceiling", Header{}, moneyStart(bearOffBoard(), domain.Cube{Owner: domain.Black, Value: 6}, [2]int{}), false},
		{"money below it", Header{}, moneyStart(bearOffBoard(), domain.Cube{Owner: domain.Black, Value: 5}, [2]int{}), true},
		{"roll on the board", Header{}, moneyStart(bearOffBoard(), domain.Cube{Owner: domain.None}, [2]int{3, 1}), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := mustMachine(t, c.h, c.start)
			if got := m.CubeAvailable(); got != c.want {
				t.Errorf("CubeAvailable = %v, want %v", got, c.want)
			}
		})
	}

	m := mustMachine(t, Header{MatchLength: 7}, at([2]int{5, 5}, domain.Cube{Owner: domain.None}))
	m = mustApply(t, m, Action{Side: domain.Black, Kind: KindDouble})
	if m.CubeAvailable() {
		t.Error("a double waiting for its answer leaves no cube to offer")
	}
}

// A forfeit at money play is worth a single at the cube's value, whatever the
// Jacoby rule says of the cube's position.
func TestMachineForfeitMoney(t *testing.T) {
	for _, jacoby := range []bool{false, true} {
		for _, c := range []struct {
			cube domain.Cube
			want int
		}{
			{centredCube(), 1},
			{domain.Cube{Owner: domain.Black, Value: 1}, 2},
			{domain.Cube{Owner: domain.Black, Value: 2}, 4},
		} {
			m := mustMachine(t, Header{Jacoby: jacoby}, moneyStart(InitialBoard(), c.cube, [2]int{}))
			m = mustApply(t, m, Action{Side: domain.Black, Kind: KindForfeit})
			if got := m.Score()[domain.White]; got != c.want {
				t.Errorf("jacoby %v, cube %+v: player 2 scores %d, want %d", jacoby, c.cube, got, c.want)
			}
		}
	}
}
