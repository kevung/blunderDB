// extract_gnubg_stats parses GS tags from a gnuBG SGF file and prints
// aggregated per-player statistics as JSON.
//
// Usage: go run ./cmd/extract_gnubg_stats <file.sgf>
//
// The GS tag format is documented in gnubg/sgf.c:WriteStatContext.
// Encoding:
//
//	GS[M:<m-values>][C:<c-values>][D:<d-values>]
//
// M section (16 values):
//
//	anUnforcedMoves[0] [1] anTotalMoves[0] [1]
//	anMoves[0][VERYBAD] [1] [0][BAD] [1] [0][DOUBTFUL] [1] [0][NONE] [1]
//	arErrorCheckerplay[0][NORM] [0][UNNORM] [1][NORM] [1][UNNORM]
//
// C section (20 ints + 24 floats):
//
//	anTotalCube[0] [1] anDouble[0] [1] anTake[0] [1] anPass[0] [1]
//	anCubeMissedDoubleDP[0] [1] anCubeMissedDoubleTG[0] [1]
//	anCubeWrongDoubleDP[0] [1] anCubeWrongDoubleTG[0] [1]
//	anCubeWrongTake[0] [1] anCubeWrongPass[0] [1]
//	(then 6 error categories × 2 players × 2 (NORM+UNNORM) = 24 floats, player0 first)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// statContext mirrors gnubg's statcontext fields that are stored in the SGF GS tag.
type statContext struct {
	// M section
	UnforcedMoves [2]int
	TotalMoves    [2]int
	VeryBad       [2]int
	Bad           [2]int
	Doubtful      [2]int
	SkillNone     [2]int
	// arErrorCheckerplay[player][0=NORM, 1=UNNORM]
	ErrorCheckerEMG [2]float64
	ErrorCheckerMWC [2]float64
	HasMoves        bool
	// C section
	TotalCube        [2]int
	Double           [2]int
	Take             [2]int
	Pass             [2]int
	MissedDoubleDP   [2]int
	MissedDoubleTG   [2]int
	WrongDoubleDP    [2]int
	WrongDoubleTG    [2]int
	WrongTake        [2]int
	WrongPass        [2]int
	ErrMissedDoubleDP [2][2]float64
	ErrMissedDoubleTG [2][2]float64
	ErrWrongDoubleDP  [2][2]float64
	ErrWrongDoubleTG  [2][2]float64
	ErrWrongTake      [2][2]float64
	ErrWrongPass      [2][2]float64
	HasCube           bool
}

// PlayerStats is the per-player output structure.
type PlayerStats struct {
	CheckerUnforced     int      `json:"checker_unforced"`
	CheckerTotal        int      `json:"checker_total"`
	CheckerForced       int      `json:"checker_forced"`
	CheckerErrors       *int     `json:"checker_errors"`
	CheckerBlunders     *int     `json:"checker_blunders"`
	CheckerEquityEMG    *float64 `json:"checker_equity_error_emg"`
	CheckerMWCLossPct   *float64 `json:"checker_mwc_loss_pct"`
	CheckerPRGnuBG1000  *float64 `json:"checker_pr_gnubg_1000"`
	CheckerPRXG500      *float64 `json:"checker_pr_xg_500"`
	TotalCube           int      `json:"total_cube"`
	DoubleDecisions     int      `json:"double_decisions"`
	TakeDecisions       int      `json:"take_decisions"`
	PassDecisions       int      `json:"pass_decisions"`
	CubeMissedDoubleDP  int      `json:"cube_missed_double_dp"`
	CubeMissedDoubleTG  int      `json:"cube_missed_double_tg"`
	CubeWrongDoubleDP   int      `json:"cube_wrong_double_dp"`
	CubeWrongDoubleTG   int      `json:"cube_wrong_double_tg"`
	CubeWrongTake       int      `json:"cube_wrong_take"`
	CubeWrongPass       int      `json:"cube_wrong_pass"`
	CubeEquityEMG       *float64 `json:"cube_equity_error_emg"`
	CubeMWCLossPct      *float64 `json:"cube_mwc_loss_pct"`
	TotalEquityEMG      *float64 `json:"total_equity_error_emg"`
	TotalMWCLossPct     *float64 `json:"total_mwc_loss_pct"`
	CloseCubeDecisions  *int     `json:"close_cube_decisions"`
	PRGnuBG             *float64 `json:"pr_gnubg"`
}

