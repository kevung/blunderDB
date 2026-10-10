package duel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// midgame is a Start off the opening board, so that the machine reads it as a
// game in progress and not as an opening.
func midgame(away [2]int, cube domain.Cube, onRoll, decision int, dice [2]int) *domain.Position {
	b := transcript.InitialBoard()
	b.Points[24].Checkers, b.Points[23].Checkers, b.Points[23].Color = 1, 1, domain.Black
	return &domain.Position{Board: b, Cube: cube, Score: away, PlayerOnRoll: onRoll, DecisionType: decision, Dice: dice}
}

func pair(a, b SideSpec) [2]SideSpec { return [2]SideSpec{a, b} }

// TestDuelAwayScore: the Away score replaces the Start's, gives the opening
// position at that score without a Start, stays none at the start of the
// match, and is refused when the length does not hold it; Crawford and
// post-Crawford keep their cube rules.
func TestDuelAwayScore(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, newStore(t), 2)
	sides := pair(external("A"), external("B"))
	create := func(length int, away [2]int, start *domain.Position) (*State, error) {
		return svc.Create(ctx, "", Settings{MatchLength: length, Away: &away, Start: start, Sides: sides})
	}

	s, err := create(7, [2]int{3, 5}, nil)
	if err != nil || s.Start == nil || s.Start.Score != [2]int{3, 5} || s.Start.Board != transcript.InitialBoard() {
		t.Fatalf("opening at 3-away 5-away: %+v, %v", s, err)
	}
	if s.Score != [2]int{4, 2} {
		t.Errorf("score %v, want the points 4-2", s.Score)
	}
	if s, err = create(7, [2]int{7, 7}, nil); err != nil || s.Start != nil {
		t.Fatalf("the start of the match keeps no Start: %+v, %v", s, err)
	}
	if s, err = create(7, [2]int{2, 6}, midgame([2]int{7, 7}, domain.Cube{Owner: domain.None}, domain.Black, domain.CheckerAction, [2]int{3, 1})); err != nil || s.Start.Score != [2]int{2, 6} {
		t.Fatalf("the Away score replaces the Start's: %+v, %v", s, err)
	}

	for _, c := range []struct {
		name   string
		length int
		away   [2]int
	}{
		{"beyond the length", 7, [2]int{8, 5}},
		{"both in the Crawford game", 7, [2]int{domain.Crawford, domain.Crawford}},
		{"a money session", 0, [2]int{3, 3}},
	} {
		if _, err := create(c.length, c.away, nil); err == nil {
			t.Errorf("%s: accepted", c.name)
		} else if k := refusalKind(t, err); k != transcript.RefusedStartScore {
			t.Errorf("%s: refused %q", c.name, k)
		}
	}

	centred := domain.Cube{Owner: domain.None}
	s, err = create(7, [2]int{domain.Crawford, 4}, midgame([2]int{7, 7}, centred, domain.Black, domain.CheckerAction, [2]int{}))
	if err != nil || s.Awaiting == nil || s.Awaiting.Kind != DecideMove {
		t.Fatalf("the Crawford game offers no cube: %+v, %v", s, err)
	}
	s, err = create(7, [2]int{domain.PostCrawford, 4}, midgame([2]int{7, 7}, centred, domain.White, domain.CheckerAction, [2]int{}))
	if err != nil || s.Awaiting == nil || s.Awaiting.Kind != DecideCube || s.Awaiting.Side != domain.White {
		t.Fatalf("post-Crawford, the trailer has the cube back: %+v, %v", s.Awaiting, err)
	}
	if _, err := create(7, [2]int{3, 5}, &domain.Position{Board: transcript.InitialBoard(), Score: [2]int{3, 5}, PlayerOnRoll: domain.Black, Cube: domain.Cube{Owner: domain.Black}}); err == nil {
		t.Error("a cube owned at 1 is accepted")
	}
	bad := midgame([2]int{7, 7}, centred, domain.Black, domain.CheckerAction, [2]int{})
	bad.Board.Points[6].Checkers = 6 // sixteen checkers
	if _, err := create(7, [2]int{3, 5}, bad); err == nil {
		t.Error("a board of sixteen checkers is accepted")
	}
}

