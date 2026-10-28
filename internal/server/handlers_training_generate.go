package server

import (
	"fmt"
	"math/rand/v2"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/training"
)

// The Training question generators over HTTP (ADR-0041): a client that drills
// a person in front of its own board asks the daemon for questions and keeps
// the clock, the answer and the grading itself. A question is never stored:
// the response carries the position AND its truth, so the caller judges
// without a second round trip, and a seed so it can replay what it graded.
//
// A generator is engine work, so it is bounded like one: a count per call,
// and for Evaluation (a 2-ply truth per question) every question is a unit of
// the shared analysis workers, charged to the tenant's analysis quota. Bearoff
// is a table walk of microseconds and runs on the request.

const (
	// maxBearoffQuestions bounds one training.generateBearoff call.
	maxBearoffQuestions = 50
	// maxEvaluationQuestions bounds one training.generateEvaluation call.
	maxEvaluationQuestions = 10
	// maxQuestionSeed keeps a seed exact as a JSON number in every client
	// (2^53 is where a double stops counting).
	maxQuestionSeed = int64(1) << 52
)

// trainingGenerateReq asks for questions from one seed source.
type trainingGenerateReq struct {
	// Source is "pool" (canonical shapes), "board" (neighbourhood of
	// Position) or "library" (the tenant's PositionIDs, as they stand).
	Source string `json:"source"`
	// Position is the seed of the board source.
	Position *domain.Position `json:"position,omitempty"`
	// PositionIDs are the tenant's positions the library source asks about,
	// one question each, in order; count is ignored for it.
	PositionIDs []int64 `json:"positionIds,omitempty"`
	// Count is how many questions to make, 1 to 50 for Bearoff and 1 to 10
	// for Evaluation (default 1). The pool and board sources draw each
	// question afresh.
	Count int `json:"count"`
	// Seed makes the call replayable: question i uses seed + i, so the
	// same request and seed give the same questions, and question i alone
	// is regenerated with count 1 and its own seed. 0 draws one, returned
	// in each question. At most 2^52.
	Seed int64 `json:"seed"`
}

// trainingBearoffQuestion is one Bearoff question and the seed that replays it.
type trainingBearoffQuestion struct {
	Seed int64 `json:"seed"`
	// PositionID is the library position the question came from (library
	// source only).
	PositionID int64 `json:"positionId,omitempty"`
	race.BearoffQuestion
}

type trainingBearoffResp struct {
	Questions []trainingBearoffQuestion `json:"questions"`
}

// trainingEvaluationQuestion is one Evaluation question and the seed that
// replays it.
type trainingEvaluationQuestion struct {
	Seed       int64 `json:"seed"`
	PositionID int64 `json:"positionId,omitempty"`
	training.EvaluationQuestion
}

type trainingEvaluationResp struct {
	Questions []trainingEvaluationQuestion `json:"questions"`
}

// seedAndCount validates the shared bounds and returns the base seed.
func (req *trainingGenerateReq) seedAndCount(max int) (int64, int, error) {
	switch req.Source {
	case race.SourcePool, race.SourceBoard, race.SourceLibrary:
	default:
		return 0, 0, fmt.Errorf("source must be pool, board or library")
	}
	if req.Seed < 0 || req.Seed > maxQuestionSeed {
		return 0, 0, fmt.Errorf("seed must be between 0 and 2^52")
	}
	n := req.Count
	if n == 0 {
		n = 1
	}
	if req.Source == race.SourceLibrary {
		n = len(req.PositionIDs)
		if n == 0 {
			return 0, 0, fmt.Errorf("the library source needs positionIds")
		}
	} else if len(req.PositionIDs) > 0 {
		return 0, 0, fmt.Errorf("positionIds belong to the library source")
	}
	if n < 1 || n > max {
		return 0, 0, fmt.Errorf("count must be between 1 and %d", max)
	}
	if req.Source == race.SourceBoard && req.Position == nil {
		return 0, 0, fmt.Errorf("the board source needs a position")
	}
	if req.Source != race.SourceBoard && req.Position != nil {
		return 0, 0, fmt.Errorf("position belongs to the board source")
	}
	seed := req.Seed
	if seed == 0 {
		// A drawn seed leaves room for the largest batch, so any served
		// question stays replayable alone.
		seed = 1 + rand.Int64N(maxQuestionSeed-int64(max)) //nolint:gosec // G404: a replayable exercise seed, not a secret
	} else if seed+int64(n)-1 > maxQuestionSeed {
		return 0, 0, fmt.Errorf("seed + count must not exceed 2^52")
	}
	return seed, n, nil
}

