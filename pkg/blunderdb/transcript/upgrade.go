package transcript

// kindOpening is the opening roll of documents before version 3: Dice[0] player 1's
// die, Dice[1] player 2's, a tie followed by another one. [Upgrade] alone reads it.
const kindOpening Kind = "opening"

// Upgrade converts a document of an earlier FormatVersion to the current one, and
// returns a current document unchanged. Every reader of a stored draft goes through
// it, so the rest of the package knows only the current shape.
//
// Before version 3 a game opened with an opening Action. It is dropped: the play
// that followed it was already made by the winner with the opening roll, and
// becomes the game's first Action, carrying the score the opening declared. A
// re-roll after a tie is dropped with it — the declared score of a tie passes to
// the game's first play all the same. An opening that cut a game short becomes a
// declared score on the next game's first Action, the one thing that still starts a
// game while another is running; an opening with nothing behind it leaves that
// boundary waiting at the end ([Document.NextScore]). The Cursor keeps its Action,
// or the end.
//
// The first play takes the opening roll, player 1's die then player 2's: played by
// the opening's loser, or with another roll, it reads back marked as it was.
func Upgrade(doc Document) Document {
	if doc.FormatVersion >= FormatVersion {
		return doc
	}
	out := doc
	out.FormatVersion = FormatVersion
	out.Actions = make([]Action, 0, len(doc.Actions))
	out.Cursor = -1

	st := newState(doc.Header)
	opened := false
	var declared *[2]int
	var roll [2]int
	for i, a := range doc.Actions {
		if i == doc.Cursor {
			out.Cursor = len(out.Actions)
		}
		if a.Kind == kindOpening {
			if !opened {
				opened, declared, roll = true, nil, [2]int{}
			}
			if a.Dice[0] != a.Dice[1] {
				roll = a.Dice
			}
			if a.Score != nil && declared == nil {
				sc := *a.Score
				declared = &sc
			}
			continue
		}
		a = a.clone()
		a.Score = nil
		if opened {
			opened = false
			if declared == nil && st.gameActive {
				sc := st.points
				declared = &sc
			}
			a.Score, declared = declared, nil
			switch {
			case a.Kind != KindChecker && a.Kind != KindDance && a.Kind != KindUnrecorded:
			case roll != [2]int{}:
				a.Dice = roll
			default:
				a.Dice = openingOrder(a)
			}
		}
		st.step(len(out.Actions), a)
		out.Actions = append(out.Actions, a)
	}
	if opened {
		switch {
		case declared != nil:
			out.NextScore = declared
		case st.gameActive:
			sc := st.points
			out.NextScore = &sc
		}
	}
	if out.Cursor < 0 || doc.Cursor >= len(doc.Actions) {
		out.Cursor = len(out.Actions)
	}
	return out
}