// MatchOutput is the top-level JSON output.
type MatchOutput struct {
	SGFFile string      `json:"sgf_file"`
	GnuBG   struct {
		Player1 PlayerStats `json:"player1"`
		Player2 PlayerStats `json:"player2"`
	} `json:"gnubg"`
	Notes string `json:"notes"`
}

var (
	reMSection = regexp.MustCompile(`GS\[M:([^\]]+)\]`)
	reCSection = regexp.MustCompile(`\[C:([^\]]+)\]`)
)

func parseInt(s string) (int, string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexAny(s, " \t\n")
	var tok string
	if idx < 0 {
		tok, s = s, ""
	} else {
		tok, s = s[:idx], s[idx+1:]
	}
	v, _ := strconv.Atoi(tok)
	return v, s
}

func parseFloat(s string) (float64, string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexAny(s, " \t\n")
	var tok string
	if idx < 0 {
		tok, s = s, ""
	} else {
		tok, s = s[:idx], s[idx+1:]
	}
	v, _ := strconv.ParseFloat(tok, 64)
	return v, s
}

func parseMSection(raw string) (statContext, bool) {
	var sc statContext
	s := raw
	sc.UnforcedMoves[0], s = parseInt(s)
	sc.UnforcedMoves[1], s = parseInt(s)
	sc.TotalMoves[0], s = parseInt(s)
	sc.TotalMoves[1], s = parseInt(s)
	sc.VeryBad[0], s = parseInt(s)
	sc.VeryBad[1], s = parseInt(s)
	sc.Bad[0], s = parseInt(s)
	sc.Bad[1], s = parseInt(s)
	sc.Doubtful[0], s = parseInt(s)
	sc.Doubtful[1], s = parseInt(s)
	sc.SkillNone[0], s = parseInt(s)
	sc.SkillNone[1], s = parseInt(s)
	sc.ErrorCheckerEMG[0], s = parseFloat(s)
	sc.ErrorCheckerMWC[0], s = parseFloat(s)
	sc.ErrorCheckerEMG[1], s = parseFloat(s)
	sc.ErrorCheckerMWC[1], _ = parseFloat(s)
	sc.HasMoves = true
	return sc, true
}

func parseCSection(raw string) (statContext, bool) {
	var sc statContext
	s := raw
	sc.TotalCube[0], s = parseInt(s)
	sc.TotalCube[1], s = parseInt(s)
	sc.Double[0], s = parseInt(s)
	sc.Double[1], s = parseInt(s)
	sc.Take[0], s = parseInt(s)
	sc.Take[1], s = parseInt(s)
	sc.Pass[0], s = parseInt(s)
	sc.Pass[1], s = parseInt(s)
	sc.MissedDoubleDP[0], s = parseInt(s)
	sc.MissedDoubleDP[1], s = parseInt(s)
	sc.MissedDoubleTG[0], s = parseInt(s)
	sc.MissedDoubleTG[1], s = parseInt(s)
	sc.WrongDoubleDP[0], s = parseInt(s)
	sc.WrongDoubleDP[1], s = parseInt(s)
	sc.WrongDoubleTG[0], s = parseInt(s)
	sc.WrongDoubleTG[1], s = parseInt(s)
	sc.WrongTake[0], s = parseInt(s)
	sc.WrongTake[1], s = parseInt(s)
	sc.WrongPass[0], s = parseInt(s)
	sc.WrongPass[1], s = parseInt(s)
	// Equity errors: player0 first, then player1
	for p := 0; p < 2; p++ {
		sc.ErrMissedDoubleDP[p][0], s = parseFloat(s)
		sc.ErrMissedDoubleDP[p][1], s = parseFloat(s)
		sc.ErrMissedDoubleTG[p][0], s = parseFloat(s)
		sc.ErrMissedDoubleTG[p][1], s = parseFloat(s)
		sc.ErrWrongDoubleDP[p][0], s = parseFloat(s)
		sc.ErrWrongDoubleDP[p][1], s = parseFloat(s)
		sc.ErrWrongDoubleTG[p][0], s = parseFloat(s)
		sc.ErrWrongDoubleTG[p][1], s = parseFloat(s)
		sc.ErrWrongTake[p][0], s = parseFloat(s)
		sc.ErrWrongTake[p][1], s = parseFloat(s)
		sc.ErrWrongPass[p][0], s = parseFloat(s)
		sc.ErrWrongPass[p][1], s = parseFloat(s)
	}
	sc.HasCube = true
	return sc, true
}