// seedPositions returns the seed position of each question: the inline board,
// or the tenant's library positions (a missing one answers 404).
func (s *Server) seedPositions(r *http.Request, scope string, req *trainingGenerateReq, n int) ([]*domain.Position, error) {
	out := make([]*domain.Position, n)
	for i := range out {
		switch req.Source {
		case race.SourceBoard:
			out[i] = req.Position
		case race.SourceLibrary:
			p, err := s.opts.Storage.Positions().Load(r.Context(), scope, req.PositionIDs[i])
			if err != nil {
				return nil, err
			}
			out[i] = p
		}
	}
	return out, nil
}

func (s *Server) trainingGenerateRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/training.generateBearoff", s.handleTrainingGenerateBearoff},
		{http.MethodPost, "/v1/training.generateEvaluation", s.handleTrainingGenerateEvaluation},
	}
}

func (s *Server) handleTrainingGenerateBearoff(w http.ResponseWriter, r *http.Request) {
	var req trainingGenerateReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	base, n, err := req.seedAndCount(maxBearoffQuestions)
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	seeds, err := s.seedPositions(r, scopeOf(r), &req, n)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	out := trainingBearoffResp{Questions: make([]trainingBearoffQuestion, n)}
	for i := range out.Questions {
		seed := base + int64(i)
		q := trainingBearoffQuestion{Seed: seed}
		if req.Source == race.SourceLibrary {
			q.PositionID = req.PositionIDs[i]
		}
		q.BearoffQuestion = race.GenerateBearoffSeeded(race.BearoffRequest{Source: req.Source, Seed: seeds[i]}, uint64(seed)) //nolint:gosec // G115: seed is within [0, 2^52]
		out.Questions[i] = q
	}
	writeJSONResp(w, out)
}

func (s *Server) handleTrainingGenerateEvaluation(w http.ResponseWriter, r *http.Request) {
	var req trainingGenerateReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	base, n, err := req.seedAndCount(maxEvaluationQuestions)
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	scope := scopeOf(r)
	seeds, err := s.seedPositions(r, scope, &req, n)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	if s.refuseAnalysis(w, scope) {
		return
	}
	out := trainingEvaluationResp{Questions: make([]trainingEvaluationQuestion, n)}
	for i := range out.Questions {
		seed := base + int64(i)
		q := trainingEvaluationQuestion{Seed: seed}
		if req.Source == race.SourceLibrary {
			q.PositionID = req.PositionIDs[i]
		}
		ereq := training.EvaluationRequest{Source: req.Source, Seed: seeds[i]}
		// One question per unit: it waits its tenant's turn on the shared
		// workers and is charged what it cost, as an evaluation is.
		ran := s.analysis.run(r.Context(), scope, func(get searcherFor) {
			cost := get.clock()
			walker, relWalker := boundSearcher(r.Context(), get, 0, 0)
			judge, relJudge := boundSearcher(r.Context(), get, gammonnet.DefaultPly, gammonnet.DefaultPruneK)
			defer func() { relWalker(); relJudge() }()
			defer func() { s.quota.charge(scope, cost()) }()
			if walker == nil || judge == nil {
				q.EvaluationQuestion = training.EvaluationQuestion{Source: req.Source, Refusal: training.RefusalNotEvaluable}
				return
			}
			q.EvaluationQuestion = training.NewGeneratorFrom(walker, judge).EvaluationSeeded(ereq, uint64(seed)) //nolint:gosec // G115: seed is within [0, 2^52]
		})
		if !ran {
			if r.Context().Err() != nil {
				return
			}
			writeErrorCode(w, CodeUnavailable, "the daemon is stopping: no question is made")
			return
		}
		out.Questions[i] = q
		if s.quota.analysisExhausted(scope) && i < n-1 {
			out.Questions = out.Questions[:i+1]
			break
		}
	}
	writeJSONResp(w, out)
}
