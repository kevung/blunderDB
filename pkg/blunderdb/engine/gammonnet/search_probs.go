// SPDX-License-Identifier: MIT

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
// holds moneyEquity(Probs(pos)) == the equity BestPlay's own recursion
// assigns to pos, rather than trusting the identity.
func (s *Searcher) Probs(pos *Position) ([NumOutputs]float32, bool) {
	// The one validation of this walk, as Searcher.Plays does for the other:
	// everything below is either this position or a play generated from it,
	// and encodeLegal takes the rest on construction.
	if !pos.Valid() {
		return [NumOutputs]float32{}, false
	}
	return s.probsAt(pos, s.cfg.Ply, 0, s.matchState(), s.cfg.CubeOwner)
}

// probsAt is the recursion. state and owner are the match state and the
// cube AS pos's OWN MOVER SEES THEM, swapped and mirrored on the way down
// exactly as gn_search.c's position_probs does and as positionEquity does —
// never re-read from the root. Until ADR-0023 this walk passed the ROOT's
// state to every level, so at 2 ply the opponent's replies were ranked from
// the root player's side of the score: exact at 1 ply (the inner call is a
// leaf) and at every symmetric score, and silently wrong for the 2-ply cube
// decision at 4-away/2-away. TestProbsMatchEquityMatchesPositionEquity is
// the identity that catches it.
func (s *Searcher) probsAt(pos *Position, depth, level int, state *MatchState, owner CubeOwner) ([NumOutputs]float32, bool) {
	if level == 0 {
		s.seedMatchState(state) // #197/C.10: fixes matchStates[0]/[1] for this whole chain
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

	// À la racine — level 0, la seule où s.workers est jamais consulté (les
	// ouvriers n'ont pas d'ouvriers à eux, #195/C.8) — les 21 lancers PROPRES
	// vont dans une file combinée au lieu d'ouvrir chacun sa propre barrière ;
	// ailleurs, la boucle sérielle ci-dessous, terme pour terme identique.
	if level == 0 && len(s.workers) > 0 {
		return s.probsAtRootParallel(pos, depth, state, owner)
	}

	cands := s.candsAt(level)

	var total [NumOutputs]float64
	for r := 0; r < NumRolls; r++ {
		roll := s.rolls[r]
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

// probsAtRootParallel est probsAt's own root loop, aplati exactement comme
// la phase trois de rankPlays l'est (deepenGroups, #195/C.8) : au lieu de
// classer chacun des 21 lancers propres tour à tour et de laisser SON
// rankPlays ouvrir puis refermer sa propre barrière avant que le suivant ne
// commence, les candidats à approfondir de chaque lancer sont d'abord
// rassemblés, puis approfondis TOUS ENSEMBLE dans une seule file.
//
// Pass A (sérielle, bon marché — un tri de plateaux frères par lancer, déjà
// groupé sur le noyau) classe chaque lancer et garde ses meilleurs candidats
// dans un brouillon qui lui est propre (s.probeCands[r]) : contrairement à
// candsAt(level), qui serait écrasé par le lancer suivant, ce brouillon doit
// rester valide jusqu'à ce que TOUS les lancers soient approfondis. Une
// danse (aucun coup légal) ne rentre dans aucun groupe ; la position après
// passage est ce qu'il y a à approfondir, et elle n'a besoin d'aucune
// recherche supplémentaire à CE niveau.
//
// Pass B (deepenGroups) approfondit tous les groupes non vides d'un coup :
// une file, une barrière, au lieu de 21.
//
// Pass C (sérielle, en index de lancer croissant) reproduit exactement la
// réduction de la boucle d'origine : trier le groupe du lancer (l'équité que
// deepenGroups vient d'écrire change l'ordre), prendre le meilleur, recourir
// à probsAt pour SA distribution, l'inverser, l'accumuler pondérée — la
// somme reste sérielle et en float64, ce que la parallélisation change
// n'est jamais que qui classe et qui approfondit chaque lancer.
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
		searched := n
		if f := s.cfg.Filter[depth-1]; f > 0 && f < searched {
			searched = f
		}
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

	var total [NumOutputs]float64
	for r := 0; r < NumRolls; r++ {
		roll := s.rolls[r]

		var result *Position
		if s.probeDanced[r] {
			result = &s.probePassed[r]
		} else {
			sortByEquity(s.probeCands[r])
			result = &s.probeCands[r][0].Play.Result
		}

		theirsProbs, ok := s.probsAt(result, depth-1, 1, theirs, theirOwner)
		if !ok {
			return [NumOutputs]float32{}, false
		}

		mine := invertProbs(&theirsProbs)
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
