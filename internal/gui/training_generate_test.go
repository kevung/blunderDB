package gui

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/training"
)

// The binding answers with a question, and a second call reuses the generator
// the first one built rather than paying for two new searchers.
func TestGenerateEvaluationQuestionAnswersFromThePool(t *testing.T) {
	app := &App{}
	first := app.GenerateEvaluationQuestion(training.EvaluationRequest{Source: training.SourcePool})
	if !first.Generated || first.CubeAnswer == "" {
		t.Fatalf("no question from the pool: %+v", first)
	}
	built := app.trainingGen
	if second := app.GenerateEvaluationQuestion(training.EvaluationRequest{Source: training.SourcePool}); !second.Generated || app.trainingGen != built {
		t.Fatalf("the second question rebuilt the generator or refused: %+v", second)
	}
	if refused := app.GenerateEvaluationQuestion(training.EvaluationRequest{Source: "elsewhere"}); refused.Generated || refused.Refusal != training.RefusalUnknownSource {
		t.Fatalf("an unknown source was not refused by name: %+v", refused)
	}
}
