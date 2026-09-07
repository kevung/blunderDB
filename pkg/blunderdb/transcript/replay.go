package transcript

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// InconsistencyKind names one of the five facts a Replay can find (fonctionnel.md §1.4).
// Every one of them is derived at each Replay, shown, and kept: none is stored on an
// Action, none is written into the saved Match, and none is ever a refusal.
type InconsistencyKind string

const (
	// IllegalMove: the board the Action left is reachable by no legal play from the
	// board before it. The Action's BoardAfter is then what really happened.
	IllegalMove InconsistencyKind = "illegal_move"
	// DoubleTurn: two consecutive Actions of the same side. Opening, take and pass are
	// out of the count — after a take the doubler rolls, which is not a double turn.
	DoubleTurn InconsistencyKind = "double_turn"
	// ImpossibleCube: a double by a side that does not hold the cube, a double in the
	// Crawford game or above the ceiling, an answer with no offer.
	ImpossibleCube InconsistencyKind = "impossible_cube"
	// PastEnd: an Action recorded after the match was won — what shortening the match
	// length produces.
	PastEnd InconsistencyKind = "past_end"
	// InconsistentDice: a checker play whose steps do not use the Action's dice, which
	// is what correcting a roll under a kept play produces. It requalifies the play as
	// illegal.
	InconsistentDice InconsistencyKind = "inconsistent_dice"
)

// Inconsistency is one derived fact about one Action, with the sentence the panel shows.
type Inconsistency struct {
	Kind   InconsistencyKind `json:"kind"`
	Detail string            `json:"detail"`
}

// ActionInfo is everything a Replay derives about one Action. None of it is stored on
// the Action, and none of it travels into the saved Match.
type ActionInfo struct {
	Index int  `json:"index"`
	Side  int  `json:"side"`
	Kind  Kind `json:"kind"`

	// Before is the Position the Action was played from — board, cube, dice, away
	// score with its Crawford sentinel, side on roll — and it is the Position the
	// saved Move carries. HasPosition is false for the Actions that produce none: an
	// opening and a resignation.
	Before      domain.Position `json:"before"`
	HasPosition bool            `json:"has_position"`

	// After is the board the Action left: the resulting board of the legal play whose
	// steps match, or the Action's own BoardAfter when no legal play reaches it.
	After domain.Board `json:"after"`

	// Notation is the play as a transcript writes it, "Cannot Move" for a dance.
	Notation string `json:"notation,omitempty"`

	GameIndex  int    `json:"game_index"`
	GameNumber int    `json:"game_number"`
	Score      [2]int `json:"score"`

	// MoveNumber is the index this Action takes among its game's Moves, or -1 when it
	// produces none (an opening, a resignation). It is how a caller lines an
	// ActionInfo's Before position up with the Move that MatchParts returns.
	MoveNumber int32 `json:"move_number"`

	Inconsistencies []Inconsistency `json:"inconsistencies,omitempty"`
}

func (i *ActionInfo) add(kind InconsistencyKind, detail string) {
	i.Inconsistencies = append(i.Inconsistencies, Inconsistency{Kind: kind, Detail: detail})
}

// GameInfo is what a Replay derives about one game of the transcription.
type GameInfo struct {
	Number       int    `json:"number"`
	InitialScore [2]int `json:"initial_score"`
	// Winner is the gnubg encoding the domain uses: 0 = player 1, 1 = player 2,
	// -1 = the game is unfinished.
	Winner    int  `json:"winner"`
	PointsWon int  `json:"points_won"`
	Crawford  bool `json:"crawford"`
	Finished  bool `json:"finished"`
	// First and Last bound the game's Actions in the document, -1 while empty.
	First int `json:"first"`
	Last  int `json:"last"`
}

// Next describes the Action the transcription is waiting for: what kind, from which
// side, and the position it would be played from. It is what the board and the
// candidate list show when the Cursor sits at the end of the document.
type Next struct {
	// Expects is the kind the panel offers first. An answer to a double is named
	// KindTake, which is one of its two forms: the other is a pass, and the gesture,
	// not the expectation, decides which.
	Expects    Kind            `json:"expects"`
	Side       int             `json:"side"`
	Position   domain.Position `json:"position"`
	GameNumber int             `json:"game_number"`
	Crawford   bool            `json:"crawford"`
	MatchOver  bool            `json:"match_over"`
}

