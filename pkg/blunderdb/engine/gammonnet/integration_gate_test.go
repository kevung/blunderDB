// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// TestIntegrationGate is #123, ADR-0014: does the whole port — codec, search,
// MET, cube — say sensible things about a real match at a real score with a
// real cube? Unit tests and gold files prove parity against the C reference;
// this replays real analysed matches and checks the answers against XG and
// gnubg, exactly as ADR-0014 lays out.
//
// Measured on 2026-08-29 (16 cores, the two gnubg fixtures + the xg fixture):
// 458.8s for 669 decisions (426 gnubg, 243 xg) at the canonical 2-ply k=12
// config — ADR-0014's "the per-decision cost... decides between a CI test
// and a pre-merge recipe step" is thereby settled: ~8 minutes here, and a
// typical CI runner has a fraction of the cores, so this is a recipe step,
// run deliberately, not part of `go test ./...`. Set BLUNDERDB_GATE to run
// it; BLUNDERDB_GATE_LIMIT truncates both corpora for a quick smoke pass.
//
// Result at the 2026-08-29 measurement (before ADR-0016's use_match): PASS.
// Criterion 3 (candidacy) 420/426 (1.4% missing, under the 5% line);
// criterion 1 (cost vs XG) 87 checked, max 0.0311 against the 0.05 block,
// with 32 decisions at a 1-away score excluded — the search was still
// money-only there while XG's stored equities are match equity, so the two
// were not comparable; criterion 2 (cube verdict vs XG) 91 checked, 0
// ND<->DP flips, 2 adjacent disagreements. See ADR-0014's "Update" section
// for the full account, including three real bugs this run found and fixed
// along the way (two in domain/moves.go's notation, one in ingest/xg.go's
// cube-response perspective) — none of them in gammonNet itself.
//
// Result at the 2026-08-31 measurement (ADR-0016's use_match, that exclusion
// deleted): FAIL, by exactly two decisions. Criterion 3 (candidacy) 421/426
// (1.2% missing, still under the line); criterion 1 (cost vs XG) 105
// checked — the 32 decisions that were entirely excluded before are now
// checked like any other, and 30 of them clear the 0.05 block cleanly — with
// two failing: score [1,5] dice [4,3] costs 0.0552, score [1,5] dice [1,1]
// costs 0.0738. Both are the SAME game, Black 1-away/White 5-away — a score
// where the cube's presence changes checker-play priorities the most, and
// this port's search is money-aware and MATCH-aware (ADR-0016) but still
// CUBELESS (search.go: "The search is CUBELESS... valuing nodes through the
// cube model at the LEAVES... is a documented future tranche" — use_cube,
// ADR-0016 point 7, deliberately deferred). XG analyses cubeful. A cubeless
// search disagreeing with a cubeful judge exactly at the score where the
// cube matters most, on 2 of 32 decisions once the other 30 already agree,
// is the SHAPE of the use_cube gap, not a new one — evidence for the next
// tranche's motivation, not a sign this one is wrong. Left failing rather
// than loosened: re-measure once use_cube lands, and only silence this if
// that tranche does not close it either.
//
// Result at the 2026-09-02 measurement (ADR-0023's use_cube, this file's own
// searcherFor now cubeful like ConfigForPosition): FAIL, by the SAME two
// decisions, at the SAME costs to the fourth decimal — 0.0552 and 0.0738.
// 765s. Criterion 3 (candidacy) 421/426 unchanged; criterion 1 106 checked
// (one more than before), median 0.0000; criterion 2 91 checked, 0 blocking
// flips, 2 adjacent disagreements.
//
// That identity REFUTES the paragraph above, and the refutation is the
// finding: every decision at score [1,5] in this fixture is in the CRAWFORD
// game (measured — the cube is absent and its owner is None throughout), so
// use_cube cannot move them by construction, and since gammonNet v1.2.1 it
// is guaranteed not to (the cube value in the Crawford game is the dead
// value, which is the cubeless one). The two outliers were never the cube's
// doing; a cubeless search agreeing with a cubeful judge at a score where
// there IS no cube is not evidence of anything. What is left to explain them
// is depth and table: this gate runs 2-ply k=12 against XG's own deeper
// setting and XG's own MET, on the score where the MET is most lopsided
// (1-away Crawford: the trailer needs the gammon, the leader needs nothing).
// Still left failing rather than loosened — the block is at 0.05 and two
// decisions sit just over it — but it is now a DEPTH/MET question to
// measure, not a missing tranche to write.
//
// Result of that depth/MET measurement (2026-09-03, #192/C.5, ADR-0029):
// THE HYPOTHESIS IS REFUTED TOO. TestMeasureGateRedCasesAtDepth replays every
// score-[1,5] checker decision of the xg fixture at three settings, and the
// two red ones do not move:
//
//	dice [4,3] "21/17 bar/22"        0.0552 / 0.0552 / 0.0552
//	dice [1,1] "10/9 13/12 6/5(2)"   0.0738 / (out of xg's net) / 0.0738
//	                                 2-ply k=12 | 2-ply unpruned | 3-ply k=12
//
// Same move chosen at every setting, same cost to the fourth decimal. And
// there is no "xg MET" left to try: engine/met.go IS Kazaross-XG2, xg's own
// default table — the sentence above should have said so. Pruning is not the
// suspect either, and not even monotone: at dice [5,2] turning it OFF made
// the cost worse, 0.0000 -> 0.0827.
//
// That is three hypotheses measured and refuted on the same two decisions —
// use_cube (ADR-0023), then depth/MET, then cube efficiency (ADR-0029, which
// cannot move them either: the Crawford game has no cube). What is left is
// the network's own judgement on two boards against xg's deeper analysis of
// them, which is not a configuration question. Still left failing, still not
// loosened; the next reader should not spend another run on the setting.
func TestIntegrationGate(t *testing.T) {
	if os.Getenv("BLUNDERDB_GATE") == "" {
		t.Skip("set BLUNDERDB_GATE to run the integration gate (~8 min at 2-ply k=12, measured 2026-08-29)")
	}

	gnubgFiles := []string{"../../../../testdata/test.sgf", "../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf"}
	xgFile := "../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg"

	cfg := DefaultConfig(2)
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	var gnubgDecisions []gateDecision
	for _, path := range gnubgFiles {
		mg, err := ingest.MapGnuBG(path)
		if err != nil {
			t.Fatalf("MapGnuBG(%s): %v", path, err)
		}
		gnubgDecisions = append(gnubgDecisions, extractDecisions(mg, "gnubg")...)
	}
	xgGraph, err := ingest.MapXG(xgFile)
	if err != nil {
		t.Fatalf("MapXG(%s): %v", xgFile, err)
	}
	xgDecisions := extractDecisions(xgGraph, "xg")

	// BLUNDERDB_GATE_LIMIT truncates both lists — a smoke run of the pipeline
	// (parsing, matching, MatchState orientation) at a fraction of the cost,
	// before committing to the full ~1h pass.
	if n, err := strconv.Atoi(os.Getenv("BLUNDERDB_GATE_LIMIT")); err == nil && n > 0 {
		if n < len(gnubgDecisions) {
			gnubgDecisions = gnubgDecisions[:n]
		}
		if n < len(xgDecisions) {
			xgDecisions = xgDecisions[:n]
		}
	}

	t.Logf("gnubg fixtures: %d decisions; xg fixture: %d decisions", len(gnubgDecisions), len(xgDecisions))

	// Criterion 3 — every chosen move appears among the arbiter's candidates
	// (judged against the gnubg fixtures, which store the widest net).
	//
	// ADR-0014 is explicit that a single miss is "a signal, not a severity":
	// gnubg's own list, wide as it is (19.4 candidates on average, up to
	// 221), is not a proof of exhaustiveness, so an occasional legitimate
	// move landing outside it is expected even from a correct port — what a
	// broken one (wrong colour, flipped perspective) produces is a HIGH
	// rate, not an isolated miss. candidacyBlockRate is the aggregate line:
	// each miss is still logged for manual inspection (every one measured
	// so far checked out as exactly that — gnubg's list simply stopping
	// short of an unusual legal move, not a matching bug), but only
	// crossing the rate fails the gate.
	const candidacyBlockRate = 0.05
	var candidacyChecked, candidacyMissing, candidacyUnresolved int
	for _, d := range gnubgDecisions {
		if d.kind != "checker" {
			continue
		}
		ours, ok := ourChoice(t, net, prune, cfg, d)
		if !ok {
			candidacyUnresolved++
			continue
		}
		candidacyChecked++
		if !inCandidateSet(ours.Notation, d.analysis.CheckerAnalysis.Moves) {
			candidacyMissing++
			t.Logf("candidacy: %s decision (score %v, dice %v): our move %q not among gnubg's %d candidates",
				d.source, d.pos.Score, d.dice, ours.Notation, len(d.analysis.CheckerAnalysis.Moves))
		}
	}
	if candidacyChecked > 0 {
		if rate := float64(candidacyMissing) / float64(candidacyChecked); rate > candidacyBlockRate {
			t.Errorf("candidacy: %d/%d (%.1f%%) of chosen moves missing from gnubg's candidates, over the %.0f%% line",
				candidacyMissing, candidacyChecked, rate*100, candidacyBlockRate*100)
		}
	}
	t.Logf("criterion 3 (candidacy): %d/%d checked, %d missing, %d unresolved (codec/search failure)",
		candidacyChecked-candidacyMissing, candidacyChecked, candidacyMissing, candidacyUnresolved)

	// Criterion 1 — no checker disagreement costs more than 0.05 equity,
	// judged by XG's own stored equities. Both are on the SAME scale now
	// (ADR-0016): the search values a match-score decision through the MET
	// (2×MWC−1), same as XG's own stored equities — there is no longer a
	// 1-away score excluded here. 30 of the 32 decisions this used to skip
	// entirely now clear the block; the other 2 do not, and the reason is
	// documented above this function (the use_cube gap, not a new bug).
	var costChecked, costOutOfNet, costUnresolved int
	var costs []float64
	const costBlock = 0.05
	for _, d := range xgDecisions {
		if d.kind != "checker" {
			continue
		}
		ours, ok := ourChoice(t, net, prune, cfg, d)
		if !ok {
			costUnresolved++
			continue
		}
		moves := d.analysis.CheckerAnalysis.Moves
		xgBest := bestEquity(moves)
		xgOurs, found := equityFor(ours.Notation, moves)
		if !found {
			costOutOfNet++
			continue
		}
		costChecked++
		cost := xgBest - xgOurs
		costs = append(costs, cost)
		if cost > costBlock {
			t.Errorf("cost: xg decision (score %v, dice %v): our move %q costs %.4f equity (xg best %.4f, ours %.4f)",
				d.pos.Score, d.dice, ours.Notation, cost, xgBest, xgOurs)
		}
	}
	sort.Float64s(costs)
	t.Logf("criterion 1 (cost vs xg): %d checked (%d outside xg's candidate net, %d unresolved), max=%.4f median=%.4f",
		costChecked, costOutOfNet, costUnresolved, percentile(costs, 1.0), percentile(costs, 0.5))

	// Criterion 2 — no ND<->DP cube verdict flip, in either direction,
	// against XG. Adjacent flips (ND<->DT, DT<->DP) are reported, not blocking.
	var cubeChecked, ndDpFlips, adjacentFlips, cubeUnresolved int
	for _, d := range xgDecisions {
		if d.kind != "cube" {
			continue
		}
		ours, ok := ourCubeAction(t, net, prune, cfg, d)
		if !ok {
			cubeUnresolved++
			continue
		}
		xgAction, ok := xgCubeBucket(d.analysis.DoublingCubeAnalysis.BestCubeAction)
		if !ok {
			cubeUnresolved++
			t.Logf("cube: unrecognised BestCubeAction %q, skipped", d.analysis.DoublingCubeAnalysis.BestCubeAction)
			continue
		}
		cubeChecked++
		if isNDDPFlip(ours, xgAction) {
			ndDpFlips++
			t.Errorf("cube: xg decision (score %v, cube %v): we say %v, xg says %v — ND<->DP flip",
				d.pos.Score, d.pos.Cube, ours, xgAction)
		} else if ours != xgAction {
			adjacentFlips++
		}
	}
	t.Logf("criterion 2 (cube verdict vs xg): %d checked (%d unresolved), %d ND<->DP flips (blocking), %d adjacent disagreements (noise)",
		cubeChecked, cubeUnresolved, ndDpFlips, adjacentFlips)
}

