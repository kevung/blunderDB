package engine

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// Game-type classification (issue #291, fiche J.1a).
//
// The plan of play a position stands in, derived from the board alone, stored
// in an indexed column, recomputed by `blunderdb repair`, never editable.
// It answers the question a bundle of saved filters cannot: "show me my errors
// in a holding game".
//
// # What is sourced and what is a convention
//
// Three boundaries are gnubg's own and are sourced in
// docs/recherche/P5-classification-type-de-jeu.md: the game being over, the
// race (the two rearmost checkers have crossed — domain.MatchesNoContact), and
// "crashed" (at most six checkers outside the side's own points 1 and 2, a
// threshold its author Joseph Heled calls arbitrary and chose to be
// non-cyclic). Everything else is a CONVENTION: P5 searched the literature and
// found the human plans qualitative and convergent on concepts but "quasiment
// dépourvues de seuils numériques". No published inter-rater agreement exists
// for this problem either, so this classifier cannot be validated against a
// standard — there isn't one.
//
// That is why every unsourced threshold below is a named, versioned constant
// rather than a literal in a condition, and why the rules are published in the
// documentation. A contested taxonomy is worse than no taxonomy; a taxonomy
// whose rules are readable and whose label is derived, non-editable and never
// exported as truth is defensible.
//
// # Berliner's lesson, and what is done about it
//
// BKG's author first categorised positions hard and found that "errors in
// comparing the two were often made" at the boundaries (P5 §D). P5's answer is
// several labels and an ambiguity flag. This file keeps ONE label, and pays for
// it by ordering the rules from the most specific to the most general so a
// position that satisfies two rules gets the more informative one — a backgame
// that is also a holding game is a backgame. Where that order is a judgement
// rather than a deduction, the rule says so.
//
// # One label, for the player on roll
//
// The label describes the plan of the side to move, because the stored column
// answers "what plan was I in when I made this decision".

// GameTypeRulesVersion names the calibration of the unsourced thresholds
// below. It is bumped whenever one of them changes, so a label stored by an
// older version can be told apart from one this version would produce.
const GameTypeRulesVersion = "heuristique_v1"

const (
	// CrunchActiveCheckersMax is gnubg's CLASS_CRASHED threshold, SOURCED:
	// "you have at most 6 checkers not on points 1 or 2" (Joseph Heled,
	// bug-gnubg, 2012-02-08, quoted in P5 §Key Findings 3).
	CrunchActiveCheckersMax = 6

	// BlitzHomePointsMin is how many home-board points the mover must have
	// made for an attack to be called a blitz. CONVENTION: no published
	// threshold exists. Three is the point from which the opponent's return
	// from the bar starts to fail.
	BlitzHomePointsMin = 3

	// PrimeLengthMin is the length a blockade must reach to count as a prime
	// in the prime-vs-prime rule. CONVENTION: the literature names four-,
	// five- and six-primes without saying where the plan begins.
	PrimeLengthMin = 4

	// AcePointPipDeficitMin is how far behind in the race the mover must be
	// for a lone ace-point anchor to be called an ace-point GAME rather than
	// an anchor that happens to be deep. CONVENTION: P5 marks this threshold
	// as one to calibrate.
	AcePointPipDeficitMin = 20
)

// ClassifyGameType returns the plan of play of the side on roll.
func ClassifyGameType(p *domain.Position) domain.GameType {
	if p == nil {
		return domain.TypeUnknown
	}
	// Everything below is written from Black's point of view — Black moves
	// 24→1, its home is 1..6, the opponent's home is 19..24 — and White is
	// classified by mirroring the board. One set of rules, read once.
	b := p.Board
	if p.PlayerOnRoll == domain.White {
		b = mirrorBoard(&p.Board)
	}
	return classifyForBlack(&b)
}

