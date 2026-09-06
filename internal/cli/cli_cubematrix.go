package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// runCubeMatrix handles the cubematrix command: the cube verdict at every
// away × away score of a match, for one position given as an XGID (issue
// #267, fiche I.11).
//
// Pure computation, like epc: no database is opened, and the grid comes from
// the same gammonnet.ComputeCubeMatrix the Eval panel's tab and the daemon's
// /v1/gammonnet.cubeMatrix call.
func (cli *CLI) runCubeMatrix(args []string) error {
	cmd := flag.NewFlagSet("cubematrix", flag.ContinueOnError)

	format := cmd.String("format", "text", "Output format: text, json")
	length := cmd.Int("match-length", 7, "Match length the grid spans (1-25)")
	ply := cmd.Int("ply", 2, "Search depth for every cell (0 or 2)")
	pruneK := cmd.Int("prune-k", 12, "Candidate moves kept by the prune network")
	jobs := cmd.Int("jobs", 0, "Parallel searches (0 = one per core)")

	cmd.Usage = func() {
		fmt.Println("Usage: blunderdb cubematrix [options] <XGID>")
		fmt.Println()
		fmt.Println("Show the cube verdict at every score of a match: for each away × away")
		fmt.Println("cell, whether this position is a double and whether it is a take.")
		fmt.Println("The position's own score is ignored — the grid replaces it — but its")
		fmt.Println("cube is kept, so the answer is about this cube, not a centred one.")
		fmt.Println()
		fmt.Println("Every cell is its own search: the engine is match-aware, so a single")
		fmt.Println("search read through different match equities would be wrong exactly")
		fmt.Println("where the score matters. The grid is post-Crawford throughout.")
		fmt.Println()
		fmt.Println("Options:")
		cmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb cubematrix 'XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10'")
		fmt.Println("  blunderdb cubematrix --match-length 5 --format json '<XGID>'")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if cmd.NArg() != 1 {
		cmd.Usage()
		return fmt.Errorf("expected exactly one XGID argument")
	}
	if *length < 1 || *length > 25 {
		return fmt.Errorf("match length must be between 1 and 25")
	}

	pos, err := domain.DecodeXGID(cmd.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid XGID: %w", err)
	}

	matrix, err := gammonnet.ComputeCubeMatrix(context.Background(), pos, *length, *ply, *pruneK, *jobs)
	if err != nil {
		return err
	}

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(matrix)
	}
	printCubeMatrix(matrix)
	return nil
}

// cubeMatrixGlyphs names the four verdicts in the two characters a grid can
// afford. The legend is printed under the grid rather than trusted to memory.
var cubeMatrixGlyphs = map[string]string{
	"no_double":   "ND",
	"double_take": "DT",
	"double_pass": "DP",
	"too_good":    "TG",
}

// printCubeMatrix renders the grid: rows are the doubling side's away score,
// columns the opponent's, so reading across a row answers "I need N, they
// need what?" — the question a player actually asks at the table.
func printCubeMatrix(m gammonnet.CubeMatrix) {
	byScore := make(map[[2]int]gammonnet.CubeMatrixCell, len(m.Cells))
	for _, c := range m.Cells {
		byScore[[2]int{c.AwayOnRoll, c.AwayOpponent}] = c
	}

	fmt.Printf("Cube matrix, %d-point match, %s (rows: away on roll, columns: away opponent)\n\n",
		m.MatchLength, gammonnet.DepthLabel(m.Ply))

	var header strings.Builder
	header.WriteString("     ")
	for j := 1; j <= m.MatchLength; j++ {
		fmt.Fprintf(&header, "%4d", j)
	}
	fmt.Println(header.String())

	for i := 1; i <= m.MatchLength; i++ {
		var row strings.Builder
		fmt.Fprintf(&row, "%4d ", i)
		for j := 1; j <= m.MatchLength; j++ {
			cell := byScore[[2]int{i, j}]
			glyph := "  ?"
			if !cell.Refused {
				glyph = " " + cubeMatrixGlyphs[cell.Verdict]
			}
			fmt.Fprintf(&row, "%4s", glyph)
		}
		fmt.Println(row.String())
	}

	fmt.Println()
	fmt.Println("ND no double   DT double, take   DP double, pass   TG too good   ? not evaluable")
	for _, c := range m.Cells {
		if c.Refused {
			fmt.Printf("  %d-away/%d-away: %s\n", c.AwayOnRoll, c.AwayOpponent, c.Reason)
		}
	}
}