// gateDecision is one checker or cube decision extracted from a match graph,
// with enough context (Crawford in particular) to build a MatchState.
type gateDecision struct {
	source   string // "gnubg" or "xg", for logging only
	kind     string // domain.Move.MoveType: "checker" or "cube"
	pos      *domain.Position
	dice     [2]int
	analysis *domain.PositionAnalysis
	crawford bool
}

// extractDecisions walks every game and move of a parsed match, keeping only
// analysed, non-forced decisions.
func extractDecisions(mg *ingest.MatchGraph, source string) []gateDecision {
	var out []gateDecision
	for _, g := range mg.Games {
		crawford := isCrawfordGame(mg, g)
		for _, mv := range g.Moves {
			if mv.Position == nil || len(mv.Analyses) == 0 {
				continue
			}
			an := mv.Analyses[0]
			switch mv.Move.MoveType {
			case "checker":
				if an.CheckerAnalysis == nil || len(an.CheckerAnalysis.Moves) < 2 {
					continue // no analysis, or forced (nothing to disagree about)
				}
			case "cube":
				if an.DoublingCubeAnalysis == nil {
					continue
				}
				// A "cube" move node is either the doubler's own evaluation
				// (Move.CubeAction "No Double" or "Double") or a synthesized
				// companion node recording the RESPONSE ("Take"/"Pass") at
				// the mirrored position. The response node reuses the
				// doubler's analysis without swapping Player/Opponent
				// chances (a real bug found via this gate — ingest/xg.go's
				// mapDoubleTakeMove and mapSingleCubeMove), and even fixed
				// it would answer a different question ("should the
				// responder take") than Decide computes ("should the player
				// on roll double"). Only the doubler's own node is a fair
				// comparison for criterion 2.
				if mv.Move.CubeAction != "No Double" && mv.Move.CubeAction != "Double" {
					continue
				}
			default:
				continue
			}
			out = append(out, gateDecision{
				source:   source,
				kind:     mv.Move.MoveType,
				pos:      mv.Position,
				dice:     [2]int{int(mv.Move.Dice[0]), int(mv.Move.Dice[1])},
				analysis: an,
				crawford: crawford,
			})
		}
	}
	return out
}

