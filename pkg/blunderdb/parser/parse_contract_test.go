package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The shared corpus (testdata/parse_corpus.json) is generated from the CURRENT
// GUI parsePosition and asserted by both sides: the GUI in
// frontend/src/__tests__/parseContract.test.js and this Go test. The Go parser
// must reproduce the GUI output field-for-field, so the two can never drift.
type cmpAnalysis struct {
	AnalysisType          string                       `json:"analysisType"`
	XGID                  string                       `json:"xgid"`
	AnalysisEngineVersion string                       `json:"analysisEngineVersion"`
	Comment               string                       `json:"comment"`
	DoublingCubeAnalysis  *domain.DoublingCubeAnalysis `json:"doublingCubeAnalysis"`
	CheckerAnalysis       []domain.CheckerMove         `json:"checkerAnalysis"`
}

func TestParsePositionContract(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "parse_corpus.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus %s: %v", path, err)
	}
	var corpus struct {
		Cases []struct {
			Name     string `json:"name"`
			Input    string `json:"input"`
			Expected struct {
				Position domain.Position `json:"position"`
				Analysis cmpAnalysis     `json:"analysis"`
			} `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("parse corpus: %v", err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("corpus has no cases")
	}

	for _, c := range corpus.Cases {
		t.Run(c.Name, func(t *testing.T) {
			res, err := ParsePosition(c.Input)
			if err != nil {
				t.Fatalf("ParsePosition: %v", err)
			}

			if !reflect.DeepEqual(res.Position, c.Expected.Position) {
				t.Errorf("position mismatch:\n got  %+v\n want %+v", res.Position, c.Expected.Position)
			}

			got := cmpAnalysis{
				AnalysisType:          res.Analysis.AnalysisType,
				XGID:                  res.Analysis.XGID,
				AnalysisEngineVersion: res.Analysis.AnalysisEngineVersion,
				Comment:               res.Comment,
				DoublingCubeAnalysis:  res.Analysis.DoublingCubeAnalysis,
				CheckerAnalysis:       movesOrEmpty(res.Analysis.CheckerAnalysis),
			}
			// Normalize empty-vs-nil so JSON [] (non-nil empty) compares equal.
			want := c.Expected.Analysis
			if want.CheckerAnalysis == nil {
				want.CheckerAnalysis = []domain.CheckerMove{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("analysis mismatch:\n got  %s\n want %s", mustJSON(got), mustJSON(want))
			}
		})
	}
}

func movesOrEmpty(ca *domain.CheckerAnalysis) []domain.CheckerMove {
	if ca == nil || ca.Moves == nil {
		return []domain.CheckerMove{}
	}
	return ca.Moves
}

func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
