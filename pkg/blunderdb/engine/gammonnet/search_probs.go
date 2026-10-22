// SPDX-License-Identifier: MIT

package gammonnet

// The pre-roll distribution at depth — ported from gn_search.c's
// position_probs/gn_search_probs (t34-videau-spec §8, step 1).
//
// A cube decision needs the five probabilities BEFORE the dice are rolled,
// not a scalar equity. This is a separate walk from positionEquity, so the
// equity path the parity tests hold byte-for-byte stays untouched; it calls
// rankPlays, so both walks always pick the same play. Paid once per cube
// decision, mostly from a warm cache.

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

// terminalProbs is the distribution of a finished game, all mass on the
// outcome that happened; like terminalEquity, Turn names the LOSER.
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

// Probs is the pre-roll distribution of pos at the searcher's depth, from
// pos.Turn's view — the §8 companion of BestPlay's equity. At depth 0 the
// network's raw answer; deeper, the roll-weighted average of the distribution
// of the play BestPlay's valuation would choose, inverted once per ply. A
// test holds moneyEquity(Probs(pos)) to BestPlay's equity.
func (s *Searcher) Probs(pos *Position) ([NumOutputs]float32, bool) {
	// The one validation of this walk, as Searcher.Plays does for the other:
	// everything below is either this position or a play generated from it,
	// and encodeLegal takes the rest on construction.
	if !pos.Valid() {
		return [NumOutputs]float32{}, false
	}
	return s.probsAt(pos, s.cfg.Ply, 0, s.matchState(), s.cfg.CubeOwner)
}

