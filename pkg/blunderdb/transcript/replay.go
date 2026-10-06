package transcript

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// InconsistencyKind names one of the seven facts a Replay can find (fonctionnel.md §1.4).
// Every one of them is derived at each Replay, shown, and kept: none is stored on an
// Action, none is written into the saved Match, and none is ever a refusal.
type InconsistencyKind string

const (
	// IllegalMove: the board the Action left is reachable by no legal play from the
	// board before it. The Action's BoardAfter is then what really happened.
	IllegalMove InconsistencyKind = "illegal_move"
	// DoubleTurn: two consecutive Actions of the same side within one game. Take, pass
	// and resignation are out of the count — after a take the doubler rolls, which is
	// not a double turn — and a game's first play follows nobody.
	DoubleTurn InconsistencyKind = "double_turn"
	// ImpossibleCube: a double by a side that does not hold the cube, a double before
	// the game's first play, in the Crawford game or above the ceiling, an answer with
	// no offer, a cube action after the game's end.
	ImpossibleCube InconsistencyKind = "impossible_cube"
	// PastEnd: an Action recorded after the match was won — what shortening the match
	// length produces.
	PastEnd InconsistencyKind = "past_end"
	// InconsistentDice: a checker play whose steps do not use the Action's dice, which
	// is what correcting a roll under a kept play produces, or a game's first play
	// rolled as a double, which no opening roll can be.
	InconsistentDice InconsistencyKind = "inconsistent_dice"
	// UnrecordedMove: the roll is known, the play is not — a KindUnrecorded Action,
	// which a .mat writes "???". Nothing is wrong with the match; the RECORD is
	// incomplete, and everything after it stands on a board nobody can check.
	UnrecordedMove InconsistencyKind = "unrecorded_move"
	// ScoreMismatch: the score declared on a game's first Action is not the derived
	// one (ADR-0053); the game is played at the declared score and the detail names
	// the derived one. It also marks an unusable declaration (money, negative).
	ScoreMismatch InconsistencyKind = "score_mismatch"
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
	// saved Move carries. HasPosition is false for the one Action that produces
	// none: a resignation.
	Before      domain.Position `json:"before"`
	HasPosition bool            `json:"has_position"`

	// After is the board the Action left: the resulting board of the legal play whose
	// steps match, or the Action's own BoardAfter when no legal play reaches it.
	After domain.Board `json:"after"`

	// Notation is the play as a transcript writes it, "Cannot Move" for a dance and
	// "???" for a play the record does not carry.
	Notation string `json:"notation,omitempty"`

	GameIndex  int    `json:"game_index"`
	GameNumber int    `json:"game_number"`
	Score      [2]int `json:"score"`

	// MoveNumber is the index this Action takes among its game's Moves, or -1 when it
	// produces none (a resignation). It is how a caller lines an
	// ActionInfo's Before position up with the Move that MatchParts returns.
	MoveNumber int32 `json:"move_number"`

	// OpensGame says the Action is its game's first: the opening roll's winner plays
	// it, and it is the one that carries a declared score.
	OpensGame bool `json:"opens_game"`

	// DecisionMS and CubeDecisionMS are the Action's own, carried to the Move
	// it produces: the one thing here a Replay does not derive.
	DecisionMS     *int64 `json:"decision_ms,omitempty"`
	CubeDecisionMS *int64 `json:"cube_decision_ms,omitempty"`

	Inconsistencies []Inconsistency `json:"inconsistencies,omitempty"`
}

func (i *ActionInfo) add(kind InconsistencyKind, detail string) {
	i.Inconsistencies = append(i.Inconsistencies, Inconsistency{Kind: kind, Detail: detail})
}

