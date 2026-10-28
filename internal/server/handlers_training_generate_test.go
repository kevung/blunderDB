package server

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/training"
)

// TestGenerateBearoffIsReplayableFromItsSeed: the same request and seed give
// the same questions, question i is regenerated alone from seed+i, and each
// question carries its truth.
func TestGenerateBearoffIsReplayableFromItsSeed(t *testing.T) {
	ts := newTestServer(t)
	req := trainingGenerateReq{Source: "pool", Count: 5, Seed: 4242}
	var a, b trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", req, &a)
	postDecode(t, ts, "/v1/training.generateBearoff", req, &b)
	if len(a.Questions) != 5 || !reflect.DeepEqual(a, b) {
		t.Fatalf("same seed, different questions: %d vs %d", len(a.Questions), len(b.Questions))
	}
	for i, q := range a.Questions {
		if q.Seed != 4242+int64(i) || !q.Generated || q.EPC.Bottom.EPC == nil {
			t.Fatalf("question %d = seed %d generated %v: want a graded question", i, q.Seed, q.Generated)
		}
	}
	var one trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Seed: a.Questions[3].Seed}, &one)
	if !reflect.DeepEqual(one.Questions[0], a.Questions[3]) {
		t.Error("question 3 is not regenerated alone from its own seed")
	}
	var drawn trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool"}, &drawn)
	if drawn.Questions[0].Seed == 0 {
		t.Error("a drawn seed is returned so the call can be replayed")
	}
}

// TestGenerateBearoffSources: the board source plays on from the position, the
// library source keeps the tenant's position as it stands, and a seed outside
// the domain is a named refusal in the answer, not an HTTP error.
func TestGenerateBearoffSources(t *testing.T) {
	ts := newTestServer(t)
	var pool trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Seed: 7}, &pool)
	seed := pool.Questions[0].Position

	var board trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "board", Position: &seed, Count: 3, Seed: 9}, &board)
	for _, q := range board.Questions {
		if !q.Generated || q.Source != "board" || q.Plies < 1 {
			t.Errorf("board question = %+v: want generated, at least one ply", q)
		}
	}

	var saved positionSaveResp
	postDecode(t, ts, "/v1/positions.save", positionReq{Position: &seed}, &saved)
	var stored domain.Position
	postDecode(t, ts, "/v1/positions.load", idReq{ID: saved.ID}, &stored)
	var lib trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "library", PositionIDs: []int64{saved.ID}}, &lib)
	if q := lib.Questions[0]; !q.Generated || q.Plies != 0 || q.PositionID != saved.ID || !sameChequers(q.Position.Board, stored.Board) {
		t.Errorf("library question = %+v: want the stored position at k = 0", q)
	}

	contact := domain.InitializePosition()
	var refused trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "board", Position: &contact}, &refused)
	if q := refused.Questions[0]; q.Generated || q.Refusal != race.RefusalNotBearoff {
		t.Errorf("contact seed = %+v: want a notBearoff refusal", q)
	}
}

