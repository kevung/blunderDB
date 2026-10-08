package transcript

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// state is the board the Replay walks. FromMAT drives it too, one Action at a time,
// because reading a .mat needs the position before each move to recognise the play
// that was written down.
type state struct {
	header Header
	board  domain.Board
	cube   domain.Cube
	points [2]int
	games  []GameInfo

	gameActive     bool
	crawfordPlayed bool
	// pendingDouble is the side of a double still waiting for its answer, -1 for none.
	pendingDouble int
	// turn is the side whose Action is expected next, -1 while unknown.
	turn       int
	moveNumber int32

	prevKind Kind
	prevSide int
	hasPrev  bool

	// jacobyScores makes the Jacoby rule reduce what a game is worth. A Replay never
	// sets it: a recorded gammon is what happened, and the rule is a flag for the
	// evaluator. A Machine sets it, because there the rule decides the score.
	jacobyScores bool

	// forfeited is set by a KindForfeit: the match is over whatever the
	// score, won by forfeitWinner.
	forfeited     bool
	forfeitWinner int
}

func newState(h Header) *state {
	return &state{
		header:        h,
		board:         InitialBoard(),
		cube:          centredCube(),
		pendingDouble: -1,
		turn:          -1,
	}
}

// clone copies a state deeply enough to be replayed from without the original moving.
// Everything in it is a value but the games, which step appends to and writes into.
func (s *state) clone() state {
	out := *s
	out.games = append([]GameInfo(nil), s.games...)
	return out
}

func (s *state) matchOver() bool {
	if s.forfeited {
		return true
	}
	L := s.header.MatchLength
	return L > 0 && (s.points[domain.Black] >= L || s.points[domain.White] >= L)
}

// winner is the side that won the match once it is over: the one a forfeit
// left, otherwise the one ahead — at money play the score alone names no one.
func (s *state) winner() int {
	if s.forfeited {
		return s.forfeitWinner
	}
	if s.points[domain.White] > s.points[domain.Black] {
		return domain.White
	}
	return domain.Black
}

// awayScores writes the score the way a Position carries it: how many points each
// player still needs, with the Crawford rule folded into the two smallest values —
// 1 in the Crawford game, 0 after it (CONTEXT.md "Away score", ADR-0045 rule 7).
func (s *state) awayScores() [2]int {
	if s.header.MatchLength <= 0 {
		return [2]int{domain.Unlimited, domain.Unlimited}
	}
	crawford := len(s.games) > 0 && s.games[len(s.games)-1].Crawford
	away := domain.AwayScores(s.header.MatchLength, s.points[domain.Black], s.points[domain.White])
	for i, v := range away {
		if v < 0 {
			v = 0
		}
		if v == domain.Crawford && !crawford {
			v = domain.PostCrawford
		}
		away[i] = v
	}
	return away
}

// position builds the Position an Action is played from. The session's rules are
// posted for the panel and evaluator, not as identity (ADR-0028).
func (s *state) position(side int, dice [2]int, decision int, cube domain.Cube) domain.Position {
	pos := domain.Position{
		Board:        s.board,
		Cube:         cube,
		Dice:         dice,
		Score:        s.awayScores(),
		PlayerOnRoll: side,
		DecisionType: decision,
		MaxCube:      s.header.MaxCube,
	}
	if s.header.MatchLength <= 0 {
		if s.header.Jacoby {
			pos.HasJacoby = 1
		}
		if s.header.Beaver {
			pos.HasBeaver = 1
		}
	}
	return pos
}

// ensureGame opens a game if none is running, deriving its initial score and whether
// it is the Crawford game: the first game a player enters one point from the match.
func (s *state) ensureGame() {
	if s.gameActive {
		return
	}
	crawford := false
	if L := s.header.MatchLength; L > 0 && !s.crawfordPlayed {
		if s.points[domain.Black] == L-1 || s.points[domain.White] == L-1 {
			crawford, s.crawfordPlayed = true, true
		}
	}
	s.games = append(s.games, GameInfo{
		Number:       len(s.games) + 1,
		InitialScore: s.points,
		DerivedScore: s.points,
		Winner:       -1,
		Crawford:     crawford,
		First:        -1,
		Last:         -1,
	})
	s.board = InitialBoard()
	s.cube = centredCube()
	s.pendingDouble = -1
	s.turn = -1
	s.moveNumber = 0
	s.gameActive = true
}