// isCrawfordGame reports whether g is the FIRST game of the match where a
// player starts at match point — the Crawford game, by definition. Derived
// from InitialScore/MatchLength rather than read from a per-game flag: both
// gnubgparser and xgparser carry one, but neither is wired through
// MatchGraph yet, and a real 7-point match has at most one Crawford game, so
// the derivation is exercised on exactly the cases that matter here.
func isCrawfordGame(mg *ingest.MatchGraph, g ingest.GameGraph) bool {
	ml := int(mg.Match.MatchLength)
	if ml <= 0 {
		return false
	}
	atMatchPoint := func(gg ingest.GameGraph) bool {
		return int(gg.Game.InitialScore[0]) == ml-1 || int(gg.Game.InitialScore[1]) == ml-1
	}
	if !atMatchPoint(g) {
		return false
	}
	for _, other := range mg.Games {
		if other.Game.GameNumber < g.Game.GameNumber && atMatchPoint(other) {
			return false // Crawford already played
		}
	}
	return true
}

// matchStateFor builds the MatchState and CubeOwner a decision needs, from
// pos.Score (away, [Black, White] — domain's xgid.go convention) and
// pos.Cube, relative to pos.PlayerOnRoll. Built on MatchStateFromScores
// (ADR-0016's one translation) rather than a second copy of the away-score
// and cube-exponent decode — this file's own past copy had the cube bug
// MatchStateFromScores fixes: pos.Cube.Value is blunderDB's log2 exponent
// convention (0,1,2,… for cube 1,2,4,…), not a literal cube value, and the
// old inline version here fed it through as if it already were one. ok is
// false for a money position (Score sentinel [-1,-1]) or a state
// MatchStateFromScores refuses: every decision in these fixtures is match
// play, so either would signal a mapping bug, not a legitimate input.
func matchStateFor(pos *domain.Position, crawford bool) (MatchState, CubeOwner, bool) {
	if pos.Score[0] < 0 || pos.Score[1] < 0 {
		return MatchState{}, CubeCentred, false
	}
	var awayOnRoll, awayOpponent int
	if pos.PlayerOnRoll == domain.Black {
		awayOnRoll, awayOpponent = pos.Score[0], pos.Score[1]
	} else {
		awayOnRoll, awayOpponent = pos.Score[1], pos.Score[0]
	}
	var owner CubeOwner
	switch pos.Cube.Owner {
	case domain.None:
		owner = CubeCentred
	case pos.PlayerOnRoll:
		owner = CubeOwned
	default:
		owner = CubeOpponent
	}
	state, ok := MatchStateFromScores(awayOnRoll, awayOpponent, pos.Cube.Value, crawford)
	return state, owner, ok
}