func addM(agg, g *statContext) {
	for p := 0; p < 2; p++ {
		agg.UnforcedMoves[p] += g.UnforcedMoves[p]
		agg.TotalMoves[p] += g.TotalMoves[p]
		agg.VeryBad[p] += g.VeryBad[p]
		agg.Bad[p] += g.Bad[p]
		agg.Doubtful[p] += g.Doubtful[p]
		agg.SkillNone[p] += g.SkillNone[p]
		agg.ErrorCheckerEMG[p] += g.ErrorCheckerEMG[p]
		agg.ErrorCheckerMWC[p] += g.ErrorCheckerMWC[p]
	}
}

func addC(agg, g *statContext) {
	for p := 0; p < 2; p++ {
		agg.TotalCube[p] += g.TotalCube[p]
		agg.Double[p] += g.Double[p]
		agg.Take[p] += g.Take[p]
		agg.Pass[p] += g.Pass[p]
		agg.MissedDoubleDP[p] += g.MissedDoubleDP[p]
		agg.MissedDoubleTG[p] += g.MissedDoubleTG[p]
		agg.WrongDoubleDP[p] += g.WrongDoubleDP[p]
		agg.WrongDoubleTG[p] += g.WrongDoubleTG[p]
		agg.WrongTake[p] += g.WrongTake[p]
		agg.WrongPass[p] += g.WrongPass[p]
		for j := 0; j < 2; j++ {
			agg.ErrMissedDoubleDP[p][j] += g.ErrMissedDoubleDP[p][j]
			agg.ErrMissedDoubleTG[p][j] += g.ErrMissedDoubleTG[p][j]
			agg.ErrWrongDoubleDP[p][j] += g.ErrWrongDoubleDP[p][j]
			agg.ErrWrongDoubleTG[p][j] += g.ErrWrongDoubleTG[p][j]
			agg.ErrWrongTake[p][j] += g.ErrWrongTake[p][j]
			agg.ErrWrongPass[p][j] += g.ErrWrongPass[p][j]
		}
	}
}

func fp(v float64) *float64 { return &v }
func ip(v int) *int         { return &v }

