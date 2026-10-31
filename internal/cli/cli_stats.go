package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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
		"contrast":    cli.runStatsContrast,
		"h2h":         cli.runStatsH2H,
		"ranking":     cli.runStatsRanking,
		"recurring":   cli.runStatsRecurring,
		"plan":        cli.runStatsPlan,
		"effect":      cli.runStatsEffect,
		"biases":      cli.runStatsBiases,
		"tournament":  cli.runStatsTournament,
		"training":    cli.runStatsTraining,
		"windows":     cli.runStatsWindows,
		"summary":     cli.runStatsSummary,
		"matches":     cli.runStatsMatches,
		"tournaments": cli.runStatsTournaments,
		"progression": cli.runStatsProgression,
		"breakdown":   cli.runStatsBreakdown,
		"report":      cli.runStatsReport,
		"engines":     cli.runStatsEngines,
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
	fmt.Fprintln(cli.stdout(), "Usage: blunderdb stats <sub-command> [options]")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Statistics computed apart from `list --type stats`.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Sub-commands:")
	fmt.Fprintln(cli.stdout(), "  contrast   Positions two players both decided, one played well and the other did not")
	fmt.Fprintln(cli.stdout(), "  h2h        Two players against each other: common matches, PR of each, record")
	fmt.Fprintln(cli.stdout(), "  ranking    Players ranked by PR above a floor of counted decisions")
	fmt.Fprintln(cli.stdout(), "  recurring  Errors grouped by plan of play and theme, costliest first")
	fmt.Fprintln(cli.stdout(), "  plan       Study plan: error families ranked by recoverable MWC, with a 95% interval")
	fmt.Fprintln(cli.stdout(), "  effect     Each studied family's loss rate before and after it was first studied")
	fmt.Fprintln(cli.stdout(), "  biases     Which way the decisions err: takes, doubles by score, blots, with 95% intervals")
	fmt.Fprintln(cli.stdout(), "  tournament One player's tournament against their usual level, by round, rank, score and pace")
	fmt.Fprintln(cli.stdout(), "  training   Quiz PR and Anki retention against real PR, by calendar window")
	fmt.Fprintln(cli.stdout(), "  windows    PR over a sliding calendar window (month, quarter)")
	fmt.Fprintln(cli.stdout(), "  summary    The filter's level without any identifier (public profile)")
	fmt.Fprintln(cli.stdout(), "  tournaments  Tournaments holding a counted decision under the filter (Dashboard count)")
	fmt.Fprintln(cli.stdout(), "  progression  PR per match, per tournament and rolling (Progression tab)")
	fmt.Fprintln(cli.stdout(), "  breakdown  PR by phase, plan of play, tag, score and cube action (Breakdowns tab)")
	fmt.Fprintln(cli.stdout(), "  report     Self-contained HTML report of the filter (--html)")
	fmt.Fprintln(cli.stdout(), "  engines    Analysis engines present in the database, as --engine expects them")
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
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb stats recurring --db <file> [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Group the errors of the filter by plan of play and theme, costliest first.")
		fmt.Fprintln(cli.stdout(), "A checker theme is the reason the explanation rules name (gammon, blots,")
		fmt.Fprintln(cli.stdout(), "point, passive); a cube theme is the direction of the cube error. An error")
		fmt.Fprintln(cli.stdout(), "no rule names is listed apart, per plan of play, outside the ranking (JSON:")
		fmt.Fprintln(cli.stdout(), "\"Unthemed\"): the rules only speak from 60 mp, above the Error threshold.")
		fmt.Fprintln(cli.stdout(), "Cost is the share of the filter's PR the group accounts for; an error is a")
		fmt.Fprintln(cli.stdout(), "counted decision costing at least the library's Error threshold.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "--quiz and --deck turn the ranking into study: --quiz draws positions at")
		fmt.Fprintln(cli.stdout(), "random from the three costliest groups (or the one --group names), --deck")
		fmt.Fprintln(cli.stdout(), "makes an Anki deck of all their positions.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats recurring --db database.db --player \"Alice\"")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats recurring --db database.db --quiz --format json")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats recurring --db database.db --group 1 --deck \"My worst group\"")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats recurring --db database.db --decision-type checker --format json")
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
			fmt.Fprintln(cli.stdout(), "\nCancelling...")
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
		fmt.Fprintln(cli.stdout(), string(data))
		return nil
	}
	cli.printRecurringErrors(res, *limit)
	if *quiz {
		fmt.Fprintln(cli.stdout())
		if len(quizIDs) == 0 {
			fmt.Fprintln(cli.stdout(), "Quiz: no position to draw from.")
		} else {
			fmt.Fprintf(cli.stdout(), "Quiz — %d positions: %s\n", len(quizIDs), joinIDs(quizIDs))
		}
	}
	if deckID != 0 {
		fmt.Fprintf(cli.stdout(), "Created deck %d (%s)\n", deckID, *deckName)
	}
	return nil
}