// searcherFor builds a Searcher over the shared networks, configured for
// state (nil for money) and for the cube owner — one per decision, since a
// Searcher is bound to one Match and one cube for its whole life exactly as
// it is bound to one Ply (SearchConfig's UseMatch/Match doc comment). Cheap:
// net/prune are the same singletons every call, so this allocates scratch
// buffers, nothing more. WithWorkers(16) matches what the single shared
// searcher this file used to build once — the two gnubg fixtures plus the xg
// fixture cross many different scores, so there is no single Searcher left to
// share.
//
// UseCube is on (ADR-0023), as it is in ConfigForPosition — the gate judges
// the search the application actually runs, and both arbiters here (XG and
// gnubg) analyse cubeful.
func searcherFor(t *testing.T, net, prune *Network, cfg SearchConfig, state *MatchState, owner CubeOwner) (*Searcher, bool) {
	t.Helper()
	if state != nil {
		if !state.IsValid() {
			return nil, false
		}
		cfg.UseMatch = true
		cfg.Match = *state
	} else {
		cfg.UseMatch = false
	}
	cfg.UseCube = true
	cfg.CubeOwner = owner
	cfg.CubeX = DefaultEfficiency(owner)
	s := newSearcherWith(cfg, net, prune)
	s.WithWorkers(16)
	return s, true
}