func classifyForBlack(b *domain.Board) domain.GameType {
	// R1 — over. SOURCED (gnubg CLASS_OVER).
	if b.Bearoff[0] >= 15 || b.Bearoff[1] >= 15 {
		return domain.TypeOver
	}

	// R2 — race. SOURCED (gnubg CONTACT/RACE): the rearmost checkers crossed.
	if noContact(b) {
		return domain.TypeRace
	}

	// R3 — bear-in against contact. The mover is bringing checkers home while
	// the opponent still holds an anchor in the mover's home board: the plan
	// is not "bear off", it is "bear in without leaving a shot".
	if backmost(b, domain.Black) <= 6 && hasAnchorIn(b, domain.White, 1, 6) {
		return domain.TypeBearIn
	}

	// R4 — crunch. SOURCED (gnubg CLASS_CRASHED). Tested BEFORE the anchor
	// rules on purpose: a side with six active checkers has no plan left but
	// its timing, whatever anchors it still holds.
	if activeCheckers(b, domain.Black) <= CrunchActiveCheckersMax {
		return domain.TypeCrunch
	}

	// R5 — backgame. SOURCED concept: two or more anchors in the opponent's
	// home board (USBGF, Magriel). The convention P5 records is applied here:
	// a second anchor OUTSIDE the opponent's home board does not make a
	// backgame, it leaves a holding game.
	if anchorsIn(b, domain.Black, 19, 24) >= 2 {
		return domain.TypeBackgame
	}

	// R6 — ace-point game. SOURCED concept (USBGF); the race deficit is a
	// convention. Holding the opponent's ace point while level or ahead is a
	// deep anchor, not an ace-point game.
	if anchorsIn(b, domain.Black, 19, 24) == 1 && anchorAt(b, domain.Black, 24) &&
		pipOf(b, domain.Black)-pipOf(b, domain.White) >= AcePointPipDeficitMin {
		return domain.TypeAcePoint
	}

	// R7 — blitz. SOURCED concept; both thresholds are conventions. The
	// opponent must have something to answer: a checker on the bar, or a blot
	// in the mover's home board about to be hit.
	if homePointsMade(b, domain.Black) >= BlitzHomePointsMin &&
		(b.Points[domain.WhiteBar].Checkers > 0 || blotsIn(b, domain.White, 1, 6) > 0) {
		return domain.TypeBlitz
	}

	// R8 — prime vs prime. SOURCED concept (USBGF); the length is a
	// convention. Both blockades must actually trap something, or two long
	// walls facing nothing are just two long walls.
	if longestPrime(b, domain.Black) >= PrimeLengthMin && longestPrime(b, domain.White) >= PrimeLengthMin &&
		trapped(b, domain.White) && trapped(b, domain.Black) {
		return domain.TypePrimeVsPrime
	}

	// R9 — holding, and mutual holding. SOURCED concept; the span of what
	// counts as a "high" anchor is P5's proposal (points 18..21 seen from the
	// holder), which is the golden point and its neighbours.
	blackHigh := anchorsIn(b, domain.Black, 18, 21)
	whiteHigh := anchorsIn(b, domain.White, 4, 7)
	if blackHigh >= 1 {
		if whiteHigh >= 1 {
			return domain.TypeMutualHolding
		}
		return domain.TypeHolding
	}

	// R10 — residual. P5 proposes "containment" here, which needs to know that
	// a checker was JUST hit; a position carries no such history (it is
	// identified by its board, and reached from many games), so the rule is
	// not written rather than written on a guess.
	return domain.TypeContact
}

// mirrorBoard turns White's position into Black's. Index i becomes 25−i, which
// maps the White bar (0) onto the Black bar (25), and the colours swap.
func mirrorBoard(b *domain.Board) domain.Board {
	var out domain.Board
	for i := 0; i <= domain.NumPoints+1; i++ {
		pt := b.Points[domain.NumPoints+1-i]
		switch pt.Color {
		case domain.Black:
			out.Points[i] = domain.Point{Checkers: pt.Checkers, Color: domain.White}
		case domain.White:
			out.Points[i] = domain.Point{Checkers: pt.Checkers, Color: domain.Black}
		default:
			out.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
		}
	}
	out.Bearoff = [2]int{b.Bearoff[1], b.Bearoff[0]}
	return out
}

// noContact reports whether the rearmost checkers have crossed. Black's
// rearmost is its highest occupied point, White's is its lowest; once Black's
// is below White's, neither can be blocked or hit again.
func noContact(b *domain.Board) bool {
	return backmost(b, domain.Black) < backmost(b, domain.White)
}

