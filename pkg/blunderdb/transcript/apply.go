package transcript

import (
	"errors"
	"fmt"

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
	// are full it starts the roll again.
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
	// GestureDelete removes the Action under the Cursor. The ones after it keep their
	// side, which is how a deletion shows up as one local double turn instead of
	// rewriting the rest of the match (ADR-0045 rule 4).
	GestureDelete GestureKind = "delete"
	// GestureFlipSide gives the Action under the Cursor to the other camp.
	GestureFlipSide GestureKind = "flip_side"
	// GestureSetLength changes the match length, money play included, and re-states
	// the session's rules.
	GestureSetLength GestureKind = "set_length"
	// GestureSwapPlayers exchanges the two players: names, every side, and the board,
	// which is why every play's steps are mirrored with them.
	GestureSwapPlayers GestureKind = "swap_players"
	// GestureSetHeader replaces the header wholesale — the metadata, the tournament,
	// and the match id a first save posts.
	GestureSetHeader GestureKind = "set_header"
	// GestureUndo and GestureRedo walk the editing session's stack. They are
	// NAMED here, so that a caller has one spelling of them, and they are the two
	// [Apply] refuses: a stack is state, and it lives in [Editor]. The session
	// layer (database/db_transcription.go) routes them to [Editor.Undo] and
	// [Editor.Redo] before ever reaching Apply.
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
	out := doc.clone()
	// Touched describes the LAST gesture and nothing else: it is cleared here and
	// set again only by the paths that write an Action, so that walking the
	// Cursor never leaves a stale index behind for the next Replay to start from.
	out.HasTouched = false
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
		if g.Die < 1 || g.Die > 6 {
			return doc, fmt.Errorf("transcript: %d is not a die", g.Die)
		}
		e := ensureEntry(&out)
		switch {
		case e.Dice[0] == 0:
			e.Dice[0] = g.Die
		case e.Dice[1] == 0:
			e.Dice[1] = g.Die
		default:
			e.Dice = [2]int{g.Die, 0}
		}
		e.Steps, e.Selected, e.Review = nil, false, false
		reroll(&out)
		return out, nil

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
		// Review is NOT cleared here. fonctionnel.md §2 marks the Action "à
		// revoir" *until validation*, and picking a candidate is the reviewing,
		// not the end of it: the panel preselects its own 0-ply rank as soon as
		// the roll comes back, so clearing the mark on a selection would clear
		// it before the user has seen anything.
		return out, nil

	case GestureEnterPlay:
		e := ensureEntry(&out)
		e.Steps = append([]domain.CheckerStep(nil), g.Steps...)
		e.Selected = true
		// A board no legal play reaches is what really happened, and it is kept as
		// given; validate() drops it again if the play turns out to be legal.
		out.pendingBoard = nil
		if g.BoardAfter != nil {
			b := *g.BoardAfter
			out.pendingBoard = &b
		}
		return out, nil

	case GestureValidate:
		return validate(out)

	case GestureDance:
		e := ensureEntry(&out)
		if e.Dice[0] == 0 || e.Dice[1] == 0 {
			return doc, ErrNoDice
		}
		return record(out, Action{Side: e.Side, Kind: KindDance, Dice: e.Dice}), nil

	case GestureDouble, GestureTake, GesturePass, GestureResign:
		return cubeGesture(out, g)

	case GestureCursorBack:
		out = commitCorrection(out)
		if out.Cursor > 0 {
			if !out.HasReturn {
				out.Return, out.HasReturn = out.Cursor, true
			}
			out.Cursor--
		}
		loadEntry(&out)
		return out, nil

	case GestureCursorForward:
		out = commitCorrection(out)
		if out.Cursor < len(out.Actions) {
			out.Cursor++
		}
		loadEntry(&out)
		return out, nil

	case GestureCorrect:
		if out.Cursor < 0 || out.Cursor >= len(out.Actions) {
			return doc, ErrNoAction
		}
		loadEntry(&out)
		return out, nil

	case GestureInsertBefore, GestureInsertAfter:
		at := out.Cursor
		if g.Kind == GestureInsertAfter && at < len(out.Actions) {
			// "After the last Action" and "at the end of the document" are the
			// same slot: a Cursor already past the last Action is not moved on
			// again, which would otherwise make `a` an error at the one place
			// the user spends most of their time.
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
		if out.Cursor < 0 || out.Cursor >= len(out.Actions) {
			return doc, ErrNoAction
		}
		out.Actions = append(out.Actions[:out.Cursor], out.Actions[out.Cursor+1:]...)
		if out.Cursor > len(out.Actions) {
			out.Cursor = len(out.Actions)
		}
		// The Cursor stays put, which after the removal is the FOLLOWING Action —
		// fonctionnel.md §2, "supprimer … Replay depuis la suivante" — and that
		// slot is what the Replay is asked to look at, since the double turn a
		// deletion leaves behind lands exactly there.
		out.Touched, out.HasTouched = out.Cursor, true
		out.Entry, out.HasReturn = nil, false
		return out, nil

	case GestureFlipSide:
		if out.Cursor < 0 || out.Cursor >= len(out.Actions) {
			return doc, ErrNoAction
		}
		out.Actions[out.Cursor].Side = opponent(out.Actions[out.Cursor].Side)
		out.Touched, out.HasTouched = out.Cursor, true
		out.Entry = nil
		return out, nil

	case GestureSetLength:
		if !g.HasLength {
			return doc, fmt.Errorf("transcript: no match length given")
		}
		out.Header.MatchLength = g.MatchLength
		// Money and match do not carry the same rules: crossing between them
		// re-states the session's flags rather than keeping the other side's.
		out.Header.Jacoby = g.MatchLength == 0 && (!g.HasRules || g.Jacoby)
		out.Header.Beaver = g.MatchLength == 0 && g.HasRules && g.Beaver
		out.Entry = nil
		return out, nil

	case GestureSwapPlayers:
		out.Header.Player1, out.Header.Player2 = out.Header.Player2, out.Header.Player1
		for i := range out.Actions {
			a := &out.Actions[i]
			a.Side = opponent(a.Side)
			if a.Kind == KindOpening {
				a.Dice[0], a.Dice[1] = a.Dice[1], a.Dice[0]
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
		out.Entry = nil
		return out, nil

	case GestureSetHeader:
		out.Header = g.Header
		return out, nil

	case GestureUndo, GestureRedo:
		return doc, ErrNotPure
	}
	return doc, fmt.Errorf("transcript: unknown gesture %q", g.Kind)
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
//
// A new entry after a decided opening starts with the opening's own roll: the winner
// of the opening plays the two dice that were just rolled, and fonctionnel.md §1.2 is
// explicit that the user does not type them again.
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
	e := Entry{At: at, Mode: EntryNew, Side: Replay(doc, 0).Next.Side}
	if at > 0 {
		if prev := doc.Actions[at-1]; prev.Kind == KindOpening && prev.Dice[0] != prev.Dice[1] {
			hi, lo := prev.Dice[0], prev.Dice[1]
			if lo > hi {
				hi, lo = lo, hi
			}
			e.Dice = [2]int{hi, lo}
		}
	}
	return e
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
// It is what makes ux.md §4.3's budgets true. "Erreur vue k tours plus tard :
// Retour, `h`×k, `j`, `l`×k" spends its last k keystrokes walking FORWARD, not
// on a validation — so walking away from a correction has to keep it, or those
// budgets would each be one keystroke short and, worse, a user who walked on
// would silently lose the play they had just picked.
//
// Two things it deliberately does not do. It does not take the correction's
// return jump: `l` is not a validation, it is a step, and the next `l` must
// carry on from the Action just corrected rather than from the end of the
// document. And it does nothing at all when the entry still says exactly what
// the Action says — walking the Transcript to READ it must not rewrite every
// cell it crosses, push an undo entry per cell, or touch the row on disk.
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
		// A correction that is not ready to be recorded — a roll half retyped,
		// no play picked — is simply left where it is. Nothing is refused and
		// nothing is written: the Action keeps what it had.
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
// The rule has two halves and they are both about not making the user retype
// what they already told the software. If the play RECORDED on the Action is
// still one of the new roll's legal plays, it is kept — reading "53" where the
// sheet said "63" must not cost the play as well as the dice. If it is not, the
// first candidate of the new roll is preselected and the entry is marked for
// review: something has to be chosen, and an empty list under a filled roll
// would leave the user with nothing to press.
//
// Nothing is written here. The Action keeps the play it has until a validation
// replaces it, which is what makes a half-finished correction losable and the
// document on disk always a document the user validated.
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
		// The new roll allows nothing: it is a dance, and the panel's own
		// candidate round trip records one. Keeping a play here would hand a
		// validation a play the roll cannot make.
		return
	}
	// Both tests are needed, and they are not the same one. The board reached
	// says the play LANDS where a legal play lands; diceCoherent says its steps
	// can be charged to the two dice — the very thing a corrected roll breaks,
	// and what §1.4's "dés incohérents" is (a 6-1 played as 8/2 6/5 reaches a
	// board a 6-1 reaches, and a corrected 5-4 does not make it a 5-4).
	if _, reached := resolveSteps(pos.Board, e.Side, played.Steps); findPlay(legal, reached) != nil &&
		diceCoherent(played.Steps, e.Dice, e.Side) {
		e.Steps = append([]domain.CheckerStep(nil), played.Steps...)
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
	var a Action
	if expectsOpening(doc, *e) {
		a = Action{Side: e.Side, Kind: KindOpening, Dice: e.Dice}
		if e.Dice[0] != e.Dice[1] {
			a.Side = domain.Black
			if e.Dice[1] > e.Dice[0] {
				a.Side = domain.White
			}
		}
	} else {
		if !e.Selected {
			return doc, ErrNoCandidate
		}
		a = Action{Side: e.Side, Kind: KindChecker, Dice: e.Dice, Steps: e.Steps}
		if doc.pendingBoard != nil {
			pos := entryPosition(doc)
			pos.Dice, pos.PlayerOnRoll, pos.DecisionType = e.Dice, e.Side, domain.CheckerAction
			if findPlay(domain.LegalMoves(&pos), *doc.pendingBoard) == nil {
				b := *doc.pendingBoard
				a.BoardAfter = &b
			}
		}
	}
	return record(doc, a), nil
}

// expectsOpening reports whether the slot the entry lands on is a game's first Action.
func expectsOpening(doc Document, e Entry) bool {
	if e.Mode == EntryReplace && e.At < len(doc.Actions) {
		return doc.Actions[e.At].Kind == KindOpening
	}
	if e.At < len(doc.Actions) {
		return false
	}
	return Replay(doc, 0).Next.Expects == KindOpening
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
	// A gesture that EDITS what is already written names the Action it touched,
	// so the Replay can land the Cursor on the Inconsistency it may have made
	// behind the user's back. A plain append does NOT: there is nothing behind
	// it, and pulling the Cursor back onto the Action just typed — a play the
	// rules mark, an Action past the end of a won match — would make the next
	// roll correct it instead of following it. The panel shows the last
	// Action's marks where they belong, under the dice.
	if mode == EntryReplace || at < len(doc.Actions) {
		doc.Touched, doc.HasTouched = at, true
	}
	if mode == EntryReplace && at < len(doc.Actions) {
		doc.Actions[at] = a
		doc.Cursor = at + 1
		if doc.HasReturn {
			doc.Cursor, doc.HasReturn = doc.Return, false
		}
	} else {
		doc.Actions = append(doc.Actions, Action{})
		copy(doc.Actions[at+1:], doc.Actions[at:])
		doc.Actions[at] = a
		doc.Cursor = at + 1
	}
	doc.Entry, doc.pendingBoard = nil, nil
	return doc
}

// cubeGesture records a double, its answer or a resignation. Each of them first
// validates a play left selected but not recorded, which is what "double" means when
// the user has already picked the play they were looking at.
func cubeGesture(doc Document, g Gesture) (Document, error) {
	if doc.Entry != nil && doc.Entry.Selected && doc.Entry.Dice[0] != 0 && doc.Entry.Dice[1] != 0 {
		v, err := validate(doc)
		if err != nil {
			return doc, err
		}
		doc = v
	} else {
		doc.Entry = nil
	}
	ann := Replay(doc, 0)
	side := ann.Next.Side
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
	doc.Entry = &Entry{Side: side, Mode: EntryNew, At: doc.Cursor}
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

	// replayer is the session's incremental [Replayer]. It belongs here for the
	// same reason the stack does: one editing session, one cache, and a gesture
	// then costs the replay of what it changed instead of the whole match.
	replayer Replayer
}

// NewEditor starts an editing session on doc.
func NewEditor(doc Document) *Editor { return &Editor{Doc: doc} }

// Replay annotates the session's document, incrementally. `from` is where the
// Cursor starts looking for an Inconsistency to land on — the Action the last
// gesture touched, which [Document.Touched] names.
func (e *Editor) Replay(from int) Annotated { return e.replayer.Replay(e.Doc, from) }

// From is where a Replay after the last gesture must start looking: the Action
// that gesture wrote, and the Cursor when it wrote none. The two differ exactly
// where it matters — a correction in place sends the Cursor back to where the
// user came from, and the Inconsistency it just created is behind that.
func (e *Editor) From() int {
	if e.Doc.HasTouched {
		return e.Doc.Touched
	}
	return e.Doc.Cursor
}

// SeekCursor puts the Cursor on an Action and loads it into the entry, exactly
// as walking there with the Cursor gestures would. It is NOT a gesture: it
// pushes nothing on the stack, because the Cursor is not a fact of the match
// and undoing a jump the software made on its own would undo nothing the user
// did. It is what makes the Replay's "jump to the first Inconsistency" real on
// the session's document and not only in what was handed back.
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
	// Where the user was reading is remembered, exactly as walking back with the
	// Cursor remembers it: a correction made on the Inconsistency they were sent
	// to then gives them their place back (fonctionnel.md §2).
	if !e.Doc.HasReturn {
		e.Doc.Return, e.Doc.HasReturn = e.Doc.Cursor, true
	}
	e.Doc.Cursor = at
	loadEntry(&e.Doc)
}

// CanUndo and CanRedo report what the panel has to draw: whether there is
// anything on either side of the stack.
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
