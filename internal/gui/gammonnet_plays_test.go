package gui

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The targeted re-run answers only the plays asked for, marked partial, so the
// panel replaces those rows and keeps the rest of its ranking.
func TestEvaluatePlaysAnswersOnlyThePlaysAsked(t *testing.T) {
	pos := racePosition(6, 19, [2]int{2, 1}, domain.White)
	a := &App{}
	full, err := a.evaluateGammonNet(pos, 0, 0, 0)
	if err != nil || len(full.Moves) < 2 {
		t.Fatalf("full evaluation: %v, %d moves", err, len(full.Moves))
	}
	asked := full.Moves[len(full.Moves)-1].Move
	got, err := a.evaluatePlaysCtx(context.Background(), pos, 1, 0, []string{asked})
	if err != nil {
		t.Fatalf("evaluatePlaysCtx: %v", err)
	}
	if !got.Partial || len(got.Moves) != 1 || got.Moves[0].Move != asked || got.Moves[0].AnalysisDepth != "1-ply" {
		t.Fatalf("got %+v", got)
	}
}
