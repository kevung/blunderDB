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

// TestIntegrationGate (ADR-0014) replays real analysed matches at real
// scores with real cubes and checks the whole port's answers against XG and
// gnubg; unit tests and gold files only prove parity with the C.
//
// ~8 minutes on 16 cores for 669 decisions at 2-ply k=12: a deliberate
// pre-merge step, not part of `go test ./...`. Set BLUNDERDB_GATE to run it;
// BLUNDERDB_GATE_LIMIT truncates both corpora for a smoke pass.
//
// It FAILS by two decisions, both in the same Crawford game at score [1,5]:
// dice [4,3] costs 0.0552 and dice [1,1] 0.0738 against the 0.05 block.
// Measured and ruled out: use_cube (ADR-0023; no cube in the Crawford game),
// depth and pruning (TestMeasureGateRedCasesAtDepth: same move, same cost at
// 2-ply unpruned and 3-ply), the MET (engine/met.go is XG's own
// Kazaross-XG2), and cube efficiency (ADR-0029). What is left is the
// network's own judgement on two boards; left failing, not loosened, and not
// worth another run on the settings.
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
	// A single miss is "a signal, not a severity" (ADR-0014): gnubg's list is
	// not exhaustive, while a broken port (wrong colour, flipped perspective)
	// misses at a HIGH rate. Misses are logged; only the rate fails the gate.
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
	// judged by XG's stored equities, on the same scale (ADR-0016). The two
	// known failures are documented above this function.
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
				// Only the doubler's own node ("No Double"/"Double") answers
				// Decide's question; a synthesized "Take"/"Pass" response
				// node answers the responder's.
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

// isCrawfordGame reports whether g is the first game where a player starts at
// match point. Derived from InitialScore/MatchLength because the parsers'
// per-game flag is not wired through MatchGraph.
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

// matchStateFor builds the MatchState and CubeOwner a decision needs from
// pos.Score (away, [Black, White]) and pos.Cube (a log2 exponent), relative
// to pos.PlayerOnRoll, through MatchStateFromScores. ok is false for money or
// a refused state — a mapping bug here, since every fixture is match play.
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

// searcherFor builds a Searcher for state (nil for money) and the cube owner,
// one per decision since the fixtures cross many scores. UseCube is on
// (ADR-0023), as in ConfigForPosition: the gate judges the search the
// application runs, and both arbiters analyse cubeful.
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

// ourChoice runs our 2-ply search on d at d's own score and returns the
// domain.LegalPlay it chose, its Notation in the stored candidates' dialect.
// ok is false when the position, dice or score is unusable, or there is
// nothing to choose between.
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
		// domain.LegalMoves leaves Result.PlayerOnRoll at the mover; our
		// generator switches it (search.go's perspective rule). Match that
		// first, or every play fails.
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

// normalizeForMatching identifies a move by its sorted multiset of "From/To"
// steps, without hit markers or "(n)" collapsing, because:
//   - gnubg's SGF candidates omit the "*" that XG's carry;
//   - domain/moves.go renders "4/2 4/2*" where a stored candidate writes
//     "4/2(2)".
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

// ourCubeAction runs our 2-ply cube decision at d's own score and buckets it
// like xgCubeBucket: TooGood folds into NoDouble (ADR-0014).
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
