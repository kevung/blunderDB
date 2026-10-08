package transcript

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// GestureKind is one of the acts of fonctionnel.md §2. Each is a pure function of the
// document: [Apply] returns the document the gesture makes, and changes nothing else.
type GestureKind string

const (
	// GestureCreate opens an empty draft. It reads the document it is given only for
	// its length, which a new draft inherits from the last one.
	GestureCreate GestureKind = "create"
	// GestureEnterDie fills the first free die of the Action being typed; once both
	// are full it starts the roll again. On a game's first play the two dice are the
	// opening roll, player 1's die then player 2's: the higher one gives the play to
	// its side, and a double names no one. Retyping the roll a first play already
	// has, in either order, leaves its camp and its order alone.
	GestureEnterDie GestureKind = "enter_die"
	// GestureClearDice empties both dice of an Action not yet validated.
	GestureClearDice GestureKind = "clear_dice"
	// GestureSelectCandidate picks one of [Candidates] as the play that was made.
	GestureSelectCandidate GestureKind = "select_candidate"
	// GestureEnterPlay records a play typed or made on the board rather than picked
	// from the candidates — the one path by which an illegal move enters.
	GestureEnterPlay GestureKind = "enter_play"
	// GestureValidate records the Action being typed at the Cursor.
	GestureValidate GestureKind = "validate"
	// GestureDance records a roll that allowed no play.
	GestureDance GestureKind = "dance"
	// GestureDouble, GestureTake, GesturePass and GestureResign record the cube and
	// the end of a game. Each validates a play left pending before it.
	GestureDouble GestureKind = "double"
	GestureTake   GestureKind = "take"
	GesturePass   GestureKind = "pass"
	GestureResign GestureKind = "resign"
	// GestureCursorBack and GestureCursorForward move the Cursor and load the Action
	// it lands on, whose dice and play the next entry corrects in place.
	GestureCursorBack    GestureKind = "cursor_back"
	GestureCursorForward GestureKind = "cursor_forward"
	// GestureCorrect loads the Action under the Cursor for correction in place.
	GestureCorrect GestureKind = "correct"
	// GestureInsertBefore and GestureInsertAfter expect a new Action beside the
	// Cursor, proposing the side that keeps the sequence coherent.
	GestureInsertBefore GestureKind = "insert_before"
	GestureInsertAfter  GestureKind = "insert_after"
	// GestureDelete removes the decision being edited (written or being typed) and
	// steps back to the previous one, loaded for correction (ADR-0050). Later
	// Actions keep their side: one local double turn (ADR-0045 rule 4).
	GestureDelete GestureKind = "delete"
	// GestureFlipSide gives the Action under the Cursor to the other camp.
	GestureFlipSide GestureKind = "flip_side"
	// GestureSetLength changes the match length (money included) and states the
	// session's rules. It is the only path to a length; a full Replay follows.
	GestureSetLength GestureKind = "set_length"
	// GestureSwapPlayers exchanges the two players: names, every side, and the board,
	// which is why every play's steps are mirrored with them.
	GestureSwapPlayers GestureKind = "swap_players"
	// GestureSetHeader writes the DESCRIPTIVE head (names, event, place, round,
	// date, transcriber, tournament). Length, rules ([GestureSetLength]) and match
	// id are kept: a form that omitted one would otherwise turn a match into money
	// play, or make a saved draft file a second Match.
	GestureSetHeader GestureKind = "set_header"
	// GestureSetScore declares (or, with a nil Gesture.Score, clears) the score of
	// the game whose first Action is Gesture.At (ADR-0053).
	GestureSetScore GestureKind = "set_score"
	// GestureSetVideo attaches the media the Repères refer to
	// (Gesture.VideoSource); an empty source detaches it (ADR-0079 rule 4).
	GestureSetVideo GestureKind = "set_video"
	// GestureSetTimecode posts Repères on the Action under the Cursor: the
	// instant of the action (Gesture.TickMS, when HasTick) and the instant of
	// the roll (Gesture.RollTickMS, when HasRollTick), a negative value
	// clearing it. It is the explicit gesture of ADR-0079 rule 3.
	GestureSetTimecode GestureKind = "set_timecode"
	// GestureUndo and GestureRedo are named here for one spelling, but [Apply]
	// refuses them: the stack lives in [Editor], and the session layer routes them
	// to [Editor.Undo] and [Editor.Redo].
	GestureUndo GestureKind = "undo"
	GestureRedo GestureKind = "redo"
)

// Gesture is a gesture and what it needs. Only the fields its Kind names are read.
type Gesture struct {
	Kind GestureKind

	// Die is the pip entered by GestureEnterDie.
	Die int
	// Candidate indexes [Candidates] for GestureSelectCandidate.
	Candidate int
	// Steps and BoardAfter carry a play entered by hand (GestureEnterPlay).
	// BoardAfter is kept only when no legal play reaches it.
	Steps      []domain.CheckerStep
	BoardAfter *domain.Board
	// Side names the camp of a cube action or a resignation; HasSide distinguishes
	// "player 1" from "not stated", since the zero value is a real side.
	Side    int
	HasSide bool
	// Level is a resignation's value in games (1, 2 or 3).
	Level int
	// MatchLength is the length GestureCreate and GestureSetLength set. HasLength
	// distinguishes a money session (0) from an unstated length.
	MatchLength int
	HasLength   bool
	// Jacoby and Beaver are the session's rules, read only when HasRules says they
	// were stated: a money draft is Jacoby by default, and a zero value must not
	// silently turn that default off.
	HasRules bool
	Jacoby   bool
	Beaver   bool
	// Header is what GestureSetHeader writes.
	Header Header
	// At is the index of the Action a gesture about one written Action names —
	// the game's first Action for GestureSetScore.
	At int
	// Score is the score GestureSetScore declares, points of player 1 then of
	// player 2; nil clears the declaration.
	Score *[2]int

	// TickMS is the instant of the video, in milliseconds, the gesture was made
	// at, read only when HasTick says a video is playing: 0 is a real instant.
	// GestureEnterDie posts it as the roll's Repère, GestureEnterPlay,
	// GestureValidate, GestureDance and the cube gestures as the action's —
	// on a NEW Action only, a correction in place keeping the Repères it had
	// (ADR-0079 rule 3). GestureSetTimecode writes it, and RollTickMS when
	// HasRollTick, on the Action under the Cursor, a negative value clearing.
	TickMS      int64
	HasTick     bool
	RollTickMS  int64
	HasRollTick bool
	// VideoSource is the media GestureSetVideo attaches, "" to detach.
	VideoSource string
}