// TestDuelStartChoices: the roll on the board is played or drawn again; a cube
// decision is begun before (the default) or after; a double the board shows
// offered is answered, or taken; player 2 on roll keeps the turn and each
// player's Away score.
func TestDuelStartChoices(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, newStore(t), 3)
	sides := pair(external("A"), external("B"))
	create := func(start *domain.Position, reroll, after bool) *State {
		t.Helper()
		s, err := svc.Create(ctx, "", Settings{MatchLength: 7, Start: start, Reroll: reroll, AfterCube: after, Sides: sides})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		return s
	}
	centred := domain.Cube{Owner: domain.None}

	rolled := midgame([2]int{5, 5}, centred, domain.Black, domain.CheckerAction, [2]int{6, 5})
	if s := create(rolled, false, false); s.Awaiting.Kind != DecideMove || s.Awaiting.Position.Dice != [2]int{6, 5} {
		t.Errorf("the board's roll: %+v", s.Awaiting)
	}
	if s := create(rolled, true, false); s.Awaiting.Kind != DecideCube || s.Start.Dice != [2]int{} {
		t.Errorf("drawn again, the cube decision comes first: %+v", s.Awaiting)
	}
	if s := create(rolled, true, true); s.Awaiting.Kind != DecideMove || s.Awaiting.Side != domain.Black {
		t.Errorf("drawn again after the cube: %+v", s.Awaiting)
	}

	cube := midgame([2]int{5, 5}, centred, domain.Black, domain.CubeAction, [2]int{})
	if s := create(cube, false, false); s.Awaiting.Kind != DecideCube || s.Awaiting.Side != domain.Black {
		t.Errorf("before the cube decision: %+v", s.Awaiting)
	}
	if s := create(cube, false, true); s.Awaiting.Kind != DecideMove || s.Awaiting.Side != domain.Black || len(s.Actions) != 0 {
		t.Errorf("after the cube decision: %+v", s.Awaiting)
	}

	for _, offered := range []domain.Cube{{Owner: domain.None, Value: 1}, {Owner: domain.White, Value: 1}} {
		p := midgame([2]int{5, 5}, offered, domain.Black, domain.CubeAction, [2]int{})
		s := create(p, false, false)
		if s.Awaiting.Kind != DecideAnswer || s.Awaiting.Side != domain.White || len(s.Actions) != 1 ||
			s.Actions[0].Kind != transcript.KindDouble || s.Start.Cube != centred {
			t.Errorf("a double offered %+v, before: %+v, actions %+v, start cube %+v", offered, s.Awaiting, s.Actions, s.Start.Cube)
		}
		s = create(p, false, true)
		if s.Awaiting.Kind != DecideMove || s.Awaiting.Side != domain.Black || s.Start.Cube != (domain.Cube{Owner: domain.White, Value: 1}) {
			t.Errorf("a double offered %+v, after: %+v, start cube %+v", offered, s.Awaiting, s.Start.Cube)
		}
	}
	redouble := midgame([2]int{5, 5}, domain.Cube{Owner: domain.White, Value: 2}, domain.Black, domain.CubeAction, [2]int{})
	if s := create(redouble, false, false); s.Awaiting.Kind != DecideAnswer || s.Start.Cube != (domain.Cube{Owner: domain.Black, Value: 1}) {
		t.Errorf("a redouble offered: %+v, start cube %+v", s.Awaiting, s.Start.Cube)
	}

	// Player 2 on roll, the human playing player 2 and the delegated Side player 1.
	second := midgame([2]int{2, 6}, centred, domain.White, domain.CheckerAction, [2]int{3, 1})
	s, err := svc.Create(ctx, "", Settings{MatchLength: 7, Start: second, Sides: pair(SideSpec{Kind: sideFirst, Name: "Bot"}, external("Me"))})
	if err != nil || s.Awaiting == nil || s.Awaiting.Side != domain.White || s.Awaiting.Kind != DecideMove {
		t.Fatalf("player 2 on roll: %+v, %v", s, err)
	}
	if s.Start.Score != [2]int{2, 6} || s.Score != [2]int{5, 1} {
		t.Errorf("each Away score stays with its player: start %v, score %v", s.Start.Score, s.Score)
	}
	if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil || s.Awaiting == nil || s.Awaiting.Side != domain.White {
		t.Fatalf("after player 2's play the delegated player 1 plays and hands back: %+v, %v", s, err)
	}
}