// ourChoice runs our own 2-ply search on d, at d's own score (ADR-0016 — a
// fresh searcherFor per decision, since the fixtures cross many scores), and
// returns the resulting domain.LegalPlay it chose — Notation in the SAME
// dialect as the stored candidate lists (domain/moves.go's notation() was
// built for exactly this comparison) and Result for identifying it. ok is
// false when the position or dice are unusable, the score is not evaluable,
// or there is nothing to choose between.
func ourChoice(t *testing.T, net, prune *Network, cfg SearchConfig, d gateDecision) (domain.LegalPlay, bool) {
	t.Helper()
	pos := *d.pos
	pos.Dice = [2]int{d.dice[0], d.dice[1]}
	legal := domain.LegalMoves(&pos)
	if len(legal) < 2 {
		return domain.LegalPlay{}, false
	}

	var state *MatchState
	if pos.Score[0] >= 0 && pos.Score[1] >= 0 {
		m, _, ok := matchStateFor(&pos, d.crawford)
		if !ok {
			return domain.LegalPlay{}, false
		}
		state = &m
	}
	s, ok := searcherFor(t, net, prune, cfg, state, CubeOwnerOf(&pos))
	if !ok {
		return domain.LegalPlay{}, false
	}

	gpos, err := FromDomain(&pos)
	if err != nil {
		t.Logf("FromDomain failed for a decision: %v", err)
		return domain.LegalPlay{}, false
	}
	best, ok, err := s.BestPlay(&gpos, d.dice[0], d.dice[1])
	if err != nil || !ok {
		t.Logf("BestPlay failed for a decision: %v", err)
		return domain.LegalPlay{}, false
	}

	opponent := domain.White
	if pos.PlayerOnRoll == domain.White {
		opponent = domain.Black
	}
	for _, play := range legal {
		// domain.LegalMoves leaves Result.PlayerOnRoll at the MOVER; our own
		// generator switches it to the opponent on the result (search.go's
		// perspective rule). Comparing without matching that convention
		// first fails every play, chosen or not — see moves_diff_test.go's
		// boardKey, which excludes the turn for the same reason.
		res := play.Result
		res.PlayerOnRoll = opponent
		gresult, err := FromDomain(&res)
		if err != nil {
			continue
		}
		if gresult == best.Play.Result {
			return play, true
		}
	}
	t.Logf("our chosen play (score %v, dice %v) matched none of %d domain.LegalMoves — codec bug?",
		d.pos.Score, d.dice, len(legal))
	return domain.LegalPlay{}, false
}

// inCandidateSet reports whether notation (our dialect) matches one of the
// stored candidates' Move fields, up to NormalizeMove's token-order folding.
func inCandidateSet(notation string, moves []domain.CheckerMove) bool {
	_, found := equityFor(notation, moves)
	return found
}