var (
	// ErrNoEntry reports a gesture that needs an Action being typed and found none.
	ErrNoEntry = errors.New("transcript: no action is being entered")
	// ErrNoDice reports a play gesture made before the roll was entered.
	ErrNoDice = errors.New("transcript: the roll is not entered")
	// ErrNoCandidate reports a validation with no play selected.
	ErrNoCandidate = errors.New("transcript: no play is selected")
	// ErrNoAction reports a gesture on the Action under the Cursor when there is none.
	ErrNoAction = errors.New("transcript: the cursor is not on an action")
	// ErrNotPure reports undo or redo handed to [Apply]. They are gestures of the
	// SESSION, not of the document: only an [Editor] holds the stack they walk.
	ErrNotPure = errors.New("transcript: undo and redo belong to the editor, not to Apply")
)

// Apply returns the document that gesture g makes of doc. It is pure: doc is never
// modified, and the same pair always gives the same result. Undo and redo are not
// here — a stack is state, and it lives in [Editor].
func Apply(doc Document, g Gesture) (Document, error) {
	out, err := apply(doc, g)
	if err == nil && movesOnly(g.Kind) && !out.HasTouched {
		// A gesture that wrote nothing leaves the Cursor where it put it:
		// jumping to the first Inconsistency would make every one a wall
		// (ADR-0054).
		out.HoldCursor = true
	}
	return out, err
}

// movesOnly reports whether a gesture moves the Cursor or edits the entry, and
// writes an Action only through the correction a step commits.
func movesOnly(k GestureKind) bool {
	switch k {
	case GestureEnterDie, GestureClearDice, GestureSelectCandidate, GestureEnterPlay,
		GestureCursorBack, GestureCursorForward, GestureCorrect,
		GestureInsertBefore, GestureInsertAfter:
		return true
	}
	return false
}

