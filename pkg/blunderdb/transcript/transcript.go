// Package transcript is the engine of a Transcription: the move-by-move record of a
// match played somewhere else — at a club, over a real board, from a video or a score
// sheet — while the user writes it down and corrects it.
//
// # What it is, and what it is not
//
// A Transcription is a draft (ADR-0045): a Document of Actions the user records, kept
// apart from the Match it will produce. This package is the whole of its logic — the
// document, the gestures that change it, the Replay that derives everything else, and
// the domain triplet a save or a .mat export is made of. It decides nothing: the dice
// were rolled at a table, the moves were made by two people, and the engine is here to
// rank candidates and to recognise what was played, never to choose it (ADR-0044).
//
// The rules therefore *check*, they never *enforce*. A move that no legal play can
// reach is a fact of the match: it is recorded as played, marked as an Inconsistency,
// and the game goes on from the board it left. Nothing in this package refuses an
// Action, and nothing deletes an Inconsistency.
//
// # Purity
//
// Its one internal dependency is [github.com/kevung/blunderdb/pkg/blunderdb/domain] —
// no SQL, no storage, and deliberately no ingest either, which imports storage. What a
// save needs is returned as domain types by [MatchParts], and it is the caller, on the
// database side, that turns them into an ingest graph and writes them. [FromMAT] adds
// one external dependency, the .mat parser the GnuBG importer already uses.
//
// # The three operations
//
//   - [Apply] takes a Document and one Gesture and returns the Document it becomes. It
//     is pure: the undo stack a user expects lives in [Editor], in memory, because a
//     function of (document, gesture) cannot hold one.
//   - [Replay] plays the Actions again in order and returns an [Annotated] document:
//     the Position each Action was played from, the board it left, the score, the
//     Crawford game, where each game ended and with how many points, the end of the
//     match — and the Inconsistencies. Nothing derived is ever stored on an Action.
//   - [MatchParts] returns the match, its games and its moves, which is exactly what
//     ingest.RenderMAT takes and what a save writes.
//
// # Sides and encodings
//
// An Action's Side is 0 for player 1 and 1 for player 2 — the domain's Black/White,
// player 1 at the bottom of the board. The domain types this package returns keep
// their own encodings, which are not the same one: domain.Move.Player is XG's +1/-1
// and domain.Game.Winner is gnubg's 0/1/-1. The conversion happens once, in
// [MatchParts].
package transcript