// Annotated is a document and everything a Replay derives from it.
type Annotated struct {
	Document Document     `json:"document"`
	Actions  []ActionInfo `json:"actions"`
	Games    []GameInfo   `json:"games"`
	Next     Next         `json:"next"`

	// Finished and Winner describe the match: a score has reached the length.
	Finished bool   `json:"finished"`
	Winner   int    `json:"winner"`
	Score    [2]int `json:"score"`

	// Cursor is where the Replay leaves the Cursor: on the first Inconsistency it
	// found at or after the Action it replayed from, and otherwise where it was.
	Cursor int `json:"cursor"`
}

// Inconsistent reports whether any Action carries an Inconsistency — what the save
// dialog warns about before writing the Match anyway.
func (a Annotated) Inconsistent() bool {
	for i := range a.Actions {
		if len(a.Actions[i].Inconsistencies) > 0 {
			return true
		}
	}
	return false
}

// Replay plays the Actions again in order and derives everything of fonctionnel.md
// §1.3: the Position each Action was played from and the board it left, the number and
// score of each game, its Crawford mention, its winner and points, the end of the
// match — and the Inconsistencies of §1.4.
//
// It replays from the first Action whatever `from` says: a document of 300 Actions
// costs well under a millisecond, so an incremental replay would buy a complexity
// nobody can measure. What `from` selects is where the Cursor lands — on the first
// Inconsistency at or after it, which is what a correction wants to show next.
func Replay(doc Document, from int) Annotated {
	s := newState(doc.Header)
	out := Annotated{Document: doc, Winner: -1, Cursor: doc.Cursor}
	out.Actions = make([]ActionInfo, 0, len(doc.Actions))
	for i, a := range doc.Actions {
		out.Actions = append(out.Actions, s.step(i, a))
	}
	out.Games = s.games
	out.Score = s.points
	if s.matchOver() {
		out.Finished = true
		out.Winner = domain.Black
		if s.points[domain.White] > s.points[domain.Black] {
			out.Winner = domain.White
		}
	}
	out.Next = s.next()
	if from < 0 {
		from = 0
	}
	for i := from; i < len(out.Actions); i++ {
		if len(out.Actions[i].Inconsistencies) > 0 {
			out.Cursor = i
			break
		}
	}
	return out
}

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
	prevTie  bool
	hasPrev  bool
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