func apply(doc Document, g Gesture) (Document, error) {
	out := doc.clone()
	// Touched describes the LAST gesture and nothing else: it is cleared here and
	// set again only by the paths that write an Action, so that walking the
	// Cursor never leaves a stale index behind for the next Replay to start from.
	out.HasTouched, out.HoldCursor = false, false
	switch g.Kind {
	case GestureCreate:
		// A new draft inherits the length of the one it follows, and 7 when there is
		// none (fonctionnel.md §1.1).
		length := DefaultMatchLength
		if doc.FormatVersion != 0 {
			length = doc.Header.MatchLength
		}
		if g.HasLength {
			length = g.MatchLength
		}
		fresh := New(length)
		if length == 0 && g.HasRules {
			fresh.Header.Jacoby, fresh.Header.Beaver = g.Jacoby, g.Beaver
		}
		return fresh, nil

	case GestureEnterDie:
		return enterDie(doc, out, g)

	case GestureClearDice:
		if out.Entry == nil {
			return doc, ErrNoEntry
		}
		out.Entry.Dice = [2]int{}
		out.Entry.Steps, out.Entry.Selected, out.Entry.Review = nil, false, false
		return out, nil

	case GestureSelectCandidate:
		ensureEntry(&out)
		cands := Candidates(out)
		if len(cands) == 0 {
			return doc, ErrNoDice
		}
		if g.Candidate < 0 || g.Candidate >= len(cands) {
			return doc, fmt.Errorf("transcript: no candidate %d among %d", g.Candidate, len(cands))
		}
		out.Entry.Steps = append([]domain.CheckerStep(nil), cands[g.Candidate].Steps...)
		out.Entry.Selected = true
		// Review is NOT cleared here but at validation (fonctionnel.md §2): the
		// panel preselects a candidate itself, which would clear it unseen.
		return out, nil

	case GestureEnterPlay:
		e := ensureEntry(&out)
		e.Steps = append([]domain.CheckerStep(nil), g.Steps...)
		e.Selected = true
		if g.HasTick {
			e.TickMS = tickOf(g.TickMS)
		}
		// A board no legal play reaches is what really happened, and it is kept as
		// given; validate() drops it again if the play turns out to be legal.
		out.pendingBoard = nil
		if g.BoardAfter != nil {
			b := *g.BoardAfter
			out.pendingBoard = &b
		}
		return out, nil

	case GestureValidate:
		if g.HasTick && out.Entry != nil {
			out.Entry.TickMS = tickOf(g.TickMS)
		}
		return validate(out)

	case GestureDance:
		e := ensureEntry(&out)
		if e.Dice[0] == 0 || e.Dice[1] == 0 {
			return doc, ErrNoDice
		}
		if g.HasTick {
			e.TickMS = tickOf(g.TickMS)
		}
		return record(out, Action{Side: e.Side, Kind: KindDance, Dice: e.Dice,
			RollTickMS: e.RollTickMS, TickMS: e.TickMS}), nil

	case GestureDouble, GestureTake, GesturePass, GestureResign:
		return cubeGesture(out, g)

	case GestureCursorBack:
		return cursorBack(commitCorrection(out)), nil

	case GestureCursorForward:
		return cursorForward(commitCorrection(out)), nil

	case GestureCorrect:
		if out.Cursor < 0 || out.Cursor >= len(out.Actions) {
			return doc, ErrNoAction
		}
		loadEntry(&out)
		return out, nil

	case GestureInsertBefore, GestureInsertAfter:
		at := out.Cursor
		if g.Kind == GestureInsertAfter && at < len(out.Actions) {
			// "After the last Action" and "end of document" are one slot: not
			// moving on again keeps `a` from erroring at the end.
			at++
		}
		if at < 0 {
			at = 0
		}
		if at > len(out.Actions) {
			at = len(out.Actions)
		}
		side := proposedSide(out, at)
		if g.HasSide {
			side = g.Side
		}
		out.Cursor = at
		out.Entry = &Entry{Side: side, Mode: EntryNew, At: at}
		return out, nil

	case GestureDelete:
		return deleteDecision(doc, out)

	case GestureFlipSide:
		if out.Cursor < 0 || out.Cursor >= len(out.Actions) {
			return doc, ErrNoAction
		}
		a := &out.Actions[out.Cursor]
		// A game's first play holds the opening roll in player order: giving it to
		// the other camp gives that camp the higher die. A roll written against its
		// camp is left as it is, so that the flip is what repairs it.
		if openingWinner(a.Dice) == a.Side && Replay(out, 0).Actions[out.Cursor].OpensGame {
			a.Dice[0], a.Dice[1] = a.Dice[1], a.Dice[0]
		}
		a.Side = opponent(a.Side)
		out.Touched, out.HasTouched = out.Cursor, true
		out.Entry = nil
		return out, nil

	case GestureSetLength:
		if !g.HasLength {
			return doc, fmt.Errorf("transcript: no match length given")
		}
		out.Header.MatchLength = g.MatchLength
		// The session's rules SURVIVE a crossing to match play (no Position
		// carries them there), so money → match → money keeps them. A document
		// reaching money without ever stating them is Jacoby (fonctionnel.md §1.1).
		switch {
		case g.HasRules:
			out.Header.Jacoby, out.Header.Beaver = g.Jacoby, g.Beaver
		case g.MatchLength == 0 && !out.Header.Jacoby && !out.Header.Beaver:
			out.Header.Jacoby = true
		}
		out.Entry = nil
		return out, nil

	case GestureSwapPlayers:
		out.Header.Player1, out.Header.Player2 = out.Header.Player2, out.Header.Player1
		ann := Replay(doc, 0)
		for i := range out.Actions {
			a := &out.Actions[i]
			a.Side = opponent(a.Side)
			// The dice are player 1's then player 2's on a game's first play only;
			// any other roll keeps its cell as written.
			if ann.Actions[i].OpensGame {
				a.Dice[0], a.Dice[1] = a.Dice[1], a.Dice[0]
			}
			if a.Score != nil {
				a.Score[0], a.Score[1] = a.Score[1], a.Score[0]
			}
			for j := range a.Steps {
				a.Steps[j].From = mirrorIndex(a.Steps[j].From)
				a.Steps[j].To = mirrorIndex(a.Steps[j].To)
			}
			if a.BoardAfter != nil {
				b := mirrorBoard(*a.BoardAfter)
				a.BoardAfter = &b
			}
		}
		if out.NextScore != nil {
			out.NextScore[0], out.NextScore[1] = out.NextScore[1], out.NextScore[0]
		}
		out.Entry = nil
		return out, nil

	case GestureSetHeader:
		out.Header = mergeHeader(out.Header, g.Header)
		return out, nil

	case GestureSetScore:
		return setScore(doc, out, g)

	case GestureSetVideo:
		out.Header.VideoSource = strings.TrimSpace(g.VideoSource)
		out.HoldCursor = true
		return out, nil

	case GestureSetTimecode:
		return setTimecode(doc, out, g)

	case GestureUndo, GestureRedo:
		return doc, ErrNotPure
	}
	return doc, fmt.Errorf("transcript: unknown gesture %q", g.Kind)
}

// setScore is GestureSetScore. It refuses only the meaningless (not a game's first
// Action, money play, negative); a score past the length is declared and marked
// (ADR-0044). It moves neither the Cursor nor the entry, and holds the Cursor.
//
// At the end slot (At = len(Actions)) it declares the score of the game the next
// Action appended opens, which is where a boundary waiting there is corrected or
// cleared ([Document.NextScore]).
func setScore(doc, out Document, g Gesture) (Document, error) {
	at := g.At
	end := at == len(out.Actions)
	if at < 0 || at > len(out.Actions) || !end && !Replay(out, 0).Actions[at].OpensGame {
		return doc, fmt.Errorf("transcript: action %d is not the first of a game", at)
	}
	if g.Score == nil {
		if end {
			out.NextScore = nil
		} else {
			out.Actions[at].Score = nil
		}
		out.HoldCursor = true
		return out, nil
	}
	if out.Header.MatchLength <= 0 {
		return doc, errors.New("transcript: a money session has no score")
	}
	if g.Score[0] < 0 || g.Score[1] < 0 {
		return doc, fmt.Errorf("transcript: %d-%d is not a score", g.Score[0], g.Score[1])
	}
	sc := *g.Score
	if end {
		out.NextScore = &sc
	} else {
		out.Actions[at].Score = &sc
	}
	out.HoldCursor = true
	return out, nil
}