func buildPlayer(sc *statContext, p int) PlayerStats {
	ps := PlayerStats{
		CheckerUnforced:    sc.UnforcedMoves[p],
		CheckerTotal:       sc.TotalMoves[p],
		CheckerForced:      sc.TotalMoves[p] - sc.UnforcedMoves[p],
		TotalCube:          sc.TotalCube[p],
		DoubleDecisions:    sc.Double[p],
		TakeDecisions:      sc.Take[p],
		PassDecisions:      sc.Pass[p],
		CubeMissedDoubleDP: sc.MissedDoubleDP[p],
		CubeMissedDoubleTG: sc.MissedDoubleTG[p],
		CubeWrongDoubleDP:  sc.WrongDoubleDP[p],
		CubeWrongDoubleTG:  sc.WrongDoubleTG[p],
		CubeWrongTake:      sc.WrongTake[p],
		CubeWrongPass:      sc.WrongPass[p],
	}

	hasCheckerStats := sc.ErrorCheckerEMG[p] != 0
	if sc.HasMoves {
		errors := sc.VeryBad[p] + sc.Bad[p] + sc.Doubtful[p]
		blunders := sc.VeryBad[p] + sc.Bad[p]
		ps.CheckerErrors = ip(errors)
		ps.CheckerBlunders = ip(blunders)
		if hasCheckerStats {
			emg := sc.ErrorCheckerEMG[p]
			mwc := sc.ErrorCheckerMWC[p] * 100
			ps.CheckerEquityEMG = fp(round6(emg))
			ps.CheckerMWCLossPct = fp(round4(mwc))
			if sc.UnforcedMoves[p] > 0 {
				ps.CheckerPRGnuBG1000 = fp(round3(1000 * emg / float64(sc.UnforcedMoves[p])))
				ps.CheckerPRXG500 = fp(round3(500 * emg / float64(sc.UnforcedMoves[p])))
			}
		}
	}

	hasCubeStats := sc.ErrMissedDoubleDP[p][0] != 0 ||
		sc.ErrMissedDoubleTG[p][0] != 0 ||
		sc.ErrWrongDoubleDP[p][0] != 0 ||
		sc.ErrWrongDoubleTG[p][0] != 0 ||
		sc.ErrWrongTake[p][0] != 0 ||
		sc.ErrWrongPass[p][0] != 0

	if sc.HasCube && hasCubeStats {
		cubeEMG := sc.ErrMissedDoubleDP[p][0] + sc.ErrMissedDoubleTG[p][0] +
			sc.ErrWrongDoubleDP[p][0] + sc.ErrWrongDoubleTG[p][0] +
			sc.ErrWrongTake[p][0] + sc.ErrWrongPass[p][0]
		cubeMWC := sc.ErrMissedDoubleDP[p][1] + sc.ErrMissedDoubleTG[p][1] +
			sc.ErrWrongDoubleDP[p][1] + sc.ErrWrongDoubleTG[p][1] +
			sc.ErrWrongTake[p][1] + sc.ErrWrongPass[p][1]
		ps.CubeEquityEMG = fp(round6(cubeEMG))
		ps.CubeMWCLossPct = fp(round4(cubeMWC * 100))

		if hasCheckerStats {
			totEMG := sc.ErrorCheckerEMG[p] + cubeEMG
			totMWC := sc.ErrorCheckerMWC[p] + cubeMWC
			ps.TotalEquityEMG = fp(round6(totEMG))
			ps.TotalMWCLossPct = fp(round4(totMWC * 100))
		}
	} else if hasCheckerStats {
		ps.TotalEquityEMG = fp(round6(sc.ErrorCheckerEMG[p]))
		ps.TotalMWCLossPct = fp(round4(sc.ErrorCheckerMWC[p] * 100))
	}

	return ps
}

func round3(v float64) float64 { return roundN(v, 1000) }
func round4(v float64) float64 { return roundN(v, 10000) }
func round6(v float64) float64 { return roundN(v, 1000000) }
func roundN(v float64, n float64) float64 {
	if v < 0 {
		return -roundN(-v, n)
	}
	return float64(int(v*n+0.5)) / n
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: extract_gnubg_stats <file.sgf>")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
	content := string(data)

	var aggM, aggC statContext

	for _, m := range reMSection.FindAllStringSubmatch(content, -1) {
		sc, _ := parseMSection(m[1])
		addM(&aggM, &sc)
		aggM.HasMoves = true
	}
	for _, m := range reCSection.FindAllStringSubmatch(content, -1) {
		sc, _ := parseCSection(m[1])
		addC(&aggC, &sc)
		aggC.HasCube = true
	}

	// Merge
	agg := aggM
	agg.TotalCube = aggC.TotalCube
	agg.Double = aggC.Double
	agg.Take = aggC.Take
	agg.Pass = aggC.Pass
	agg.MissedDoubleDP = aggC.MissedDoubleDP
	agg.MissedDoubleTG = aggC.MissedDoubleTG
	agg.WrongDoubleDP = aggC.WrongDoubleDP
	agg.WrongDoubleTG = aggC.WrongDoubleTG
	agg.WrongTake = aggC.WrongTake
	agg.WrongPass = aggC.WrongPass
	agg.ErrMissedDoubleDP = aggC.ErrMissedDoubleDP
	agg.ErrMissedDoubleTG = aggC.ErrMissedDoubleTG
	agg.ErrWrongDoubleDP = aggC.ErrWrongDoubleDP
	agg.ErrWrongDoubleTG = aggC.ErrWrongDoubleTG
	agg.ErrWrongTake = aggC.ErrWrongTake
	agg.ErrWrongPass = aggC.ErrWrongPass
	agg.HasCube = aggC.HasCube

	out := MatchOutput{SGFFile: os.Args[1]}
	out.GnuBG.Player1 = buildPlayer(&agg, 0)
	out.GnuBG.Player2 = buildPlayer(&agg, 1)
	out.Notes = "Generated by cmd/extract_gnubg_stats. close_cube_decisions (anCloseCube) not stored in SGF — pr_gnubg unavailable."

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "json encode: %v\n", err)
		os.Exit(1)
	}
}
