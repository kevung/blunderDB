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
		"recurring":   cli.runStatsRecurring,
		"training":    cli.runStatsTraining,
		"progression": cli.runStatsProgression,
		"breakdown":   cli.runStatsBreakdown,
		"report":      cli.runStatsReport,
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
	fmt.Println("  training   Quiz PR and Anki retention against real PR, by calendar window")
	fmt.Println("  progression  PR per match, per tournament and rolling (Progression tab)")
	fmt.Println("  breakdown  PR by phase, plan of play, tag, score and cube action (Breakdowns tab)")
	fmt.Println("  report     Self-contained HTML report of the filter (--html)")
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
	quiz := fs.Bool("quiz", false, "Draw a quiz: position ids picked at random from the worst groups, for the Decision exercise or quiz_grade")
	quizSize := fs.Int("quiz-size", storage.StudyQuizSize, "Number of positions --quiz draws")
	deckName := fs.String("deck", "", "Create an Anki deck of this name from the positions of the worst groups")
	group := fs.Int("group", 0, "With --quiz or --deck: the rank of one group (1 = costliest) instead of the three costliest")
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
		fmt.Println("--quiz and --deck turn the ranking into study: --quiz draws positions at")
		fmt.Println("random from the three costliest groups (or the one --group names), --deck")
		fmt.Println("makes an Anki deck of all their positions.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats recurring --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats recurring --db database.db --quiz --format json")
		fmt.Println("  blunderdb stats recurring --db database.db --group 1 --deck \"My worst group\"")
		fmt.Println("  blunderdb stats recurring --db database.db --decision-type checker --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	filter, err := buildStatsFilter(*player, *tournament, *from, *to, *decisionType)
	if err != nil {
		return err
	}
	if *group < 0 {
		return fmt.Errorf("invalid --group %d: want 0 or a rank from 1", *group)
	}
	if *quizSize < 1 {
		return fmt.Errorf("invalid --quiz-size %d: want at least 1", *quizSize)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	textOutput := strings.ToLower(*format) != "json"
	var res *storage.RecurringErrors
	err = withInterruptibleContext(func() {
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
	var quizIDs []int64
	if *quiz {
		if quizIDs, err = cli.db.StudyPositionIDs(filter, *group, *quizSize); err != nil {
			return fmt.Errorf("draw the quiz: %w", err)
		}
	}
	var deckID int64
	if *deckName != "" {
		ids := res.StudyPositionIDs(*group)
		if len(ids) == 0 {
			return fmt.Errorf("no group to make a deck from")
		}
		var err error
		if deckID, err = cli.db.CreateStudyDeck(*deckName, ids); err != nil {
			return err
		}
	}
	if !textOutput {
		out := struct {
			*storage.RecurringErrors
			Quiz   []int64 `json:"Quiz,omitempty"`
			DeckID int64   `json:"DeckID,omitempty"`
		}{res, quizIDs, deckID}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal recurring errors: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	printRecurringErrors(res, *limit)
	if *quiz {
		fmt.Println()
		if len(quizIDs) == 0 {
			fmt.Println("Quiz: no position to draw from.")
		} else {
			fmt.Printf("Quiz — %d positions: %s\n", len(quizIDs), joinIDs(quizIDs))
		}
	}
	if deckID != 0 {
		fmt.Printf("Created deck %d (%s)\n", deckID, *deckName)
	}
	return nil
}

func joinIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprint(id)
	}
	return strings.Join(parts, ",")
}

// buildStatsFilter reads the filter flags the stats sub-commands share.
func buildStatsFilter(player, tournament, from, to, decisionType string) (StatsFilter, error) {
	filter := StatsFilter{PlayerName: player, DateFrom: from, DateTo: to, DecisionType: -1}
	switch strings.ToLower(decisionType) {
	case "all":
	case "checker":
		filter.DecisionType = 0
	case "cube":
		filter.DecisionType = 1
	default:
		return filter, fmt.Errorf("invalid --decision-type %q: want all, checker or cube", decisionType)
	}
	if tournament != "" {
		ids, err := parseIDList(tournament)
		if err != nil {
			return filter, fmt.Errorf("invalid --tournament: %w", err)
		}
		filter.TournamentIDs = ids
	}
	return filter, nil
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

func (cli *CLI) runStatsTraining(args []string) error {
	fs := flag.NewFlagSet("stats training", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	player := fs.String("player", "", "Only this player's matches (the real PR series)")
	tournament := fs.String("tournament", "", "Filter the matches by tournament IDs, comma-separated")
	from := fs.String("from", "", "Start date filter YYYY-MM-DD (matches)")
	to := fs.String("to", "", "End date filter YYYY-MM-DD (matches)")
	decisionType := fs.String("decision-type", "all", "Decision type of the matches: all, checker, or cube")
	window := fs.String("window", storage.TrainingWindowWeek, "Calendar window: week or month")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats training --db <file> [options]")
		fmt.Println()
		fmt.Println("The Decision quiz PR, the real PR of the matches and the Anki retention,")
		fmt.Println("folded by calendar window so the three can be read side by side. The quiz")
		fmt.Println("PR is on the real PR's scale; the retention is the share of review-state")
		fmt.Println("card reviews rated Hard or better. Each series carries its own count: a")
		fmt.Println("window with no decision has a count of 0, not a value. The filter options")
		fmt.Println("restrict the matches only; the quiz and Anki journals carry no player.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats training --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats training --db database.db --window month --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if *window != storage.TrainingWindowWeek && *window != storage.TrainingWindowMonth {
		return fmt.Errorf("invalid --window %q: want week or month", *window)
	}
	filter, err := buildStatsFilter(*player, *tournament, *from, *to, *decisionType)
	if err != nil {
		return err
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	var res *storage.TrainingStats
	err = withInterruptibleContext(func() {}, func(ctx context.Context) error {
		var err error
		res, err = cli.db.ComputeTrainingStatsCtx(ctx, filter, *window)
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("stats cancelled")
		}
		return fmt.Errorf("training stats: %w", err)
	}
	if strings.ToLower(*format) == "json" {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal training stats: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	printTrainingStats(res)
	return nil
}

// printTrainingStats writes one line per window, the oldest first.
func printTrainingStats(res *storage.TrainingStats) {
	if len(res.Periods) == 0 {
		fmt.Println("No quiz session, match or Anki review yet.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s\tQUIZ PR\tDECISIONS\tMATCH PR\tDECISIONS\tANKI RETENTION\tREVIEWS\n", strings.ToUpper(res.Window))
	for _, p := range res.Periods {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\t%s\t%d\n", p.Start,
			optional(p.QuizDecisions, fmt.Sprintf("%.2f", p.QuizPR)), p.QuizDecisions,
			optional(p.MatchDecisions, fmt.Sprintf("%.2f", p.MatchPR)), p.MatchDecisions,
			optional(p.AnkiReviews, fmt.Sprintf("%.1f%%", 100*p.AnkiRetention)), p.AnkiReviews)
	}
	w.Flush()
}

// optional shows a series value only when it has a sample behind it.
func optional(count int, value string) string {
	if count == 0 {
		return "-"
	}
	return value
}