// enterDie is GestureEnterDie: one die of the roll being typed. doc is the
// document as it was, returned with an error; out is its working copy.
func enterDie(doc, out Document, g Gesture) (Document, error) {
	if g.Die < 1 || g.Die > 6 {
		return doc, fmt.Errorf("transcript: %d is not a die", g.Die)
	}
	e := ensureEntry(&out)
	// The roll's Repère is the FIRST die typed: retyping or correcting a
	// die is not the dice falling again.
	if g.HasTick && e.RollTickMS == nil {
		e.RollTickMS = tickOf(g.TickMS)
	}
	switch {
	case e.Dice[0] == 0:
		e.Dice[0] = g.Die
	case e.Dice[1] == 0:
		e.Dice[1] = g.Die
	default:
		e.Dice = [2]int{g.Die, 0}
	}
	e.Steps, e.Selected, e.Review = nil, false, false
	if d := e.Dice; d[1] != 0 && slotOpensGame(Replay(out, 0), e.At, e.Mode == EntryReplace) {
		// The roll the play already has is kept as written — unless it is written
		// against its camp, which retyping it in player order repairs.
		stored := Action{Side: -1}
		if e.Mode == EntryReplace && e.At < len(out.Actions) {
			stored = out.Actions[e.At]
		}
		if sameRoll(d, stored.Dice) && openingWinner(stored.Dice) == stored.Side {
			e.Dice, e.Side = stored.Dice, stored.Side
		} else if w := openingWinner(d); w >= 0 {
			e.Side = w
		}
	}
	reroll(&out)
	return out, nil
}

// setTimecode is GestureSetTimecode: it writes the Repères the gesture states on
// the Action under the Cursor and nothing else. A Repère out of order is kept and
// marked by the Replay (TimecodeBackwards), never refused (ADR-0079 rule 2).
func setTimecode(doc, out Document, g Gesture) (Document, error) {
	at := out.Cursor
	if at < 0 || at >= len(out.Actions) {
		return doc, ErrNoAction
	}
	a := &out.Actions[at]
	if g.HasRollTick {
		if !rolls(a.Kind) {
			return doc, fmt.Errorf("transcript: a %s has no roll to time", a.Kind)
		}
		a.RollTickMS = tickOf(g.RollTickMS)
	}
	if g.HasTick {
		a.TickMS = tickOf(g.TickMS)
	}
	out.HoldCursor = true
	return out, nil
}

// tickOf is the Repère a gesture states: the instant itself, or nil — cleared —
// for a negative one.
func tickOf(ms int64) *int64 {
	if ms < 0 {
		return nil
	}
	return &ms
}

// rolls reports whether an Action of kind k starts with a roll, and so carries
// the roll's Repère besides the action's.
func rolls(k Kind) bool {
	return k == KindChecker || k == KindDance || k == KindUnrecorded
}

// sameRoll reports whether two rolls are the same two dice, in either order.
func sameRoll(a, b [2]int) bool {
	return a == b || a == [2]int{b[1], b[0]}
}

// mergeHeader writes the descriptive fields of `in` over `cur` and keeps length,
// rules and match id (see GestureSetHeader; fonctionnel.md §1.1).
func mergeHeader(cur, in Header) Header {
	out := cur
	out.Player1, out.Player2 = in.Player1, in.Player2
	out.Event, out.Location, out.Round = in.Event, in.Location, in.Round
	out.Date, out.Transcriber = in.Date, in.Transcriber
	out.TournamentID = in.TournamentID
	// A form that does not name the video keeps it: detaching is
	// GestureSetVideo's, so a header form unaware of the video cannot drop it.
	if v := strings.TrimSpace(in.VideoSource); v != "" {
		out.VideoSource = v
	}
	return out
}

// ensureEntry returns the Action being typed, opening one at the Cursor if none is.
func ensureEntry(doc *Document) *Entry {
	if doc.Entry == nil {
		e := proposedEntry(*doc)
		doc.Entry = &e
	}
	return doc.Entry
}

// proposedEntry is the Action the document expects where the Cursor stands: the one
// already recorded there, which a new entry corrects in place, or a new one at the end
// of the document.
func proposedEntry(doc Document) Entry {
	at := doc.Cursor
	if at < 0 {
		at = 0
	}
	if at > len(doc.Actions) {
		at = len(doc.Actions)
	}
	if at < len(doc.Actions) {
		a := doc.Actions[at]
		return Entry{
			Side:     a.Side,
			Dice:     a.Dice,
			Steps:    append([]domain.CheckerStep(nil), a.Steps...),
			Selected: len(a.Steps) > 0,
			Mode:     EntryReplace,
			At:       at,
		}
	}
	return Entry{At: at, Mode: EntryNew, Side: Replay(doc, 0).Next.Side}
}

