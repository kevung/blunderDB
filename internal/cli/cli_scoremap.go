package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// runScoreMap handles the scoremap command: the best decision of one
// position at every away × away score of a match — the best play when the
// position has dice, the cube verdict otherwise. A command of its own rather
// than a cubematrix flag, so cubematrix's scripts keep their output and
// defaults. Pure computation: gammonnet.ComputeScoreMap, as the GUI and
// daemon call it.
func (cli *CLI) runScoreMap(args []string) error {
	cmd := flag.NewFlagSet("scoremap", flag.ContinueOnError)

	format := cmd.String("format", "text", "Output format: text, json")
	length := cmd.Int("match-length", gammonnet.DefaultMapMatchLength, "Match length the grid spans (odd, 3-15)")
	ply := cmd.Int("ply", 2, "Search depth for every cell (0-2)")
	pruneK := cmd.Int("prune-k", 12, "Candidate moves kept by the prune network")
	jobs := cmd.Int("jobs", 0, "Parallel searches (0 = one per core)")

	cmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb scoremap [options] <XGID|OGID|ArkID>")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Show the best decision of one position at every score of a match. With")
		fmt.Fprintln(cli.stdout(), "dice, each away × away cell gives the best play, its equity, the second")
		fmt.Fprintln(cli.stdout(), "play and the gap; without dice, the cube verdict, as cubematrix does.")
		fmt.Fprintln(cli.stdout(), "The position's score is replaced by the grid; its cube is kept, and a")
		fmt.Fprintln(cli.stdout(), "cell where that cube cannot stand is marked not evaluable. With dice, a")
		fmt.Fprintln(cli.stdout(), "cell at 1-away against more is the Crawford game (\"crawford\": true in")
		fmt.Fprintln(cli.stdout(), "JSON, a C in text); a cube cell is post-Crawford, as in cubematrix.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		cmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb scoremap 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:6:4:0:9:10'")
		fmt.Fprintln(cli.stdout(), "  blunderdb scoremap --match-length 5 --ply 0 --format json '<XGID>'")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if cmd.NArg() != 1 {
		cmd.Usage()
		return fmt.Errorf("expected exactly one XGID, OGID or ArkID argument")
	}
	if err := gammonnet.ValidateMapMatchLength(*length); err != nil {
		return err
	}
	if err := gammonnet.ValidateMapPly(*ply); err != nil {
		return err
	}

	pos, err := domain.DecodePositionID(cmd.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid XGID, OGID or ArkID: %w", err)
	}

	m, err := gammonnet.ComputeScoreMap(context.Background(), pos, nil, *length, *ply, *pruneK,
		gammonnet.LocalExec(*jobs, *ply, *pruneK), nil)
	if err != nil {
		return err
	}

	if *format == "json" {
		enc := json.NewEncoder(cli.stdout())
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}
	if m.Decision == "cube" {
		matrix := gammonnet.CubeMatrix{MatchLength: m.MatchLength, Ply: m.Ply}
		for _, c := range m.Cells {
			matrix.Cells = append(matrix.Cells, c.CubeMatrixCell)
		}
		cli.printCubeMatrix(matrix)
		return nil
	}
	cli.printScoreMap(m)
	return nil
}

// printScoreMap lists the checker cells, one per line: a grid of plays would
// not fit a terminal.
func (cli *CLI) printScoreMap(m gammonnet.ScoreMap) {
	fmt.Fprintf(cli.stdout(), "Score map, %d-point match, %s (away on roll / away opponent)\n\n",
		m.MatchLength, gammonnet.DepthLabel(m.Ply))
	fmt.Fprintf(cli.stdout(), "%-7s %9s  %-22s %7s  %s\n", "score", "equity", "best play", "gap", "second play")
	for _, c := range m.Cells {
		score := fmt.Sprintf("%d/%d", c.AwayOnRoll, c.AwayOpponent)
		if c.Crawford {
			score += " C"
		}
		if c.Refused {
			fmt.Fprintf(cli.stdout(), "%-7s not evaluable: %s\n", score, c.Reason)
			continue
		}
		gap := "-"
		if c.Gap != nil {
			gap = fmt.Sprintf("%.3f", *c.Gap)
		}
		fmt.Fprintf(cli.stdout(), "%-7s %+9.3f  %-22s %7s  %s\n", score, c.Equity, c.Move, gap, c.SecondMove)
	}
}