func (cli *CLI) runStatsPlan(args []string) error {
	fs := flag.NewFlagSet("stats plan", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	player := fs.String("player", "", "Only this player's decisions")
	tournament := fs.String("tournament", "", "Filter by tournament IDs, comma-separated")
	from := fs.String("from", "", "Start date filter YYYY-MM-DD")
	to := fs.String("to", "", "End date filter YYYY-MM-DD")
	decisionType := fs.String("decision-type", "all", "Decision type: all, checker, or cube")
	limit := fs.Int("limit", 10, "Maximum number of families shown (text only; 0 = all)")
	format := fs.String("format", "text", "Output format: text or json")
	quiz := fs.Bool("quiz", false, "Draw a quiz: position ids picked at random from the plan's first families")
	quizSize := fs.Int("quiz-size", storage.StudyQuizSize, "Number of positions --quiz draws")
	deckName := fs.String("deck", "", "Create an Anki deck of this name from the positions of the plan's first families")
	queue := fs.Bool("queue", false, "List the study queue of the plan's first families, the largest excess first")
	family := fs.Int("family", 0, "With --quiz, --deck or --queue: the rank of one family (1 = first) instead of the first three")
	fs.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb stats plan --db <file> [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Answer \"what should I work on now?\": the recurring-error families (plan of play")
		fmt.Fprintln(cli.stdout(), "x theme) ranked by the winning chances studying them would recover (ADR-0077).")
		fmt.Fprintln(cli.stdout(), "A family's recoverable MWC is the sum, over its errors, of the loss minus the")
		fmt.Fprintln(cli.stdout(), "difficulty: what a reference player would have lost in the same positions.")
		fmt.Fprintln(cli.stdout(), "It comes with a 95% interval; a family enters the plan with at least")
		fmt.Fprintf(cli.stdout(), "%d priced errors and an interval above zero, ranked by the interval's lower\n", storage.StudyPlanMinErrors)
		fmt.Fprintln(cli.stdout(), "bound. The others are listed apart, to confirm. Money-play errors carry no")
		fmt.Fprintln(cli.stdout(), "MWC and are only counted (unpriced).")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats plan --db database.db --player \"Alice\"")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats plan --db database.db --player \"Alice\" --quiz --format json")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats plan --db database.db --family 1 --deck \"Plan: first family\"")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats plan --db database.db --family 2 --queue")
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
	if *family < 0 {
		return fmt.Errorf("invalid --family %d: want 0 or a rank from 1", *family)
	}
	if *quizSize < 1 {
		return fmt.Errorf("invalid --quiz-size %d: want at least 1", *quizSize)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	textOutput := strings.ToLower(*format) != "json"
	var plan *storage.StudyPlan
	err = withInterruptibleContext(func() {
		if textOutput {
			fmt.Fprintln(cli.stdout(), "\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		plan, err = cli.db.ComputeStudyPlanCtx(ctx, filter)
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("stats cancelled")
		}
		return fmt.Errorf("study plan: %w", err)
	}
	var quizIDs []int64
	if *quiz {
		quizIDs = storage.DrawStudyQuiz(plan.StudyPositionIDs(*family), *quizSize, nil)
	}
	var deckID int64
	if *deckName != "" {
		ids := plan.StudyPositionIDs(*family)
		if len(ids) == 0 {
			return fmt.Errorf("no family in the plan to make a deck from")
		}
		if deckID, err = cli.db.CreateStudyDeck(*deckName, ids); err != nil {
			return err
		}
	}
	var entries []domain.StudyQueueEntry
	if *queue {
		entries = plan.QueueEntries(*family)
	}
	if !textOutput {
		out := struct {
			*storage.StudyPlan
			Quiz   []int64                  `json:"Quiz,omitempty"`
			DeckID int64                    `json:"DeckID,omitempty"`
			Queue  []domain.StudyQueueEntry `json:"Queue,omitempty"`
		}{plan, quizIDs, deckID, entries}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal study plan: %w", err)
		}
		fmt.Fprintln(cli.stdout(), string(data))
		return nil
	}
	cli.printStudyPlan(plan, *limit)
	if *quiz {
		fmt.Fprintln(cli.stdout())
		if len(quizIDs) == 0 {
			fmt.Fprintln(cli.stdout(), "Quiz: no position to draw from.")
		} else {
			fmt.Fprintf(cli.stdout(), "Quiz — %d positions: %s\n", len(quizIDs), joinIDs(quizIDs))
		}
	}
	if *queue {
		fmt.Fprintln(cli.stdout())
		w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "POSITION\tMATCH\tERROR (mp)\tKIND")
		for _, e := range entries {
			kind := "checker"
			if e.IsCube {
				kind = "cube"
			}
			fmt.Fprintf(w, "%d\t%s\t%d\t%s\n", e.PositionID, e.Label, e.ErrorMP, kind)
		}
		w.Flush()
	}
	if deckID != 0 {
		fmt.Fprintf(cli.stdout(), "Created deck %d (%s)\n", deckID, *deckName)
	}
	return nil
}

