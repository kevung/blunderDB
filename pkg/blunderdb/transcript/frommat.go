package transcript

import (
	"fmt"
	"strings"
	"time"

	"github.com/kevung/gnubgparser"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// FromMAT reads a Jellyfish/gnubg .mat transcript back into a Document — the same
// parser the GnuBG importer uses, and this package's one external dependency.
//
// It is the inverse of [MatchParts] followed by ingest.RenderMAT, and it exists for
// two reasons: it makes the round trip testable on real files, and it is what lets a
// .mat be REPLAYED to have its Inconsistencies pointed out rather than merely imported.
//
// Two things the format does not carry are reconstructed rather than invented:
//
//   - The opening roll. A .mat starts a game at the first play, so the opening is
//     rebuilt from it — the two dice, given to the player who moved first. A first
//     roll of doubles (which no opening can be) is written down as it stands and reads
//     as a tie; the play that follows keeps its own side and roll either way.
//   - The resignation. The format has no token for it: a game whose recorded plays do
//     not end it, yet which announces a winner, ended by a resignation, and the level
//     is what the announced points and the cube say it was.
func FromMAT(text string) (Document, error) {
	parsed, err := gnubgparser.ParseMAT(strings.NewReader(text))
	if err != nil {
		return Document{}, fmt.Errorf("transcript: parse .mat: %w", err)
	}
	return fromParsedMAT(parsed)
}

func fromParsedMAT(parsed *gnubgparser.Match) (Document, error) {
	doc := New(parsed.Metadata.MatchLength)
	doc.Header.Player1 = parsed.Metadata.Player1
	doc.Header.Player2 = parsed.Metadata.Player2
	doc.Header.Event = parsed.Metadata.Event
	doc.Header.Location = parsed.Metadata.Place
	doc.Header.Round = parsed.Metadata.Round
	doc.Header.Transcriber = parsed.Metadata.Annotator
	if d, err := time.Parse("2006-01-02", parsed.Metadata.Date); err == nil {
		doc.Header.Date = d
	}

	st := newState(doc.Header)
	push := func(a Action) {
		st.step(len(doc.Actions), a)
		doc.Actions = append(doc.Actions, a)
	}

	for gi := range parsed.Games {
		game := &parsed.Games[gi]
		if rec := firstCheckerRecord(game.Moves); rec != nil {
			push(openingAction(rec))
		} else {
			// Nothing to rebuild an opening from: close the running game by hand so
			// this one starts on a fresh board all the same.
			st.endGame(-1, 0)
			st.ensureGame()
		}

		for i := range game.Moves {
			rec := &game.Moves[i]
			switch rec.Type {
			case gnubgparser.MoveTypeNormal:
				push(checkerAction(rec))
			case gnubgparser.MoveTypeDouble:
				push(Action{Side: rec.Player, Kind: KindDouble})
			case gnubgparser.MoveTypeTake:
				push(Action{Side: rec.Player, Kind: KindTake})
			case gnubgparser.MoveTypeDrop:
				push(Action{Side: rec.Player, Kind: KindPass})
			case gnubgparser.MoveTypeResign:
				_, points := statedResult(parsed.Games, gi, doc.Header.MatchLength)
				push(Action{Side: rec.Player, Kind: KindResign, Level: resignLevel(points, cubeValue(st.cube))})
			}
		}

		// The plays do not end a game the file says was won: it was resigned.
		if winner, points := statedResult(parsed.Games, gi, doc.Header.MatchLength); st.gameActive && points > 0 &&
			(winner == domain.Black || winner == domain.White) {
			push(Action{
				Side:  opponent(winner),
				Kind:  KindResign,
				Level: resignLevel(points, cubeValue(st.cube)),
			})
		}
	}

	doc.Cursor = len(doc.Actions)
	return doc, nil
}

// firstCheckerRecord returns the first play of a game, which is the only trace its
// opening roll left in the file.
func firstCheckerRecord(records []gnubgparser.MoveRecord) *gnubgparser.MoveRecord {
	for i := range records {
		if records[i].Type == gnubgparser.MoveTypeNormal {
			return &records[i]
		}
	}
	return nil
}

// openingAction rebuilds the opening from the first play: the higher die goes to the
// player who moved, which is what having won the opening means.
func openingAction(rec *gnubgparser.MoveRecord) Action {
	hi, lo := rec.Dice[0], rec.Dice[1]
	if lo > hi {
		hi, lo = lo, hi
	}
	a := Action{Side: rec.Player, Kind: KindOpening, Dice: [2]int{hi, lo}}
	if rec.Player == domain.White {
		a.Dice = [2]int{lo, hi}
	}
	return a
}

// statedResult is what the FILE says a game was worth. The score line of the next game
// is the reliable statement of it: a "Wins N points" line standing on its own is
// attributed by the parser to whoever acted last, which is not the winner when a game
// ended on a resignation, and a game ended by a drop leaves it no points at all. The
// score lines never lie, and they are what the transcription must reproduce. The last
// game of a file has no successor and falls back to what the parser made of it.
func statedResult(games []gnubgparser.Game, i, matchLength int) (winner, points int) {
	if i+1 < len(games) {
		gained0 := games[i+1].Score[0] - games[i].Score[0]
		gained1 := games[i+1].Score[1] - games[i].Score[1]
		switch {
		case gained0 > 0 && gained1 <= 0:
			return domain.Black, gained0
		case gained1 > 0 && gained0 <= 0:
			return domain.White, gained1
		}
	}
	winner, points = games[i].Winner, games[i].Points
	// The last game has no successor to state its result. What it does have, when it
	// is the game that ended the match, is one player it CAN belong to: the file's
	// "and the match" says so, the parser drops it, and the arithmetic recovers it.
	if matchLength > 0 && points > 0 {
		ends0 := games[i].Score[0]+points >= matchLength
		ends1 := games[i].Score[1]+points >= matchLength
		if ends0 != ends1 {
			winner = domain.White
			if ends0 {
				winner = domain.Black
			}
		}
	}
	return winner, points
}

// checkerAction turns one play of the file into an Action. The from/to pairs are kept
// as the file wrote them — the Replay recomputes the hits and the board they leave, and
// says whether any legal play reaches it.
//
// A cell with no play in it is TWO different facts, and the difference is the whole of
// this function. A cell holding nothing but its dice (or the "Cannot Move" some writers
// put there) says the player could not play: a dance. A cell holding "???" says gnubg
// did not write down what was played — the player moved, and the record is silent. The
// parser's decoded Move is all -1 in both cases, so the raw MoveString is what separates
// them, and it is the only thing that does.
func checkerAction(rec *gnubgparser.MoveRecord) Action {
	side := rec.Player
	dice := [2]int{rec.Dice[0], rec.Dice[1]}
	if isUnrecorded(rec.MoveString) {
		return Action{Side: side, Kind: KindUnrecorded, Dice: dice}
	}
	steps := absoluteSteps(rec.Move, side)
	if len(steps) == 0 {
		return Action{Side: side, Kind: KindDance, Dice: dice}
	}
	return Action{Side: side, Kind: KindChecker, Dice: dice, Steps: steps}
}

// isUnrecorded recognises gnubg's mark for a play it did not record: a cell made of
// question marks and nothing else. The count is not fixed at three on purpose — the
// mark is what it means, not how long it is — but an empty cell is a dance, never this.
func isUnrecorded(moveString string) bool {
	s := strings.TrimSpace(moveString)
	return s != "" && strings.Trim(s, "?") == ""
}

// absoluteSteps converts the parser's player-relative points (0..23 for points 1..24,
// 24 for the bar, -1 for off) into the absolute board indices the domain uses, where
// player 2's frame is player 1's mirrored.
func absoluteSteps(move [8]int, side int) []domain.CheckerStep {
	var steps []domain.CheckerStep
	for i := 0; i < len(move); i += 2 {
		if move[i] == -1 {
			break
		}
		steps = append(steps, domain.CheckerStep{
			From: absolutePoint(move[i], side),
			To:   absolutePoint(move[i+1], side),
		})
	}
	return steps
}

func absolutePoint(rel, side int) int {
	switch {
	case rel == -1:
		return domain.Off
	case rel == 24:
		return barOf(side)
	default:
		point := rel + 1
		if side == domain.White {
			return 25 - point
		}
		return point
	}
}

// resignLevel reads back the level a resignation must have had for the game to be
// worth the points the file announces.
func resignLevel(points, cube int) int {
	if cube <= 0 {
		cube = 1
	}
	level := points / cube
	if level < 1 {
		level = 1
	}
	if level > 3 {
		level = 3
	}
	return level
}
