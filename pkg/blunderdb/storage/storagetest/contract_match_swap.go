package storagetest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// swapSeat is everything the readers say of one seat of a match.
type swapSeat struct {
	Name string
	// Per game, in order: the seat's initial score and whether it won.
	Scores   []int32
	Won      []bool
	Elo      *float64
	Exp      *int
	Detail   storage.MatchPlayerDetailStats
	Review   storage.PlayerReview
	BadgePR  float64
	BadgeMWC float64
	BadgeL7  domain.MWC7
	Time     storage.PlayerTimeSummary
	Row      storage.MatchStatsRow
}

// swapSnapshot is a match as its readers see it: per seat, and per move.
type swapSnapshot struct {
	Seats [2]swapSeat
	// Per move id: the position it stands on, its player, and the seat the
	// decision losses attribute it to.
	PosByMove    map[int64]int64
	PlayerByMove map[int64]int32
	LossByMove   map[int64]storage.DecisionLoss
	Moves        map[int64]domain.Move
	Positions    map[int64]domain.Position
	Analyses     map[int64]*domain.PositionAnalysis
	PositionRows int
	Origin       *storage.MatchOrigin
}

// testMatchSwapKeepsPositions: a position is stored from the side on roll
// (domain.Position.NormalizeForStorage), so swapping a match's seats must
// change no position, no analysis and no played-decision score — each person
// keeps every metric, to the bit, under the other seat — and swapping twice
// must give the match back exactly. The match crosses the Crawford game, and
// a draft open on it or an origin (a match played here) refuses the swap.
func testMatchSwapKeepsPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	elo1, elo2 := 1720.5, 1480.0
	exp1, exp2 := 350, 40
	m := domain.Match{Player1Name: "Kevin", Player2Name: "Gildas", MatchLength: 7,
		Player1Elo: &elo1, Player2Elo: &elo2, Player1Experience: &exp1, Player2Experience: &exp2}
	matchID, err := ms.Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1,
		InitialScore: [2]int32{1, 3}, Winner: 1, PointsWon: 2})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	checker := func(slot int, score [2]int, cube domain.Cube, played string, loss float64) int64 {
		pos := statsDecisionPos(t, slot)
		pos.Score = score
		pos.Cube = cube
		id, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save checker position (slot %d): %v", slot, err)
		}
		a := domain.PositionAnalysis{PlayedMoves: []string{played},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "8/6 6/4", Equity: 0.50},
				{Move: played, Equity: 0.50 - loss, EquityError: &loss},
			}}}
		if err := s.Analyses().Save(ctx, "", id, &a); err != nil {
			t.Fatalf("Save checker analysis (slot %d): %v", slot, err)
		}
		return id
	}
	cubeDecision := func(score [2]int, cube domain.Cube, played string, dca domain.DoublingCubeAnalysis) int64 {
		pos := statsDecisionPos(t, 0)
		pos.DecisionType = domain.CubeAction
		pos.Dice = [2]int{0, 0}
		pos.Score = score
		pos.Cube = cube
		id, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save cube position: %v", err)
		}
		if err := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{AnalysisType: "DoublingCube",
			PlayedCubeActions: []string{played}, DoublingCubeAnalysis: &dca}); err != nil {
			t.Fatalf("Save cube analysis: %v", err)
		}
		return id
	}

	// Scores differ between the seats and the cube is owned, from the side on
	// roll: a swap that rewrote either would land on other positions.
	open := checker(0, [2]int{6, 4}, domain.Cube{Value: 0, Owner: domain.None}, "13/11 24/23", 0.05)
	reply := checker(1, [2]int{4, 6}, domain.Cube{Value: 0, Owner: domain.None}, "13/11 24/23", 0.12)
	redouble := cubeDecision([2]int{6, 4}, domain.Cube{Value: 1, Owner: 0}, "Double", domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 0.5, CubefulDoubleTakeEquity: 0.4, CubefulDoublePassEquity: 1.0,
		CubefulDoubleTakeError: 0.1, CubefulDoublePassError: 0.5})
	take := cubeDecision([2]int{4, 6}, domain.Cube{Value: 1, Owner: 0}, "Take", domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 0.5, CubefulDoubleTakeEquity: 1.1, CubefulDoublePassEquity: 1.0,
		CubefulDoubleTakeError: 0.6, CubefulDoublePassError: 0.5})
	owned := checker(2, [2]int{4, 6}, domain.Cube{Value: 2, Owner: 1}, "13/11 24/23", 0.02)

	i32 := func(v int32) *int32 { return &v }
	i64 := func(v int64) *int64 { return &v }
	moves := []domain.Move{
		{MoveType: "checker", PositionID: open, Player: 1, Dice: [2]int32{1, 1}, CheckerMove: "13/11 24/23",
			LuckMP: i32(120), DecisionMS: i64(3000), RollTickMS: i64(1000), TickMS: i64(4000)},
		{MoveType: "checker", PositionID: reply, Player: -1, Dice: [2]int32{1, 2}, CheckerMove: "13/11 24/23",
			LuckMP: i32(-80), DecisionMS: i64(5000)},
		{MoveType: "cube", PositionID: redouble, Player: 1, CubeAction: "Double", DecisionMS: i64(2000)},
		{MoveType: "cube", PositionID: take, Player: -1, CubeAction: "Take", DecisionMS: i64(7000)},
		{MoveType: "checker", PositionID: owned, Player: -1, Dice: [2]int32{1, 3}, CheckerMove: "13/11 24/23",
			LuckMP: i32(30), DecisionMS: i64(1500)},
	}
	for i := range moves {
		moves[i].GameID, moves[i].MoveNumber = gameID, int32(i+1)
		if _, err := ms.CreateMove(ctx, "", &moves[i]); err != nil {
			t.Fatalf("CreateMove %d: %v", i, err)
		}
	}

	// The Crawford game (Kevin 1-away, no cube), then a post-Crawford game,
	// where the trailer's double is priced at the stored PostCrawford sentinel.
	crawford, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 2,
		InitialScore: [2]int32{6, 3}, Winner: -1, PointsWon: 1})
	if err != nil {
		t.Fatalf("CreateGame Crawford: %v", err)
	}
	postCrawford, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 3,
		InitialScore: [2]int32{6, 4}, Winner: 1, PointsWon: 2})
	if err != nil {
		t.Fatalf("CreateGame post-Crawford: %v", err)
	}
	none := domain.Cube{Value: 0, Owner: domain.None}
	later := []domain.Move{
		{GameID: crawford, MoveType: "checker", Player: 1, Dice: [2]int32{1, 4}, CheckerMove: "13/11 24/23", LuckMP: i32(55),
			PositionID: checker(3, [2]int{domain.Crawford, 4}, none, "13/11 24/23", 0.07)},
		{GameID: crawford, MoveType: "checker", Player: -1, Dice: [2]int32{1, 5}, CheckerMove: "13/11 24/23", LuckMP: i32(-210),
			PositionID: checker(4, [2]int{4, domain.Crawford}, none, "13/11 24/23", 0.09)},
		{GameID: postCrawford, MoveType: "cube", Player: -1, CubeAction: "Double",
			PositionID: cubeDecision([2]int{3, domain.PostCrawford}, none, "Double", domain.DoublingCubeAnalysis{
				CubefulNoDoubleEquity: 0.5, CubefulDoubleTakeEquity: 0.3, CubefulDoublePassEquity: 1.0,
				CubefulDoubleTakeError: 0.2, CubefulDoublePassError: 0.5})},
		{GameID: postCrawford, MoveType: "checker", Player: 1, Dice: [2]int32{1, 6}, CheckerMove: "13/11 24/23", LuckMP: i32(15),
			PositionID: checker(5, [2]int{domain.PostCrawford, 3}, domain.Cube{Value: 1, Owner: 0}, "13/11 24/23", 0.11)},
	}
	for i := range later {
		later[i].MoveNumber = int32(i + 1)
		if _, err := ms.CreateMove(ctx, "", &later[i]); err != nil {
			t.Fatalf("CreateMove (later game) %d: %v", i, err)
		}
	}

	// Another match reaches the opening position: it must not see the swap.
	other := domain.Match{Player1Name: "Erin", Player2Name: "Bob", MatchLength: 7}
	otherID, err := ms.Save(ctx, "", &other)
	if err != nil {
		t.Fatalf("Save other match: %v", err)
	}
	otherGame, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: otherID, GameNumber: 1, Winner: -1, PointsWon: 1})
	if err != nil {
		t.Fatalf("CreateGame other: %v", err)
	}
	if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: otherGame, MoveNumber: 1, MoveType: "checker",
		PositionID: open, Player: 1, Dice: [2]int32{1, 1}, CheckerMove: "13/11 24/23", LuckMP: i32(-37)}); err != nil {
		t.Fatalf("CreateMove other: %v", err)
	}

	swap := func(stage string) {
		t.Helper()
		if err := ms.SwapPlayers(ctx, "", matchID); err != nil {
			t.Fatalf("%s: SwapPlayers: %v", stage, err)
		}
	}
	snap := func(stage string, id int64) swapSnapshot {
		t.Helper()
		sn, err := takeSwapSnapshot(ctx, s, id)
		if err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		return sn
	}

	before := snap("before", matchID)
	otherBefore := snap("before (other match)", otherID)
	for seat, st := range before.Seats {
		if st.Detail.TotalDecisions == 0 || st.Review.Decisions == 0 || st.BadgeMWC == 0 || !st.BadgeL7.Available {
			t.Fatalf("seat %d has nothing to compare: %+v", seat+1, st)
		}
	}

	swap("first swap")
	after := snap("after one swap", matchID)
	if !reflect.DeepEqual(after.PosByMove, before.PosByMove) {
		t.Errorf("the swap moved decisions to other positions: %v, was %v", after.PosByMove, before.PosByMove)
	}
	if !reflect.DeepEqual(after.Positions, before.Positions) {
		t.Errorf("the swap rewrote positions")
	}
	if !reflect.DeepEqual(after.Analyses, before.Analyses) {
		t.Errorf("the swap changed analyses")
	}
	if after.PositionRows != before.PositionRows {
		t.Errorf("the swap left %d positions, was %d", after.PositionRows, before.PositionRows)
	}
	for id, p := range before.PlayerByMove {
		if after.PlayerByMove[id] != -p {
			t.Errorf("move %d: player %d after the swap, want %d", id, after.PlayerByMove[id], -p)
		}
		mv, was := after.Moves[id], before.Moves[id]
		mv.Player = was.Player
		if !reflect.DeepEqual(mv, was) {
			t.Errorf("move %d changed beyond its player:\n got %+v\nwant %+v", id, mv, was)
		}
		l, wl := after.LossByMove[id], before.LossByMove[id]
		if l.Player != 1-wl.Player {
			t.Errorf("move %d: decision loss on seat %d after the swap, was %d", id, l.Player, wl.Player)
		}
		l.Player = wl.Player
		if !reflect.DeepEqual(l, wl) {
			t.Errorf("move %d: decision loss changed beyond its seat:\n got %+v\nwant %+v", id, l, wl)
		}
	}
	for seat := range 2 {
		got, want := after.Seats[seat], before.Seats[1-seat]
		got.Row.Seat = want.Row.Seat
		if !reflect.DeepEqual(got, want) {
			t.Errorf("seat %d after the swap is not seat %d before:\n got %+v\nwant %+v", seat+1, 2-seat, got, want)
		}
	}
	if o := snap("after one swap (other match)", otherID); !reflect.DeepEqual(o, otherBefore) {
		t.Errorf("swapping a match changed another match sharing a position")
	}

	swap("second swap")
	if again := snap("after two swaps", matchID); !reflect.DeepEqual(again, before) {
		t.Errorf("two swaps do not give the match back:\n got %+v\nwant %+v", again, before)
	}

	refused := func(stage string, want error, was swapSnapshot) {
		t.Helper()
		err := ms.SwapPlayers(ctx, "", matchID)
		if !errors.Is(err, want) || !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s: SwapPlayers = %v, want %v", stage, err, want)
		}
		if got := snap(stage, matchID); !reflect.DeepEqual(got, was) {
			t.Errorf("%s: the refused swap changed the match", stage)
		}
	}
	// A draft open on the match replaces it at its Finish, in the order the
	// draft holds: the swap would be silently undone.
	draft, err := s.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1",
		MatchID: matchID, Label: "Kevin — Gildas", Document: "{}"})
	if err != nil {
		t.Fatalf("Save draft: %v", err)
	}
	refused("draft open", storage.ErrSwapDraftOpen, before)
	if err := s.Transcriptions().Delete(ctx, "", draft); err != nil {
		t.Fatalf("Delete draft: %v", err)
	}
	// A match played here: its origin names the seats as they were played.
	if err := s.Duels().SetOrigin(ctx, "", &storage.MatchOrigin{MatchID: matchID, DiceSeed: "seed", OverTime: 2,
		DeclaredBots: []storage.DeclaredBot{{Player: 2, Configuration: "thorough", Engine: "v1"}}}); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	refused("played here", storage.ErrSwapPlayedHere, snap("with an origin", matchID))
}