// printStudyPlan writes the plan, then the families still to confirm. MWC
// figures are shown in percentage points of match winning chances.
func (cli *CLI) printStudyPlan(plan *storage.StudyPlan, limit int) {
	fmt.Fprintf(cli.stdout(), "Study plan — %d counted decisions, error threshold %d mp\n", plan.NumDecisions, plan.ThresholdMP)
	fmt.Fprintf(cli.stdout(), "Recoverable MWC = Σ(loss − difficulty) over a family's errors; 95%% interval; at least %d errors.\n\n", plan.MinErrors)
	row := func(w *tabwriter.Writer, rank string, f storage.StudyPlanFamily) {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%.2f%%\t%.2f%%\t%.2f%%\t[%.2f%%, %.2f%%]\t%s\n", rank,
			f.GameType, f.Kind, f.Theme, f.Errors, f.Avoidable, 100*f.MeanLoss, 100*f.MeanDifficulty,
			100*f.Recoverable, 100*f.Low, 100*f.High, idPreview(planIDs(f), 5))
	}
	header := "RANK\tPLAN\tKIND\tTHEME\tERRORS\tAVOIDABLE\tMEAN LOSS\tMEAN DIFFICULTY\tRECOVERABLE\t95% INTERVAL\tPOSITIONS"
	if len(plan.Families) == 0 {
		fmt.Fprintln(cli.stdout(), "No family has enough evidence yet.")
	} else {
		w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, header)
		shown := plan.Families
		if limit > 0 && limit < len(shown) {
			shown = shown[:limit]
		}
		for i, f := range shown {
			row(w, fmt.Sprint(i+1), f)
		}
		w.Flush()
		if len(shown) < len(plan.Families) {
			fmt.Fprintf(cli.stdout(), "\n(Showing %d of %d families, use --limit 0 to see all)\n", len(shown), len(plan.Families))
		}
	}
	if len(plan.Tentative) > 0 {
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "To confirm (too few errors, or an interval reaching zero):")
		w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, header)
		for _, f := range plan.Tentative {
			row(w, "-", f)
		}
		w.Flush()
	}
	if plan.Unthemed > 0 || plan.Unpriced > 0 {
		fmt.Fprintf(cli.stdout(), "\nOutside the plan: %d errors without a theme, %d unpriced (money play or no option costs).\n", plan.Unthemed, plan.Unpriced)
	}
}

