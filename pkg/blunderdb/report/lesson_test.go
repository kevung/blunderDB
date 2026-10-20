package report

import (
	"fmt"
	"regexp"
	"strconv"
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

func TestDiagramTurnsCubeAndBarWithThePlayerOnRoll(t *testing.T) {
	cubeY := func(svg string) int {
		m := regexp.MustCompile(`<rect x="\d+" y="(\d+)" width="28" height="28"`).FindStringSubmatch(svg)
		if m == nil {
			t.Fatal("no cube drawn")
		}
		y, _ := strconv.Atoi(m[1])
		return y
	}
	barCheckers := func(svg string) (above, below []string) {
		barX := margin + cubeCol + 6*pointW + barW/2
		midY := margin + headerH + numH + boardH/2
		re := regexp.MustCompile(fmt.Sprintf(`<circle cx="%d" cy="(\d+)" r="\d+" fill="(#[0-9a-f]+)"`, barX))
		for _, m := range re.FindAllStringSubmatch(svg, -1) {
			y, _ := strconv.Atoi(m[1])
			if y < midY {
				above = append(above, m[2])
			} else {
				below = append(below, m[2])
			}
		}
		return
	}
	mid := margin + headerH + numH + boardH/2
	for _, onRoll := range []int{domain.Black, domain.White} {
		for _, owner := range []int{domain.Black, domain.White} {
			p := domain.InitializePosition()
			p.PlayerOnRoll, p.Cube.Owner, p.Cube.Value = onRoll, owner, 1
			p.Board.Points[0] = domain.Point{Checkers: 1, Color: domain.Black}
			p.Board.Points[25] = domain.Point{Checkers: 2, Color: domain.White}
			svg := Diagram(&p)
			if got := cubeY(svg) < mid; got != (owner != onRoll) {
				t.Errorf("onRoll %d owner %d: cube on the wrong end (top=%v)", onRoll, owner, got)
			}
			above, below := barCheckers(svg)
			wantAbove, wantBelow := 1, 2 // Points[0] above, Points[25] below, half a turn when White rolls
			if onRoll == domain.White {
				wantAbove, wantBelow = 2, 1
			}
			if len(above) != wantAbove || len(below) != wantBelow {
				t.Errorf("onRoll %d: bar above %v below %v", onRoll, above, below)
			}
		}
	}
	for away, want := range map[int]string{domain.Crawford: "-1 Cr", domain.PostCrawford: "-1 post-Cr", 3: "-3", domain.Unlimited: "∞"} {
		if got := scoreText(away); got != want {
			t.Errorf("scoreText(%d) = %q, want %q", away, got, want)
		}
	}
}
