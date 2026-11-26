package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// runRolls handles the rolls command: the 21 rolls of a position about to be
// rolled, each with its best play, the equity after it and whether it loses
// the market. Pure computation, no database: gammonnet.ComputeRollMap, as the
// GUI and daemon call it.
func (cli *CLI) runRolls(args []string) error {
	cmd := flag.NewFlagSet("rolls", flag.ContinueOnError)

	format := cmd.String("format", "text", "Output format: text, json")
	ply := cmd.Int("ply", 2, "Search depth for every roll (0-2)")
	pruneK := cmd.Int("prune-k", 12, "Candidate moves kept by the prune network")
	jobs := cmd.Int("jobs", 0, "Parallel searches (0 = one per core)")
	roll := cmd.String("roll", "", "Play this roll first (e.g. 31) and show the opponent's replies")
	move := cmd.String("move", "", "With --roll: the play made (any legal one; empty for a dance)")

	cmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb rolls [options] <XGID|OGID>")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Show what each of the 21 rolls does for the player about to roll: the")
		fmt.Fprintln(cli.stdout(), "best play, the equity after it, and whether it loses the market (after")
		fmt.Fprintln(cli.stdout(), "it, a double would be passed). The market is shown only when the player")
		fmt.Fprintln(cli.stdout(), "may double. The dice of the position are ignored.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "The roll-weighted average, cube not turned, is a search one ply deeper")
		fmt.Fprintln(cli.stdout(), "with the same settings: the position's 1- or 2-ply equity at --ply 0")
		fmt.Fprintln(cli.stdout(), "or 1; at --ply 2, a 3-ply search with the 2-ply filters, which is not")
		fmt.Fprintln(cli.stdout(), "the engine's published 3-ply. With --roll and --move, the position is")
		fmt.Fprintln(cli.stdout(), "first played and the table lists the opponent's replies.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		cmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb rolls 'XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10'")
		fmt.Fprintln(cli.stdout(), "  blunderdb rolls --ply 1 --format json '<XGID>'")
		fmt.Fprintln(cli.stdout(), "  blunderdb rolls --roll 31 --move '8/5 6/5' '<XGID>'  # the replies to 8/5 6/5")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if cmd.NArg() != 1 {
		cmd.Usage()
		return fmt.Errorf("expected exactly one XGID or OGID argument")
	}
	if err := gammonnet.ValidateMapPly(*ply); err != nil {
		return err
	}

	pos, err := domain.DecodePositionID(cmd.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid XGID or OGID: %w", err)
	}
	if *roll != "" {
		d1, d2, err := parseRoll(*roll)
		if err != nil {
			return err
		}
		if pos, err = gammonnet.PositionAfterPlay(pos, d1, d2, *move); err != nil {
			return err
		}
	} else if *move != "" {
		return fmt.Errorf("--move needs --roll")
	}

	m, err := gammonnet.ComputeRollMap(context.Background(), pos, nil, *ply, *pruneK,
		gammonnet.LocalExec(*jobs, *ply, *pruneK), nil)
	if err != nil {
		return err
	}

	if *format == "json" {
		enc := json.NewEncoder(cli.stdout())
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}
	cli.printRollMap(m)
	return nil
}

// parseRoll reads "31", "3-1" or "3 1" as two dice.
func parseRoll(s string) (int, int, error) {
	var digits []int
	for _, r := range s {
		switch {
		case r >= '1' && r <= '6':
			digits = append(digits, int(r-'0'))
		case r == '-' || r == ' ' || r == ',':
		default:
			return 0, 0, fmt.Errorf("invalid roll %q: want two dice such as 31", s)
		}
	}
	if len(digits) != 2 {
		return 0, 0, fmt.Errorf("invalid roll %q: want two dice such as 31", s)
	}
	return digits[0], digits[1], nil
}

func (cli *CLI) printRollMap(m gammonnet.RollMap) {
	fmt.Fprintf(cli.stdout(), "Rolls, %s (equity of the player on roll after the roll)\n\n", gammonnet.DepthLabel(m.Ply))
	fmt.Fprintf(cli.stdout(), "%-5s %4s %9s  %-7s %s\n", "roll", "/36", "equity", "market", "best play")
	for _, r := range m.Rows {
		market := "-"
		if r.MarketLost != nil {
			market = "kept"
			if *r.MarketLost {
				market = "lost"
			}
		}
		play := r.Move
		if r.Dance {
			play = "(dance)"
		}
		fmt.Fprintf(cli.stdout(), "%d-%d   %4d %+9.3f  %-7s %s\n", r.Die2, r.Die1, r.Weight, r.Equity, market, play)
	}
	fmt.Fprintf(cli.stdout(), "\nAverage %+.3f\n", m.Average)
}
