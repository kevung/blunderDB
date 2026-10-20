package report

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestRenderLessonFoldsTheAnalysisAndRecallsTheOrigin(t *testing.T) {
	eq := -0.05
	a := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: "8/5 6/5", Equity: 0.1}, {Move: "24/20", Equity: 0.05, EquityError: &eq},
	}}}
	l := &domain.Lesson{Name: "Primes"}
	out := renderLesson("en", l, []lessonStepView{{Title: "T", Text: "x", Boards: []lessonBoard{{Diagram: "<svg/>", Analysis: analysisHTML("en", a)}}}}, "Course <1>")
	for _, want := range []string{"<details><summary>", "8/5 6/5", "24/20", "Course &lt;1&gt;", "Origin of the file:"} {
		if !strings.Contains(out, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(out, "<details open") {
		t.Error("the analysis must be folded by default")
	}
	if strings.Contains(renderLesson("en", l, nil, ""), "Origin of the file:") {
		t.Error("no watermark, no origin line")
	}
}
