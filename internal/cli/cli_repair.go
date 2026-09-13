package cli

import (
	"flag"
	"fmt"
	"strings"
)

// runRepair is `blunderdb repair`: recompute what the database DERIVES from
// what it stores — the scalar columns of every analysis, from the JSON they
// are a projection of, the phase of every position, from its board, and the
// Crawford sentinel of every away score, from the match the position came
// from or, for a position no match points at, the XGID it came in with from
// another program (#338, #360).
//
// The JSON stays intact, so a bug in the projection is repairable without
// re-importing anything — it has been needed once already, when the XG
// importer's "Double No" was read as a real double and gave the column the
// error of a double that never happened (#115). Fixing the reader did nothing
// for the rows already written.
//
// Never automatic, and that is the point: rewriting every analysis column on
// the mere act of opening a database is not something a tool should do behind
// its user's back. It is a schema-preserving repair, not a migration.
//
// The daemon has served this as /v1/analyses.repair since it existed; the CLI
// and the GUI could not reach it until the reverse parity check went looking
// (G.14, #242).
func (cli *CLI) runRepair(args []string) error {
	repairCmd := flag.NewFlagSet("repair", flag.ContinueOnError)

	dbPath := repairCmd.String("db", "", "Path to the database file (required)")
	format := repairCmd.String("format", "text", "Output format: text or json")

	repairCmd.Usage = func() {
		fmt.Println("Usage: blunderdb repair [options]")
		fmt.Println()
		fmt.Println("Recompute what the database derives from what it stores:")
		fmt.Println("the scalar columns of every analysis, from the JSON they are")
		fmt.Println("a projection of, the phase of every position, from its board,")
		fmt.Println("and the Crawford sentinel of every away score, from the match")
		fmt.Println("the position came from or the XGID it came in with from")
		fmt.Println("another program. Useful after a fix to how an imported")
		fmt.Println("analysis is read, after a change to how a phase is decided,")
		fmt.Println("and once, for the databases imported before the importers")
		fmt.Println("wrote the sentinel: a post-Crawford position stored as a")
		fmt.Println("Crawford one is read cube-dead. Correcting it changes the")
		fmt.Println("position's Zobrist hash, so such a position is rehashed and")
		fmt.Println("merged with its correct twin when the database holds one.")
		fmt.Println("Nothing runs it automatically.")
		fmt.Println()
		fmt.Println("Options:")
		repairCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb repair --db database.db")
		fmt.Println("  blunderdb repair --db database.db --format json")
	}

	if err := repairCmd.Parse(args); err != nil {
		return err
	}

	if *dbPath == "" {
		repairCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	repaired, err := cli.db.RepairAnalyses()
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}
	phases, err := cli.db.RepairGamePhases()
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}
	crawford, err := cli.db.RepairCrawfordSentinel()
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}

	if formatLower == "json" {
		return printJSON(struct {
			Repaired int `json:"repaired"`
			Phases   int `json:"phases"`
			Crawford int `json:"crawford"`
		}{repaired, phases, crawford})
	}
	switch repaired {
	case 0:
		fmt.Println("Every analysis column already matched its analysis; nothing to repair.")
	case 1:
		fmt.Println("1 analysis repaired.")
	default:
		fmt.Printf("%d analyses repaired.\n", repaired)
	}
	switch phases {
	case 0:
		fmt.Println("Every position already carried the right phase.")
	case 1:
		fmt.Println("1 position reclassified.")
	default:
		fmt.Printf("%d positions reclassified.\n", phases)
	}
	switch crawford {
	case 0:
		fmt.Println("Every position already carried the right Crawford sentinel.")
	case 1:
		fmt.Println("1 post-Crawford position rehashed.")
	default:
		fmt.Printf("%d post-Crawford positions rehashed.\n", crawford)
	}
	return nil
}
