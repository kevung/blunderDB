package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// runSearch handles the search command
func (cli *CLI) runSearch(args []string) error {
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)

	// Define flags
	dbPath := searchCmd.String("db", "", "Path to the database file (required)")
	outputDB := searchCmd.String("export", "", "Export results to a new database file")
	limit := searchCmd.Int("limit", 0, "Maximum number of results (0 = no limit)")
	format := searchCmd.String("format", "table", "Output format: table, json, xgid")

	// Filter flags
	decisionType := searchCmd.String("decision", "", "Filter by decision type: checker, cube")
	pipMin := searchCmd.Int("pip-min", 0, "Minimum pip count difference")
	pipMax := searchCmd.Int("pip-max", 0, "Maximum pip count difference")
	winRateMin := searchCmd.Float64("winrate-min", 0, "Minimum win rate (%)")
	winRateMax := searchCmd.Float64("winrate-max", 0, "Maximum win rate (%)")
	cubeValue := searchCmd.Int("cube", 0, "Filter by cube value")
	score1 := searchCmd.Int("score1", -1, "Filter by player 1 score")
	score2 := searchCmd.Int("score2", -1, "Filter by player 2 score")
	matchLength := searchCmd.Int("match-length", 0, "Filter by match length")
	errorMin := searchCmd.Float64("error-min", 0, "Minimum equity error (blunders)")
	moveErrorMin := searchCmd.Float64("move-error-min", 0, "Minimum played move error (millipoints)")
	moveErrorMax := searchCmd.Float64("move-error-max", 0, "Maximum played move error (millipoints)")
	hasAnalysis := searchCmd.Bool("has-analysis", false, "Only positions with analysis")
	checkerOff1Min := searchCmd.Int("off1-min", 0, "Minimum checkers off for player 1")
	checkerOff2Min := searchCmd.Int("off2-min", 0, "Minimum checkers off for player 2")
	matchIDsFlag := searchCmd.String("match-ids", "", "Filter by match IDs (comma-separated, e.g. '1,3,5' or range '2,7')")
	tournamentIDsFlag := searchCmd.String("tournament-ids", "", "Filter by tournament IDs (comma-separated, e.g. '1,3' or range '1,5')")

	searchCmd.Usage = func() {
		fmt.Println("Usage: blunderdb search [options]")
		fmt.Println()
		fmt.Println("Search for positions in the database using filters.")
		fmt.Println()
		fmt.Println("Options:")
		searchCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # List all positions")
		fmt.Println("  blunderdb search --db database.db")
		fmt.Println()
		fmt.Println("  # Search cube decisions")
		fmt.Println("  blunderdb search --db database.db --decision cube")
		fmt.Println()
		fmt.Println("  # Search positions with errors >= 0.1")
		fmt.Println("  blunderdb search --db database.db --error-min 0.1")
		fmt.Println()
		fmt.Println("  # Search and export to new database")
		fmt.Println("  blunderdb search --db database.db --decision cube --export cubes.db")
		fmt.Println()
		fmt.Println("  # Search bearoff positions")
		fmt.Println("  blunderdb search --db database.db --off1-min 1 --off2-min 1")
		fmt.Println()
		fmt.Println("  # Output as JSON")
		fmt.Println("  blunderdb search --db database.db --format json --limit 10")
		fmt.Println()
		fmt.Println("  # Search in specific matches")
		fmt.Println("  blunderdb search --db database.db --match-ids 2,5")
		fmt.Println()
		fmt.Println("  # Search in a tournament")
		fmt.Println("  blunderdb search --db database.db --tournament-ids 1")
	}

	if err := searchCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		searchCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Build filter parameters for LoadPositionsByFilters
	// Create a base filter position with EMPTY board (no checker position filtering)
	// This is different from InitializePosition() which sets up starting position
	filter := Position{
		Board:        Board{Points: [26]Point{}}, // Empty board - matches any position
		Cube:         Cube{Owner: None, Value: 0},
		Dice:         [2]int{0, 0},
		Score:        [2]int{-1, -1}, // -1 means no score filter
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}

	// Set decision type filter
	decisionTypeFilter := false
	if *decisionType != "" {
		decisionTypeFilter = true
		switch strings.ToLower(*decisionType) {
		case "checker":
			filter.DecisionType = CheckerAction
		case "cube":
			filter.DecisionType = CubeAction
		default:
			return fmt.Errorf("invalid decision type: %s (must be 'checker' or 'cube')", *decisionType)
		}
	}

	// Build filter strings for the search function
	var pipCountFilter string
	if *pipMin > 0 || *pipMax > 0 {
		if *pipMin > 0 && *pipMax > 0 {
			pipCountFilter = fmt.Sprintf("p%d,%d", *pipMin, *pipMax)
		} else if *pipMin > 0 {
			pipCountFilter = fmt.Sprintf("p>%d", *pipMin)
		} else {
			pipCountFilter = fmt.Sprintf("p<%d", *pipMax)
		}
	}

	var winRateFilter string
	if *winRateMin > 0 || *winRateMax > 0 {
		if *winRateMin > 0 && *winRateMax > 0 {
			winRateFilter = fmt.Sprintf("w%f,%f", *winRateMin, *winRateMax)
		} else if *winRateMin > 0 {
			winRateFilter = fmt.Sprintf("w>%f", *winRateMin)
		} else {
			winRateFilter = fmt.Sprintf("w<%f", *winRateMax)
		}
	}

	var moveErrorFilter string
	if *moveErrorMin > 0 || *moveErrorMax > 0 {
		if *moveErrorMin > 0 && *moveErrorMax > 0 {
			moveErrorFilter = fmt.Sprintf("E%f,%f", *moveErrorMin, *moveErrorMax)
		} else if *moveErrorMin > 0 {
			moveErrorFilter = fmt.Sprintf("E>%f", *moveErrorMin)
		} else {
			moveErrorFilter = fmt.Sprintf("E<%f", *moveErrorMax)
		}
	}

	var player1CheckerOffFilter string
	if *checkerOff1Min > 0 {
		player1CheckerOffFilter = fmt.Sprintf("o>%d", *checkerOff1Min-1)
	}

	var player2CheckerOffFilter string
	if *checkerOff2Min > 0 {
		player2CheckerOffFilter = fmt.Sprintf("O>%d", *checkerOff2Min-1)
	}

	// Set cube value filter
	includeCube := false
	if *cubeValue > 0 {
		includeCube = true
		filter.Cube.Value = *cubeValue
	}

	// Set score filter
	includeScore := false
	if *score1 >= 0 || *score2 >= 0 || *matchLength > 0 {
		includeScore = true
		if *score1 >= 0 {
			filter.Score[0] = *score1
		}
		if *score2 >= 0 {
			filter.Score[1] = *score2
		}
	}

	// Use the core implementation to get analysis data in the same query, avoiding
	// per-row LoadAnalysis calls for errorMin and hasAnalysis filtering.
	positions, analysisMap, err := cli.db.LoadPositionsByFiltersCore(SearchFilters{
		Filter:                  filter,
		IncludeCube:             includeCube,
		IncludeScore:            includeScore,
		PipCountFilter:          pipCountFilter,
		WinRateFilter:           winRateFilter,
		MoveErrorFilter:         moveErrorFilter,
		Player1CheckerOffFilter: player1CheckerOffFilter,
		Player2CheckerOffFilter: player2CheckerOffFilter,
		DecisionTypeFilter:      decisionTypeFilter,
		MatchIDsFilter:          *matchIDsFlag,
		TournamentIDsFilter:     *tournamentIDsFlag,
	})
	if err != nil {
		return fmt.Errorf("failed to search positions: %v", err)
	}

	// Apply errorMin / hasAnalysis using the analysis map from the JOIN (no extra DB queries).
	var filteredPositions []Position
	for _, pos := range positions {
		if *errorMin > 0 || *hasAnalysis {
			analysis := analysisMap[pos.ID]
			if analysis == nil {
				if *hasAnalysis {
					continue
				}
			} else if *errorMin > 0 {
				hasError := false
				if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 1 {
					if analysis.CheckerAnalysis.Moves[1].EquityError != nil {
						if math.Round(*analysis.CheckerAnalysis.Moves[1].EquityError*1000)/1000 >= *errorMin {
							hasError = true
						}
					}
				}
				if analysis.DoublingCubeAnalysis != nil {
					if math.Round(analysis.DoublingCubeAnalysis.CubefulNoDoubleError*1000)/1000 >= *errorMin ||
						math.Round(analysis.DoublingCubeAnalysis.CubefulDoubleTakeError*1000)/1000 >= *errorMin ||
						math.Round(analysis.DoublingCubeAnalysis.CubefulDoublePassError*1000)/1000 >= *errorMin {
						hasError = true
					}
				}
				if !hasError {
					continue
				}
			}
		}

		filteredPositions = append(filteredPositions, pos)
	}

	// Apply limit
	if *limit > 0 && len(filteredPositions) > *limit {
		filteredPositions = filteredPositions[:*limit]
	}

	// Output results
	fmt.Printf("Found %d position(s)\n\n", len(filteredPositions))

	if len(filteredPositions) == 0 {
		return nil
	}

	// Format output
	switch strings.ToLower(*format) {
	case "json":
		type PositionResult struct {
			ID           int64   `json:"id"`
			XGID         string  `json:"xgid,omitempty"`
			Score        [2]int  `json:"score"`
			Cube         int     `json:"cube"`
			DecisionType string  `json:"decision_type"`
			Dice         [2]int  `json:"dice"`
			BestMove     string  `json:"best_move,omitempty"`
			Equity       float64 `json:"equity,omitempty"`
		}

		var results []PositionResult
		for _, pos := range filteredPositions {
			result := PositionResult{
				ID:    pos.ID,
				Score: pos.Score,
				Cube:  pos.Cube.Value,
				Dice:  pos.Dice,
			}

			if pos.DecisionType == CheckerAction {
				result.DecisionType = "checker"
			} else {
				result.DecisionType = "cube"
			}

			// Get analysis if available
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err == nil && analysis != nil {
				result.XGID = analysis.XGID
				if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
					result.BestMove = analysis.CheckerAnalysis.Moves[0].Move
					result.Equity = analysis.CheckerAnalysis.Moves[0].Equity
				}
			}

			results = append(results, result)
		}

		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %v", err)
		}
		fmt.Println(string(jsonData))

	case "xgid":
		for _, pos := range filteredPositions {
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err == nil && analysis != nil && analysis.XGID != "" {
				fmt.Println(analysis.XGID)
			}
		}

	default: // table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tScore\tCube\tType\tDice\tBest Move\tEquity")
		fmt.Fprintln(w, "--\t-----\t----\t----\t----\t---------\t------")

		for _, pos := range filteredPositions {
			decType := "checker"
			if pos.DecisionType == CubeAction {
				decType = "cube"
			}

			diceStr := ""
			if pos.Dice[0] > 0 {
				diceStr = fmt.Sprintf("%d-%d", pos.Dice[0], pos.Dice[1])
			}

			bestMove := ""
			equityStr := ""

			// Get analysis if available
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err == nil && analysis != nil {
				if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
					bestMove = analysis.CheckerAnalysis.Moves[0].Move
					equityStr = fmt.Sprintf("%.3f", analysis.CheckerAnalysis.Moves[0].Equity)
				} else if analysis.DoublingCubeAnalysis != nil {
					bestMove = analysis.DoublingCubeAnalysis.BestCubeAction
					equityStr = fmt.Sprintf("%.3f", analysis.DoublingCubeAnalysis.CubefulNoDoubleEquity)
				}
			}

			fmt.Fprintf(w, "%d\t%d-%d\t%d\t%s\t%s\t%s\t%s\n",
				pos.ID, pos.Score[0], pos.Score[1], pos.Cube.Value, decType, diceStr, bestMove, equityStr)
		}
		w.Flush()
	}

	// Export to new database if requested
	if *outputDB != "" {
		fmt.Printf("\nExporting %d positions to: %s\n", len(filteredPositions), *outputDB)

		// Get metadata from source database
		metadata, _ := cli.db.LoadMetadata()
		metadata["description"] = fmt.Sprintf("Exported from search: %d positions", len(filteredPositions))
		metadata["dateOfCreation"] = time.Now().Format("2006-01-02 15:04:05")

		err = cli.db.ExportDatabase(ExportOptions{
			ExportPath:         *outputDB,
			Positions:          filteredPositions,
			Metadata:           metadata,
			IncludeAnalysis:    true,
			IncludeComments:    true,
			IncludePlayedMoves: true,
		})
		if err != nil {
			return fmt.Errorf("failed to export database: %v", err)
		}

		fmt.Println("Export completed successfully")
	}

	return nil
}