func (s *state) endGame(winner, points int) {
	if !s.gameActive {
		return
	}
	g := &s.games[len(s.games)-1]
	g.Winner, g.PointsWon, g.Finished = winner, points, true
	if winner == domain.Black || winner == domain.White {
		s.points[winner] += points
	}
	s.gameActive = false
	s.pendingDouble = -1
	s.turn = -1
}

// gamePoints is what winning by bearing off is worth: a single, a gammon when the
// loser bore nothing off, a backgammon when a loser's checker is still on the bar or
// in the winner's home board — times the value of the cube.
//
// A Replay leaves Jacoby out (fonctionnel.md §1.3): it is a Position flag the
// evaluator applies (ADR-0028), never a reduction of a recorded gammon. A Machine,
// which decides the score rather than reading it, applies it ([state.jacobyOnly]).
func (s *state) gamePoints(winner int) int {
	loser := opponent(winner)
	base := 1
	if s.board.Bearoff[loser] == 0 {
		base = 2
		if s.trapped(loser, winner) {
			base = 3
		}
	}
	if s.jacobyOnly() {
		base = 1
	}
	return base * cubeValue(s.cube)
}

// jacobyOnly reports whether a gammon or a backgammon counts as a single game: the
// Jacoby rule of a money session, under a cube nobody has turned yet.
func (s *state) jacobyOnly() bool {
	return s.jacobyScores && s.header.MatchLength <= 0 && s.header.Jacoby &&
		s.cube.Owner == domain.None && s.cube.Value == 0
}

// trapped reports whether the loser still has a checker on the bar or inside the
// winner's home board — the backgammon condition.
func (s *state) trapped(loser, winner int) bool {
	if s.board.Points[barOf(loser)].Checkers > 0 {
		return true
	}
	lo, hi := 1, 6 // Black's home board
	if winner == domain.White {
		lo, hi = 19, 24
	}
	for i := lo; i <= hi; i++ {
		pt := s.board.Points[i]
		if pt.Color == loser && pt.Checkers > 0 {
			return true
		}
	}
	return false
}

// bearsTurn reports whether an Action counts for the double-turn rule. A take or a pass answers the other side's offer and leaves the turn
// where it was, which is why the doubler playing right after a take is not a double
// turn (fonctionnel.md §1.4).
//
// A resignation is out of the count too: it may come at any moment, and counting it
// would mark every resignation after its author's own play — a false positive.
func bearsTurn(k Kind) bool {
	switch k {
	case KindTake, KindPass, KindResign, KindForfeit:
		return false
	}
	return true
}

