package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runList handles the list command
func (cli *CLI) runList(args []string) error {
	listCmd := flag.NewFlagSet("list", flag.ContinueOnError)

	// Define flags
	dbPath := listCmd.String("db", "", "Path to the database file (required)")
	listType := listCmd.String("type", "", "List type: matches, tournaments, positions, moves, analyses, imports, stats, players, tags (required)")
	limit := listCmd.Int("limit", 10, "Maximum number of items to list")

	// Stats-specific flags (only used when --type stats)
	statsMetric := listCmd.String("metric", "pr", "Metric to display: pr or mwc (stats only)")
	statsPlayer := listCmd.String("player", "", "Filter by player name (stats only)")
	statsTournament := listCmd.String("tournament", "", "Filter by tournament IDs, comma-separated (stats only)")
	statsFrom := listCmd.String("from", "", "Start date filter YYYY-MM-DD (stats only)")
	statsTo := listCmd.String("to", "", "End date filter YYYY-MM-DD (stats only)")
	statsDecisionType := listCmd.String("decision-type", "all", "Decision type: all, checker, or cube (stats only)")
	statsTopBlunders := listCmd.Int("top-blunders", 10, "Number of top blunders to show (stats only)")
	importQueue := listCmd.Bool("queue", false,
		"With --type imports --batch <id>: the study queue that follows the report — what to look at now, in order")
	statsFormat := listCmd.String("format", "text", "Output format: text, json or csv (stats, players and imports only)")

	// Imports-specific flag (only used when --type imports)
	batchID := listCmd.Int64("batch", 0, "Show the full report of one import batch instead of the list (imports only)")

	listCmd.Usage = func() {
		fmt.Println("Usage: blunderdb list [options]")
		fmt.Println()
		fmt.Println("List database contents.")
		fmt.Println()
		fmt.Println("Options:")
		listCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # List all matches")
		fmt.Println("  blunderdb list --db database.db --type matches")
		fmt.Println()
		fmt.Println("  # List all tournaments")
		fmt.Println("  blunderdb list --db database.db --type tournaments")
		fmt.Println()
		fmt.Println("  # List first 20 positions")
		fmt.Println("  blunderdb list --db database.db --type positions --limit 20")
		fmt.Println()
		fmt.Println("  # List the recorded imports, then read one's report")
		fmt.Println("  blunderdb list --db database.db --type imports")
		fmt.Println("  blunderdb list --db database.db --type imports --batch 3")
		fmt.Println()
		fmt.Println("  # What to look at now, in the order to look at it")
		fmt.Println("  blunderdb list --db database.db --type imports --batch 3 --queue")
		fmt.Println()
		fmt.Println("  # Show database statistics")
		fmt.Println("  blunderdb list --db database.db --type stats")
		fmt.Println()
		fmt.Println("  # Show stats as JSON")
		fmt.Println("  blunderdb list --db database.db --type stats --format json")
		fmt.Println()
		fmt.Println("  # Show stats in MWC with player filter")
		fmt.Println("  blunderdb list --db database.db --type stats --metric mwc --player \"Alice\"")
		fmt.Println()
		fmt.Println("  # One statistics row per player, over a competition's dates")
		fmt.Println("  blunderdb list --db database.db --type players --from 2026-03-01 --to 2026-03-08")
		fmt.Println()
		fmt.Println("  # The same table as CSV, for a spreadsheet or a script")
		fmt.Println("  blunderdb list --db database.db --type players --format csv")
		fmt.Println()
		fmt.Println("  # The tag vocabulary of this database, most used first")
		fmt.Println("  blunderdb list --db database.db --type tags")
		fmt.Println()
		fmt.Println("  # Tabular exports, for a notebook or a spreadsheet")
		fmt.Println("  blunderdb list --db database.db --type positions --format csv > positions.csv")
		fmt.Println("  blunderdb list --db database.db --type moves     --format csv > moves.csv")
		fmt.Println("  blunderdb list --db database.db --type analyses  --format csv > analyses.csv")
	}

	if err := listCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		listCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *listType == "" {
		listCmd.Usage()
		return fmt.Errorf("missing required flag: --type")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform listing based on type
	switch strings.ToLower(*listType) {
	case "matches":
		return cli.listMatches(*limit)
	case "tournaments":
		return cli.listTournaments(*limit)
	case "positions":
		if strings.ToLower(*statsFormat) == "csv" {
			return cli.exportPositionsCSV(exportLimit(listCmd, *limit))
		}
		return cli.listPositions(*limit)
	case "moves":
		if strings.ToLower(*statsFormat) != "csv" {
			return fmt.Errorf("--type moves is a tabular export: add --format csv")
		}
		return cli.exportMovesCSV(exportLimit(listCmd, *limit))
	case "analyses":
		if strings.ToLower(*statsFormat) != "csv" {
			return fmt.Errorf("--type analyses is a tabular export: add --format csv")
		}
		return cli.exportAnalysesCSV(exportLimit(listCmd, *limit))
	case "imports":
		return cli.listImports(*limit, *batchID, strings.ToLower(*statsFormat), *importQueue)
	case "stats":
		// Build StatsFilter from flags
		filter := StatsFilter{
			PlayerName:   *statsPlayer,
			DateFrom:     *statsFrom,
			DateTo:       *statsTo,
			DecisionType: -1, // default: all
		}
		switch strings.ToLower(*statsDecisionType) {
		case "checker":
			filter.DecisionType = 0
		case "cube":
			filter.DecisionType = 1
		}
		if *statsTournament != "" {
			ids, err := parseIDList(*statsTournament)
			if err != nil {
				return fmt.Errorf("invalid --tournament: %w", err)
			}
			filter.TournamentIDs = ids
		}
		return cli.showStats(filter, *statsMetric, *statsFormat, *statsTopBlunders)
	case "tags":
		return cli.listTags(strings.ToLower(*statsFormat))
	case "players":
		// Only the match-level filters matter here; the players table covers
		// every player and splits checker from cube into its own columns, so
		// --player and --decision-type are deliberately not applied.
		filter := StatsFilter{
			DateFrom:     *statsFrom,
			DateTo:       *statsTo,
			DecisionType: -1,
		}
		if *statsTournament != "" {
			ids, err := parseIDList(*statsTournament)
			if err != nil {
				return fmt.Errorf("invalid --tournament: %w", err)
			}
			filter.TournamentIDs = ids
		}
		return cli.showPlayerTable(filter, *statsFormat)
	default:
		return fmt.Errorf("unknown list type: %s (must be 'matches', 'tournaments', 'positions', 'moves', 'analyses', 'imports', 'stats', 'players', or 'tags')", *listType)
	}
}

// listMatches lists all matches in the database
func (cli *CLI) listMatches(limit int) error {
	matches, err := cli.db.GetAllMatches()
	if err != nil {
		return fmt.Errorf("failed to get matches: %w", err)
	}

	if len(matches) == 0 {
		fmt.Println("No matches found in database")
		return nil
	}

	fmt.Printf("Found %d match(es):\n\n", len(matches))

	displayCount := len(matches)
	if limit > 0 && limit < len(matches) {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		match := matches[i]
		fmt.Printf("ID: %d\n", match.ID)
		fmt.Printf("  Players: %s vs %s\n", match.Player1Name, match.Player2Name)
		if match.Event != "" {
			fmt.Printf("  Event: %s\n", match.Event)
		}
		if match.Location != "" {
			fmt.Printf("  Location: %s\n", match.Location)
		}
		fmt.Printf("  Match Length: %d\n", match.MatchLength)
		fmt.Printf("  Games: %d\n", match.GameCount)
		fmt.Printf("  Imported: %s\n", match.ImportDate.Format("2006-01-02 15:04:05"))
		if match.FilePath != "" {
			fmt.Printf("  File: %s\n", match.FilePath)
		}
		fmt.Println()
	}

	if limit > 0 && len(matches) > limit {
		fmt.Printf("(Showing %d of %d matches, use --limit to see more)\n", displayCount, len(matches))
	}

	return nil
}

// listTournaments lists all tournaments in the database
func (cli *CLI) listTournaments(limit int) error {
	tournaments, err := cli.db.GetAllTournaments()
	if err != nil {
		return fmt.Errorf("failed to get tournaments: %w", err)
	}

	if len(tournaments) == 0 {
		fmt.Println("No tournaments found in database")
		return nil
	}

	fmt.Printf("Found %d tournament(s):\n\n", len(tournaments))

	displayCount := len(tournaments)
	if limit > 0 && limit < len(tournaments) {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		tour := tournaments[i]
		fmt.Printf("ID: %d\n", tour.ID)
		fmt.Printf("  Name: %s\n", tour.Name)
		if tour.Date != "" {
			fmt.Printf("  Date: %s\n", tour.Date)
		}
		if tour.Location != "" {
			fmt.Printf("  Location: %s\n", tour.Location)
		}
		fmt.Printf("  Matches: %d\n", tour.MatchCount)
		fmt.Println()
	}

	if limit > 0 && len(tournaments) > limit {
		fmt.Printf("(Showing %d of %d tournaments, use --limit to see more)\n", displayCount, len(tournaments))
	}

	return nil
}

// listPositions lists positions in the database
func (cli *CLI) listPositions(limit int) error {
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	if len(positions) == 0 {
		fmt.Println("No positions found in database")
		return nil
	}

	fmt.Printf("Found %d position(s):\n\n", len(positions))

	displayCount := len(positions)
	if limit > 0 && limit < len(positions) {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		pos := positions[i]

		fmt.Printf("ID: %d\n", pos.ID)
		fmt.Printf("  Score: %d-%d\n", pos.Score[0], pos.Score[1])
		fmt.Printf("  Player on roll: %d\n", pos.PlayerOnRoll)
		if pos.DecisionType == CheckerAction {
			fmt.Printf("  Decision: Checker play\n")
		} else {
			fmt.Printf("  Decision: Cube action\n")
		}
		fmt.Println()
	}

	if limit > 0 && len(positions) > limit {
		fmt.Printf("(Showing %d of %d positions, use --limit to see more)\n", displayCount, len(positions))
	}

	return nil
}

// showStats displays database statistics using ComputeStats.
//
// metric is "pr" or "mwc", format is "text" or "json", topN is the number of
// top blunders to display (only relevant for text format, JSON always includes
// the full TopBlunders slice).
func (cli *CLI) showStats(filter StatsFilter, metric, format string, topN int) error {
	textOutput := strings.ToLower(format) != "json"
	var result *StatsResult
	err := withInterruptibleContext(func() {
		if textOutput {
			fmt.Println("\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		result, err = cli.db.ComputeStatsCtx(ctx, filter)
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("stats cancelled")
		}
		return fmt.Errorf("failed to compute stats: %w", err)
	}

	if strings.ToLower(format) == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal stats: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// ── Text format ──────────────────────────────────────────────────────────
	useMWC := strings.ToLower(metric) == "mwc"
	metricLabel := "PR"
	if useMWC {
		metricLabel = "MWC"
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// 1. Header
	fmt.Fprintln(w, "=== blunderDB Statistics ===")
	if filter.PlayerName != "" {
		fmt.Fprintf(w, "Player:\t%s\n", filter.PlayerName)
	}
	if filter.DateFrom != "" || filter.DateTo != "" {
		from, to := filter.DateFrom, filter.DateTo
		if from == "" {
			from = "—"
		}
		if to == "" {
			to = "—"
		}
		fmt.Fprintf(w, "Date range:\t%s → %s\n", from, to)
	}
	switch filter.DecisionType {
	case 0:
		fmt.Fprintln(w, "Decision type:\tchecker only")
	case 1:
		fmt.Fprintln(w, "Decision type:\tcube only")
	}
	fmt.Fprintf(w, "Metric:\t%s\n", metricLabel)
	w.Flush()
	fmt.Println()

	// 2. Totals
	fmt.Println("── Totals ──")
	fmt.Fprintf(w, "  Positions:\t%d\n", result.Totals.NumPositions)
	fmt.Fprintf(w, "  Matches:\t%d\n", result.Totals.NumMatches)
	fmt.Fprintf(w, "  Tournaments:\t%d\n", result.Totals.NumTournaments)
	fmt.Fprintf(w, "  Decisions:\t%d\n", result.Totals.NumDecisions)
	w.Flush()
	fmt.Println()

	// 3. PR / MWC global
	fmt.Printf("── %s ──\n", metricLabel)
	if useMWC {
		mwcStr := func(v float64) string {
			if !result.MWCAvailable {
				return "—"
			}
			return fmt.Sprintf("%.4f", v)
		}
		fmt.Fprintf(w, "  Global:\t%s\n", mwcStr(result.MWCGlobal))
		fmt.Fprintf(w, "  Checker:\t%s\n", mwcStr(result.MWCChecker))
		fmt.Fprintf(w, "  Cube:\t%s\n", mwcStr(result.MWCCube))
	} else {
		fmt.Fprintf(w, "  Global:\t%.3f\n", result.PRGlobal)
		fmt.Fprintf(w, "  Checker:\t%.3f\n", result.PRChecker)
		fmt.Fprintf(w, "  Cube:\t%.3f\n", result.PRCube)
		fmt.Fprintf(w, "  Snowie ER:\t%.3f\n", result.SnowieGlobal)
	}
	w.Flush()
	fmt.Println()

	// 4. Rolling
	rollingNs := []int{5, 10, 50, 100, 250, 500, 1000}
	fmt.Printf("── Rolling %s ──\n", metricLabel)
	fmt.Fprintln(w, "  N\tDecisions used\tValue")
	fmt.Fprintln(w, "  —\t——————————————\t—————")
	for _, n := range rollingNs {
		var val string
		if useMWC {
			if v, ok := result.MWCRolling[n]; ok {
				if result.MWCAvailable {
					val = fmt.Sprintf("%.4f", v)
				} else {
					val = "—"
				}
			} else {
				val = "n/a"
			}
		} else {
			if v, ok := result.PRRolling[n]; ok {
				val = fmt.Sprintf("%.3f", v)
			} else {
				val = "n/a"
			}
		}
		actualN := n
		if actualN > result.Totals.NumDecisions {
			actualN = result.Totals.NumDecisions
		}
		fmt.Fprintf(w, "  %d\t%d\t%s\n", n, actualN, val)
	}
	w.Flush()
	fmt.Println()

	// 5. Top blunders
	fmt.Printf("── Top %d Blunders ──\n", topN)
	fmt.Fprintln(w, "  Pos ID\tType\tError (EMG)\tMWC Loss\tDate\tPlayers")
	fmt.Fprintln(w, "  ——————\t————\t———————————\t————————\t————\t———————")
	limit := topN
	if len(result.TopBlunders) < limit {
		limit = len(result.TopBlunders)
	}
	for _, b := range result.TopBlunders[:limit] {
		dt := "checker"
		if b.DecisionType == 1 {
			dt = "cube"
		}
		errEMG := fmt.Sprintf("%.3f", float64(b.ErrorMP)/1000)
		mwcStr := "—"
		if result.MWCAvailable && b.MWCLoss != 0 {
			mwcStr = fmt.Sprintf("%.4f", b.MWCLoss)
		}
		date := b.MatchDate
		if date == "" {
			date = "—"
		}
		fmt.Fprintf(w, "  %d\t%s\t%s\t%s\t%s\t%s\n",
			b.PositionID, dt, errEMG, mwcStr, date, b.PlayerNames)
	}
	w.Flush()
	fmt.Println()

	// 6. Cube action breakdown
	if len(result.CubeActionBreakdown) > 0 {
		fmt.Println("── Cube Action Breakdown ──")
		fmt.Fprintln(w, "  Action\tDecisions\tBlunders\tBlunder %\tPR\tMWC")
		fmt.Fprintln(w, "  ——————\t—————————\t————————\t—————————\t——\t———")
		for _, ca := range result.CubeActionBreakdown {
			blunderPct := 0.0
			if ca.NumDecisions > 0 {
				blunderPct = 100 * float64(ca.BlunderCount) / float64(ca.NumDecisions)
			}
			mwcStr := "—"
			if result.MWCAvailable {
				mwcStr = fmt.Sprintf("%.4f", ca.MWC)
			}
			fmt.Fprintf(w, "  %s\t%d\t%d\t%.1f%%\t%.3f\t%s\n",
				ca.Action, ca.NumDecisions, ca.BlunderCount, blunderPct, ca.PR, mwcStr)
		}
		w.Flush()
		fmt.Println()
	}

	// 6b. The three breakdowns of #266: the same decisions, sliced.
	if len(result.PerPhase) > 0 {
		fmt.Println("── By Game Phase ──")
		fmt.Fprintln(w, "  Phase\tDecisions\tBlunders\tPR")
		fmt.Fprintln(w, "  —————\t—————————\t————————\t——")
		for _, ph := range result.PerPhase {
			fmt.Fprintf(w, "  %s\t%d\t%d\t%.3f\n", ph.Phase, ph.NumDecisions, ph.BlunderCount, ph.PR)
		}
		w.Flush()
		fmt.Println()
	}
	if len(result.PerTag) > 0 {
		fmt.Println("── By Tag ──")
		fmt.Fprintln(w, "  Tag\tDecisions\tBlunders\tPR")
		fmt.Fprintln(w, "  ———\t—————————\t————————\t——")
		for _, tag := range result.PerTag {
			fmt.Fprintf(w, "  %s\t%d\t%d\t%.3f\n", tag.Tag, tag.NumDecisions, tag.BlunderCount, tag.PR)
		}
		w.Flush()
		// A tag labels; it does not partition. Saying so is the difference
		// between a reader who trusts the column and one who adds it up and
		// concludes the tool is broken.
		fmt.Println("  (a position may carry several tags; these rows do not sum to the total)")
		fmt.Println()
	}
	if len(result.PerScore) > 0 {
		fmt.Println("── By Score (away × away, from the player on roll's side) ──")
		fmt.Fprintln(w, "  Score\tDecisions\tBlunders\tPR")
		fmt.Fprintln(w, "  —————\t—————————\t————————\t——")
		for _, c := range result.PerScore {
			label := fmt.Sprintf("%d-away/%d-away", c.MoverAway, c.OpponentAway)
			if c.MoverAway == 0 && c.OpponentAway == 0 {
				label = "money"
			}
			// A cell too small to read is still printed WITH its count: hiding
			// it would make the omission unauditable.
			thin := ""
			if c.NumDecisions < storage.MinCellDecisions {
				thin = "  (thin)"
			}
			fmt.Fprintf(w, "  %s\t%d\t%d\t%.3f%s\n", label, c.NumDecisions, c.BlunderCount, c.PR, thin)
		}
		w.Flush()
		fmt.Println()
	}

	// 7. Error histogram
	if len(result.ErrorHistogram) > 0 {
		fmt.Println("── Error Histogram ──")
		fmt.Fprintln(w, "  Range (EMG)\tCount")
		fmt.Fprintln(w, "  ——————————\t—————")
		for _, b := range result.ErrorHistogram {
			var rangeStr string
			if b.MaxMP == -1 {
				rangeStr = fmt.Sprintf("≥%.3f", float64(b.MinMP)/1000)
			} else {
				rangeStr = fmt.Sprintf("%.3f–%.3f", float64(b.MinMP)/1000, float64(b.MaxMP)/1000)
			}
			fmt.Fprintf(w, "  %s\t%d\n", rangeStr, b.Count)
		}
		w.Flush()
	}

	return nil
}

// showPlayerTable prints one statistics row per player. CSV is the format that
// matters here: the request behind this table came from someone collecting a
// competition's match logs, and a spreadsheet is where such a ranking ends up.
func (cli *CLI) showPlayerTable(filter StatsFilter, format string) error {
	rows, err := cli.db.GetPlayerTable(filter)
	if err != nil {
		return fmt.Errorf("failed to compute player table: %w", err)
	}

	switch strings.ToLower(format) {
	case "json":
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal player table: %w", err)
		}
		fmt.Println(string(data))
		return nil

	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		if err := w.Write([]string{
			"player", "matches", "wins", "losses", "decisions",
			"pr", "pr_checker", "pr_cube", "snowie_er", "errors", "blunders",
			"luck_rate_mp", "luck_rolls",
		}); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, r := range rows {
			// An unmeasured figure is written as an empty field rather than a
			// zero: a spreadsheet averaging this column must not count a player
			// nobody measured as perfectly average.
			luck := ""
			if r.LuckKnown {
				luck = strconv.FormatFloat(r.LuckRateMP, 'f', 1, 64)
			}
			rate := func(v float64, known bool) string {
				if !known {
					return ""
				}
				return strconv.FormatFloat(v, 'f', 2, 64)
			}
			if err := w.Write([]string{
				r.Name,
				strconv.Itoa(r.Matches), strconv.Itoa(r.Wins), strconv.Itoa(r.Losses),
				strconv.Itoa(r.Decisions),
				rate(r.PR, r.Decisions > 0),
				rate(r.PRChecker, r.CheckerDecisions > 0),
				rate(r.PRCube, r.CubeDecisions > 0),
				rate(r.SnowieER, r.Decisions > 0),
				strconv.Itoa(r.Errors), strconv.Itoa(r.Blunders),
				luck, strconv.Itoa(r.LuckRolls),
			}); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		return w.Error()
	}

	// ── Text format ──────────────────────────────────────────────────────────
	if len(rows) == 0 {
		fmt.Println("No player found for this filter.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "=== Players ===")
	if filter.DateFrom != "" || filter.DateTo != "" {
		fmt.Fprintf(w, "Period:\t%s → %s\n", orDash(filter.DateFrom), orDash(filter.DateTo))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Player\tMatches\tW-L\tDec.\tPR\tChecker\tCube\tSnowie\tBlunders\tLuck")

	fmtRate := func(v float64, known bool) string {
		if !known {
			return "—"
		}
		return fmt.Sprintf("%.2f", v)
	}
	for _, r := range rows {
		luck := "—"
		if r.LuckKnown {
			luck = fmt.Sprintf("%+.1f", r.LuckRateMP)
		}
		fmt.Fprintf(w, "%s\t%d\t%d-%d\t%d\t%s\t%s\t%s\t%s\t%d\t%s\n",
			r.Name, r.Matches, r.Wins, r.Losses, r.Decisions,
			fmtRate(r.PR, r.Decisions > 0),
			fmtRate(r.PRChecker, r.CheckerDecisions > 0),
			fmtRate(r.PRCube, r.CubeDecisions > 0),
			fmtRate(r.SnowieER, r.Decisions > 0),
			r.Blunders, luck)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "\"—\" marks a figure that was never measured, which is not the same as zero.")
	return w.Flush()
}

// orDash renders an empty filter bound as an open one.
func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// exportLimit is --limit as a tabular export must read it: unbounded unless
// the user asked for a bound.
//
// `--limit` defaults to 10 because a `list` printed to a terminal should not
// scroll a database past the reader. An export is the opposite case: it is
// redirected to a file and read by a program, and silently truncating it at
// ten rows would be a trap nobody notices until the figures are wrong. So the
// default is ignored here, and only a --limit the user actually typed applies.
func exportLimit(fs *flag.FlagSet, limit int) int {
	explicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "limit" {
			explicit = true
		}
	})
	if !explicit {
		return 0
	}
	return limit
}