// normalizeForMatching identifies a move by its multiset of "From/To" steps,
// stripped of hit markers and "(n)" collapsing, sorted. Two gaps in a plain
// engine.NormalizeMove comparison, both found against test.sgf while
// building #123:
//
//   - gnubg's SGF candidates render "6/3 3/1" for a move that hits on the
//     way, never "6/3* 3/1*" — XG's candidates DO carry the "*". The hit is
//     a derived fact about the board, not part of which move was chosen.
//   - domain/moves.go collapses repeated steps by their FULL token
//     (including the hit marker), so a double move landing on the same point
//     twice — hitting only the first time, since the second lands on what is
//     now a friendly point — renders as two separate tokens, "4/2 4/2*",
//     where a stored candidate collapses both into "4/2(2)". Expanding every
//     "(n)" back into n repeated bare tokens before sorting makes the two
//     forms compare equal regardless of which side collapsed what.
func normalizeForMatching(move string) string {
	var steps []string
	for _, tok := range strings.Fields(move) {
		tok = strings.ReplaceAll(tok, "*", "")
		count := 1
		if i := strings.IndexByte(tok, '('); i >= 0 && strings.HasSuffix(tok, ")") {
			if n, err := strconv.Atoi(tok[i+1 : len(tok)-1]); err == nil && n > 0 {
				count = n
			}
			tok = tok[:i]
		}
		for i := 0; i < count; i++ {
			steps = append(steps, tok)
		}
	}
	sort.Strings(steps)
	return strings.Join(steps, " ")
}

// equityFor returns the stored equity of the candidate matching notation.
func equityFor(notation string, moves []domain.CheckerMove) (float64, bool) {
	norm := normalizeForMatching(notation)
	for _, m := range moves {
		if normalizeForMatching(m.Move) == norm {
			return m.Equity, true
		}
	}
	return 0, false
}

// bestEquity is the top-ranked candidate's equity. Candidate lists in these
// fixtures are stored best-first; sorting defensively costs nothing here.
func bestEquity(moves []domain.CheckerMove) float64 {
	best := math.Inf(-1)
	for _, m := range moves {
		if m.Equity > best {
			best = m.Equity
		}
	}
	return best
}

// ourCubeAction runs our own 2-ply cube decision (Probs at depth, then
// Decide), at d's own score (ADR-0016 — Probs is match-aware too, so this
// needs its own searcherFor exactly as ourChoice does) and buckets it the
// same coarse way xgCubeBucket does: TooGood folds into NoDouble, matching
// the "don't offer the cube" reading ADR-0014 treats them as sharing for
// this gate's purposes.
func ourCubeAction(t *testing.T, net, prune *Network, cfg SearchConfig, d gateDecision) (CubeAction, bool) {
	t.Helper()
	state, owner, ok := matchStateFor(d.pos, d.crawford)
	if !ok {
		return 0, false
	}
	s, ok := searcherFor(t, net, prune, cfg, &state, owner)
	if !ok {
		return 0, false
	}
	gpos, err := FromDomain(d.pos)
	if err != nil {
		t.Logf("FromDomain failed for a cube decision: %v", err)
		return 0, false
	}
	probs, ok := s.Probs(&gpos)
	if !ok {
		t.Logf("Probs failed for a cube decision (score %v)", d.pos.Score)
		return 0, false
	}
	dec, ok := Decide(&probs, owner, &state, DefaultEfficiency(owner), d.pos.HasJacoby != 0)
	if !ok {
		return 0, false
	}
	if dec.Action == TooGood {
		return NoDouble, true
	}
	return dec.Action, true
}

// xgCubeBucket maps XG's BestCubeAction label onto our CubeAction, folding
// "too good" into NoDouble — see engine.BestCubeVerdict, the single place
// that already knows every spelling these labels come in.
func xgCubeBucket(label string) (CubeAction, bool) {
	v, ok := engine.BestCubeVerdict(label)
	if !ok {
		return 0, false
	}
	if !v.ShouldDouble {
		return NoDouble, true
	}
	if v.ShouldPass {
		return DoublePass, true
	}
	return DoubleTake, true
}

// isNDDPFlip is ADR-0014's blocking cube criterion: a disagreement between
// No Double and Double/Pass, in either direction. Adjacent disagreements
// (ND<->DT, DT<->DP) are boundary noise and return false.
func isNDDPFlip(a, b CubeAction) bool {
	return (a == NoDouble && b == DoublePass) || (a == DoublePass && b == NoDouble)
}

// percentile is a simple sorted-slice percentile (q in [0,1]); empty input
// reports 0 rather than NaN, which reads more cleanly in a log line.
func percentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	return sorted[idx]
}