import (
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// FormatVersion is the version of the Document's shape. It travels in the document
// itself: a change to the form of a Transcription is a version of the document, never
// a DatabaseVersion migration (ADR-0045 rule 1).
const FormatVersion = 1

// DefaultMatchLength is the length a first draft is offered, when no previous draft
// says otherwise (fonctionnel.md §1.1).
const DefaultMatchLength = 7

// Kind is what an Action is: one player's act at one moment of the match.
type Kind string

const (
	// KindOpening is the opening roll: Dice[0] is player 1's die, Dice[1] player 2's.
	// It produces neither Move nor Position; it fixes who starts and with which roll,
	// and it opens a game. A tie stays in the document and is followed by another
	// opening ("relance").
	KindOpening Kind = "opening"
	// KindChecker is a roll and the checker play it was used for.
	KindChecker Kind = "checker"
	// KindDance is a roll that allowed no play at all.
	KindDance Kind = "dance"
	// KindUnrecorded is a roll whose play was NOT WRITTEN DOWN. It is not a dance:
	// the player did move, and the record simply does not say how. gnubg spells it
	// "???" in a .mat cell, and a transcription that reads one must give it back
	// unchanged — rendering it as a dance would put a play in the file that nobody
	// made. The board it leaves is unknown, so the replay carries the previous one
	// forward and marks the Action (fonctionnel.md §1.4, "coup non consigné").
	KindUnrecorded Kind = "unrecorded"
	// KindDouble is a double or a redouble. Its answer is a separate Action.
	KindDouble Kind = "double"
	// KindTake accepts the double that precedes it; the cube doubles and changes hands.
	KindTake Kind = "take"
	// KindPass declines it; the game ends at the value the cube had before the offer.
	KindPass Kind = "pass"
	// KindResign gives the game up at Level (1 = single, 2 = gammon, 3 = backgammon).
	// It produces no Move — only the winner and the points of the Game (ADR-0045 §6).
	KindResign Kind = "resign"
)

// UnrecordedNotation is how a play the record does not carry is written: the three
// question marks gnubg puts in the cell. It is the notation a KindUnrecorded Action
// takes, the CheckerMove the saved Move carries, and what ingest.RenderMAT writes
// back — one spelling, so a "???" read from a .mat comes out of the round trip as
// the same "???".
const UnrecordedNotation = "???"

// Action is one player's act. Side and Kind are common to all of them; the remaining
// fields belong to one Kind each (fonctionnel.md §1.2).
//
// Side is OWNED by the Action: proposed at entry from whose turn it is, it changes
// afterwards only by the "change side" gesture. Inserting or deleting an Action
// therefore never flips the side of the ones after it — it creates one local
// Inconsistency instead, which is the correction the user actually wants to see.
type Action struct {
	Side int  `json:"side"`
	Kind Kind `json:"kind"`

	// Dice carries the roll of an opening, a checker play or a dance.
	Dice [2]int `json:"dice,omitempty"`

	// Steps is the checker play, in absolute board indices (domain.CheckerStep).
	Steps []domain.CheckerStep `json:"steps,omitempty"`

	// BoardAfter is set ONLY when the play is illegal — when no legal play from the
	// previous position reaches it. It is then what the board really became, and it
	// takes precedence over anything the rules could compute.
	BoardAfter *domain.Board `json:"board_after,omitempty"`

	// Level is the resignation's value in games: 1, 2 or 3.
	Level int `json:"level,omitempty"`
}

// Header is the head of the document (fonctionnel.md §1.1). Only MatchLength is asked
// for at creation; everything else is filled in later or never.
type Header struct {
	// MatchLength is the points the match is played to; 0 is a money session.
	MatchLength int `json:"match_length"`
	// Jacoby and Beaver are rules of the session, asked for in money play only. They
	// are flags posted on every Position and never an Action (ADR-0028, ADR-0044).
	Jacoby bool `json:"jacoby"`
	Beaver bool `json:"beaver"`
	// MaxCube is the session's cube ceiling as a log2 exponent, 0 for none. A double
	// that would pass it is an Inconsistency, never a refusal.
	MaxCube int `json:"max_cube,omitempty"`

	Player1 string `json:"player1,omitempty"`
	Player2 string `json:"player2,omitempty"`

	Event       string    `json:"event,omitempty"`
	Location    string    `json:"location,omitempty"`
	Round       string    `json:"round,omitempty"`
	Date        time.Time `json:"date,omitempty"`
	Transcriber string    `json:"transcriber,omitempty"`

	// TournamentID attaches the saved Match to a tournament; MatchID is the Match this
	// draft owns, posted at the first save and stable from then on.
	TournamentID *int64 `json:"tournament_id,omitempty"`
	MatchID      *int64 `json:"match_id,omitempty"`
}

// Document is a Transcription: its header, its Actions in order, and the Cursor — the
// Action the transcription is currently about. It is what one row of the transcription
// table holds, as JSON, rewritten after every gesture that changes an Action.
//
// Entry and the correction's return position are NOT persisted: they are the Action
// being typed, which a crash is allowed to lose. Reopening a draft puts the Cursor at
// the end and replays everything.
type Document struct {
	FormatVersion int      `json:"format_version"`
	Header        Header   `json:"header"`
	Actions       []Action `json:"actions"`
	Cursor        int      `json:"cursor"`

	Entry     *Entry `json:"-"`
	Return    int    `json:"-"`
	HasReturn bool   `json:"-"`

	// Touched is the index of the Action the last gesture EDITED — replaced,
	// inserted between two others, deleted, given to the other camp — and
	// HasTouched says a gesture edited one at all. It is what a Replay is asked
	// to start looking for Inconsistencies from (fonctionnel.md §1.4 — after a
	// Replay the Cursor jumps to the first one).
	//
	// It is not the Cursor: a correction in place sends the Cursor back where
	// the user came from, several Actions further on, and the Inconsistency the
	// correction just created sits behind it. And a plain APPEND sets neither,
	// deliberately: there is nothing behind the last Action, and pulling the
	// Cursor onto the play just typed — one the rules happen to mark — would
	// make the next roll correct it instead of following it.
	Touched    int  `json:"-"`
	HasTouched bool `json:"-"`

	// pendingBoard is the board a hand-entered play left, held until validation
	// decides whether the play was legal — a legal play's board is derived, and only
	// an illegal one is written on the Action.
	pendingBoard *domain.Board
}

// EntryMode says what validating the Entry does to the document.
type EntryMode int

const (
	// EntryNew inserts the Action at At.
	EntryNew EntryMode = iota
	// EntryReplace overwrites the Action at At (a correction in place).
	EntryReplace
)

// Entry is the Action being typed: the dice as they are entered one die at a time and
// the candidate picked for them. It is a draft of a draft — never written to the
// database, never part of a Replay.
type Entry struct {
	Side     int
	Dice     [2]int
	Steps    []domain.CheckerStep
	Selected bool
	Mode     EntryMode
	At       int

	// Review marks a play the user has to look at again: correcting a roll
	// under a recorded play, when the play is no longer legal for the new
	// roll, preselects the first candidate of that roll rather than keeping a
	// play the dice no longer allow (fonctionnel.md §2, "corriger un jet").
	// The Action itself is NOT marked — nothing is written until validation,
	// and a Replay of a document is the same whether an entry is under review
	// or not.
	Review bool
}

// New returns an empty draft of the given length, with an opening expected. A length of
// 0 is a money session, where Jacoby is on by default and beaver off.
func New(matchLength int) Document {
	return Document{
		FormatVersion: FormatVersion,
		Header: Header{
			MatchLength: matchLength,
			Jacoby:      matchLength == 0,
		},
	}
}

// clone copies a document deeply enough that the copy can be edited without the
// original moving: Apply is pure, and a shared Actions array would break that quietly.
func (d Document) clone() Document {
	out := d
	out.Actions = make([]Action, len(d.Actions))
	for i, a := range d.Actions {
		out.Actions[i] = a.clone()
	}
	if d.pendingBoard != nil {
		b := *d.pendingBoard
		out.pendingBoard = &b
	}
	if d.Entry != nil {
		e := *d.Entry
		e.Steps = append([]domain.CheckerStep(nil), d.Entry.Steps...)
		out.Entry = &e
	}
	return out
}

func (a Action) clone() Action {
	out := a
	out.Steps = append([]domain.CheckerStep(nil), a.Steps...)
	if a.BoardAfter != nil {
		b := *a.BoardAfter
		out.BoardAfter = &b
	}
	return out
}

// InitialBoard is the standard opening board, in the domain's absolute geometry: Black
// (player 1) moves 24→1 and bears off past 1, White (player 2) mirrors it.
func InitialBoard() domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: domain.None}
	}
	put := func(idx, n, color int) { b.Points[idx] = domain.Point{Checkers: n, Color: color} }
	put(24, 2, domain.Black)
	put(13, 5, domain.Black)
	put(8, 3, domain.Black)
	put(6, 5, domain.Black)
	put(1, 2, domain.White)
	put(12, 5, domain.White)
	put(17, 3, domain.White)
	put(19, 5, domain.White)
	return b
}