// step replays one Action against the state and returns what it derives about it.
func (s *state) step(i int, a Action) ActionInfo {
	info := ActionInfo{Index: i, Side: a.Side, Kind: a.Kind, MoveNumber: -1, GameIndex: -1,
		DecisionMS: a.DecisionMS, CubeDecisionMS: a.CubeDecisionMS}

	if s.matchOver() {
		info.add(PastEnd, "the match is already won")
	}
	// No game opens on a cube action: one recorded after its game ended is kept in
	// that game and marked, and the next play opens the next game.
	if a.Score == nil && !s.gameActive && len(s.games) > 0 && isCubeKind(a.Kind) {
		return s.lateCube(i, a, info)
	}
	// An Action opens a game when none is running, or when it declares the score of
	// one: the running game then stops there, unfinished.
	opens := !s.gameActive || a.Score != nil
	if !opens && s.hasPrev && bearsTurn(s.prevKind) && bearsTurn(a.Kind) && s.prevSide == a.Side {
		info.add(DoubleTurn, fmt.Sprintf("player %d acts twice in a row", a.Side+1))
	}
	if opens {
		s.endGame(-1, 0)
		// A declared score is posted BEFORE the game opens, so that the Crawford
		// mention of the game is decided on the score it is played at.
		derived := s.points
		declared := a.Score != nil && s.declare(&info, *a.Score)
		s.ensureGame()
		if declared {
			g := &s.games[len(s.games)-1]
			g.Declared, g.DerivedScore = true, derived
		}
		info.OpensGame = true
	}

	switch a.Kind {
	case KindChecker, KindDance, KindUnrecorded:
		s.ensureGame()
		dice := a.Dice
		if opens {
			// The opening order (player 1's die first) is the document's; the
			// Position, the saved Move and the .mat carry the roll high die first,
			// as every other source writes an opening play.
			dice = [2]int{max(dice[0], dice[1]), min(dice[0], dice[1])}
		}
		pos := s.position(a.Side, dice, domain.CheckerAction, s.cube)
		info.Before, info.HasPosition = pos, true
		// A game's first play, and a roll the side could not double before,
		// had no cube decision (ADR-0073), so no duration is deduced for one.
		info.cubeChoice = !opens && s.pendingDouble < 0 && s.mayDouble(a.Side)
		legal := domain.LegalMoves(&pos)
		if opens && a.Dice[0] != 0 && a.Dice[0] == a.Dice[1] {
			info.add(InconsistentDice, fmt.Sprintf("the game's first play is rolled %d%d: no opening roll is a double", a.Dice[0], a.Dice[1]))
		}
		if w := openingWinner(a.Dice); opens && w >= 0 && w != a.Side {
			info.add(InconsistentDice, fmt.Sprintf("player %d won the opening roll and plays first", w+1))
		}

		if a.Kind == KindUnrecorded {
			// The board is unknown: carry the last one forward, check nothing,
			// and report an incomplete record, not a player error.
			info.After = s.board
			info.Notation = UnrecordedNotation
			info.add(UnrecordedMove, "the record does not say what was played")
		} else if a.Kind == KindDance {
			info.After = s.board
			info.Notation = "Cannot Move"
			if len(legal) > 0 {
				info.add(IllegalMove, "the roll allows a play, so no dance is possible")
			}
		} else {
			if !diceCoherent(s.board, a.Steps, a.Dice, a.Side) {
				info.add(InconsistentDice, fmt.Sprintf("the play does not use the roll %d%d", a.Dice[0], a.Dice[1]))
			}
			resolved, reached := resolveSteps(s.board, a.Side, a.Steps)
			// Notate the steps as RECORDED, not the legal play reaching the same
			// board: one board has several spellings ("16/10 10/7", "13/7 16/13").
			info.Notation = domain.Notation(resolved, a.Side)
			if play := findPlay(legal, reached); play != nil {
				info.After = play.Result.Board
			} else {
				info.add(IllegalMove, "no legal play from the previous position reaches this board")
				info.After = reached
				if a.BoardAfter != nil {
					info.After = *a.BoardAfter
				}
			}
		}
		s.board = info.After
		info.MoveNumber, s.moveNumber = s.moveNumber, s.moveNumber+1
		s.turn = opponent(a.Side)
		if s.board.Bearoff[a.Side] >= domain.CheckersPerPlayer {
			s.endGame(a.Side, s.gamePoints(a.Side))
		}

	case KindDouble:
		s.ensureGame()
		info.Before, info.HasPosition = s.position(a.Side, [2]int{}, domain.CubeAction, s.cube), true
		info.After = s.board
		switch {
		case opens:
			info.add(ImpossibleCube, "no one doubles before the game's first play")
		case s.pendingDouble >= 0:
			info.add(ImpossibleCube, "a double is already waiting for its answer")
		case s.cube.Owner != domain.None && s.cube.Owner != a.Side:
			info.add(ImpossibleCube, fmt.Sprintf("player %d does not hold the cube", a.Side+1))
		}
		if len(s.games) > 0 && s.games[len(s.games)-1].Crawford {
			info.add(ImpossibleCube, "the cube is dead in the Crawford game")
		}
		if s.header.MaxCube > 0 && s.cube.Value >= s.header.MaxCube {
			info.add(ImpossibleCube, fmt.Sprintf("the cube is capped at %d", 1<<s.header.MaxCube))
		}
		s.pendingDouble = a.Side
		info.MoveNumber, s.moveNumber = s.moveNumber, s.moveNumber+1
		s.turn = opponent(a.Side)

	case KindTake, KindPass:
		s.ensureGame()
		offered := answeredCube(s.cube)
		info.Before, info.HasPosition = s.position(a.Side, [2]int{}, domain.CubeAction, offered), true
		info.After = s.board
		if s.pendingDouble < 0 || s.pendingDouble == a.Side {
			info.add(ImpossibleCube, "no double from the other side to answer")
		}
		info.MoveNumber, s.moveNumber = s.moveNumber, s.moveNumber+1
		if a.Kind == KindTake {
			s.cube = domain.Cube{Owner: a.Side, Value: s.cube.Value + 1}
			s.pendingDouble = -1
			// The doubler rolls next.
			s.turn = opponent(a.Side)
		} else {
			winner := s.pendingDouble
			if winner < 0 {
				winner = opponent(a.Side)
			}
			// A pass is worth the cube as it stood BEFORE the offer.
			s.endGame(winner, cubeValue(s.cube))
		}

	case KindResign:
		s.ensureGame()
		// A resignation is a fact of the Game, not a Move: no Position, no notation,
		// nothing in `move` (ADR-0045 §6). Before is filled for the panel only, and
		// HasPosition stays false.
		info.Before = s.position(a.Side, [2]int{}, domain.CubeAction, s.cube)
		info.After = s.board
		level := a.Level
		if level < 1 {
			level = 1
		}
		if level > 3 {
			level = 3
		}
		if s.jacobyOnly() {
			level = 1
		}
		s.endGame(opponent(a.Side), level*cubeValue(s.cube))

	case KindForfeit:
		s.ensureGame()
		info.Before = s.position(a.Side, [2]int{}, domain.CubeAction, s.cube)
		info.After = s.board
		winner := opponent(a.Side)
		points := cubeValue(s.cube)
		if L := s.header.MatchLength; L > 0 {
			points = max(L-s.points[winner], 1)
		}
		s.endGame(winner, points)
		s.forfeited, s.forfeitWinner = true, winner

	default:
		info.add(IllegalMove, fmt.Sprintf("unknown action kind %q", a.Kind))
	}

	if len(s.games) > 0 {
		gi := len(s.games) - 1
		g := &s.games[gi]
		info.GameIndex, info.GameNumber, info.Score = gi, g.Number, g.InitialScore
		if g.First < 0 {
			g.First = i
		}
		g.Last = i
	}

	s.prevKind, s.prevSide, s.hasPrev = a.Kind, a.Side, true
	return info
}