// backmost is the point a side still has to travel from: Black's highest
// occupied point, White's distance expressed on the same axis (so the two are
// directly comparable). A checker on the bar is behind every point.
func backmost(b *domain.Board, color int) int {
	if color == domain.Black {
		if b.Points[domain.BlackBar].Checkers > 0 {
			return domain.NumPoints + 1
		}
		for i := domain.NumPoints; i >= 1; i-- {
			if b.Points[i].Checkers > 0 && b.Points[i].Color == domain.Black {
				return i
			}
		}
		return 0
	}
	if b.Points[domain.WhiteBar].Checkers > 0 {
		return 0
	}
	for i := 1; i <= domain.NumPoints; i++ {
		if b.Points[i].Checkers > 0 && b.Points[i].Color == domain.White {
			return i
		}
	}
	return domain.NumPoints + 1
}

// activeCheckers counts a side's checkers outside its own points 1 and 2 —
// gnubg's measure of how much freedom it has left. Borne-off checkers are not
// counted: they are not on the board, and counting them would make a side that
// bore off cleanly look crunched.
func activeCheckers(b *domain.Board, color int) int {
	low, high := 3, domain.NumPoints+1 // Black: everything but 1, 2 and off
	if color == domain.White {
		low, high = 0, domain.NumPoints-2
	}
	n := 0
	for i := low; i <= high; i++ {
		if b.Points[i].Checkers > 0 && b.Points[i].Color == color {
			n += b.Points[i].Checkers
		}
	}
	return n
}

// anchorsIn counts the points in [from, to] where a side has two or more
// checkers — the definition of an anchor.
func anchorsIn(b *domain.Board, color, from, to int) int {
	n := 0
	for i := from; i <= to; i++ {
		if b.Points[i].Color == color && b.Points[i].Checkers >= 2 {
			n++
		}
	}
	return n
}

func anchorAt(b *domain.Board, color, point int) bool {
	return b.Points[point].Color == color && b.Points[point].Checkers >= 2
}

func hasAnchorIn(b *domain.Board, color, from, to int) bool {
	return anchorsIn(b, color, from, to) > 0
}

// homePointsMade counts the points a side has made in its own home board.
func homePointsMade(b *domain.Board, color int) int {
	if color == domain.Black {
		return anchorsIn(b, color, 1, 6)
	}
	return anchorsIn(b, color, 19, 24)
}

// blotsIn counts a side's lone checkers in [from, to].
func blotsIn(b *domain.Board, color, from, to int) int {
	n := 0
	for i := from; i <= to; i++ {
		if b.Points[i].Color == color && b.Points[i].Checkers == 1 {
			n++
		}
	}
	return n
}

// longestPrime is the longest run of consecutive points a side holds.
func longestPrime(b *domain.Board, color int) int {
	best, run := 0, 0
	for i := 1; i <= domain.NumPoints; i++ {
		if b.Points[i].Color == color && b.Points[i].Checkers >= 2 {
			run++
			if run > best {
				best = run
			}
		} else {
			run = 0
		}
	}
	return best
}

// trapped reports whether a side still has a checker behind the opponent's
// blockade — on the bar, or on a point the opponent's rearmost blocking point
// stands in front of.
func trapped(b *domain.Board, color int) bool {
	if color == domain.Black {
		// Black is trapped by White's points ahead of Black's rearmost checker
		// (White blocks from below on Black's axis).
		back := backmost(b, domain.Black)
		for i := back - 1; i >= 1; i-- {
			if b.Points[i].Color == domain.White && b.Points[i].Checkers >= 2 {
				return true
			}
		}
		return false
	}
	back := backmost(b, domain.White)
	for i := back + 1; i <= domain.NumPoints; i++ {
		if b.Points[i].Color == domain.Black && b.Points[i].Checkers >= 2 {
			return true
		}
	}
	return false
}

// pipOf is the side's pip count, borne-off checkers excluded.
func pipOf(b *domain.Board, color int) int {
	pips := 0
	if color == domain.Black {
		pips += b.Points[domain.BlackBar].Checkers * 25
		for i := 1; i <= domain.NumPoints; i++ {
			if b.Points[i].Color == domain.Black {
				pips += b.Points[i].Checkers * i
			}
		}
		return pips
	}
	pips += b.Points[domain.WhiteBar].Checkers * 25
	for i := 1; i <= domain.NumPoints; i++ {
		if b.Points[i].Color == domain.White {
			pips += b.Points[i].Checkers * (domain.NumPoints + 1 - i)
		}
	}
	return pips
}