// deleteDecision is GestureDelete: it removes the decision being edited and steps
// back to the one before (ADR-0050). doc is the document as it was, returned
// unchanged with the error; out is its working copy.
func deleteDecision(doc, out Document) (Document, error) {
	// An unwritten decision (open insertion, continued-game slot, roll typed at
	// the end) is abandoned; nothing written is touched.
	if e := out.Entry; e != nil && e.Mode == EntryNew {
		out.Cursor = clampSlot(e.At, len(out.Actions))
		out.Entry, out.pendingBoard, out.HasReturn = nil, nil, false
		stepBack(&out)
		return out, nil
	}
	if out.Cursor >= len(out.Actions) {
		// End of document, nothing typed: step back onto the last Action so
		// that Del held down walks back through what was typed.
		if len(out.Actions) == 0 {
			return doc, ErrNoAction
		}
		out.Cursor = len(out.Actions)
		out.Entry, out.pendingBoard, out.HasReturn = nil, nil, false
		stepBack(&out)
		return out, nil
	}
	if out.Cursor < 0 {
		return doc, ErrNoAction
	}
	at := out.Cursor
	// A game's first Action that carried its declared score hands it to the Action
	// after it, which now opens the game: deleting a play is not undeclaring a score.
	if gone := out.Actions[at]; gone.Score != nil && at+1 < len(out.Actions) && out.Actions[at+1].Score == nil {
		if ann := Replay(out, 0); ann.Actions[at+1].GameIndex == ann.Actions[at].GameIndex {
			out.Actions[at+1].Score = gone.Score
		}
	}
	out.Actions = append(out.Actions[:at], out.Actions[at+1:]...)
	out.Touched, out.HasTouched = at, true
	out.Entry, out.pendingBoard, out.HasReturn = nil, nil, false
	stepBack(&out)
	return out, nil
}

// stepBack puts the Cursor on the decision before the deleted one and loads it for
// correction (ADR-0050); on the first slot it stays on what follows. HoldCursor
// keeps the Replay from pulling it onto the double turn ahead.
func stepBack(doc *Document) {
	if doc.Cursor > 0 {
		doc.Cursor--
	}
	loadEntry(doc)
	doc.HoldCursor = true
}

// cursorBack is GestureCursorBack once the correction it walks away from is
// committed. The hole of a double turn is a stop of its own, between the two
// Actions it separates (ADR-0054).
func cursorBack(out Document) Document {
	if out.Cursor > 0 {
		if !out.HasReturn {
			out.Return, out.HasReturn = out.Cursor, true
		}
		if !onHole(out) && holeBefore(out, out.Cursor) {
			openHole(&out, out.Cursor)
			return out
		}
		out.Cursor--
	}
	loadEntry(&out)
	return out
}

// cursorForward is GestureCursorForward once the correction it walks away from
// is committed. Past a hole is the Action it stands before, at the same index.
func cursorForward(out Document) Document {
	if !onHole(out) && out.Cursor < len(out.Actions) {
		out.Cursor++
		if holeBefore(out, out.Cursor) {
			openHole(&out, out.Cursor)
			return out
		}
	}
	loadEntry(&out)
	return out
}

// holeBefore reports whether the Action at `at` and the one before it are turns of
// the same side — the Replay's DoubleTurn, read on two Actions — leaving the other
// side's turn missing (ADR-0054).
func holeBefore(doc Document, at int) bool {
	if at <= 0 || at >= len(doc.Actions) {
		return false
	}
	prev, a := doc.Actions[at-1], doc.Actions[at]
	return bearsTurn(prev.Kind) && bearsTurn(a.Kind) && prev.Side == a.Side
}

// onHole reports whether the Cursor stands on the hole in front of the Action it
// names: an insertion open at the Cursor, where a double turn leaves one.
func onHole(doc Document) bool {
	e := doc.Entry
	return e != nil && e.Mode == EntryNew && e.At == doc.Cursor && holeBefore(doc, doc.Cursor)
}

// openHole puts the Cursor on the hole in front of the Action at `at` and opens
// the insertion that fills it, for the side whose turn is missing. Nothing is
// written: walking off the hole abandons the slot, as it does any insertion.
func openHole(doc *Document, at int) {
	doc.Cursor = at
	doc.Entry = &Entry{Side: proposedSide(*doc, at), Mode: EntryNew, At: at}
	doc.pendingBoard = nil
}

// gameEndsAt reports whether the Action at `at` closes its game or the match: a
// Replay of the prefix expects a new game next. It costs a prefix Replay, so it
// runs only on writes INSIDE the document.
func gameEndsAt(doc Document, at int) bool {
	if at < 0 || at >= len(doc.Actions) {
		return false
	}
	prefix := Document{FormatVersion: doc.FormatVersion, Header: doc.Header, Actions: doc.Actions[:at+1]}
	next := Replay(prefix, 0).Next
	return next.MatchOver || next.GameStart
}

// continueGame opens the slot right after `at` when the Action written there leaves
// its game running although the next Action started the next game before the
// correction (a pass turned take, ADR-0050): rolls are INSERTED until the game ends,
// rather than overwriting that game's first play. `nextScore` is the score that
// game started at; it is declared on its first play, so the game being continued
// cannot swallow it.
func continueGame(doc *Document, at int, nextScore *[2]int) bool {
	if nextScore == nil || at+1 >= len(doc.Actions) || gameEndsAt(*doc, at) {
		return false
	}
	if doc.Actions[at+1].Score == nil {
		doc.Actions[at+1].Score = nextScore
	}
	doc.Cursor, doc.HasReturn = at+1, false
	doc.Entry, doc.pendingBoard = &Entry{Side: proposedSide(*doc, at+1), Mode: EntryNew, At: at + 1}, nil
	doc.HoldCursor = true
	return true
}

// loadEntry puts the Action under the Cursor into the entry, so the dice and the play
// the panel shows are the ones recorded there, and the next entry corrects them.
func loadEntry(doc *Document) {
	if doc.Cursor < 0 || doc.Cursor >= len(doc.Actions) {
		doc.Entry = nil
		return
	}
	e := proposedEntry(*doc)
	doc.Entry = &e
}

