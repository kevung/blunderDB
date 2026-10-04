package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// provenanceFlags declares --engine and --min-depth on fs and returns what
// applies them to a filter.
func provenanceFlags(fs *flag.FlagSet) func(*StatsFilter) {
	engine := fs.String("engine", "", "Only the decisions analysed by this engine (exact name, as stored)")
	minDepth := fs.Int("min-depth", 0, "Only the decisions analysed at least this deep (plies)")
	return func(f *StatsFilter) {
		f.AnalysisEngine = *engine
		f.MinAnalysisDepth = *minDepth
	}
}

// corpusStatsFlags declares the flags the corpus views share: --db, the
// match-level filter and --format.
type corpusStatsFlags struct {
	db, tournament, from, to, decisionType, format *string
	provenance                                     func(*StatsFilter)
}

func newCorpusStatsFlags(fs *flag.FlagSet) corpusStatsFlags {
	return corpusStatsFlags{
		db:           fs.String("db", "", "Path to the database file (required)"),
		tournament:   fs.String("tournament", "", "Filter the matches by tournament IDs, comma-separated"),
		from:         fs.String("from", "", "Start date filter YYYY-MM-DD (matches)"),
		to:           fs.String("to", "", "End date filter YYYY-MM-DD (matches)"),
		decisionType: fs.String("decision-type", "all", "Decision type: all, checker, or cube"),
		format:       fs.String("format", "text", "Output format: text or json"),
		provenance:   provenanceFlags(fs),
	}
}

// open validates the flags, opens the database and returns the filter.
func (c corpusStatsFlags) open(cli *CLI, fs *flag.FlagSet, player string) (StatsFilter, error) {
	if *c.db == "" {
		fs.Usage()
		return StatsFilter{}, fmt.Errorf("missing required flag: --db")
	}
	filter, err := buildStatsFilter(player, *c.tournament, *c.from, *c.to, *c.decisionType)
	if err != nil {
		return filter, err
	}
	c.provenance(&filter)
	return filter, cli.initDatabase(*c.db)
}

func (c corpusStatsFlags) json() bool { return strings.ToLower(*c.format) == "json" }

func statsErr(what string, err error) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("stats cancelled")
	}
	return fmt.Errorf("%s: %w", what, err)
}

func (cli *CLI) runStatsH2H(args []string) error {
	fs := flag.NewFlagSet("stats h2h", flag.ContinueOnError)
	player := fs.String("player", "", "First player (required)")
	opponent := fs.String("opponent", "", "Second player (required)")
	c := newCorpusStatsFlags(fs)
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats h2h --db <file> --player <name> --opponent <name> [options]")
		fmt.Println()
		fmt.Println("The matches the two players played against each other, each one's PR in")
		fmt.Println("each match and over all of them, and the record (decided matches only).")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats h2h --db database.db --player \"Alice\" --opponent \"Bob\"")
		fmt.Println("  blunderdb stats h2h --db database.db --player \"Alice\" --opponent \"Bob\" --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *player == "" || *opponent == "" {
		fs.Usage()
		return fmt.Errorf("missing required flags: --player and --opponent")
	}
	filter, err := c.open(cli, fs, "")
	if err != nil {
		return err
	}
	var res *storage.HeadToHead
	err = withInterruptibleContext(func() {}, func(ctx context.Context) error {
		var err error
		res, err = cli.db.HeadToHeadCtx(ctx, *player, *opponent, filter)
		return err
	})
	if err != nil {
		return statsErr("head to head", err)
	}
	if c.json() {
		return printJSON(res)
	}
	fmt.Printf("%s vs %s — %d matches, record %d-%d\n", res.PlayerA, res.PlayerB, len(res.Matches), res.WinsA, res.WinsB)
	fmt.Printf("PR %s: %s over %d decisions; PR %s: %s over %d decisions\n\n",
		res.PlayerA, prCell(res.PRA, res.DecisionsA), res.DecisionsA, res.PlayerB, prCell(res.PRB, res.DecisionsB), res.DecisionsB)
	if len(res.Matches) == 0 {
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "MATCH\tDATE\tLENGTH\tWINNER\tPR %s\tPR %s\n", res.PlayerA, res.PlayerB)
	for _, m := range res.Matches {
		winner := "-"
		switch m.Outcome {
		case 1:
			winner = res.PlayerA
		case -1:
			winner = res.PlayerB
		}
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n", m.ID, m.Date, m.MatchLength, winner, prCell(m.PRA, m.DecisionsA), prCell(m.PRB, m.DecisionsB))
	}
	return w.Flush()
}

