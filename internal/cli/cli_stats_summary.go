package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runStatsSummary prints the public part of the filter's statistics
// (storage.StatsSummary): what a public profile or a business card shows
// without naming a position, a match or an opponent.
func (cli *CLI) runStatsSummary(args []string) error {
	fs := flag.NewFlagSet("stats summary", flag.ContinueOnError)
	player := fs.String("player", "", "Only this player's decisions")
	c := newCorpusStatsFlags(fs)
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats summary --db <file> [options]")
		fmt.Println()
		fmt.Println("The filter's level without any identifier: counts, PR (checker, cube) with")
		fmt.Println("its 95% interval, MWC loss per 7-point match, PR by game phase and by score.")
		fmt.Println("No position, match, tournament, opponent or tag appears in it.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats summary --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats summary --db database.db --player \"Alice\" --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	filter, err := c.open(cli, fs, *player)
	if err != nil {
		return err
	}
	var s *storage.StatsSummary
	err = withInterruptibleContext(func() {}, func(ctx context.Context) error {
		var err error
		s, err = cli.db.StatsSummaryCtx(ctx, filter)
		return err
	})
	if err != nil {
		return statsErr("stats summary", err)
	}
	if c.json() {
		return printJSON(s)
	}
	fmt.Printf("Matches: %d  Decisions: %d\n", s.Totals.NumMatches, s.Totals.NumDecisions)
	fmt.Printf("PR: %s%s  checker %.2f  cube %.2f\n", prCell(s.PRGlobal, s.Totals.NumDecisions), intervalText(s.PRInterval), s.PRChecker, s.PRCube)
	fmt.Printf("MWC loss (7 pt): %s\n", formatMWC7(s.MWC7))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nPHASE\tPR\t95% INTERVAL\tDECISIONS")
	for _, r := range s.PerPhase {
		fmt.Fprintf(w, "%s\t%.2f\t%s\t%d\n", r.Phase, r.PR, intervalText(r.PRInterval), r.NumDecisions)
	}
	fmt.Fprintln(w, "\nSCORE (AWAY x AWAY)\tPR\t95% INTERVAL\tDECISIONS")
	for _, r := range s.PerScore {
		score := fmt.Sprintf("%d-away vs %d-away", r.MoverAway, r.OpponentAway)
		if r.Money {
			score = "money"
		}
		fmt.Fprintf(w, "%s\t%.2f\t%s\t%d\n", score, r.PR, intervalText(r.PRInterval), r.NumDecisions)
	}
	return w.Flush()
}

// intervalText writes a 95% interval, or a dash when too few units back it.
func intervalText(iv domain.Interval) string {
	if !iv.Available {
		return " (-)"
	}
	return fmt.Sprintf(" [%.2f-%.2f]", iv.Low, iv.High)
}
