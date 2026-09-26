package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Le mode quiz côté démon. Le jugement est celui d'engine, comme au bureau,
// pour qu'un PR vaille la même chose d'un client à l'autre. Les routes
// n'écrivent rien : une session de quiz appartient à celui qui la fait.

type quizCheckerReq struct {
	PositionID int64        `json:"positionId"`
	Played     domain.Board `json:"played"`
}

type quizCheckerMoveReq struct {
	PositionID int64  `json:"positionId"`
	Move       string `json:"move"`
}

type explainReq struct {
	PositionID int64  `json:"positionId"`
	Played     string `json:"played"`
}

type quizCubeReq struct {
	PositionID int64  `json:"positionId"`
	Action     string `json:"action"`
}

func (s *Server) quizRoutes() []route {
	ps := func() storage.PositionStore { return s.opts.Storage.Positions() }
	as := func() storage.AnalysisStore { return s.opts.Storage.Analyses() }

	// analysisOf treats a missing analysis as "none", not an error: a quiz runs
	// through a library with gaps, and "not matched" is the honest verdict.
	analysisOf := func(ctx context.Context, scope string, id int64) *domain.PositionAnalysis {
		ana, err := as().Load(ctx, scope, id)
		if err != nil {
			return nil
		}
		return ana
	}

	return []route{
		{http.MethodPost, "/v1/quiz.gradeChecker", rpc(func(ctx context.Context, scope string, req quizCheckerReq) (engine.QuizVerdict, error) {
			pos, err := ps().Load(ctx, scope, req.PositionID)
			if err != nil {
				return engine.QuizVerdict{}, err
			}
			return engine.GradeCheckerAnswer(pos, analysisOf(ctx, scope, req.PositionID), req.Played), nil
		})},
		{http.MethodPost, "/v1/quiz.gradeCheckerMove", rpc(func(ctx context.Context, scope string, req quizCheckerMoveReq) (engine.QuizVerdict, error) {
			pos, err := ps().Load(ctx, scope, req.PositionID)
			if err != nil {
				return engine.QuizVerdict{}, err
			}
			return engine.GradeCheckerAnswerNotation(pos, analysisOf(ctx, scope, req.PositionID), req.Move), nil
		})},
		// Rend un THÈME et ses écarts mesurés, jamais une phrase : le client
		// la rédige dans sa langue à partir d'un gabarit.
		{http.MethodPost, "/v1/positions.explain", rpc(func(ctx context.Context, scope string, req explainReq) (engine.Explanation, error) {
			pos, err := ps().Load(ctx, scope, req.PositionID)
			if err != nil {
				return engine.Explanation{}, err
			}
			ana := analysisOf(ctx, scope, req.PositionID)
			if ana == nil {
				return engine.Explanation{}, nil
			}
			if ana.DoublingCubeAnalysis != nil {
				return engine.ExplainCube(ana, req.Played), nil
			}
			return engine.ExplainChecker(pos, ana, req.Played), nil
		})},
		{http.MethodPost, "/v1/quiz.gradeCube", rpc(func(ctx context.Context, scope string, req quizCubeReq) (engine.QuizVerdict, error) {
			if _, err := ps().Load(ctx, scope, req.PositionID); err != nil {
				return engine.QuizVerdict{}, err
			}
			return engine.GradeCubeAnswer(analysisOf(ctx, scope, req.PositionID), req.Action), nil
		})},
	}
}