// commitCorrection writes a pending correction back onto the Action it corrects,
// and leaves the Cursor exactly where it was.
//
// Walking away from a correction keeps it, or a user who walked on would silently
// lose the play they picked (ux.md §4.3 budgets). It takes no return jump (`l` is a
// step, not a validation), and does nothing when the entry equals the Action, so
// reading the Transcript rewrites nothing and pushes no undo entry.
func commitCorrection(doc Document) Document {
	e := doc.Entry
	if e == nil || e.Mode != EntryReplace || e.At < 0 || e.At >= len(doc.Actions) {
		return doc
	}
	if !entryDiffers(doc.Actions[e.At], *e) {
		return doc
	}
	cursor, ret, hasReturn := doc.Cursor, doc.Return, doc.HasReturn
	doc.Cursor, doc.HasReturn = e.At, false
	out, err := validate(doc)
	if err != nil {
		// An unready correction is left as is; the Action keeps what it had.
		doc.Cursor, doc.Return, doc.HasReturn = cursor, ret, hasReturn
		return doc
	}
	out.Cursor, out.Return, out.HasReturn = cursor, ret, hasReturn
	return out
}

// entryDiffers reports whether the entry says anything the Action does not.
func entryDiffers(a Action, e Entry) bool {
	if e.Dice != a.Dice || len(e.Steps) != len(a.Steps) {
		return true
	}
	for i := range e.Steps {
		if e.Steps[i] != a.Steps[i] {
			return true
		}
	}
	return false
}

// reroll decides what play stands under a roll that is being corrected in place
// (fonctionnel.md §2, "corriger un jet").
//
// The recorded play is kept if still legal for the new roll; otherwise the roll's
// first candidate is preselected and the entry marked for review. Nothing is
// written until validation, so the document on disk is always one the user
// validated.
func reroll(doc *Document) {
	e := doc.Entry
	if e == nil || e.Mode != EntryReplace || e.Dice[0] == 0 || e.Dice[1] == 0 {
		return
	}
	if e.At < 0 || e.At >= len(doc.Actions) {
		return
	}
	played := doc.Actions[e.At]
	if played.Kind != KindChecker || len(played.Steps) == 0 {
		return
	}

	pos := entryPosition(*doc)
	pos.Dice, pos.PlayerOnRoll, pos.DecisionType = e.Dice, e.Side, domain.CheckerAction
	legal := domain.LegalMoves(&pos)
	if len(legal) == 0 {
		// The new roll allows nothing: a dance, which the panel records.
		return
	}
	// Both tests are needed: the board says where the play LANDS, diceCoherent
	// that its steps fit the dice (§1.4 "dés incohérents": a 6-1 played as
	// 8/2 6/5 lands where a 6-1 lands, yet is no 5-4).
	steps := played.Steps
	if _, reached := resolveSteps(pos.Board, e.Side, steps); findPlay(legal, reached) != nil &&
		diceCoherent(pos.Board, steps, e.Dice, e.Side) {
		e.Steps = append([]domain.CheckerStep(nil), steps...)
		e.Selected = true
		return
	}
	e.Steps = append([]domain.CheckerStep(nil), legal[0].Steps...)
	e.Selected, e.Review = true, true
}

// proposedSide is the side that keeps the sequence coherent at index at: the camp
// facing the neighbouring Action. It is a PROPOSAL — once recorded, the side belongs
// to the Action and only "change side" moves it.
func proposedSide(doc Document, at int) int {
	if at > 0 && at-1 < len(doc.Actions) {
		prev := doc.Actions[at-1]
		if bearsTurn(prev.Kind) {
			return opponent(prev.Side)
		}
		if prev.Kind == KindTake {
			// After a take the doubler rolls again.
			return opponent(prev.Side)
		}
	}
	if at < len(doc.Actions) {
		return doc.Actions[at].Side
	}
	return Replay(doc, 0).Next.Side
}

// validate records the Action being typed, at the place the entry names.
func validate(doc Document) (Document, error) {
	e := doc.Entry
	if e == nil {
		return doc, ErrNoEntry
	}
	if e.Dice[0] == 0 || e.Dice[1] == 0 {
		return doc, ErrNoDice
	}
	// Confirming a cell that says exactly what it already holds is a READ, not
	// a correction, and must write nothing — the same rule commitCorrection
	// applies to walking away, here for Enter pressed on a cell nobody
	// changed. Without it, Enter sent the Cursor to Return and cleared the
	// Entry, so the NEXT keystroke reopened whatever Action the Cursor landed
	// on instead of writing a new one, however far from the correction that
	// landing was. The last Action is the one exception (ADR-0051): its
	// validation always reaches the end, changed or not.
	if e.Mode == EntryReplace && e.At < len(doc.Actions) && e.At+1 != len(doc.Actions) && !entryDiffers(doc.Actions[e.At], *e) {
		return doc, nil
	}
	if !e.Selected {
		return doc, ErrNoCandidate
	}
	a := Action{Side: e.Side, Kind: KindChecker, Dice: e.Dice, Steps: e.Steps,
		RollTickMS: e.RollTickMS, TickMS: e.TickMS}
	if doc.pendingBoard != nil {
		pos := entryPosition(doc)
		pos.Dice, pos.PlayerOnRoll, pos.DecisionType = e.Dice, e.Side, domain.CheckerAction
		if findPlay(domain.LegalMoves(&pos), *doc.pendingBoard) == nil {
			b := *doc.pendingBoard
			a.BoardAfter = &b
		}
	}
	return record(doc, a), nil
}

// clampSlot holds an index inside [0, n] — the slots of a document of n Actions,
// the last one being the end.
func clampSlot(at, n int) int {
	if at < 0 {
		return 0
	}
	if at > n {
		return n
	}
	return at
}

