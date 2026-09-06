package engine

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// Game phase classification (issue #264, fiche I.8).
//
// The phase is a DERIVED label: it is computed from the board alone, stored in
// an indexed column so a search can use it, recomputed on import and by
// `blunderdb repair`, and never editable. Nothing outside this file decides
// what phase a position is in.
//
// Only the board is read — never the move number of the game the position came
// from. A position is identified by its Zobrist hash and can therefore be
// reached at move 3 of one match and move 30 of another; a label derived from
// the visit would contradict itself between two rows that are the same row.
//
// Three of the four boundaries are the ones gnubg draws and are sourced
// (docs/recherche/P5-classification-type-de-jeu.md, §Key Findings 2):
// contact is decided by whether the two rearmost checkers have crossed, which
// is exactly domain.Position.MatchesNoContact. The fourth — where the opening
// stops — has no published threshold, so it is stated here as one named
// constant and documented as a convention (see OpeningDisplacementMax).

// OpeningDisplacementMax is the largest number of checkers either side may
// have moved off its starting points for the position to still count as the
// opening.
//
// It is a CONVENTION, not a sourced threshold: P5 found none published, and
// recommends that every unsourced threshold be a named, versioned parameter
// rather than a literal buried in a condition. Four is the count a side
// reaches after two ordinary rolls, or after one doublet — the opening move
// and its reply, which is what backgammon literature calls the opening.
//
// Raising or lowering it re-labels positions; it does not change any stored
// analysis. `blunderdb repair` recomputes every phase, so a change of this
// constant is a repair away from being applied to an existing database.
const OpeningDisplacementMax = 4

// ClassifyGamePhase returns the phase of a position. The position need not be
// normalized: the classification is symmetric, so it holds whichever side is
// on roll.
func ClassifyGamePhase(p *domain.Position) domain.GamePhase {
	if p == nil {
		return domain.PhaseUnknown
	}
	if p.MatchesNoContact() {
		if bothSidesHome(&p.Board) {
			return domain.PhaseBearoff
		}
		return domain.PhaseRace
	}
	if isOpening(&p.Board) {
		return domain.PhaseOpening
	}
	return domain.PhaseMiddlegame
}

// bothSidesHome reports whether every checker still on the board stands in its
// own home board. That is the point from which no checker can be blocked or
// hit any more and the game is a pure bear-off — the boundary gnubg's bear-off
// databases are indexed on.
func bothSidesHome(b *domain.Board) bool {
	for i := 0; i <= domain.NumPoints+1; i++ {
		pt := b.Points[i]
		if pt.Checkers == 0 {
			continue
		}
		if pt.Color == domain.Black && (i < 1 || i > 6) {
			return false
		}
		if pt.Color == domain.White && (i < 19 || i > 24) {
			return false
		}
	}
	return true
}

// isOpening reports whether the position is still within the opening: nothing
// borne off, nothing on the bar, and neither side has moved more than
// OpeningDisplacementMax checkers off its starting points.
//
// Displacement is counted as the number of checkers MISSING from the starting
// points, which is the number of checkers that have left them — a checker that
// moved from the 13-point to the 11-point counts once, wherever it landed.
func isOpening(b *domain.Board) bool {
	if b.Bearoff[0] > 0 || b.Bearoff[1] > 0 {
		return false
	}
	if b.Points[0].Checkers > 0 || b.Points[25].Checkers > 0 {
		return false
	}
	var displaced [2]int
	for i := 1; i <= domain.NumPoints; i++ {
		s := startingBoard.Points[i]
		if s.Checkers == 0 {
			continue
		}
		have := 0
		if b.Points[i].Color == s.Color {
			have = b.Points[i].Checkers
		}
		if have < s.Checkers {
			displaced[s.Color] += s.Checkers - have
		}
	}
	return displaced[domain.Black] <= OpeningDisplacementMax &&
		displaced[domain.White] <= OpeningDisplacementMax
}

// startingBoard is the opening arrangement, read only, taken from the domain's
// own constructor so the two can never drift.
var startingBoard = domain.InitializePosition().Board
