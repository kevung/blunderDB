package engine

import (
	"math"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Expliquer un blunder en une phrase (#298, fiche J.8), et la moitié « thèmes
// d'erreur » de J.1b (#291).
//
// # Ce que ce fichier rend, et ce qu'il ne rend pas
//
// Il ne rend PAS une phrase. Il rend un THÈME et les écarts mesurés qui le
// justifient ; la phrase est écrite dans la langue de l'utilisateur par
// l'interface, à partir d'un gabarit. C'est la séparation que P18 relève chez
// les outils qui expliquent sans LLM (DecodeChess, Fritz) : des gabarits à
// trous alimentés par une base de règles. Rendre une phrase ici en aurait fait
// une phrase française dans un moteur qui parle neuf langues.
//
// # La règle qui compte : se taire
//
// « Ne parler que quand une règle est confiante » est la contrainte de la
// fiche, et P18 en fait un principe : gnubg lui-même laisse son menu d'analyse
// VIDE quand rien ne mérite d'être dit — « There was nothing wrong with not
// doubling ». Chaque seuil ci-dessous est donc un seuil de PRISE DE PAROLE,
// pas un seuil de classement : au-dessous, Explanation.Theme est vide et
// l'interface n'affiche rien plutôt qu'une banalité.
//
// # Six thèmes, pas vingt
//
// P18 constate que les écosystèmes échecs et poker convergent sur très peu de
// catégories et sur une unité unique de gravité — la perte de probabilité de
// gain — que le backgammon possède déjà (l'équité normalisée). Une taxonomie
// trop fine produit des faux positifs, et un faux positif d'explication coûte
// plus cher que le silence : il apprend quelque chose de faux.

// Explanation is what a rule found, or nothing.
type Explanation struct {
	// Theme is a stable token the interface renders into a sentence, or "" when
	// no rule applied confidently. Never a translated string.
	Theme string `json:"theme"`
	// CostMP is what the played decision cost, in millipoints of normalised
	// equity — the unit the rest of blunderDB already charges errors in.
	CostMP int `json:"costMp"`
	// Blots, Points and GammonPct carry the measured differences the sentence
	// quotes: "leaves three blots where the best move leaves one". They are
	// filled only for the theme that used them.
	Blots      int `json:"blots,omitempty"`
	BestBlots  int `json:"bestBlots,omitempty"`
	GammonPct  int `json:"gammonPct,omitempty"`
	Points     int `json:"points,omitempty"`
	BestPoints int `json:"bestPoints,omitempty"`
	// Best is the move or cube action the analysis prefers, for the correction.
	Best string `json:"best"`
}

const (
	// ExplainMinCostMP is the cost below which nothing is said at all. It is
	// gnubg's "bad" threshold (0.060 normalised equity, its own default since
	// the thresholds were lowered), which is the point from which every engine
	// in the field agrees the decision was wrong rather than merely second.
	ExplainMinCostMP = 60

	// ExplainBlotSwingMin is how many more blots the played move must leave
	// for exposure to be named as the reason. One blot of difference is noise
	// — every alternative differs by one somewhere.
	ExplainBlotSwingMin = 2

	// ExplainGammonSwingMin is the gammon-chance difference, in percentage
	// points, from which the gammon is named as the reason.
	ExplainGammonSwingMin = 4

	// ExplainWinSwingMin is the win-chance difference, in percentage points,
	// from which a safer move is called too passive rather than merely safe.
	ExplainWinSwingMin = 3
)

// ExplainChecker explains a checker decision, or says nothing.
func ExplainChecker(pos *domain.Position, ana *domain.PositionAnalysis, played string) Explanation {
	if pos == nil || ana == nil || ana.CheckerAnalysis == nil || len(ana.CheckerAnalysis.Moves) == 0 {
		return Explanation{}
	}
	playedMove, bestMove := findPlayedAndBest(ana.CheckerAnalysis.Moves, played)
	if playedMove == nil || bestMove == nil || playedMove.Move == bestMove.Move {
		return Explanation{}
	}
	cost := 0
	if playedMove.EquityError != nil {
		cost = int(math.Round(math.Abs(*playedMove.EquityError) * 1000))
	}
	if cost < ExplainMinCostMP {
		return Explanation{}
	}
	out := Explanation{CostMP: cost, Best: bestMove.Move}

	playedBoard, okPlayed := boardAfter(pos, playedMove.Move)
	bestBoard, okBest := boardAfter(pos, bestMove.Move)

	// Gammon first: it is the one axis the analysis states directly, so it
	// needs no board reconstruction and cannot be wrong about what it saw.
	gammonSwing := int(math.Round(bestMove.PlayerGammonChance - playedMove.PlayerGammonChance))
	if gammonSwing >= ExplainGammonSwingMin {
		out.Theme = "gammon"
		out.GammonPct = gammonSwing
		return out
	}

	if okPlayed && okBest {
		mover := pos.PlayerOnRoll
		blots, bestBlots := countBlots(&playedBoard, mover), countBlots(&bestBoard, mover)
		points, bestPoints := countHomePoints(&playedBoard, mover), countHomePoints(&bestBoard, mover)

		if blots-bestBlots >= ExplainBlotSwingMin {
			out.Theme = "blots"
			out.Blots, out.BestBlots = blots, bestBlots
			return out
		}
		if bestPoints > points {
			out.Theme = "point"
			out.Points, out.BestPoints = points, bestPoints
			return out
		}
		// Safer AND worse: the move that exposes less loses win chances. This
		// is the only rule that reads two axes at once, because "too passive"
		// is exactly the conjunction — a safer move that keeps its win chances
		// is not passive, it is right.
		winSwing := int(math.Round(bestMove.PlayerWinChance - playedMove.PlayerWinChance))
		if bestBlots > blots && winSwing >= ExplainWinSwingMin {
			out.Theme = "passive"
			out.Blots, out.BestBlots = blots, bestBlots
			return out
		}
	}

	// Nothing confident to say. The cost is real, the reason is not one of the
	// six — so the sentence is not written. Silence is the answer P18 records
	// from every tool that explains without a language model.
	return Explanation{}
}

// ExplainCube explains a cube decision, or says nothing.
//
// The cube needs no board reading: gnubg and XG already decompose a cube error
// into its own kinds, and the analysis carries the three errors. The theme is
// the DIRECTION of the mistake, which is what a player learns from.
func ExplainCube(ana *domain.PositionAnalysis, played string) Explanation {
	if ana == nil || ana.DoublingCubeAnalysis == nil {
		return Explanation{}
	}
	c := ana.DoublingCubeAnalysis
	var err float64
	switch strings.ToLower(strings.TrimSpace(played)) {
	case CubeActionNoDouble:
		err = c.CubefulNoDoubleError
	case CubeActionDoubleTake:
		err = c.CubefulDoubleTakeError
	case CubeActionDoublePass:
		err = c.CubefulDoublePassError
	default:
		return Explanation{}
	}
	cost := int(math.Round(math.Abs(err) * 1000))
	if cost < ExplainMinCostMP {
		return Explanation{}
	}
	best := strings.ToLower(c.BestCubeAction)
	out := Explanation{CostMP: cost, Best: c.BestCubeAction}
	switch {
	case played == CubeActionNoDouble && strings.Contains(best, "double"):
		out.Theme = "doubletoolate"
	case played != CubeActionNoDouble && strings.Contains(best, "no double"):
		out.Theme = "doubletooearly"
	case played == CubeActionDoubleTake && strings.Contains(best, "pass"):
		out.Theme = "taketooloose"
	case played == CubeActionDoublePass && strings.Contains(best, "take"):
		out.Theme = "passtootight"
	default:
		return Explanation{}
	}
	return out
}

// findPlayedAndBest locates the played move and the analysis's own best.
func findPlayedAndBest(moves []domain.CheckerMove, played string) (*domain.CheckerMove, *domain.CheckerMove) {
	norm := NormalizeMove(played)
	var playedMove, bestMove *domain.CheckerMove
	bestErr := math.Inf(1)
	for i := range moves {
		m := &moves[i]
		e := 0.0
		if m.EquityError != nil {
			e = math.Abs(*m.EquityError)
		}
		if e < bestErr {
			bestErr, bestMove = e, m
		}
		if norm != "" && strings.EqualFold(NormalizeMove(m.Move), norm) {
			playedMove = m
		}
	}
	return playedMove, bestMove
}

// boardAfter replays a move notation through the legal-move generator and
// returns the board it produces. A notation the generator does not produce —
// an imported analysis can spell a move a way this port does not — yields
// ok=false, and the rules that need a board simply do not fire.
func boardAfter(pos *domain.Position, notation string) (domain.Board, bool) {
	norm := NormalizeMove(notation)
	for _, play := range domain.LegalMoves(pos) {
		if strings.EqualFold(NormalizeMove(play.Notation), norm) {
			return play.Result.Board, true
		}
	}
	return domain.Board{}, false
}

// countBlots counts a side's lone checkers on the board.
func countBlots(b *domain.Board, color int) int {
	n := 0
	for i := 1; i <= domain.NumPoints; i++ {
		if b.Points[i].Color == color && b.Points[i].Checkers == 1 {
			n++
		}
	}
	return n
}

// countHomePoints counts the points a side holds in its own home board.
func countHomePoints(b *domain.Board, color int) int {
	from, to := 1, 6
	if color == domain.White {
		from, to = 19, 24
	}
	n := 0
	for i := from; i <= to; i++ {
		if b.Points[i].Color == color && b.Points[i].Checkers >= 2 {
			n++
		}
	}
	return n
}