// record writes the Action at the place the entry names and moves the Cursor on. A
// correction sends the Cursor back where it stood before the user walked back to it.
func record(doc Document, a Action) Document {
	e := doc.Entry
	at := doc.Cursor
	mode := EntryNew
	if e != nil {
		at, mode = e.At, e.Mode
	}
	if at < 0 {
		at = 0
	}
	if at > len(doc.Actions) {
		at = len(doc.Actions)
	}
	// Only an EDIT names the Action it touched, so the Replay can land on the
	// Inconsistency it made behind the Cursor. An append does not (see
	// Document.Touched).
	if mode == EntryReplace || at < len(doc.Actions) {
		doc.Touched, doc.HasTouched = at, true
	}
	if mode == EntryReplace && at < len(doc.Actions) {
		// A corrected Action keeps the score declared on it: the entry retypes the
		// roll and the play, and they are all it knows about.
		if a.Score == nil {
			a.Score = doc.Actions[at].Score
		}
		// …and its Repères: a correction retypes what was played, not when
		// (ADR-0079 rule 3).
		// A correction into a kind without a roll has no roll to time.
		a.RollTickMS, a.TickMS = nil, doc.Actions[at].TickMS
		if rolls(a.Kind) {
			a.RollTickMS = doc.Actions[at].RollTickMS
		}
		var nextScore *[2]int
		if at+1 < len(doc.Actions) && gameEndsAt(doc, at) {
			ann := Replay(doc, 0)
			if gi := ann.Actions[at+1].GameIndex; gi >= 0 {
				sc := ann.Games[gi].InitialScore
				nextScore = &sc
			}
		}
		doc.Actions[at] = a
		if continueGame(&doc, at, nextScore) {
			return doc
		}
		doc.Cursor = at + 1
		if at+1 == len(doc.Actions) {
			// The LAST Action re-edited: the next roll is appended (ADR-0051).
			// The Cursor goes to the end whatever Return says, and is held.
			doc.HasReturn, doc.HoldCursor = false, true
		} else if doc.HasReturn {
			doc.Cursor, doc.HasReturn = doc.Return, false
		}
	} else {
		// Inserted in front of a game's first Action, the new one opens the game in
		// its place and takes over the score declared there.
		if at < len(doc.Actions) && doc.Actions[at].Score != nil && a.Score == nil &&
			slotOpensGame(Replay(doc, 0), at, false) {
			a.Score, doc.Actions[at].Score = doc.Actions[at].Score, nil
		}
		// A boundary waiting at the end goes to the Action that opens the game.
		if at == len(doc.Actions) && doc.NextScore != nil {
			if a.Score == nil {
				a.Score = doc.NextScore
			}
			doc.NextScore = nil
		}
		doc.Actions = append(doc.Actions, Action{})
		copy(doc.Actions[at+1:], doc.Actions[at:])
		doc.Actions[at] = a
		doc.Cursor = at + 1
		// …until the game it fills ends: past that, the next Action is the next
		// game's first play, and the Cursor rests on it rather than slipping a
		// roll of this game in front of it (ADR-0050).
		if doc.Cursor < len(doc.Actions) && !gameEndsAt(doc, at) {
			// An insertion in the MIDDLE goes on inserting: the user is filling
			// a skipped passage, not typing over what follows. The slot is drawn
			// (EntryInfo), so it is a visible state, not a mode.
			doc.Entry, doc.pendingBoard = &Entry{Side: proposedSide(doc, doc.Cursor), Mode: EntryNew, At: doc.Cursor}, nil
			// Held like an append: the inserted Action may carry a mark.
			doc.HoldCursor = true
			return doc
		}
	}
	doc.Entry, doc.pendingBoard = nil, nil
	return doc
}

// cubeGesture records a double, its answer or a resignation. Each of them first
// validates a play left selected but not recorded, which is what "double" means when
// the user has already picked the play they were looking at.
//
// It writes at the ENTRY'S SLOT, like a roll: on a walked-back cell `t` replaces
// the pass rather than inserting a take (ADR-0048 decision 1); beside an insertion
// it fills it; at the end it appends.
func cubeGesture(doc Document, g Gesture) (Document, error) {
	mode, at, slotSide := EntryNew, doc.Cursor, -1
	if e := doc.Entry; e != nil {
		mode, at = e.Mode, e.At
		if at < len(doc.Actions) {
			// Inside the document the camp is the SLOT's; Next.Side speaks of
			// the end of the match.
			slotSide = e.Side
		}
	}
	if doc.Entry != nil && doc.Entry.Selected && doc.Entry.Dice[0] != 0 && doc.Entry.Dice[1] != 0 {
		v, err := validate(doc)
		if err != nil {
			return doc, err
		}
		doc = v
		// The cube Action FOLLOWS the play just recorded, wherever the
		// validation left the Cursor.
		mode, at = EntryNew, clampSlot(at+1, len(doc.Actions))
		slotSide = -1
		if at < len(doc.Actions) {
			slotSide = proposedSide(doc, at)
		}
	} else {
		doc.Entry = nil
	}
	ann := Replay(doc, 0)
	side := ann.Next.Side
	if slotSide == domain.Black || slotSide == domain.White {
		side = slotSide
	}
	switch g.Kind {
	case GestureTake, GesturePass:
		if last := lastActionOfKind(doc, KindDouble); last >= 0 {
			side = opponent(doc.Actions[last].Side)
		}
	}
	if g.HasSide {
		side = g.Side
	}
	a := Action{Side: side}
	if g.HasTick {
		a.TickMS = tickOf(g.TickMS)
	}
	switch g.Kind {
	case GestureDouble:
		a.Kind = KindDouble
	case GestureTake:
		a.Kind = KindTake
	case GesturePass:
		a.Kind = KindPass
	case GestureResign:
		a.Kind, a.Level = KindResign, g.Level
		if a.Level < 1 {
			a.Level = 1
		}
	}
	doc.Entry = &Entry{Side: side, Mode: mode, At: at}
	return record(doc, a), nil
}