// GameInfo is what a Replay derives about one game of the transcription.
type GameInfo struct {
	Number int `json:"number"`
	// InitialScore is the score the game was PLAYED at: the declared one when its
	// first Action carries one (ADR-0053), the one the previous games give otherwise.
	InitialScore [2]int `json:"initial_score"`
	// Declared says the first Action declared InitialScore; DerivedScore is what the
	// previous games give, equal to InitialScore when nothing was declared. The
	// two differ exactly where the first Action carries a ScoreMismatch.
	Declared     bool   `json:"declared"`
	DerivedScore [2]int `json:"derived_score"`
	// Winner is the encoding of domain.Game.Winner: 1 = player 1, -1 = player 2,
	// 0 = the game is unfinished.
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
	Expects Kind `json:"expects"`
	// GameStart says no game is running: the next Action is a game's first play,
	// whose side is the opening roll's winner ([Next.Side] is then only a default).
	GameStart  bool            `json:"game_start"`
	Side       int             `json:"side"`
	Position   domain.Position `json:"position"`
	GameNumber int             `json:"game_number"`
	Crawford   bool            `json:"crawford"`
	MatchOver  bool            `json:"match_over"`
}

// EntryInfo is the Action being TYPED, as the panel needs to draw it. The Entry
// itself is not serialisable (a crash is allowed to lose it, ADR-0045 rule 1),
// so it travels here rather than on the Document, and it is nil when nothing is
// being typed.
//
// The engine, not the panel (rule 9), says whether the play under a corrected roll
// was written down or preselected.
type EntryInfo struct {
	// At is the slot the entry lands on, and Replacing says whether validating
	// it overwrites the Action there (a correction in place) or inserts.
	At        int  `json:"at"`
	Replacing bool `json:"replacing"`

	Side     int    `json:"side"`
	Dice     [2]int `json:"dice"`
	Selected bool   `json:"selected"`
	// Review marks a play the user has to look at again — see [Entry].
	Review bool `json:"review"`

	// Kind is what validating the entry would write: always KindChecker, a cube
	// action or a resignation being a gesture of its own.
	Kind Kind `json:"kind"`
	// GameStart says the entry is a game's first play: its two dice are typed as
	// the opening roll, player 1's then player 2's, and the higher one names the
	// side ([GestureEnterDie]).
	GameStart bool `json:"game_start"`

	// Notation is the play picked so far, written as a Transcript writes it, and
	// "" while none is.
	//
	// It lets the Transcript DRAW the Action being typed where it will land;
	// nothing is written before validation.
	Notation string `json:"notation,omitempty"`
}