func isCubeKind(k Kind) bool {
	return k == KindDouble || k == KindTake || k == KindPass
}

// lateCube steps a cube action recorded after its game ended. It changes neither
// the cube nor the score: the game is over, and it stays a Move of that game so the
// user finds it where the record put it.
func (s *state) lateCube(i int, a Action, info ActionInfo) ActionInfo {
	info.Before, info.HasPosition = s.position(a.Side, [2]int{}, domain.CubeAction, s.cube), true
	info.After = s.board
	info.add(ImpossibleCube, "the game is already over")
	info.MoveNumber, s.moveNumber = s.moveNumber, s.moveNumber+1
	gi := len(s.games) - 1
	g := &s.games[gi]
	info.GameIndex, info.GameNumber, info.Score = gi, g.Number, g.InitialScore
	g.Last = i
	s.prevKind, s.prevSide, s.hasPrev = a.Kind, a.Side, true
	return info
}

// declare posts the score a game's first Action declares as the score of play, and
// reports whether it did. An unusable score (money, negative) is marked and left
// aside; a used one is marked when it differs from the derived score (ADR-0053).
// A score reaching the length is used; what follows is marked PastEnd.
//
// In money play a 0-0 is no declaration at all: it is what keeps two games of a
// session apart when the first one stops short, and it says nothing wrong.
func (s *state) declare(info *ActionInfo, score [2]int) bool {
	switch {
	case s.header.MatchLength <= 0:
		if score != [2]int{} {
			info.add(ScoreMismatch, "a money session has no score")
		}
		return false
	case score[0] < 0 || score[1] < 0:
		info.add(ScoreMismatch, fmt.Sprintf("the declared score %d-%d is negative", score[0], score[1]))
		return false
	}
	if score != s.points {
		info.add(ScoreMismatch, fmt.Sprintf("the declared score is %d-%d, the previous games give %d-%d",
			score[0], score[1], s.points[0], s.points[1]))
	}
	s.points = score
	return true
}

