package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runRepair is `blunderdb repair`: recompute what the database derives from
// what it stores — each analysis's scalar columns from its JSON, each
// position's phase from its board, and the Crawford sentinel of each away
// score from its match or, lacking one, from its XGID. The stored JSON stays
// intact, so a projection bug is fixed without re-importing.
//
// Never automatic: it is a schema-preserving repair the user asks for, not a
// migration run on open.
func (cli *CLI) runRepair(args []string) error {
	repairCmd := flag.NewFlagSet("repair", flag.ContinueOnError)

	dbPath := repairCmd.String("db", "", "Path to the database file (required)")
	format := repairCmd.String("format", "text", "Output format: text or json")
	rebuildStats := repairCmd.Bool("stats", false, "Also recompute the per-match statistics (PR, decisions, blunders, luck of each seat) from scratch")
	duplicates := repairCmd.Bool("duplicates", false,
		"Only list the pairs of matches whose dice say they are one: same dice under other player names, or a truncated match and its longer version (nothing is merged)")

	repairCmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb repair [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Recompute what the database derives from what it stores:")
		fmt.Fprintln(cli.stdout(), "the scalar columns of every analysis, from the JSON they are")
		fmt.Fprintln(cli.stdout(), "a projection of, the phase of every position, from its board,")
		fmt.Fprintln(cli.stdout(), "and the Crawford sentinel of every away score, from the match")
		fmt.Fprintln(cli.stdout(), "the position came from or the XGID it came in with from")
		fmt.Fprintln(cli.stdout(), "another program. Useful after a fix to how an imported")
		fmt.Fprintln(cli.stdout(), "analysis is read, after a change to how a phase is decided,")
		fmt.Fprintln(cli.stdout(), "and once, for the databases imported before the importers")
		fmt.Fprintln(cli.stdout(), "wrote the sentinel: a post-Crawford position stored as a")
		fmt.Fprintln(cli.stdout(), "Crawford one is read cube-dead. Correcting it changes the")
		fmt.Fprintln(cli.stdout(), "position's Zobrist hash, so such a position is rehashed and")
		fmt.Fprintln(cli.stdout(), "merged with its correct twin when the database holds one.")
		fmt.Fprintln(cli.stdout(), "A signed comment on such a position then reads unverified:")
		fmt.Fprintln(cli.stdout(), "its signature covers the position's hash. The report counts them.")
		fmt.Fprintln(cli.stdout(), "Nothing runs it automatically.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		repairCmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb repair --db database.db")
		fmt.Fprintln(cli.stdout(), "  blunderdb repair --db database.db --format json")
		fmt.Fprintln(cli.stdout(), "  blunderdb repair --db database.db --stats")
		fmt.Fprintln(cli.stdout(), "  blunderdb repair --db database.db --duplicates")
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

	if *duplicates {
		return cli.repairDuplicates(formatLower)
	}

	repaired, err := cli.db.RepairAnalyses()
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}
	phases, err := cli.db.RepairGamePhases()
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}
	crawfordRep, err := cli.db.RepairCrawfordSentinel()
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}
	crawford, unverified := crawfordRep.Rehashed, crawfordRep.SignaturesUnverified

	// The counter is in the JSON report only when the pass ran.
	matchStats := 0
	var matchStatsReport *int
	if *rebuildStats {
		if matchStats, err = cli.db.RebuildMatchStats(); err != nil {
			return fmt.Errorf("repair failed: %w", err)
		}
		matchStatsReport = &matchStats
	}

	if formatLower == "json" {
		return cli.printJSON(struct {
			Repaired   int  `json:"repaired"`
			Phases     int  `json:"phases"`
			Crawford   int  `json:"crawford"`
			Unverified int  `json:"signatures_unverified"`
			MatchStats *int `json:"match_stats,omitempty"`
		}{repaired, phases, crawford, unverified, matchStatsReport})
	}
	switch repaired {
	case 0:
		fmt.Fprintln(cli.stdout(), "Every analysis column already matched its analysis; nothing to repair.")
	case 1:
		fmt.Fprintln(cli.stdout(), "1 analysis repaired.")
	default:
		fmt.Fprintf(cli.stdout(), "%d analyses repaired.\n", repaired)
	}
	switch phases {
	case 0:
		fmt.Fprintln(cli.stdout(), "Every position already carried the right phase.")
	case 1:
		fmt.Fprintln(cli.stdout(), "1 position reclassified.")
	default:
		fmt.Fprintf(cli.stdout(), "%d positions reclassified.\n", phases)
	}
	switch crawford {
	case 0:
		fmt.Fprintln(cli.stdout(), "Every position already carried the right Crawford sentinel.")
	case 1:
		fmt.Fprintln(cli.stdout(), "1 position rehashed onto the right Crawford sentinel.")
	default:
		fmt.Fprintf(cli.stdout(), "%d positions rehashed onto the right Crawford sentinel.\n", crawford)
	}
	switch unverified {
	case 0:
	case 1:
		fmt.Fprintln(cli.stdout(), "1 signed comment sits on a rehashed position: it now reads unverified, its text untouched.")
	default:
		fmt.Fprintf(cli.stdout(), "%d signed comments sit on rehashed positions: they now read unverified, their text untouched.\n", unverified)
	}
	if *rebuildStats {
		fmt.Fprintf(cli.stdout(), "Per-match statistics recomputed for %d matches.\n", matchStats)
	}
	return nil
}

// repairDuplicates lists the suspected duplicate pairs (repair --duplicates).
func (cli *CLI) repairDuplicates(format string) error {
	suspects, err := cli.db.FindDuplicateMatches()
	if err != nil {
		return fmt.Errorf("finding duplicate matches: %w", err)
	}
	if format == "json" {
		if suspects == nil {
			suspects = []domain.DuplicateSuspect{}
		}
		return cli.printJSON(struct {
			Suspects []domain.DuplicateSuspect `json:"suspects"`
		}{suspects})
	}
	if len(suspects) == 0 {
		fmt.Fprintln(cli.stdout(), "No suspected duplicate match.")
		return nil
	}
	for _, s := range suspects {
		switch s.Kind {
		case domain.DuplicateLonger:
			fmt.Fprintf(cli.stdout(), "#%d (%s) is a longer version of #%d (%s)\n", s.MatchID, s.Players, s.OtherID, s.OtherPlayers)
		default:
			fmt.Fprintf(cli.stdout(), "#%d (%s) has the dice of #%d (%s)\n", s.MatchID, s.Players, s.OtherID, s.OtherPlayers)
			// Each reading of who is who, as the alias commands that record it.
			for i, p := range s.Pairings {
				if len(s.Pairings) > 1 {
					fmt.Fprintf(cli.stdout(), "  reading %d:\n", i+1)
				}
				for _, a := range p.Aliases {
					fmt.Fprintf(cli.stdout(), "    blunderdb players alias add --db FILE %q %q\n", a.Alias, a.Canonical)
				}
			}
		}
	}
	fmt.Fprintf(cli.stdout(), "%d suspected pair(s); nothing was merged.\n", len(suspects))
	return nil
}