// probsAt is the recursion. state and owner are as pos's OWN mover sees
// them, swapped and mirrored on the way down as gn_search.c's position_probs
// does — never re-read from the root, which is silently wrong only at an
// asymmetric score from 2 ply (TestProbsMatchEquityMatchesPositionEquity).
func (s *Searcher) probsAt(pos *Position, depth, level int, state *MatchState, owner CubeOwner) ([NumOutputs]float32, bool) {
	if level == 0 {
		s.seedMatchState(state) // fixes matchStates[0]/[1] for this whole chain
	}
	if pos.isOver() {
		return terminalProbs(pos), true
	}
	if depth <= 0 {
		var probs [NumOutputs]float32
		if !s.cache.lookup(pos, &probs) {
			encodeLegal(pos, &s.feat)
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

	// À la racine d'un chercheur parallèle, les 21 lancers vont dans une
	// file combinée ; ailleurs, un lancer par emplacement, sur plusieurs
	// cœurs à partir de splitDepth, accumulés après la barrière en index de
	// lancer croissant : terme pour terme la boucle sérielle.
	if level == 0 && s.parallel() {
		return s.probsAtRootParallel(pos, depth, state, owner)
	}

	var mine [NumRolls][NumOutputs]float32
	if depth >= splitDepth && s.parallel() {
		ok := s.fork(NumRolls, func(w *Searcher, k int) bool {
			r := rollsByCost[k]
			var ok bool
			mine[r], ok = w.rollProbs(pos, depth, level, r, state, owner)
			return ok
		})
		if !ok {
			return [NumOutputs]float32{}, false
		}
	} else {
		for r := 0; r < NumRolls; r++ {
			var ok bool
			if mine[r], ok = s.rollProbs(pos, depth, level, r, state, owner); !ok {
				return [NumOutputs]float32{}, false
			}
		}
	}
	return s.weighRolls(&mine), true
}

// rollProbs is one roll's term of probsAt, from pos's mover's view: the best
// reply's distribution at depth-1, inverted.
func (s *Searcher) rollProbs(pos *Position, depth, level, r int, state *MatchState, owner CubeOwner) ([NumOutputs]float32, bool) {
	roll := s.rolls[r]
	cands := s.candsAt(level)
	n := s.rankPlays(pos, int(roll.d1), int(roll.d2), depth-1, level, state, owner, cands)
	if n < 0 {
		return [NumOutputs]float32{}, false
	}
	var theirs [NumOutputs]float32
	var ok bool
	if n > 0 {
		// The best play's own distribution, at the depth its equity was
		// scored at (depth-1) — mirroring -V(result, depth-1).
		theirs, ok = s.probsAt(&cands[0].Play.Result, depth-1, level+1, s.childMatchState(level), owner.Mirror())
	} else {
		// No legal play: the turn passes, exactly as the scalar
		// recursion does — dropping the branch would bias the average.
		passed := *pos
		passed.swapTurn()
		theirs, ok = s.probsAt(&passed, depth-1, level+1, s.childMatchState(level), owner.Mirror())
	}
	if !ok {
		return [NumOutputs]float32{}, false
	}
	return invertProbs(&theirs), true
}

// weighRolls is the roll-weighted average of the 21 terms, accumulated in
// float64 in ascending roll index, whoever computed them.
func (s *Searcher) weighRolls(mine *[NumRolls][NumOutputs]float32) [NumOutputs]float32 {
	var total [NumOutputs]float64
	for r := 0; r < NumRolls; r++ {
		for i := range total {
			total[i] += float64(s.rolls[r].weight * float64(mine[r][i]))
		}
	}
	var out [NumOutputs]float32
	for i := range out {
		out[i] = float32(total[i])
	}
	return out
}

// probsAtRootParallel est la racine de probsAt, aplatie comme la phase trois
// de rankPlays (deepenGroups) :
//
// Pass A (sérielle) classe chaque lancer dans son propre brouillon
// (s.probeCands[r]), qui doit survivre jusqu'à la fin de B ; un lancer qui
// danse ne rentre dans aucun groupe.
//
// Pass B (deepenGroups) approfondit tous les groupes : une file, une
// barrière.
//
// Pass C reproduit la réduction d'origine : retrier le groupe, prendre le
// meilleur, recourir à probsAt (un lancer par tâche), inverser, puis
// accumuler pondéré en float64 en index de lancer croissant après la
// barrière. La parallélisation ne change que qui calcule, jamais l'ordre de
// la somme.
func (s *Searcher) probsAtRootParallel(pos *Position, depth int, state *MatchState, owner CubeOwner) ([NumOutputs]float32, bool) {
	// Always level 0 — probsAt only ever branches here from its own level ==
	// 0 check, which has already seeded matchStates[0]/[1] for this chain.
	theirs := s.childMatchState(0)
	theirOwner := owner.Mirror()

	groups := s.probeGroups[:0]
	scratch := s.candsAt(0)

	for r := 0; r < NumRolls; r++ {
		roll := s.rolls[r]
		n := s.rankPlaysShallow(pos, int(roll.d1), int(roll.d2), depth-1, 0, state, owner, scratch)
		if n < 0 {
			return [NumOutputs]float32{}, false
		}
		if n == 0 {
			s.probeDanced[r] = true
			s.probePassed[r] = *pos
			s.probePassed[r].swapTurn()
			s.probeCands[r] = s.probeCands[r][:0]
			continue
		}
		s.probeDanced[r] = false
		searched := s.cfg.filterSurvivors(depth-1, scratch, n)
		if cap(s.probeCands[r]) < searched {
			s.probeCands[r] = make([]Candidate, searched)
		}
		s.probeCands[r] = s.probeCands[r][:searched]
		copy(s.probeCands[r], scratch[:searched])
		groups = append(groups, s.probeCands[r])
	}
	s.probeGroups = groups

	if !s.deepenGroups(groups, depth-1, theirs, theirOwner) {
		return [NumOutputs]float32{}, false
	}

	// Pass C : le tri de chaque groupe d'abord, sur l'appelant ; puis la
	// distribution du meilleur coup de chaque lancer, un emplacement par
	// lancer, accumulée après la barrière en index croissant.
	var results [NumRolls]*Position
	for r := 0; r < NumRolls; r++ {
		if s.probeDanced[r] {
			results[r] = &s.probePassed[r]
		} else {
			sortByEquity(s.probeCands[r])
			results[r] = &s.probeCands[r][0].Play.Result
		}
	}
	var mine [NumRolls][NumOutputs]float32
	ok := s.fork(NumRolls, func(w *Searcher, k int) bool {
		r := rollsByCost[k]
		theirsProbs, ok := w.probsAt(results[r], depth-1, 1, theirs, theirOwner)
		mine[r] = invertProbs(&theirsProbs)
		return ok
	})
	if !ok {
		return [NumOutputs]float32{}, false
	}
	return s.weighRolls(&mine), true
}