// next describes the Action the document is waiting for, on a projection of the state:
// when a game has just ended, the score and the Crawford mention of the game to come
// are already the ones the panel must show.
func (s *state) next(nextScore *[2]int) Next {
	proj := *s
	proj.games = append([]GameInfo(nil), s.games...)
	gameStart := !s.gameActive
	if nextScore != nil {
		// A boundary waiting at the end: the next Action opens a game at that score.
		proj.endGame(-1, 0)
		if proj.header.MatchLength > 0 && nextScore[0] >= 0 && nextScore[1] >= 0 {
			proj.points = *nextScore
		}
		gameStart = true
	}
	active := proj.gameActive
	proj.ensureGame()

	n := Next{MatchOver: s.matchOver(), GameNumber: len(proj.games)}
	if len(proj.games) > 0 {
		n.Crawford = proj.games[len(proj.games)-1].Crawford
	}
	n.MatchOver = proj.matchOver()
	switch {
	case gameStart || !active:
		n.Expects, n.Side, n.GameStart = KindChecker, domain.Black, true
	case s.pendingDouble >= 0:
		n.Expects, n.Side = KindTake, opponent(s.pendingDouble)
	default:
		n.Expects, n.Side = KindChecker, s.turn
		if n.Side < 0 {
			n.Side = domain.Black
		}
	}
	decision := domain.CheckerAction
	cube := proj.cube
	if n.Expects == KindTake {
		decision = domain.CubeAction
		cube = answeredCube(cube)
	}
	n.Position = proj.position(n.Side, [2]int{}, decision, cube)
	return n
}

// answeredCube is the cube a take/pass decision is recorded with. The answerer
// decides on the cube AT THE LEVEL OFFERED (fonctionnel.md §1.2), and it is
// held by no one, as every importer records it: owned by the answerer, the
// position would be the very one of their own redouble decision on that
// board, and share its Zobrist hash and its analysis.
func answeredCube(c domain.Cube) domain.Cube {
	return domain.Cube{Owner: domain.None, Value: c.Value + 1}
}

// openingWinner is the side an opening roll names — the dice are player 1's then
// player 2's, the higher one wins — or -1 for a double or an incomplete roll.
func openingWinner(d [2]int) int {
	switch {
	case d[0] == 0 || d[1] == 0 || d[0] == d[1]:
		return -1
	case d[0] > d[1]:
		return domain.Black
	}
	return domain.White
}

// findPlay returns the legal play that reaches board, or nil. The comparison is by
// RESULTING BOARD and never by notation: LegalMoves deduplicates by board, so two
// notations of one play are one entry, and a notation comparison would miss it.
func findPlay(plays []domain.LegalPlay, board domain.Board) *domain.LegalPlay {
	for i := range plays {
		if plays[i].Result.Board == board {
			return &plays[i]
		}
	}
	return nil
}

// resolveSteps plays steps on a board without judging them, and returns them in the
// order they were playable with their hits filled in, alongside the board they leave.
//
// It is NOT domain's applier, which may assume legal-generator input: steps here are
// arbitrary. They are ordered to stay applicable ("bar/24 24/18" either way round),
// hits are recomputed from the board, and a step with no checker is played last and
// yields whatever board the Replay then reports as illegal.
func resolveSteps(b domain.Board, mover int, steps []domain.CheckerStep) ([]domain.CheckerStep, domain.Board) {
	remaining := append([]domain.CheckerStep(nil), steps...)
	resolved := make([]domain.CheckerStep, 0, len(steps))
	for len(remaining) > 0 {
		pick := 0
		for i, s := range remaining {
			if s.From >= 0 && s.From <= 25 && b.Points[s.From].Color == mover && b.Points[s.From].Checkers > 0 {
				pick = i
				break
			}
		}
		s := remaining[pick]
		remaining = append(remaining[:pick], remaining[pick+1:]...)
		s.Hit = s.To >= 0 && s.To <= 25 &&
			b.Points[s.To].Color == opponent(mover) && b.Points[s.To].Checkers == 1
		b = applyStep(b, mover, s)
		resolved = append(resolved, s)
	}
	return resolved, b
}

