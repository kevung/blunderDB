package engine

import (
	"math"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Mode quiz (#294, fiche J.4).
//
// Anki fait mémoriser, le quiz TESTE : on joue le coup, et l'erreur est
// mesurée contre l'analyse enregistrée. Tout ce qui suit est pur — une
// position, une analyse, une réponse — donc l'interface, la ligne de commande
// et le démon jugent la même chose de la même façon, et le jugement se teste
// sans base de données.
//
// Le quiz n'invente aucune échelle. L'erreur est celle que l'analyse porte
// déjà, en millipoints d'équité normalisée, et le PR de session est calculé
// par la formule que les statistiques appliquent au jeu réel : 500 × erreur
// moyenne. C'est ce qui rend les deux nombres comparables, et c'était le
// point de la fiche — le rapport P10 dit la même chose autrement : dériver la
// note des seuils natifs des moteurs plutôt que d'inventer une échelle.

// QuizVerdict is the judgement of one answer.
type QuizVerdict struct {
	// Legal is false when the played board is not the result of any legal
	// play for these dice. It is a different failure from Matched: an illegal
	// move is a rules mistake, an unanalysed one is a gap in the database.
	Legal bool `json:"legal"`
	// Matched is true when the played move appears in the stored analysis, so
	// its error is known. A legal move the engine did not rank has no error.
	Matched bool `json:"matched"`
	// Notation is the played move as the generator writes it.
	Notation string `json:"notation"`
	// Best is the analysis's own best answer, for the correction.
	Best string `json:"best"`
	// ErrorMP is the cost of the answer in millipoints of normalised equity,
	// never negative. Zero means the best answer was played.
	ErrorMP int `json:"errorMp"`
}

// CubeActions are the three answers a cube question accepts.
const (
	CubeActionNoDouble   = "nd"
	CubeActionDoubleTake = "dt"
	CubeActionDoublePass = "dp"
)

// GradeCheckerAnswer judges a checker play given by the board it produced.
//
// The board is the answer, not a notation: the user moved chequers, and asking
// the interface to spell the move it just made would be a second notation to
// keep in step with the generator's. The generator dedups by resulting
// position, so a board identifies at most one play.
func GradeCheckerAnswer(pos *domain.Position, ana *domain.PositionAnalysis, played domain.Board) QuizVerdict {
	var v QuizVerdict
	for _, play := range domain.LegalMoves(pos) {
		if play.Result.Board != played {
			continue
		}
		v.Legal = true
		v.Notation = play.Notation
		break
	}
	if ana == nil || ana.CheckerAnalysis == nil || len(ana.CheckerAnalysis.Moves) == 0 {
		return v
	}
	best, bestErr := "", math.Inf(1)
	for _, m := range ana.CheckerAnalysis.Moves {
		e := 0.0
		if m.EquityError != nil {
			e = math.Abs(*m.EquityError)
		}
		if e < bestErr {
			best, bestErr = m.Move, e
		}
	}
	v.Best = best
	if !v.Legal {
		return v
	}
	norm := NormalizeMove(v.Notation)
	for _, m := range ana.CheckerAnalysis.Moves {
		if !strings.EqualFold(NormalizeMove(m.Move), norm) {
			continue
		}
		v.Matched = true
		if m.EquityError != nil {
			v.ErrorMP = int(math.Round(math.Abs(*m.EquityError) * 1000))
		}
		break
	}
	return v
}

// GradeCheckerAnswerNotation judges a checker play given by its notation.
//
// The board form above is the one a board gesture will call; this one is what
// a keyboard answers with, and both go through the SAME matching: the notation
// is resolved against the generator's legal plays first, so a move nobody
// could play is refused as illegal before any comparison with the analysis.
// Accepting a notation the generator does not produce would let the quiz mark
// an impossible move as merely "unranked".
func GradeCheckerAnswerNotation(pos *domain.Position, ana *domain.PositionAnalysis, notation string) QuizVerdict {
	norm := NormalizeMove(notation)
	for _, play := range domain.LegalMoves(pos) {
		if !strings.EqualFold(NormalizeMove(play.Notation), norm) {
			continue
		}
		return GradeCheckerAnswer(pos, ana, play.Result.Board)
	}
	// Not a legal play: still report the best answer, which is the correction
	// the user needs most when they proposed something impossible.
	v := QuizVerdict{Notation: notation}
	if ana != nil && ana.CheckerAnalysis != nil {
		best, bestErr := "", math.Inf(1)
		for _, m := range ana.CheckerAnalysis.Moves {
			e := 0.0
			if m.EquityError != nil {
				e = math.Abs(*m.EquityError)
			}
			if e < bestErr {
				best, bestErr = m.Move, e
			}
		}
		v.Best = best
	}
	return v
}

// GradeCubeAnswer judges a cube answer ("nd", "dt", "dp").
//
// The three errors are read from the analysis rather than recomputed: the
// stored numbers are what the statistics already charge the player for, and a
// quiz that graded on a second computation would grade a different game.
func GradeCubeAnswer(ana *domain.PositionAnalysis, action string) QuizVerdict {
	v := QuizVerdict{Legal: true, Notation: action}
	if ana == nil || ana.DoublingCubeAnalysis == nil {
		return v
	}
	c := ana.DoublingCubeAnalysis
	v.Best = c.BestCubeAction
	var err float64
	switch strings.ToLower(strings.TrimSpace(action)) {
	case CubeActionNoDouble:
		err = c.CubefulNoDoubleError
	case CubeActionDoubleTake:
		err = c.CubefulDoubleTakeError
	case CubeActionDoublePass:
		err = c.CubefulDoublePassError
	default:
		v.Legal = false
		return v
	}
	v.Matched = true
	v.ErrorMP = int(math.Round(math.Abs(err) * 1000))
	return v
}

// QuizPR is the session's Performance Rating, on the SAME scale as the one the
// statistics compute for real play: 500 × mean error in normalised equity.
// Zero decisions gives 0, which a caller must read together with the count
// rather than as a perfect score — the same caveat storage.pr carries.
func QuizPR(sumErrMP, decisions int) float64 {
	if decisions == 0 {
		return 0
	}
	return 500 * float64(sumErrMP) / 1000 / float64(decisions)
}
