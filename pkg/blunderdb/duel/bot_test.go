package duel

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

func emptyBoard() domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: domain.None}
	}
	return b
}

// TestTwoBotsPlayAMatchInOneCall: a Duel of two Bots is played whole by the
// call that creates it, through the default resolver, and the Match names
// each delegated Side by its Configuration, whatever name was asked.
func TestTwoBotsPlayAMatchInOneCall(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := New(st, Options{Rand: fixedRand(5)})
	bot := SideSpec{Kind: SideBot, Level: "instant", Name: "ignored"}
	s, err := svc.Create(ctx, "", Settings{MatchLength: 3, Sides: [2]SideSpec{bot, bot}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.Awaiting != nil || s.Ended == nil || s.Ended.MatchID == 0 {
		t.Fatalf("two Bots end the match in the call that creates it: %+v", s)
	}
	if s.Score[0] < 3 && s.Score[1] < 3 {
		t.Fatalf("score %v: the match is not won", s.Score)
	}
	m, err := st.Matches().Get(ctx, "", s.Ended.MatchID)
	if err != nil || m.Player1Name != "gammonNet instant" || m.Player2Name != "gammonNet instant" {
		t.Fatalf("match = %+v, %v", m, err)
	}
	o, err := st.Duels().Origin(ctx, "", s.Ended.MatchID)
	if err != nil || o.BotLevel != "instant" || o.BotEngine != gammonnet.PolicyEngineVersion {
		t.Fatalf("origin = %+v, %v; want the Bots' level and policy version", o, err)
	}
}

// TestBotAgainstAnExternalSide: the Bot plays in the same operation as the
// external Side's Play that gives it the turn; only the external Side's
// Decisions are ever awaited.
func TestBotAgainstAnExternalSide(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := New(st, Options{Rand: fixedRand(9)})
	s, err := svc.Create(ctx, "", Settings{MatchLength: 1,
		Sides: [2]SideSpec{external("Alice"), {Kind: SideBot, Level: "instant"}}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for i := 0; s.Ended == nil; i++ {
		if i > 500 {
			t.Fatal("the match never ended")
		}
		if s.Awaiting == nil || s.Awaiting.Side != domain.Black {
			t.Fatalf("only the external Side is awaited: %+v", s.Awaiting)
		}
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	m, err := st.Matches().Get(ctx, "", s.Ended.MatchID)
	if err != nil || m.Player1Name != "Alice" || m.Player2Name != "gammonNet instant" {
		t.Fatalf("match = %+v, %v", m, err)
	}
	o, err := st.Duels().Origin(ctx, "", s.Ended.MatchID)
	if err != nil || o.BotLevel != "instant" || o.BotEngine != gammonnet.PolicyEngineVersion {
		t.Fatalf("origin = %+v, %v; want the Bot's level and policy version", o, err)
	}
}

// TestBotOrigin: the origin names a Bot's level and policy only when a Bot
// played, and keeps both levels of two Bots that differ.
func TestBotOrigin(t *testing.T) {
	bot := func(l string) SideSpec { return SideSpec{Kind: SideBot, Level: l} }
	for _, c := range []struct {
		sides         [2]SideSpec
		level, engine string
	}{
		{[2]SideSpec{external("A"), external("B")}, "", ""},
		{[2]SideSpec{bot("normal"), external("B")}, "normal", gammonnet.PolicyEngineVersion},
		{[2]SideSpec{bot("thorough"), bot("thorough")}, "thorough", gammonnet.PolicyEngineVersion},
		{[2]SideSpec{bot("instant"), bot("normal")}, "instant/normal", gammonnet.PolicyEngineVersion},
	} {
		if l, e := botOrigin(c.sides); l != c.level || e != c.engine {
			t.Errorf("botOrigin(%+v) = %q, %q; want %q, %q", c.sides, l, e, c.level, c.engine)
		}
	}
}

// TestBotRefusesAnUnknownLevel: no weakened or misspelt level stands in for a
// named one (ADR-0072 rule 7).
func TestBotRefusesAnUnknownLevel(t *testing.T) {
	if _, err := Resolve(SideSpec{Kind: SideBot, Level: "weak"}); !errors.Is(err, ErrUnknownLevel) {
		t.Fatalf("Resolve(weak) = %v, want ErrUnknownLevel", err)
	}
}

// TestBotDecisions: each kind of Decision the Arbiter hands a Side gets the
// Play the policy answers, translated to the Arbiter's terms.
func TestBotDecisions(t *testing.T) {
	ctx := context.Background()
	bot, err := NewBot("instant")
	if err != nil {
		t.Fatal(err)
	}

	// A double at the opening, answered by White: the Decision is the
	// taker's, the cube turned to 2 and owned by them.
	opening := domain.Position{Board: transcript.InitialBoard(), Score: [2]int{5, 5},
		PlayerOnRoll: domain.White, DecisionType: domain.CubeAction,
		Cube: domain.Cube{Owner: domain.White, Value: 1}}
	if p, ok, err := bot.Decide(ctx, Decision{Side: domain.White, Kind: DecideAnswer, Position: opening}); err != nil || !ok || p.Kind != PlayTake {
		t.Errorf("opening double: %+v, %v, %v; want a take", p, ok, err)
	}

	// Black, fifteen checkers on the 8-point, nothing off; White one checker
	// on its ace point. White is off next roll whatever the dice, and Black
	// cannot bear one checker off first: a certain gammon.
	lost := emptyBoard()
	lost.Points[8] = domain.Point{Checkers: 15, Color: domain.Black}
	lost.Points[24] = domain.Point{Checkers: 1, Color: domain.White}
	lost.Bearoff = [2]int{0, 14}
	pos := domain.Position{Board: lost, Score: [2]int{5, 5}, PlayerOnRoll: domain.Black,
		DecisionType: domain.CubeAction, Cube: domain.Cube{Owner: domain.None}}
	if p, ok, err := bot.Decide(ctx, Decision{Side: domain.Black, Kind: DecideCube, Position: pos}); err != nil || !ok || p.Kind != PlayResign || p.Level != 2 {
		t.Errorf("certain gammon: %+v, %v, %v; want a resignation at 2", p, ok, err)
	}

	// The opening 3-1 has a best play: its steps are one of the legal plays.
	move := opening
	move.PlayerOnRoll, move.Cube = domain.Black, domain.Cube{Owner: domain.None}
	move.Dice, move.DecisionType = [2]int{3, 1}, domain.CheckerAction
	p, ok, err := bot.Decide(ctx, Decision{Side: domain.Black, Kind: DecideMove, Position: move})
	if err != nil || !ok || p.Kind != PlayMove || len(p.Steps) != 2 {
		t.Fatalf("opening 3-1: %+v, %v, %v", p, ok, err)
	}
}

// TestTwoBotsRefuseAMoneySession: no call would end a money session between
// two delegated Sides, so it is refused at creation, by name.
func TestTwoBotsRefuseAMoneySession(t *testing.T) {
	svc := New(newStore(t), Options{Rand: fixedRand(5)})
	bot := SideSpec{Kind: SideBot, Level: "instant"}
	_, err := svc.Create(context.Background(), "", Settings{MatchLength: 0, Sides: [2]SideSpec{bot, bot}})
	if refusalKind(t, err) != RefusedEndless {
		t.Fatalf("Create = %v, want %s", err, RefusedEndless)
	}
}

// TestBotResignsWhereNoDecisionIsPosed: the cube is the opponent's, so the
// Arbiter poses no cube decision before Black's roll; a Bot is asked anyway
// and resigns its certain gammon, an external Side is never asked.
func TestBotResignsWhereNoDecisionIsPosed(t *testing.T) {
	lost := emptyBoard()
	lost.Points[8] = domain.Point{Checkers: 15, Color: domain.Black}
	lost.Points[24] = domain.Point{Checkers: 1, Color: domain.White}
	lost.Bearoff = [2]int{0, 14}
	start := &domain.Position{Board: lost, Score: [2]int{7, 7}, PlayerOnRoll: domain.Black,
		DecisionType: domain.CheckerAction, Cube: domain.Cube{Owner: domain.White, Value: 1}}

	ctx := context.Background()
	for _, tc := range []struct {
		black  SideSpec
		resign bool
	}{{SideSpec{Kind: SideBot, Level: "instant"}, true}, {external("Alice"), false}} {
		svc := New(newStore(t), Options{Rand: fixedRand(5)})
		s, err := svc.Create(ctx, "", Settings{MatchLength: 7, Start: start,
			Sides: [2]SideSpec{tc.black, external("Bob")}})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		resigned := len(s.Actions) > 0 && s.Actions[0].Kind == transcript.KindResign
		if resigned != tc.resign || (resigned && (s.Actions[0].Side != domain.Black || s.Actions[0].Level != 2)) {
			t.Errorf("%s: actions %+v, want a resignation: %v", tc.black.Kind, s.Actions, tc.resign)
		}
	}
}

func TestLevelInfosFollowGammonNetTable(t *testing.T) {
	infos := LevelInfos()
	if len(infos) != len(BotLevels) {
		t.Fatalf("got %d levels, want %d", len(infos), len(BotLevels))
	}
	for i, info := range infos {
		want, ok := gammonnet.Level(BotLevels[i])
		if !ok {
			t.Fatalf("level %q missing from gammonNet's table", BotLevels[i])
		}
		if info.Name != BotLevels[i] || info.Ply != want.Ply || info.PruneK != want.PruneK {
			t.Errorf("%+v does not match %+v", info, want)
		}
	}
}