func (cli *CLI) runStatsContrast(args []string) error {
	fs := flag.NewFlagSet("stats contrast", flag.ContinueOnError)
	player := fs.String("player", "", "First player (required)")
	opponent := fs.String("opponent", "", "Second player (required)")
	limit := fs.Int("limit", 20, "Maximum number of positions shown (text only; 0 = all)")
	c := newCorpusStatsFlags(fs)
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats contrast --db <file> --player <name> --opponent <name> [options]")
		fmt.Println()
		fmt.Println("The positions both players decided, whoever they played against, where one")
		fmt.Println("played well and the other did not: the widest gap first. A player's error on")
		fmt.Println("a position is their worst play of it; \"well\" means below the library's Error")
		fmt.Println("threshold. Open the positions with --format json (position_id).")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats contrast --db database.db --player \"Alice\" --opponent \"Bob\"")
		fmt.Println("  blunderdb stats contrast --db database.db --player \"Alice\" --opponent \"Bob\" --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *player == "" || *opponent == "" {
		fs.Usage()
		return fmt.Errorf("missing required flags: --player and --opponent")
	}
	filter, err := c.open(cli, fs, "")
	if err != nil {
		return err
	}
	var res *storage.PlayerContrast
	err = withInterruptibleContext(func() {}, func(ctx context.Context) error {
		var err error
		res, err = cli.db.PlayerContrastCtx(ctx, *player, *opponent, filter)
		return err
	})
	if err != nil {
		return statsErr("player contrast", err)
	}
	if c.json() {
		return printJSON(res)
	}
	fmt.Printf("%s / %s — %d common positions, %d where one played well and the other did not (error threshold %d mp)\n\n",
		res.PlayerA, res.PlayerB, res.CommonPositions, len(res.Positions), res.ThresholdMP)
	if len(res.Positions) == 0 {
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "POSITION\tERR %s (mp)\tERR %s (mp)\tWELL PLAYED BY\n", res.PlayerA, res.PlayerB)
	shown := res.Positions
	if *limit > 0 && len(shown) > *limit {
		shown = shown[:*limit]
	}
	for _, p := range shown {
		who := res.PlayerA
		if p.WellPlayed == "b" {
			who = res.PlayerB
		}
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\n", p.PositionID, p.ErrorMPA, p.ErrorMPB, who)
	}
	return w.Flush()
}

func (cli *CLI) runStatsWindows(args []string) error {
	fs := flag.NewFlagSet("stats windows", flag.ContinueOnError)
	player := fs.String("player", "", "Only this player's decisions")
	window := fs.String("window", "month", "Sliding window: month, quarter, or a number of months")
	c := newCorpusStatsFlags(fs)
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats windows --db <file> [options]")
		fmt.Println()
		fmt.Println("The PR over a sliding calendar window, one line per month: each line covers")
		fmt.Println("that month and the ones before it in the window. A window without a counted")
		fmt.Println("decision shows a dash, not a PR.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats windows --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats windows --db database.db --player \"Alice\" --window quarter --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	months := 0
	switch strings.ToLower(*window) {
	case "month":
		months = 1
	case "quarter":
		months = 3
	default:
		if _, err := fmt.Sscanf(*window, "%d", &months); err != nil || months < 1 {
			return fmt.Errorf("invalid --window %q: want month, quarter or a number of months", *window)
		}
	}
	filter, err := c.open(cli, fs, *player)
	if err != nil {
		return err
	}
	var res []storage.WindowStats
	err = withInterruptibleContext(func() {}, func(ctx context.Context) error {
		var err error
		res, err = cli.db.PRByWindowCtx(ctx, filter, months)
		return err
	})
	if err != nil {
		return statsErr("PR by window", err)
	}
	if c.json() {
		return printJSON(res)
	}
	if len(res) == 0 {
		fmt.Println("No dated match with a counted decision in this filter.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FROM\tTO\tMATCHES\tDECISIONS\tPR")
	for _, p := range res {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n", p.From, p.To, p.NumMatches, p.NumDecisions, prCell(p.PR, p.NumDecisions))
	}
	return w.Flush()
}

func (cli *CLI) runStatsRanking(args []string) error {
	fs := flag.NewFlagSet("stats ranking", flag.ContinueOnError)
	minDecisions := fs.Int("min-decisions", 500, "Rank only the players with at least this many counted decisions")
	limit := fs.Int("limit", 0, "Show only the first N ranks (0: all)")
	c := newCorpusStatsFlags(fs)
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats ranking --db <file> [options]")
		fmt.Println()
		fmt.Println("The players ranked by PR, the lowest first, among those with at least")
		fmt.Println("--min-decisions counted decisions: a PR over a few decisions is noise.")
		fmt.Println("Equal PRs share a rank.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats ranking --db database.db --min-decisions 1000 --limit 20")
		fmt.Println("  blunderdb stats ranking --db database.db --from 2024-01-01 --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *minDecisions < 0 || *limit < 0 {
		return fmt.Errorf("--min-decisions and --limit must not be negative")
	}
	filter, err := c.open(cli, fs, "")
	if err != nil {
		return err
	}
	var res []storage.RankedPlayer
	err = withInterruptibleContext(func() {}, func(ctx context.Context) error {
		var err error
		res, err = cli.db.PlayerRankingCtx(ctx, filter, *minDecisions)
		return err
	})
	if err != nil {
		return statsErr("ranking", err)
	}
	if *limit > 0 && len(res) > *limit {
		res = res[:*limit]
	}
	if c.json() {
		return printJSON(res)
	}
	if len(res) == 0 {
		fmt.Printf("No player with at least %d counted decisions in this filter.\n", *minDecisions)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RANK\tPLAYER\tPR\tDECISIONS\tMATCHES")
	for _, r := range res {
		fmt.Fprintf(w, "%d\t%s\t%.2f\t%d\t%d\n", r.Rank, r.Name, r.PR, r.Decisions, r.Matches)
	}
	return w.Flush()
}
