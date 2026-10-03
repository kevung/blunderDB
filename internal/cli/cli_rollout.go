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
	preset := cmd.String("preset", "fast", "Starting settings, both at 0 ply: fast (216 games, truncated at 7) or standard (1296 games, truncated at 11); the flags below override it, --ply plays deeper")
	games := cmd.Int("games", 0, "Most games per candidate (0 = the preset's)")
	minGames := cmd.Int("min-games", -1, "Games before the JSD rule may stop a candidate (-1 = the preset's)")
	truncation := cmd.Int("truncation", -1, "Half-moves per game before the engine values it; 0 plays to the end (-1 = the preset's)")
	jsd := cmd.Float64("jsd", -1, "Stop a candidate when its gap to the best reaches this many standard deviations; 0 never stops early (-1 = the preset's)")
	ply := cmd.Int("ply", -1, "gammonNet depth of the plays, cube actions and leaves inside the games (-1 = the preset's)")
	candidates := cmd.Int("candidates", 0, "Plays rolled out when no --move is given, best first at max(--ply, 2) (0 = the preset's)")
	seed := cmd.Uint64("seed", rollout.DefaultSeed, "Dice seed: the same seed gives the same numbers")
	jobs := cmd.Int("jobs", 0, "Games played at once (0 = one per core); never changes the numbers")
	var moves moveList
	cmd.Var(&moves, "move", "A play to roll out, in blunderDB notation (repeatable)")
	dbPath := cmd.String("db", "", "Database to read the position from (with --id)")
	positionID := cmd.Int64("id", 0, "Position of --db to roll out, in place of an XGID or OGID")
	store := cmd.Bool("store", false, "Write the finished rollout on the position of --db, beside its analysis (never replacing it)")
	listStored := cmd.Bool("list", false, "Print the rollouts stored on the position of --db instead of rolling it out")

	cmd.Usage = func() {
		fmt.Println("Usage: blunderdb rollout [options] <XGID|OGID>")
		fmt.Println("       blunderdb rollout --db <file> --id <position> [--store] [options]")
		fmt.Println()
		fmt.Println("Roll a position out with gammonNet: its plays when the position has dice,")
		fmt.Println("its cube decision otherwise. Each candidate plays the same dice; the luck")
		fmt.Println("of every roll is taken out of each game (variance reduction); the first two")
		fmt.Println("rolls are stratified; a game stops where the two-sided bearoff table")
		fmt.Println("covers it. The cube is played inside the games (cubeful): trust the")
		fmt.Println("ranking more than the absolute equity.")
		fmt.Println()
		fmt.Println("Equities are money points per unit of the position's cube, or normalised")
		fmt.Println("equity at a match score. Ctrl-C prints what the games finished so far")
		fmt.Println("concluded. With --store, a finished rollout is written on the position as a")
		fmt.Println("second analysis with its own settings, beside the imported or evaluated one;")
		fmt.Println("an interrupted rollout is never stored. A rerun with the same settings")
		fmt.Println("replaces the earlier one.")
		fmt.Println()
		fmt.Println("Options:")
		cmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb rollout 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10'")
		fmt.Println("  blunderdb rollout --move '8/5 6/5' --move '24/23 13/10' '<XGID>'")
		fmt.Println("  blunderdb rollout --preset standard --format json '<XGID>'")
		fmt.Println("  blunderdb rollout --games 648 --truncation 0 --ply 1 '<XGID>'")
		fmt.Println("  blunderdb rollout --db library.db --id 42 --preset standard --store")
		fmt.Println("  blunderdb rollout --db library.db --id 42 --list")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	fromDB := *positionID != 0
	switch {
	case fromDB && *dbPath == "":
		return fmt.Errorf("--id names a position of --db: give the database")
	case fromDB && cmd.NArg() != 0:
		return fmt.Errorf("--id and an XGID/OGID argument both name the position: give one")
	case (*store || *listStored) && !fromDB:
		return fmt.Errorf("--store and --list need a position of the database: give --db and --id")
	case *store && *listStored:
		return fmt.Errorf("--list reads what is stored and rolls nothing out: drop --store")
	case !fromDB && cmd.NArg() != 1:
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

	if err := s.Validate(); err != nil {
		return err
	}
	var pos domain.Position
	if fromDB {
		if err := cli.initDatabase(*dbPath); err != nil {
			return err
		}
		if *listStored {
			return cli.printStoredRollouts(*positionID, *format == "json")
		}
	} else {
		var err error
		pos, err = domain.DecodePositionID(cmd.Arg(0))
		if err != nil {
			return fmt.Errorf("invalid XGID or OGID: %w", err)
		}
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
		if fromDB {
			res, err = cli.db.RolloutPosition(ctx, *positionID, s, moves, *store, opt.Progress)
		} else {
			res, err = rollout.Run(ctx, pos, s, opt)
		}
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
	stored := *store && runErr == nil

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	printRollout(res)
	switch {
	case stored:
		fmt.Printf("\nStored on position %d.\n", *positionID)
	case *store:
		fmt.Println("\nInterrupted: not stored.")
	}
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

// printStoredRollouts lists the rollouts written on a position, newest first.
func (cli *CLI) printStoredRollouts(positionID int64, asJSON bool) error {
	list, err := cli.db.LoadRollouts(positionID)
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}
	if len(list) == 0 {
		fmt.Printf("No rollout stored on position %d.\n", positionID)
		return nil
	}
	for i, r := range list {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("%s — %d games, stopped on %s, %s\n", r.AnalysisDepth, r.Games, r.Stop, r.Date.Format("2006-01-02 15:04"))
		fmt.Println(r.Signature)
		fmt.Printf("  %-24s %9s %9s %7s %7s\n", "Candidate", "Equity", "±95%", "Games", "JSD")
		for _, c := range r.Candidates {
			fmt.Printf("  %-24s %+9.4f %9.4f %7d %7.2f\n", c.Move, c.Equity, c.CI95, c.Games, c.JSD)
		}
		if r.BestCubeAction != "" {
			fmt.Printf("Best cube action: %s\n", r.BestCubeAction)
		}
	}
	return nil
}