// TestDuelSingleGame: the Duel ends with its first game, the Match written
// with that game alone at the score it was played at; a money session of two
// delegated Sides is then no longer endless.
func TestDuelSingleGame(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 5)
	away := [2]int{4, 6}
	s, err := svc.Create(ctx, "", Settings{MatchLength: 7, Away: &away, SingleGame: true, Sides: pair(external("A"), SideSpec{Kind: sideFirst, Name: "Bot"})})
	if err != nil || !s.SingleGame {
		t.Fatalf("Create: %+v, %v", s, err)
	}
	for i := 0; s.Ended == nil; i++ {
		if i > 2000 || s.Awaiting == nil {
			t.Fatalf("no end after %d plays: %+v", i, s.Awaiting)
		}
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	if s.Ended.MatchID == 0 || len(s.Games) != 1 || !s.Games[0].Finished || s.Awaiting != nil {
		t.Fatalf("ending %+v, games %+v", s.Ended, s.Games)
	}
	if s.Games[0].InitialScore != [2]int{3, 1} {
		t.Errorf("played at %v, want 3-1", s.Games[0].InitialScore)
	}
	if left, _ := svc.List(ctx, ""); len(left) != 0 {
		t.Errorf("the draft is left: %+v", left)
	}

	s, err = svc.Create(ctx, "", Settings{SingleGame: true, DiscardAtEnd: true,
		Sides: pair(SideSpec{Kind: sideFirst, Name: "A"}, SideSpec{Kind: sideFirst, Name: "B"})})
	if err != nil || s.Ended == nil || !s.Ended.Discarded || len(s.Games) != 1 {
		t.Fatalf("a single money game between two delegated Sides: %+v, %v", s, err)
	}
}

// TestDuelDraftVersion: a draft is written at the oldest format holding it —
// 3 only for a single game — so an older build refuses no more than it must,
// and a version 2 draft replayed and saved stays version 2.
func TestDuelDraftVersion(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 5)
	version := func(id int64) (string, int) {
		t.Helper()
		row, err := st.Duels().Get(ctx, "", id)
		if err != nil {
			t.Fatalf("Get row: %v", err)
		}
		var doc struct {
			FormatVersion int `json:"format_version"`
		}
		if err := json.Unmarshal([]byte(row.Document), &doc); err != nil {
			t.Fatalf("document: %v", err)
		}
		return row.FormatVersion, doc.FormatVersion
	}

	s, err := svc.Create(ctx, "", Settings{MatchLength: 7, Sides: pair(external("A"), SideSpec{Kind: sideFirst, Name: "Bot"})})
	if err != nil || s.Awaiting == nil {
		t.Fatalf("Create: %+v, %v", s, err)
	}
	if row, doc := version(s.ID); row != "2" || doc != 2 {
		t.Errorf("a match draft is written at version %s/%d, want 2", row, doc)
	}
	if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if row, doc := version(s.ID); row != "2" || doc != 2 {
		t.Errorf("replayed and saved, the draft is at version %s/%d, want 2", row, doc)
	}

	s, err = svc.Create(ctx, "", Settings{MatchLength: 7, SingleGame: true, Sides: pair(external("A"), SideSpec{Kind: sideFirst, Name: "Bot"})})
	if err != nil {
		t.Fatalf("Create single game: %v", err)
	}
	if row, doc := version(s.ID); row != "3" || doc != 3 {
		t.Errorf("a single-game draft is written at version %s/%d, want 3", row, doc)
	}
}
