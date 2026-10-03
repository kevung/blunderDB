package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runStats dispatches `blunderdb stats <sub-command>`. The global statistics
// stay under `list --type stats`; what lives here are the views that are
// computed apart from them.
func (cli *CLI) runStats(args []string) error {
	if len(args) < 1 {
		cli.printStatsUsage()
		return fmt.Errorf("missing stats sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printStatsUsage()
		return nil
	}
	run, ok := cli.statsHandlers()[sub]
	if !ok {
		cli.printStatsUsage()
		return fmt.Errorf("unknown stats sub-command: %s", args[0])
	}
	return run(args[1:])
}

// statsHandlers returns the sub-command table of `blunderdb stats`. It is a
// map, like handlers(), so the parity test can walk it.
func (cli *CLI) statsHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"recurring": cli.runStatsRecurring,
	}
}

// StatsSubcommands returns the sub-commands of `blunderdb stats`, sorted — the
// exported view cmd/cli-doc-gen walks.
func (cli *CLI) StatsSubcommands() []string {
	h := cli.statsHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printStatsUsage() {
	fmt.Println("Usage: blunderdb stats <sub-command> [options]")
	fmt.Println()
	fmt.Println("Statistics computed apart from `list --type stats`.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  recurring  Errors grouped by plan of play and theme, costliest first")
}

func (cli *CLI) runStatsRecurring(args []string) error {
	fs := flag.NewFlagSet("stats recurring", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	player := fs.String("player", "", "Only this player's decisions")
	tournament := fs.String("tournament", "", "Filter by tournament IDs, comma-separated")
	from := fs.String("from", "", "Start date filter YYYY-MM-DD")
	to := fs.String("to", "", "End date filter YYYY-MM-DD")
	decisionType := fs.String("decision-type", "all", "Decision type: all, checker, or cube")
	limit := fs.Int("limit", 20, "Maximum number of groups shown (text only; 0 = all)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats recurring --db <file> [options]")
		fmt.Println()
		fmt.Println("Group the errors of the filter by plan of play and theme, costliest first.")
		fmt.Println("A checker theme is the reason the explanation rules name (gammon, blots,")
		fmt.Println("point, passive); a cube theme is the direction of the cube error. An error")
		fmt.Println("no rule names is listed apart, per plan of play, outside the ranking (JSON:")
		fmt.Println("\"Unthemed\"): the rules only speak from 60 mp, above the Error threshold.")
		fmt.Println("Cost is the share of the filter's PR the group accounts for; an error is a")
		fmt.Println("counted decision costing at least the library's Error threshold.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats recurring --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats recurring --db database.db --decision-type checker --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	filter := StatsFilter{PlayerName: *player, DateFrom: *from, DateTo: *to, DecisionType: -1}
	switch strings.ToLower(*decisionType) {
	case "all":
	case "checker":
		filter.DecisionType = 0
	case "cube":
		filter.DecisionType = 1
	default:
		return fmt.Errorf("invalid --decision-type %q: want all, checker or cube", *decisionType)
	}
	if *tournament != "" {
		ids, err := parseIDList(*tournament)
		if err != nil {
			return fmt.Errorf("invalid --tournament: %w", err)
		}
		filter.TournamentIDs = ids
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	textOutput := strings.ToLower(*format) != "json"
	var res *storage.RecurringErrors
	err := withInterruptibleContext(func() {
		if textOutput {
			fmt.Println("\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		res, err = cli.db.ComputeRecurringErrorsCtx(ctx, filter)
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("stats cancelled")
		}
		return fmt.Errorf("recurring errors: %w", err)
	}
	if !textOutput {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal recurring errors: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	printRecurringErrors(res, *limit)
	return nil
}

// printRecurringErrors writes the groups as a table, the costliest first.
func printRecurringErrors(res *storage.RecurringErrors, limit int) {
	fmt.Printf("Recurring errors — %d counted decisions, error threshold %d mp\n\n", res.NumDecisions, res.ThresholdMP)
	if len(res.Groups) == 0 && len(res.Unthemed) == 0 {
		fmt.Println("No error in this filter.")
		return
	}
	if len(res.Groups) > 0 {
		printRecurringRanking(res, limit)
	}
	if len(res.Unthemed) > 0 {
		fmt.Println()
		fmt.Printf("No theme identified (the explanation rules speak from %d mp), per plan of play:\n", engine.ExplainMinCostMP)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PLAN\tERRORS\tCOST (PR)\tPOSITIONS")
		for _, g := range res.Unthemed {
			fmt.Fprintf(w, "%s\t%d\t%.2f\t%s\n", g.GameType, g.Count, g.PRCost, idPreview(g.PositionIDs, 5))
		}
		w.Flush()
	}
}

// printRecurringRanking writes the themed groups, the costliest first.
func printRecurringRanking(res *storage.RecurringErrors, limit int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLAN\tKIND\tTHEME\tERRORS\tCOST (PR)\tPOSITIONS")
	shown := res.Groups
	if limit > 0 && limit < len(shown) {
		shown = shown[:limit]
	}
	for _, g := range shown {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%.2f\t%s\n", g.GameType, g.Kind, g.Theme, g.Count, g.PRCost, idPreview(g.PositionIDs, 5))
	}
	w.Flush()
	if len(shown) < len(res.Groups) {
		fmt.Printf("\n(Showing %d of %d groups, use --limit 0 to see all)\n", len(shown), len(res.Groups))
	}
}

// idPreview lists the first n ids, the worst errors first, and how many more.
func idPreview(ids []int64, n int) string {
	parts := make([]string, 0, n)
	for i, id := range ids {
		if i == n {
			return strings.Join(parts, ",") + fmt.Sprintf(" (+%d)", len(ids)-n)
		}
		parts = append(parts, fmt.Sprint(id))
	}
	return strings.Join(parts, ",")
}
