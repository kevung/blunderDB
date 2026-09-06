package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Le mode quiz (#294, fiche J.4), côté démon.
//
// Le jugement est celui d'engine — le même que le bureau appelle — parce que
// le PR d'une session doit valoir la même chose d'un client à l'autre. Ce que
// ces deux routes ajoutent est l'accès à la position et à son analyse, et rien
// d'autre : elles n'écrivent pas, ne comptent pas les sessions, ne retiennent
// pas qui a répondu quoi. Une session de quiz appartient à celui qui la fait.

type quizCheckerReq struct {
	PositionID int64        `json:"positionId"`
	Played     domain.Board `json:"played"`
}

type quizCheckerMoveReq struct {
	PositionID int64  `json:"positionId"`
	Move       string `json:"move"`
}

type quizCubeReq struct {
	PositionID int64  `json:"positionId"`
	Action     string `json:"action"`
}

func (s *Server) quizRoutes() []route {
	ps := func() storage.PositionStore { return s.opts.Storage.Positions() }
	as := func() storage.AnalysisStore { return s.opts.Storage.Analyses() }

	// analysisOf reads the analysis, treating a missing one as "no analysis"
	// rather than an error: a quiz must keep running through a library with
	// gaps, and a verdict that says "not matched" is the honest answer for a
	// position nobody has evaluated.
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
		{http.MethodPost, "/v1/quiz.gradeCube", rpc(func(ctx context.Context, scope string, req quizCubeReq) (engine.QuizVerdict, error) {
			if _, err := ps().Load(ctx, scope, req.PositionID); err != nil {
				return engine.QuizVerdict{}, err
			}
			return engine.GradeCubeAnswer(analysisOf(ctx, scope, req.PositionID), req.Action), nil
		})},
	}
}
