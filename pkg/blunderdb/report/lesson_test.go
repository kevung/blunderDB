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

func TestAnalysisHTMLRecallsThePlayedMoveAndTranslatesTheCube(t *testing.T) {
	eq := 0.363
	a := &domain.PositionAnalysis{
		PlayedMoves: []string{"24/20"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/5 6/5", Equity: 0.1}, {Move: "24/20", Equity: -0.2, EquityError: &eq},
		}},
	}
	out := analysisHTML("fr", a)
	for _, want := range []string{"Joué</b> : 24/20", "(-0.363)"} {
		if !strings.Contains(out, want) {
			t.Errorf("analysis lacks %q: %s", want, out)
		}
	}
	cube := &domain.PositionAnalysis{PlayedCubeActions: []string{"No Double"}, DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "Too good to double, pass"}}
	for _, lang := range Languages() {
		out := analysisHTML(lang, cube)
		if lang != "en" && (strings.Contains(out, "Too good") || strings.Contains(out, "No Double")) {
			t.Errorf("%s: cube action left in English: %s", lang, out)
		}
		if lang == "en" && !strings.Contains(out, "Too good, Pass") {
			t.Errorf("en: %s", out)
		}
	}
}

func TestDiagramNumbersThePointsAndKeepsDiceClearOfStacks(t *testing.T) {
	p := domain.InitializePosition()
	p.Dice = [2]int{5, 3}
	svg := Diagram(&p)
	for _, want := range []string{">24</text>", ">1</text>", ">-7</text>"} {
		if !strings.Contains(svg, want) {
			t.Errorf("diagram lacks %q", want)
		}
	}
}