func (s *state) matchOver() bool {
	L := s.header.MatchLength
	return L > 0 && (s.points[domain.Black] >= L || s.points[domain.White] >= L)
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
// posted on it and are not part of its identity (ADR-0028): they are stated here
// because the panel and the evaluator read them, never because they distinguish a
// board from another.
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
// Jacoby is deliberately absent: fonctionnel.md §1.3 states this formula without it,
// and the rule stays what ADR-0028 made it, a flag posted on the Position that the
// evaluator applies. Nothing here reduces a gammon the user recorded.
func (s *state) gamePoints(winner int) int {
	loser := opponent(winner)
	base := 1
	if s.board.Bearoff[loser] == 0 {
		base = 2
		if s.trapped(loser, winner) {
			base = 3
		}
	}
	return base * cubeValue(s.cube)
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

// bearsTurn reports whether an Action counts for the double-turn rule. An opening is
// nobody's turn; a take or a pass answers the other side's offer and leaves the turn
// where it was, which is why the doubler playing right after a take is not a double
// turn (fonctionnel.md §1.4).
//
// A resignation is out of the count too, which §1.4's list does not say in so many
// words: it is given up at any moment, one's own roll included, so counting it would
// mark every resignation that follows its author's last play — a false positive on
// ordinary matches, and the Inconsistencies are worth exactly what their silence is.
func bearsTurn(k Kind) bool {
	switch k {
	case KindOpening, KindTake, KindPass, KindResign:
		return false
	}
	return true
}

// step replays one Action against the state and returns what it derives about it.
func (s *state) step(i int, a Action) ActionInfo {
	info := ActionInfo{Index: i, Side: a.Side, Kind: a.Kind, MoveNumber: -1, GameIndex: -1}

	if s.matchOver() {
		info.add(PastEnd, "the match is already won")
	}
	if s.hasPrev && bearsTurn(s.prevKind) && bearsTurn(a.Kind) && s.prevSide == a.Side {
		info.add(DoubleTurn, fmt.Sprintf("player %d acts twice in a row", a.Side+1))
	}

	tie := false
	switch a.Kind {
	case KindOpening:
		// A tie is followed by another opening in the SAME game; any other opening
		// while a game is running closes that game unfinished and starts the next.
		if s.gameActive && !s.prevTie {
			s.endGame(-1, 0)
		}
		s.ensureGame()
		// An opening produces neither Move nor Position; Before still describes the
		// board it is rolled from, because that is what the panel shows.
		info.Before = s.position(a.Side, a.Dice, domain.CheckerAction, s.cube)
		info.After = s.board
		tie = a.Dice[domain.Black] == a.Dice[domain.White]
		if !tie {
			s.turn = domain.Black
			if a.Dice[domain.White] > a.Dice[domain.Black] {
				s.turn = domain.White
			}
		}

	case KindChecker, KindDance:
		s.ensureGame()
		pos := s.position(a.Side, a.Dice, domain.CheckerAction, s.cube)
		info.Before, info.HasPosition = pos, true
		legal := domain.LegalMoves(&pos)

		if a.Kind == KindDance {
			info.After = s.board
			info.Notation = "Cannot Move"
			if len(legal) > 0 {
				info.add(IllegalMove, "the roll allows a play, so no dance is possible")
			}
		} else {
			if !diceCoherent(a.Steps, a.Dice, a.Side) {
				info.add(InconsistentDice, fmt.Sprintf("the play does not use the roll %d%d", a.Dice[0], a.Dice[1]))
			}
			resolved, reached := resolveSteps(s.board, a.Side, a.Steps)
			// The notation is of the steps as they were RECORDED, never of the legal
			// play that happens to reach the same board: one board has several
			// spellings ("16/10 10/7" and "13/7 16/13" move the same checkers to the
			// same points), and the transcript owes the reader the one that was
			// written down at the table.
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
		// The answerer decides on the cube AT THE LEVEL OFFERED (fonctionnel.md §1.2):
		// what they weigh is the doubled cube they are being handed.
		offered := domain.Cube{Owner: a.Side, Value: s.cube.Value + 1}
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
		s.endGame(opponent(a.Side), level*cubeValue(s.cube))

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

	s.prevKind, s.prevSide, s.prevTie, s.hasPrev = a.Kind, a.Side, tie, true
	return info
}

// next describes the Action the document is waiting for, on a projection of the state:
// when a game has just ended, the score and the Crawford mention of the game to come
// are already the ones the panel must show.
func (s *state) next() Next {
	proj := *s
	proj.games = append([]GameInfo(nil), s.games...)
	proj.ensureGame()

	n := Next{MatchOver: s.matchOver(), GameNumber: len(proj.games)}
	if len(proj.games) > 0 {
		n.Crawford = proj.games[len(proj.games)-1].Crawford
	}
	switch {
	case !s.gameActive || (s.hasPrev && s.prevTie):
		n.Expects, n.Side = KindOpening, domain.Black
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
		cube = domain.Cube{Owner: n.Side, Value: cube.Value + 1}
	}
	n.Position = proj.position(n.Side, [2]int{}, decision, cube)
	return n
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
// It is deliberately NOT domain's own applier: that one is fed by the legal-move
// generator and may assume its input, while a transcription's steps are whatever the
// user typed or a file wrote. So the steps are taken in an order that keeps them
// applicable — "bar/24 24/18" whichever way round it was written — hits are recomputed
// from the board rather than trusted from the step, and a step with no checker to move
// is played last and simply produces the board it produces, which the Replay then
// reports as illegal.
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

// diceCoherent reports whether every step can be charged to a distinct die of the
// roll. A play that fails it was produced by correcting a roll under a kept play, and
// §1.4 requalifies it as illegal.
func diceCoherent(steps []domain.CheckerStep, dice [2]int, mover int) bool {
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
	var assign func(k int) bool
	assign = func(k int) bool {
		if k == len(steps) {
			return true
		}
		for j := range avail {
			if used[j] || !domain.StepUsesDie(steps[k], mover, avail[j]) {
				continue
			}
			used[j] = true
			if assign(k + 1) {
				return true
			}
			used[j] = false
		}
		return false
	}
	return assign(0)
}
