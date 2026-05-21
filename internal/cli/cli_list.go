package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// runList handles the list command
func (cli *CLI) runList(args []string) error {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	// Define flags
	dbPath := listCmd.String("db", "", "Path to the database file (required)")
	listType := listCmd.String("type", "", "List type: matches, positions, stats (required)")
	limit := listCmd.Int("limit", 10, "Maximum number of items to list")

	// Stats-specific flags (only used when --type stats)
	statsMetric := listCmd.String("metric", "pr", "Metric to display: pr or mwc (stats only)")
	statsPlayer := listCmd.String("player", "", "Filter by player name (stats only)")
	statsTournament := listCmd.String("tournament", "", "Filter by tournament IDs, comma-separated (stats only)")
	statsFrom := listCmd.String("from", "", "Start date filter YYYY-MM-DD (stats only)")
	statsTo := listCmd.String("to", "", "End date filter YYYY-MM-DD (stats only)")
	statsDecisionType := listCmd.String("decision-type", "all", "Decision type: all, checker, or cube (stats only)")
	statsTopBlunders := listCmd.Int("top-blunders", 10, "Number of top blunders to show (stats only)")
	statsFormat := listCmd.String("format", "text", "Output format: text or json (stats only)")

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
		fmt.Println("  # List first 20 positions")
		fmt.Println("  blunderdb list --db database.db --type positions --limit 20")
		fmt.Println()
		fmt.Println("  # Show database statistics")
		fmt.Println("  blunderdb list --db database.db --type stats")
		fmt.Println()
		fmt.Println("  # Show stats as JSON")
		fmt.Println("  blunderdb list --db database.db --type stats --format json")
		fmt.Println()
		fmt.Println("  # Show stats in MWC with player filter")
		fmt.Println("  blunderdb list --db database.db --type stats --metric mwc --player \"Alice\"")
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
	case "positions":
		return cli.listPositions(*limit)
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
				return fmt.Errorf("invalid --tournament: %v", err)
			}
			filter.TournamentIDs = ids
		}
		return cli.showStats(filter, *statsMetric, *statsFormat, *statsTopBlunders)
	default:
		return fmt.Errorf("unknown list type: %s (must be 'matches', 'positions', or 'stats')", *listType)
	}
}

// listMatches lists all matches in the database
func (cli *CLI) listMatches(limit int) error {
	matches, err := cli.db.GetAllMatches()
	if err != nil {
		return fmt.Errorf("failed to get matches: %v", err)
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

// listPositions lists positions in the database
func (cli *CLI) listPositions(limit int) error {
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %v", err)
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
	result, err := cli.db.ComputeStats(filter)
	if err != nil {
		return fmt.Errorf("failed to compute stats: %v", err)
	}

	if strings.ToLower(format) == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal stats: %v", err)
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
