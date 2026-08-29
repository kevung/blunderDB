package gammonnet

// The pre-roll distribution at depth — ported from gn_search.c's
// position_probs/gn_search_probs (t34-videau-spec §8, step 1).
//
// A cube decision needs the five probabilities of the position BEFORE the
// dice are rolled, not the scalar equity the move-choice recursion returns —
// P(gammon) and P(backgammon) matter to a cube decision in their own right,
// separately from what they contribute to an equity. This is a SEPARATE walk
// from positionEquity, re-searching one subtree per node, because widening
// positionEquity to also carry a distribution would touch the equity path
// that the search's own parity tests hold byte-for-byte. The distribution
// follows exactly the play the scalar recursion would choose — rankPlays is
// called, never reimplemented, so the two walks can never pick different
// moves out from under each other.
//
// The price is paid only when a caller actually asks for a distribution:
// once per cube decision, never per move choice. With the evaluation cache
// already warm from ranking the same plays, most of the re-walk's network
// evaluations are hits.

// invertProbs is the same distribution, seen from the other side of the
// table. The nested encoding makes this a swap plus one complement: my
// gammon losses are the opponent's gammon wins, and P(win) partitions.
func invertProbs(in *[NumOutputs]float32) [NumOutputs]float32 {
	var out [NumOutputs]float32
	out[PWin] = 1.0 - in[PWin]
	out[PWinGammon] = in[PLoseGammon]
	out[PWinBackgammon] = in[PLoseBackgammon]
	out[PLoseGammon] = in[PWinGammon]
	out[PLoseBackgammon] = in[PWinBackgammon]
	return out
}

// InvertProbs is invertProbs, exported for cold-path callers (the Eval panel,
// #125) that need a candidate's resulting-position distribution — which
// Candidate.Probs documents as the OPPONENT's point of view — flipped back to
// the mover's, without keeping a second copy of the perspective swap this
// package is otherwise so careful about.
func InvertProbs(in *[NumOutputs]float32) [NumOutputs]float32 {
	return invertProbs(in)
}

// terminalProbs is the distribution of a finished game: all mass on the one
// outcome that happened. Computed, never evaluated, like terminalEquity —
// and sharing its convention that Turn names the LOSER at a terminal
// position.
func terminalProbs(p *Position) [NumOutputs]float32 {
	stake := gameValue(p)
	weWon := int(p.Turn) == p.winner()

	var out [NumOutputs]float32
	if weWon {
		out[PWin] = 1
	}
	if weWon && stake >= 2 {
		out[PWinGammon] = 1
	}
	if weWon && stake >= 3 {
		out[PWinBackgammon] = 1
	}
	if !weWon && stake >= 2 {
		out[PLoseGammon] = 1
	}
	if !weWon && stake >= 3 {
		out[PLoseBackgammon] = 1
	}
	return out
}

// Probs is the pre-roll distribution of pos at the searcher's configured
// depth, from pos.Turn's point of view — the §8 companion of BestPlay's
// equity. At depth 0 it is exactly the network's raw answer for pos. Deeper,
// it is the roll-weighted average of the best play's own distribution, one
// perspective inversion per ply: the best play being the one the SAME
// valuation BestPlay uses would choose, so the distribution describes the
// game the search would actually play.
//
// Both money and match valuations are linear in this distribution — a test
// holds MoneyEquity(Probs(pos)) == the equity BestPlay's own recursion
// assigns to pos, rather than trusting the identity.
func (s *Searcher) Probs(pos *Position) ([NumOutputs]float32, bool) {
	return s.probsAt(pos, s.cfg.Ply, 0)
}

func (s *Searcher) probsAt(pos *Position, depth, level int) ([NumOutputs]float32, bool) {
	if pos.isOver() {
		return terminalProbs(pos), true
	}
	if depth <= 0 {
		var probs [NumOutputs]float32
		if !s.cache.lookup(pos, &probs) {
			if !Encode(pos, &s.feat) {
				return [NumOutputs]float32{}, false
			}
			if err := s.ev.Evaluate(s.feat[:], &probs); err != nil {
				return [NumOutputs]float32{}, false
			}
			s.evals++
			s.cache.store(pos, &probs)
		} else {
			s.cacheHits++
		}
		return probs, true
	}
	if level >= len(s.cands) {
		return [NumOutputs]float32{}, false
	}
	cands := s.cands[level]

	var total [NumOutputs]float64
	for r := 0; r < NumRolls; r++ {
		roll := s.rolls[r]
		n := s.rankPlays(pos, int(roll.d1), int(roll.d2), depth-1, level, cands)
		if n < 0 {
			return [NumOutputs]float32{}, false
		}

		var theirs [NumOutputs]float32
		var ok bool
		if n > 0 {
			// The best play's own distribution, at the depth its equity was
			// scored at (depth-1) — mirroring -V(result, depth-1).
			theirs, ok = s.probsAt(&cands[0].Play.Result, depth-1, level+1)
		} else {
			// No legal play: the turn passes, exactly as the scalar
			// recursion does — dropping the branch would bias the average.
			passed := *pos
			passed.swapTurn()
			theirs, ok = s.probsAt(&passed, depth-1, level+1)
		}
		if !ok {
			return [NumOutputs]float32{}, false
		}

		mine := invertProbs(&theirs)
		for i := range total {
			total[i] += roll.weight * float64(mine[i])
		}
	}

	var out [NumOutputs]float32
	for i := range out {
		out[i] = float32(total[i])
	}
	return out, true
}