func applyStep(b domain.Board, mover int, s domain.CheckerStep) domain.Board {
	if s.From >= 0 && s.From <= 25 {
		from := &b.Points[s.From]
		if from.Checkers > 0 {
			from.Checkers--
			if from.Checkers == 0 {
				from.Color = domain.None
			}
		}
	}
	if s.To == domain.Off {
		if mover == domain.Black || mover == domain.White {
			b.Bearoff[mover]++
		}
		return b
	}
	if s.To < 0 || s.To > 25 {
		return b
	}
	dst := &b.Points[s.To]
	if s.Hit && dst.Color == opponent(mover) && dst.Checkers == 1 {
		bar := barOf(opponent(mover))
		b.Points[bar].Checkers++
		b.Points[bar].Color = opponent(mover)
		dst.Checkers, dst.Color = 0, domain.None
	}
	if dst.Color == opponent(mover) && dst.Checkers > 0 {
		// The point belongs to the other side: an illegal play is recorded as it was
		// made, so the checker lands there and the board says so.
		return b
	}
	dst.Checkers++
	dst.Color = mover
	return b
}

// diceCoherent reports whether every step can be charged to distinct dice of the
// roll. A step is one die, or several played through intermediate points the mover
// may land on in b ("24/14" on 6-4 is 24/18/14). A play that fails it was produced
// by correcting a roll under a kept play, and §1.4 requalifies it as illegal.
func diceCoherent(b domain.Board, steps []domain.CheckerStep, dice [2]int, mover int) bool {
	if len(steps) == 0 {
		return true
	}
	if dice[0] < 1 || dice[0] > 6 || dice[1] < 1 || dice[1] > 6 {
		return false
	}
	avail := []int{dice[0], dice[1]}
	if dice[0] == dice[1] {
		avail = []int{dice[0], dice[0], dice[0], dice[0]}
	}
	if len(steps) > len(avail) {
		return false
	}
	used := make([]bool, len(avail))
	dir := 1
	if mover == domain.Black {
		dir = -1
	}
	open := func(pt int) bool {
		c := b.Points[pt]
		return !(c.Color == opponent(mover) && c.Checkers >= 2)
	}
	// need is the pip distance from a point to bearing off.
	need := func(at int) int {
		if mover == domain.Black {
			return at
		}
		return 25 - at
	}
	onBoard := func(pt int) bool { return pt >= 1 && pt <= 24 }
	// chain covers the step from at to its target with unused dice, each non-final
	// die landing on a point the mover may use. A bar start is at the bar's index, so
	// the first die enters; a bear-off ends on the die that exactly or over-covers.
	var chain func(s domain.CheckerStep, at int, next func() bool) bool
	chain = func(s domain.CheckerStep, at int, next func() bool) bool {
		for j := range avail {
			if used[j] {
				continue
			}
			to := at + dir*avail[j]
			used[j] = true
			switch {
			case s.To == domain.Off:
				if avail[j] >= need(at) {
					if next() {
						return true
					}
				} else if open(to) && chain(s, to, next) {
					return true
				}
			case to == s.To:
				if next() {
					return true
				}
			case dir*(s.To-to) > 0 && onBoard(to) && open(to) && chain(s, to, next):
				return true
			}
			used[j] = false
		}
		return false
	}
	var assign func(k int) bool
	assign = func(k int) bool {
		if k == len(steps) {
			return true
		}
		s := steps[k]
		if (onBoard(s.From) || s.From == barOf(mover)) && (onBoard(s.To) || s.To == domain.Off) {
			return chain(s, s.From, func() bool { return assign(k + 1) })
		}
		return false
	}
	return assign(0)
}
