package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

// moveList collects a repeatable --move flag.
type moveList []string

func (m *moveList) String() string { return strings.Join(*m, "; ") }
func (m *moveList) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// runRollout handles the rollout command: one position given as an XGID or
// OGID, rolled out by pkg/blunderdb/engine/rollout. Pure computation, no
// database: nothing is stored.
func (cli *CLI) runRollout(args []string) error {
	cmd := flag.NewFlagSet("rollout", flag.ContinueOnError)

	format := cmd.String("format", "text", "Output format: text, json")
	preset := cmd.String("preset", "fast", "Starting settings: fast (216 games, truncated at 7) or standard (1296 games, truncated at 11); the flags below override it")
	games := cmd.Int("games", 0, "Most games per candidate (0 = the preset's)")
	minGames := cmd.Int("min-games", -1, "Games before the JSD rule may stop a candidate (-1 = the preset's)")
	truncation := cmd.Int("truncation", -1, "Half-moves per game before the engine values it; 0 plays to the end (-1 = the preset's)")
	jsd := cmd.Float64("jsd", -1, "Stop a candidate when its gap to the best reaches this many standard deviations; 0 never stops early (-1 = the preset's)")
	ply := cmd.Int("ply", -1, "gammonNet depth of the plays, cube actions and leaves inside the games (-1 = the preset's)")
	candidates := cmd.Int("candidates", 0, "Plays rolled out when no --move is given, best first (0 = the preset's)")
	seed := cmd.Uint64("seed", rollout.DefaultSeed, "Dice seed: the same seed gives the same numbers")
	jobs := cmd.Int("jobs", 0, "Games played at once (0 = one per core); never changes the numbers")
	var moves moveList
	cmd.Var(&moves, "move", "A play to roll out, in blunderDB notation (repeatable)")

	cmd.Usage = func() {
		fmt.Println("Usage: blunderdb rollout [options] <XGID|OGID>")
		fmt.Println()
		fmt.Println("Roll a position out with gammonNet: its plays when the position has dice,")
		fmt.Println("its cube decision otherwise. Each candidate plays the same dice; the luck")
		fmt.Println("of every roll is taken out of each game (variance reduction); the first two")
		fmt.Println("rolls are stratified; a game stops where the two-sided bearoff table")
		fmt.Println("covers it. The cube is played inside the games (cubeful): trust the")
		fmt.Println("ranking more than the absolute equity.")
		fmt.Println()
		fmt.Println("Equities are money points per unit of the position's cube, or normalised")
		fmt.Println("equity at a match score. Nothing is stored. Ctrl-C prints what the games")
		fmt.Println("finished so far concluded.")
		fmt.Println()
		fmt.Println("Options:")
		cmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb rollout 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10'")
		fmt.Println("  blunderdb rollout --move '8/5 6/5' --move '24/23 13/10' '<XGID>'")
		fmt.Println("  blunderdb rollout --preset standard --format json '<XGID>'")
		fmt.Println("  blunderdb rollout --games 648 --truncation 0 --ply 1 '<XGID>'")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if cmd.NArg() != 1 {
		cmd.Usage()
		return fmt.Errorf("expected exactly one XGID or OGID argument")
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("unknown format %q (text, json)", *format)
	}

	s, ok := rollout.Preset(*preset)
	if !ok {
		return fmt.Errorf("unknown preset %q (fast, standard)", *preset)
	}
	if *games > 0 {
		s.MaxGames = *games
		s.MinGames = min(s.MinGames, s.MaxGames)
	}
	if *minGames >= 0 {
		s.MinGames = *minGames
	}
	if *truncation >= 0 {
		s.Truncation = *truncation
	}
	if *jsd >= 0 {
		s.JSDLimit = *jsd
	}
	if *ply >= 0 {
		s.Ply = *ply
	}
	if *candidates > 0 {
		s.Candidates = *candidates
	}
	s.Seed = *seed
	s.Workers = *jobs

	pos, err := domain.DecodePositionID(cmd.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid XGID or OGID: %w", err)
	}

	opt := rollout.Options{Moves: moves}
	if *format == "text" {
		opt.Progress = func(p rollout.Progress) {
			fmt.Fprintf(os.Stderr, "\r%d/%d games", p.Games, p.MaxGames)
		}
	}

	var res *rollout.Result
	runErr := withInterruptibleContext(nil, func(ctx context.Context) error {
		var err error
		res, err = rollout.Run(ctx, pos, s, opt)
		return err
	})
	if opt.Progress != nil {
		fmt.Fprintln(os.Stderr)
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		return runErr
	}
	if res == nil {
		return runErr
	}

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	printRollout(res)
	return nil
}

// printRollout renders a rollout as XG does: equity, its 95 % interval, the
// games played, and how many standard deviations separate it from the best.
func printRollout(r *rollout.Result) {
	fmt.Printf("Rollout, %d games, stopped on %s\n", r.Games, r.Stop)
	fmt.Println(r.Signature)
	fmt.Println()
	fmt.Printf("  %-24s %9s %9s %7s %7s\n", "Candidate", "Equity", "±95%", "Games", "JSD")
	if r.Cube != nil {
		row := func(name string, e rollout.Estimate) {
			fmt.Printf("  %-24s %+9.4f %9.4f %7d %7.2f\n", name, e.Equity, e.CI95, e.Games, e.JSD)
		}
		row(rollout.NoDouble, r.Cube.NoDouble)
		row(rollout.DoubleTake, r.Cube.DoubleTake)
		row(rollout.DoublePass, r.Cube.DoublePass)
		fmt.Println()
		fmt.Printf("Best cube action: %s\n", r.Cube.Action)
	} else {
		for _, c := range r.Candidates {
			fmt.Printf("  %-24s %+9.4f %9.4f %7d %7.2f\n", c.Move, c.Equity, c.CI95, c.Games, c.JSD)
		}
	}
	if r.CubefulBias {
		fmt.Println()
		fmt.Println("Cubeful: the cube model plays inside the games — the ranking is more")
		fmt.Println("reliable than the absolute equity.")
	}
}