func planIDs(f storage.StudyPlanFamily) []int64 {
	ids := make([]int64, len(f.Positions))
	for i, p := range f.Positions {
		ids[i] = p.PositionID
	}
	return ids
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
func (cli *CLI) printRecurringErrors(res *storage.RecurringErrors, limit int) {
	fmt.Fprintf(cli.stdout(), "Recurring errors — %d counted decisions, error threshold %d mp\n\n", res.NumDecisions, res.ThresholdMP)
	if len(res.Groups) == 0 && len(res.Unthemed) == 0 {
		fmt.Fprintln(cli.stdout(), "No error in this filter.")
		return
	}
	if len(res.Groups) > 0 {
		cli.printRecurringRanking(res, limit)
	}
	if len(res.Unthemed) > 0 {
		fmt.Fprintln(cli.stdout())
		fmt.Fprintf(cli.stdout(), "No theme identified (the explanation rules speak from %d mp), per plan of play:\n", engine.ExplainMinCostMP)
		w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PLAN\tERRORS\tCOST (PR)\tPOSITIONS")
		for _, g := range res.Unthemed {
			fmt.Fprintf(w, "%s\t%d\t%.2f\t%s\n", g.GameType, g.Count, g.PRCost, idPreview(g.PositionIDs, 5))
		}
		w.Flush()
	}
}

// printRecurringRanking writes the themed groups, the costliest first.
func (cli *CLI) printRecurringRanking(res *storage.RecurringErrors, limit int) {
	w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
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
		fmt.Fprintf(cli.stdout(), "\n(Showing %d of %d groups, use --limit 0 to see all)\n", len(shown), len(res.Groups))
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
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb stats training --db <file> [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "The Decision quiz PR, the real PR of the matches and the Anki retention,")
		fmt.Fprintln(cli.stdout(), "folded by calendar window so the three can be read side by side. The quiz")
		fmt.Fprintln(cli.stdout(), "PR is on the real PR's scale; the retention is the share of review-state")
		fmt.Fprintln(cli.stdout(), "card reviews rated Hard or better. Each series carries its own count: a")
		fmt.Fprintln(cli.stdout(), "window with no decision has a count of 0, not a value. The filter options")
		fmt.Fprintln(cli.stdout(), "restrict the matches only; the quiz and Anki journals carry no player.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats training --db database.db --player \"Alice\"")
		fmt.Fprintln(cli.stdout(), "  blunderdb stats training --db database.db --window month --format json")
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
		fmt.Fprintln(cli.stdout(), string(data))
		return nil
	}
	cli.printTrainingStats(res)
	return nil
}

// printTrainingStats writes one line per window, the oldest first.
func (cli *CLI) printTrainingStats(res *storage.TrainingStats) {
	if len(res.Periods) == 0 {
		fmt.Fprintln(cli.stdout(), "No quiz session, match or Anki review yet.")
		return
	}
	w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s\tQUIZ PR\tDECISIONS\tMATCH PR\tDECISIONS\tANKI RETENTION\tREVIEWS\n", strings.ToUpper(res.Window))
	for _, p := range res.Periods {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\t%s\t%d\n", p.Start,
			optional(p.QuizDecisions, fmt.Sprintf("%.2f", p.QuizPR)), p.QuizDecisions,
			optional(p.MatchDecisions, fmt.Sprintf("%.2f", p.MatchPR)), p.MatchDecisions,
			optional(p.AnkiReviews, fmt.Sprintf("%.1f%%", 100*p.AnkiRetention)), p.AnkiReviews)
	}
	w.Flush()
	if len(res.Themes) == 0 {
		return
	}
	fmt.Fprintln(cli.stdout())
	w = tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "THEME\tQUIZ PR\tDECISIONS")
	for _, th := range res.Themes {
		fmt.Fprintf(w, "%s\t%.2f\t%d\n", th.Theme, th.PR, th.Decisions)
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