// centredCube is the cube of a game that starts: value 1 (exponent 0), owned by nobody.
func centredCube() domain.Cube { return domain.Cube{Owner: domain.None, Value: 0} }

// cubeValue turns the stored log2 exponent into the value the players say out loud.
func cubeValue(c domain.Cube) int { return 1 << c.Value }

func opponent(side int) int { return 1 - side }

// barOf returns the bar index of a side in the absolute geometry (they differ: Black
// re-enters from 25, White from 0).
func barOf(side int) int {
	if side == domain.Black {
		return domain.BlackBar
	}
	return domain.WhiteBar
}

// mirrorIndex maps a board index to the one it holds when the two players are
// exchanged: a point becomes its mirror, each bar becomes the other, Off stays Off.
func mirrorIndex(idx int) int {
	switch idx {
	case domain.Off:
		return domain.Off
	case domain.BlackBar:
		return domain.WhiteBar
	case domain.WhiteBar:
		return domain.BlackBar
	default:
		return 25 - idx
	}
}

// mirrorBoard turns the board around: every checker changes colour and every point
// becomes its mirror. It is what "swap the players" does to a board, and applying it
// to the steps of every Action is what keeps a swapped document replayable.
func mirrorBoard(b domain.Board) domain.Board {
	var out domain.Board
	for i := range out.Points {
		out.Points[i] = domain.Point{Color: domain.None}
	}
	for i := range b.Points {
		pt := b.Points[i]
		if pt.Checkers == 0 || (pt.Color != domain.Black && pt.Color != domain.White) {
			continue
		}
		out.Points[mirrorIndex(i)] = domain.Point{Checkers: pt.Checkers, Color: opponent(pt.Color)}
	}
	out.Bearoff = [2]int{b.Bearoff[domain.White], b.Bearoff[domain.Black]}
	return out
}
