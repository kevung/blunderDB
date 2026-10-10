package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// MatchSubcommands lists the sub-commands of `match`, beside its own flags
// (CLI_USAGE.md is generated from it).
func (cli *CLI) MatchSubcommands() []string { return []string{"repair"} }

// runMatchRepair is `match repair`: the match panel's repair, through the
// same RepairMatch as the daemon's matches.repair.
func (cli *CLI) runMatchRepair(args []string) error {
	fs := flag.NewFlagSet("match repair", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	matchID := fs.Int64("id", 0, "Match ID to repair")
	list := fs.Bool("list", false, "List the matches that differ from their source file (writes nothing)")
	dryRun := fs.Bool("dry-run", false, "Say what the repair would change, without writing")
	format := fs.String("format", "text", "Output format: text, json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb match repair --db FILE (--id N [--dry-run] | --list)")
		fmt.Println()
		fmt.Println("Rewrite a match's positions, analyses and errors from its source file,")
		fmt.Println("keeping its id, tournament, names, comments, transcription and video marks.")
		fmt.Println("Refused when the source file is missing or no longer holds the match.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb match repair --db base.db --list")
		fmt.Println("  blunderdb match repair --db base.db --id 42 --dry-run")
		fmt.Println("  blunderdb match repair --db base.db --id 42")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" || (*matchID <= 0) == !*list {
		fs.Usage()
		return fmt.Errorf("usage: blunderdb match repair --db FILE (--id N [--dry-run] | --list)")
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("invalid --format %q: text or json", *format)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	if *list {
		found, err := cli.db.FindMatchesToRepair()
		if err != nil {
			return err
		}
		if *format == "json" {
			return json.NewEncoder(os.Stdout).Encode(found)
		}
		for _, m := range found {
			if m.Error != "" {
				fmt.Printf("%d\t%s\t%s\n", m.MatchID, m.FilePath, m.Error)
				continue
			}
			fmt.Printf("%d\t%s\t%s\n", m.MatchID, m.FilePath, repairLine(m.RepairReport))
		}
		fmt.Printf("%d match(es) to repair or without a usable source\n", len(found))
		return nil
	}

	rep, err := cli.db.RepairMatch(*matchID, *dryRun)
	if err != nil {
		return err
	}
	if *format == "json" {
		return json.NewEncoder(os.Stdout).Encode(rep)
	}
	verb := "repaired"
	if rep.DryRun {
		verb = "would change"
	}
	if !rep.Differs() {
		verb = "matches its source"
	}
	fmt.Printf("match %d %s: %s\n", rep.MatchID, verb, repairLine(rep))
	return nil
}

// repairLine is a report on one line: moves on a position and analysed
// moves, before → after.
func repairLine(r ingest.RepairReport) string {
	return fmt.Sprintf("%d moves, %d changed, analysed %s → %s, comments moved %d",
		r.After.Moves, r.ChangedMoves, pct(r.Before), pct(r.After), r.RehomedComments)
}

func pct(c ingest.RepairCounts) string {
	if c.Moves == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(c.Scored)/float64(c.Moves))
}