// TestGenerateRefusesWhatIsOutOfBounds: bounds and malformed requests are 400
// before any engine work, a missing library position is 404.
func TestGenerateRefusesWhatIsOutOfBounds(t *testing.T) {
	ts := newTestServer(t)
	pos := domain.InitializePosition()
	ids := func(n int) []int64 { return make([]int64, n) }
	cases := []struct {
		name, path string
		body       trainingGenerateReq
		status     int
	}{
		{"bearoff count over bound", "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Count: maxBearoffQuestions + 1}, 400},
		{"evaluation count over bound", "/v1/training.generateEvaluation", trainingGenerateReq{Source: "pool", Count: maxEvaluationQuestions + 1}, 400},
		{"negative count", "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Count: -1}, 400},
		{"library ids over bound", "/v1/training.generateEvaluation", trainingGenerateReq{Source: "library", PositionIDs: ids(maxEvaluationQuestions + 1)}, 400},
		{"unknown source", "/v1/training.generateBearoff", trainingGenerateReq{Source: "dice"}, 400},
		{"negative seed", "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Seed: -3}, 400},
		{"seed beyond 2^52", "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Seed: maxQuestionSeed + 1}, 400},
		{"seed + count beyond 2^52", "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Count: 2, Seed: maxQuestionSeed}, 400},
		{"board without position", "/v1/training.generateEvaluation", trainingGenerateReq{Source: "board"}, 400},
		{"library without ids", "/v1/training.generateBearoff", trainingGenerateReq{Source: "library"}, 400},
		{"position on the pool", "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Position: &pos}, 400},
		{"missing library position", "/v1/training.generateBearoff", trainingGenerateReq{Source: "library", PositionIDs: []int64{987654}}, http.StatusNotFound},
	}
	for _, c := range cases {
		r := post(t, ts, c.path, c.body)
		r.Body.Close()
		if r.StatusCode != c.status {
			t.Errorf("%s: status %d, want %d", c.name, r.StatusCode, c.status)
		}
	}
}

// TestGenerateEvaluationCarriesItsTruthAndReplays: a question comes with the
// graded answer, and the same seed gives the same question and truth.
func TestGenerateEvaluationCarriesItsTruthAndReplays(t *testing.T) {
	ts := newTestServer(t)
	req := trainingGenerateReq{Source: "pool", Count: 2, Seed: 31337}
	var a, b trainingEvaluationResp
	postDecode(t, ts, "/v1/training.generateEvaluation", req, &a)
	postDecode(t, ts, "/v1/training.generateEvaluation", req, &b)
	if len(a.Questions) != 2 || !reflect.DeepEqual(a, b) {
		t.Fatalf("same seed, different questions (%d)", len(a.Questions))
	}
	for i, q := range a.Questions {
		if !q.Generated || q.Seed != 31337+int64(i) || q.WinChance < 0 || q.WinChance > 100 || q.Depth == "" || q.CubeAnswer == "" || q.Regime == "" {
			t.Errorf("question %d = %+v: want a position with win chance, depth, cube answer and regime", i, q)
		}
	}
	var one trainingEvaluationResp
	postDecode(t, ts, "/v1/training.generateEvaluation", trainingGenerateReq{Source: "pool", Seed: a.Questions[1].Seed}, &one)
	if !reflect.DeepEqual(one.Questions[0], a.Questions[1]) {
		t.Error("question 1 is not regenerated alone from its own seed")
	}
}

// TestGenerateEvaluationSources: the library source asks the tenant's money
// position as it stands; a board that is not a game is a named refusal.
func TestGenerateEvaluationSources(t *testing.T) {
	ts := newTestServer(t)
	start := domain.InitializePosition()
	start.Dice = [2]int{0, 0}
	start.Score = [2]int{-1, -1}
	start.DecisionType = domain.CubeAction
	var saved positionSaveResp
	postDecode(t, ts, "/v1/positions.save", positionReq{Position: &start}, &saved)

	var lib trainingEvaluationResp
	postDecode(t, ts, "/v1/training.generateEvaluation", trainingGenerateReq{Source: "library", PositionIDs: []int64{saved.ID}}, &lib)
	if q := lib.Questions[0]; !q.Generated || q.Plies != 0 || q.PositionID != saved.ID || q.CubeAnswer == "" {
		t.Errorf("library question = %+v: want the saved position, graded", q)
	}

	var board trainingEvaluationResp
	postDecode(t, ts, "/v1/training.generateEvaluation", trainingGenerateReq{Source: "board", Position: &start, Seed: 5}, &board)
	if q := board.Questions[0]; !q.Generated || q.Plies < 1 {
		t.Errorf("board question = %+v: want a walk of at least one ply", q)
	}

	var empty domain.Position
	postDecode(t, ts, "/v1/training.generateEvaluation", trainingGenerateReq{Source: "board", Position: &empty, Seed: 5}, &board)
	if q := board.Questions[0]; !q.Generated || q.Source != "pool" || q.Refusal != training.RefusalEmptyBoard {
		t.Errorf("empty board = %+v: want the pool fallback that says why", q)
	}
}

// sameChequers compares where the chequers stand: the colour of an empty
// point is a layout detail of the walk, not a difference.
func sameChequers(a, b domain.Board) bool {
	if a.Bearoff != b.Bearoff {
		return false
	}
	for i := range a.Points {
		if a.Points[i].Checkers != b.Points[i].Checkers || (a.Points[i].Checkers > 0 && a.Points[i].Color != b.Points[i].Color) {
			return false
		}
	}
	return true
}

// TestGenerateReplaysAtTheSeedBound: the last question of a batch served at
// the top of the seed range replays alone from the seed it carries.
func TestGenerateReplaysAtTheSeedBound(t *testing.T) {
	ts := newTestServer(t)
	var batch, one trainingBearoffResp
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Count: maxBearoffQuestions, Seed: maxQuestionSeed - maxBearoffQuestions + 1}, &batch)
	last := batch.Questions[maxBearoffQuestions-1]
	if last.Seed != maxQuestionSeed {
		t.Fatalf("last seed = %d, want 2^52", last.Seed)
	}
	postDecode(t, ts, "/v1/training.generateBearoff", trainingGenerateReq{Source: "pool", Seed: last.Seed}, &one)
	if !reflect.DeepEqual(one.Questions[0], last) {
		t.Error("the last question of the batch does not replay alone")
	}
}