// Annotated is a document and everything a Replay derives from it.
type Annotated struct {
	Document Document     `json:"document"`
	Actions  []ActionInfo `json:"actions"`
	Games    []GameInfo   `json:"games"`
	Next     Next         `json:"next"`

	// Entry is the Action being typed, nil when none is.
	Entry *EntryInfo `json:"entry,omitempty"`

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
// It replays from the first Action every time; `from` selects only where the Cursor
// lands (the first Inconsistency at or after it). At ~340 µs per checker Action
// ([domain.LegalMoves]), a caller replaying the same document at every keystroke
// holds a [Replayer] instead.
func Replay(doc Document, from int) Annotated {
	var r Replayer
	return r.Replay(doc, from)
}

// Replayer is a [Replay] that remembers. It keeps, for the document it last replayed,
// the state before each Action and what that Action derived; the next call replays
// only from the first Action that changed.
//
// It returns exactly what [Replay] returns (TestReplayIncrementalMatchesFull). The
// cache is sound because an Action's derivation depends only on the state left by the
// one before it — any new field read by [state.step] must keep that true.
//
// A Replayer is NOT safe for concurrent use; it belongs to one editing session.
type Replayer struct {
	// doc is a deep copy of the document the cache describes, so that a caller editing
	// its Actions in place cannot pass the cache off as still valid.
	doc Document
	// states[i] is the state BEFORE Action i, so states has one entry more than infos.
	// It is empty until the first Replay.
	states []state
	infos  []ActionInfo
	// replayed is how many Actions the last call stepped through, the work measure
	// TestReplayIncrementalCostsOneAction holds (not a duration).
	replayed int
}

// Replay returns the annotation of doc, replaying only the Actions whose derivation
// the previous call cannot supply.
func (r *Replayer) Replay(doc Document, from int) Annotated {
	reuse := r.reusable(doc)
	if reuse == 0 {
		r.states = append(r.states[:0], *newState(doc.Header))
		r.infos = r.infos[:0]
	} else {
		r.states = r.states[:reuse+1]
		r.infos = r.infos[:reuse]
	}

	s := r.states[reuse].clone()
	r.replayed = len(doc.Actions) - reuse
	for i := reuse; i < len(doc.Actions); i++ {
		r.infos = append(r.infos, s.step(i, doc.Actions[i]))
		r.states = append(r.states, s.clone())
	}
	r.doc = doc.clone()

	out := Annotated{Document: doc, Winner: -1, Cursor: doc.Cursor}
	// The caller gets copies: the cache must survive whatever is done to what it
	// handed out, and an Annotated is passed around and serialised.
	out.Actions = append(make([]ActionInfo, 0, len(r.infos)), r.infos...)
	out.Games = append([]GameInfo(nil), s.games...)
	out.Score = s.points
	if s.matchOver() {
		out.Finished, out.Winner = true, s.winner()
	}
	out.Next = s.next(doc.NextScore)
	if e := doc.Entry; e != nil {
		out.Entry = &EntryInfo{
			At:        e.At,
			Replacing: e.Mode == EntryReplace,
			Side:      e.Side,
			Dice:      e.Dice,
			Selected:  e.Selected,
			Review:    e.Review,
			Kind:      KindChecker,
			GameStart: slotOpensGame(out, e.At, e.Mode == EntryReplace),
		}
		// states[i] is the state before the entry's slot: no replay needed.
		if at := clampSlot(e.At, len(doc.Actions)); len(e.Steps) > 0 {
			resolved, _ := resolveSteps(r.states[at].board, e.Side, e.Steps)
			out.Entry.Notation = domain.Notation(resolved, e.Side)
		}
	}
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

// slotOpensGame reports whether an Action written at slot `at` would be its game's
// first: the one already there when the entry replaces it, otherwise whether the
// game before the slot is over (or there is none).
func slotOpensGame(ann Annotated, at int, replacing bool) bool {
	n := len(ann.Actions)
	switch {
	case replacing && at >= 0 && at < n:
		return ann.Actions[at].OpensGame
	case at >= n:
		return ann.Next.GameStart
	case at <= 0:
		return true
	}
	gi := ann.Actions[at-1].GameIndex
	if gi < 0 || gi >= len(ann.Games) {
		return true
	}
	// A game closed by the next one's declared score, unfinished, has no winner:
	// the slot after it still belongs to it.
	g := ann.Games[gi]
	return g.Finished && g.Winner >= 0 && g.Last == at-1
}

// reusable returns how many leading Actions the cache still describes: the length of
// the common prefix, or 0 when the head changed or nothing is cached yet.
func (r *Replayer) reusable(doc Document) int {
	if len(r.states) == 0 || !sameReplayRules(r.doc.Header, doc.Header) {
		return 0
	}
	n := min(len(r.infos), len(doc.Actions))
	for i := 0; i < n; i++ {
		if !sameAction(r.doc.Actions[i], doc.Actions[i]) {
			return i
		}
	}
	return n
}

// sameReplayRules compares the header fields a Replay READS. The rest of it — the
// names, the event, the tournament, the Match the draft owns — is carried through
// untouched and derives nothing, so changing it must not throw the cache away.
func sameReplayRules(a, b Header) bool {
	return a.MatchLength == b.MatchLength && a.Jacoby == b.Jacoby &&
		a.Beaver == b.Beaver && a.MaxCube == b.MaxCube
}

// sameAction compares two Actions by value, following the three references an Action
// carries: its steps, the board an illegal play left, and a declared score.
func sameAction(a, b Action) bool {
	if a.Side != b.Side || a.Kind != b.Kind || a.Dice != b.Dice || a.Level != b.Level {
		return false
	}
	if (a.Score == nil) != (b.Score == nil) || (a.Score != nil && *a.Score != *b.Score) {
		return false
	}
	if len(a.Steps) != len(b.Steps) {
		return false
	}
	for i := range a.Steps {
		if a.Steps[i] != b.Steps[i] {
			return false
		}
	}
	switch {
	case a.BoardAfter == nil && b.BoardAfter == nil:
		return true
	case a.BoardAfter == nil || b.BoardAfter == nil:
		return false
	default:
		return *a.BoardAfter == *b.BoardAfter
	}
}