// lastActionOfKind returns the index of the last Action before the Cursor of that
// kind, or -1.
func lastActionOfKind(doc Document, k Kind) int {
	end := doc.Cursor
	if end > len(doc.Actions) {
		end = len(doc.Actions)
	}
	for i := end - 1; i >= 0; i-- {
		if doc.Actions[i].Kind == k {
			return i
		}
	}
	return -1
}

// EntryPosition is the Position the Action being typed would be played from: the one
// under the Cursor when it stands on an Action, and the one the match has reached
// otherwise. It is what the board shows and what [Candidates] reads.
func EntryPosition(doc Document) domain.Position {
	return entryPosition(doc)
}

func entryPosition(doc Document) domain.Position {
	at := doc.Cursor
	if doc.Entry != nil {
		at = doc.Entry.At
	}
	ann := Replay(doc, 0)
	if at >= 0 && at < len(ann.Actions) {
		return ann.Actions[at].Before
	}
	return ann.Next.Position
}

// Candidates lists the legal plays of the roll being entered, in the order the
// generator produces them. They are what the user picks from: the engine RANKS and
// RECOGNISES, it never chooses (ADR-0044).
func Candidates(doc Document) []domain.LegalPlay {
	e := proposedEntry(doc)
	if doc.Entry != nil {
		e = *doc.Entry
	}
	pos := entryPosition(doc)
	pos.Dice = e.Dice
	pos.PlayerOnRoll = e.Side
	pos.DecisionType = domain.CheckerAction
	return domain.LegalMoves(&pos)
}

// Editor is the undo stack around [Apply]. The stack is in memory and nowhere else: a
// crash loses it, and reopening a draft replays the document as it was last written
// (ADR-0045 rule 1).
type Editor struct {
	Doc    Document
	past   []Document
	future []Document

	// replayer is the session's incremental [Replayer]: one session, one cache,
	// so a gesture replays only what it changed.
	replayer Replayer
}

// NewEditor starts an editing session on doc.
func NewEditor(doc Document) *Editor { return &Editor{Doc: doc} }

// Replay annotates the session's document, incrementally. `from` is where the
// Cursor starts looking for an Inconsistency to land on — the Action the last
// gesture touched, which [Document.Touched] names.
func (e *Editor) Replay(from int) Annotated { return e.replayer.Replay(e.Doc, from) }

// From is where a Replay after the last gesture must start looking: the Action
// that gesture wrote, else the Cursor (see Document.Touched). A held Cursor
// (Document.HoldCursor) answers the end of the document, so it stays put.
func (e *Editor) From() int {
	if e.Doc.HoldCursor {
		return len(e.Doc.Actions)
	}
	if e.Doc.HasTouched {
		return e.Doc.Touched
	}
	return e.Doc.Cursor
}

// SeekCursor puts the Cursor on an Action and loads it into the entry, as walking
// there would. It is NOT a gesture and pushes nothing on the stack: the Cursor is
// not a fact of the match. It applies the Replay's jump to the session's document.
func (e *Editor) SeekCursor(at int) {
	if at < 0 {
		at = 0
	}
	if at > len(e.Doc.Actions) {
		at = len(e.Doc.Actions)
	}
	if at == e.Doc.Cursor {
		return
	}
	// Remember where the user was reading, so a correction there gives it back
	// (fonctionnel.md §2).
	if !e.Doc.HasReturn {
		e.Doc.Return, e.Doc.HasReturn = e.Doc.Cursor, true
	}
	e.Doc.Cursor = at
	loadEntry(&e.Doc)
}

// CanUndo reports whether there is a gesture to undo.
func (e *Editor) CanUndo() bool { return len(e.past) > 0 }

// CanRedo reports whether a gesture has been undone and not redone.
func (e *Editor) CanRedo() bool { return len(e.future) > 0 }

// Apply records the gesture, pushing the previous document on the undo stack. A
// gesture that fails changes nothing, stack included.
func (e *Editor) Apply(g Gesture) error {
	next, err := Apply(e.Doc, g)
	if err != nil {
		return err
	}
	e.past = append(e.past, e.Doc)
	e.future = nil
	e.Doc = next
	return nil
}

// Undo steps back one gesture, reporting whether there was one.
func (e *Editor) Undo() bool {
	if len(e.past) == 0 {
		return false
	}
	e.future = append(e.future, e.Doc)
	e.Doc = e.past[len(e.past)-1]
	e.past = e.past[:len(e.past)-1]
	return true
}

// Redo steps forward again, reporting whether there was anything to redo.
func (e *Editor) Redo() bool {
	if len(e.future) == 0 {
		return false
	}
	e.past = append(e.past, e.Doc)
	e.Doc = e.future[len(e.future)-1]
	e.future = e.future[:len(e.future)-1]
	return true
}

// SetMatchID states which Match the draft has produced, on the document AND on
// every document of the undo stack. It is a fact about the library, not a
// gesture: undoing past a save must not forget the Match, or the next save
// would create a second one. It pushes nothing.
func (e *Editor) SetMatchID(id *int64) {
	set := func(doc *Document) {
		if id == nil {
			doc.Header.MatchID = nil
			return
		}
		v := *id
		doc.Header.MatchID = &v
	}
	set(&e.Doc)
	for i := range e.past {
		set(&e.past[i])
	}
	for i := range e.future {
		set(&e.future[i])
	}
}