func takeSwapSnapshot(ctx context.Context, s storage.Storage, matchID int64) (swapSnapshot, error) {
	sn := swapSnapshot{PosByMove: map[int64]int64{}, PlayerByMove: map[int64]int32{},
		LossByMove: map[int64]storage.DecisionLoss{}, Moves: map[int64]domain.Move{}}
	m, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		return sn, fmt.Errorf("get match: %w", err)
	}
	sn.Seats[0].Name, sn.Seats[1].Name = m.Player1Name, m.Player2Name
	sn.Seats[0].Elo, sn.Seats[1].Elo = m.Player1Elo, m.Player2Elo
	sn.Seats[0].Exp, sn.Seats[1].Exp = m.Player1Experience, m.Player2Experience

	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			return sn, fmt.Errorf("games: %w", err)
		}
		for seat, side := range [2]int32{1, -1} {
			sn.Seats[seat].Scores = append(sn.Seats[seat].Scores, g.InitialScore[seat])
			sn.Seats[seat].Won = append(sn.Seats[seat].Won, g.Winner == side)
		}
	}
	switch o, err := s.Duels().Origin(ctx, "", matchID); {
	case err == nil:
		sn.Origin = o
	case !errors.Is(err, storage.ErrNotFound):
		return sn, fmt.Errorf("origin: %w", err)
	}

	var ids []int64
	for mv, err := range s.Matches().MovesByMatch(ctx, "", matchID) {
		if err != nil {
			return sn, fmt.Errorf("MovesByMatch: %w", err)
		}
		sn.PosByMove[mv.ID] = mv.PositionID
		sn.PlayerByMove[mv.ID] = mv.Player
		sn.Moves[mv.ID] = *mv
		ids = append(ids, mv.PositionID)
	}
	positions, err := s.Positions().LoadByIDs(ctx, "", ids)
	if err != nil {
		return sn, fmt.Errorf("LoadByIDs: %w", err)
	}
	sn.Positions = map[int64]domain.Position{}
	for _, p := range positions {
		sn.Positions[p.ID] = p
	}
	if sn.Analyses, err = s.Analyses().LoadMany(ctx, "", ids); err != nil {
		return sn, fmt.Errorf("LoadMany: %w", err)
	}
	if sn.PositionRows, err = s.Positions().Count(ctx, ""); err != nil {
		return sn, fmt.Errorf("count positions: %w", err)
	}

	detail, err := s.Stats().MatchDetail(ctx, "", matchID)
	if err != nil {
		return sn, fmt.Errorf("MatchDetail: %w", err)
	}
	sn.Seats[0].Detail, sn.Seats[1].Detail = detail.Player1, detail.Player2
	review, err := s.Stats().MatchReview(ctx, "", matchID)
	if err != nil {
		return sn, fmt.Errorf("MatchReview: %w", err)
	}
	sn.Seats[0].Review, sn.Seats[1].Review = review.Players[0], review.Players[1]
	badges, err := s.Stats().MatchBadges(ctx, "", []int64{matchID})
	if err != nil {
		return sn, fmt.Errorf("MatchBadges: %w", err)
	}
	b := badges[matchID]
	sn.Seats[0].BadgePR, sn.Seats[0].BadgeMWC, sn.Seats[0].BadgeL7 = b.PR, b.MWCLoss, b.MWC7
	sn.Seats[1].BadgePR, sn.Seats[1].BadgeMWC, sn.Seats[1].BadgeL7 = b.PR2, b.MWCLoss2, b.MWC7P2
	times, err := s.Stats().MatchTimeSummary(ctx, "", matchID)
	if err != nil {
		return sn, fmt.Errorf("MatchTimeSummary: %w", err)
	}
	sn.Seats[0].Time, sn.Seats[1].Time = times.Players[0], times.Players[1]
	rows, err := s.Stats().MatchStats(ctx, "", []int64{matchID})
	if err != nil {
		return sn, fmt.Errorf("MatchStats: %w", err)
	}
	for _, r := range rows {
		if r.Seat == 1 || r.Seat == 2 {
			sn.Seats[r.Seat-1].Row = r
		}
	}
	losses, err := s.Stats().MatchDecisionLosses(ctx, "", matchID)
	if err != nil {
		return sn, fmt.Errorf("MatchDecisionLosses: %w", err)
	}
	for _, l := range losses {
		sn.LossByMove[l.MoveID] = l
	}
	return sn, nil
}
