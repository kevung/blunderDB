package cli

// cli_search_flags.go — the flags of `blunderdb search`, and what they mean.
//
// parseSearchFlags was 280 lines and 180 statements, the tallest function left
// in the tree after B.15 split find and Compute, and the one that set
// .golangci.yml's statement ceiling. Three of its four parts were not logic at
// all: thirty flag declarations, fifty lines of help text, and the construction
// of a SearchFilters from what was parsed. They are here; what stays in
// cli_search.go is the decision-making — which of --query and the filter flags
// the user chose, and what to refuse (B.15, #183).

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

type searchFlags struct {
	dbPath   *string
	outputDB *string
	limit    *int
	offset   *int
	format   *string

	decisionType      *string
	pipMin            *int
	pipMax            *int
	winRateMin        *float64
	winRateMax        *float64
	cubeValue         *int
	score1            *int
	score2            *int
	matchLength       *int
	errorMin          *float64
	moveErrorMin      *float64
	moveErrorMax      *float64
	hasAnalysis       *bool
	checkerOff1Min    *int
	checkerOff2Min    *int
	matchIDsFlag      *string
	tournamentIDsFlag *string
	positionIDsFlag   *string
	diceFlag          *string
	individual        *bool
	flagged           *bool
	hasComment        *bool
	noComment         *bool
	query             *string
	queryHelp         *bool
}

// defineSearchFlags declares every flag of `blunderdb search` on fs and returns
// them in one place. Splitting the declarations out of parseSearchFlags is what
// lets the rest of that function be read: thirty flags and their help strings
// are a table, not logic.
func defineSearchFlags(fs *flag.FlagSet) *searchFlags {
	return &searchFlags{
		dbPath:   fs.String("db", "", "Path to the database file (required)"),
		outputDB: fs.String("export", "", "Export results to a new database file"),
		limit:    fs.Int("limit", 0, "Maximum number of results (0 = no limit)"),
		offset:   fs.Int("offset", 0, "Skip this many results before the first one returned (paging, with --limit)"),
		format:   fs.String("format", "table", "Output format: table, json, xgid"),

		// Filter flags
		decisionType:      fs.String("decision", "", "Filter by decision type: checker, cube"),
		pipMin:            fs.Int("pip-min", 0, "Minimum pip count difference"),
		pipMax:            fs.Int("pip-max", 0, "Maximum pip count difference"),
		winRateMin:        fs.Float64("winrate-min", 0, "Minimum win rate (%)"),
		winRateMax:        fs.Float64("winrate-max", 0, "Maximum win rate (%)"),
		cubeValue:         fs.Int("cube", 0, "Filter by cube value"),
		score1:            fs.Int("score1", -1, "Filter by player 1 score"),
		score2:            fs.Int("score2", -1, "Filter by player 2 score"),
		matchLength:       fs.Int("match-length", 0, "Filter by match length"),
		errorMin:          fs.Float64("error-min", 0, "Minimum equity error (blunders)"),
		moveErrorMin:      fs.Float64("move-error-min", 0, "Minimum played move error (millipoints)"),
		moveErrorMax:      fs.Float64("move-error-max", 0, "Maximum played move error (millipoints)"),
		hasAnalysis:       fs.Bool("has-analysis", false, "Only positions with analysis"),
		checkerOff1Min:    fs.Int("off1-min", 0, "Minimum checkers off for player 1"),
		checkerOff2Min:    fs.Int("off2-min", 0, "Minimum checkers off for player 2"),
		matchIDsFlag:      fs.String("match-ids", "", "Filter by match IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'"),
		tournamentIDsFlag: fs.String("tournament-ids", "", "Filter by tournament IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'"),
		positionIDsFlag:   fs.String("position-ids", "", "Filter by position IDs (range '2,7' or explicit list '5;10;15')"),
		diceFlag:          fs.String("dice", "", "Filter by dice roll: '5,3' matches both dice (any order); '5' matches positions where 5 was rolled on either die"),
		individual:        fs.Bool("individual", false, "Only positions imported on their own, not as part of a match"),
		flagged:           fs.Bool("flagged", false, "Only positions you marked for study in the source tool (eXtreme Gammon flags)"),
		hasComment:        fs.Bool("has-comment", false, "Only positions carrying a comment (whatever its origin — yours or an imported note)"),
		noComment:         fs.Bool("no-comment", false, "Only positions carrying no comment"),
		query:             fs.String("query", "", "Search with the interface's own query language, e.g. 's cube p>30 E>0.05' (see --query-help); exclusive with the filter flags"),
		queryHelp:         fs.Bool("query-help", false, "List the tokens --query understands, and exit"),
	}
}

// printSearchUsage prints `blunderdb search --help`: the option list flag.FlagSet
// generates, then the worked examples. Prose, not logic — kept out of
// parseSearchFlags for the same reason as the declarations above.
func printSearchUsage(fs *flag.FlagSet) {
	fmt.Println("Usage: blunderdb search [options]")
	fmt.Println()
	fmt.Println("Search for positions in the database using filters.")
	fmt.Println()
	fmt.Println("Options:")
	fs.PrintDefaults()
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
	fmt.Println("  # Search in specific matches (2, 5, and 9)")
	fmt.Println("  blunderdb search --db database.db --match-ids 2,5,9")
	fmt.Println()
	fmt.Println("  # Search in a tournament")
	fmt.Println("  blunderdb search --db database.db --tournament-ids 1")
	fmt.Println()
	fmt.Println("  # Search positions where dice were 6-5")
	fmt.Println("  blunderdb search --db database.db --dice 6,5")
	fmt.Println()
	fmt.Println("  # Find the positions you imported yourself, not the ones matches brought in")
	fmt.Println("  blunderdb search --db database.db --individual")
	fmt.Println()
	fmt.Println("  # Search positions where a 6 was rolled on either die")
	fmt.Println("  blunderdb search --db database.db --dice 6")
	fmt.Println()
	fmt.Println("  # Positions flagged for study in XG")
	fmt.Println("  blunderdb search --db database.db --flagged")
	fmt.Println()
	fmt.Println("  # Find every commented position")
	fmt.Println("  blunderdb search --db database.db --has-comment")
	fmt.Println()
	fmt.Println("  # Blunders still waiting to be annotated")
	fmt.Println("  blunderdb search --db database.db --no-comment --error-min 0.1")
	fmt.Println()
	fmt.Println("  # The interface's own query language: cube decisions, 30+ pips behind, 50 millipoints of error")
	fmt.Println("  blunderdb search --db database.db --query 's cube p>30 E>50'")
	fmt.Println()
	fmt.Println("  # Filters no flag exposes: a move pattern, a comment tag, a player, a date")
	fmt.Println("  blunderdb search --db database.db --query 's m\"13/11\" t\"blunder\" pl\"Alice\" T>2026/01/01'")
}

// toFilters turns the parsed flags into the SearchFilters the storage layer
// takes. It is the whole of what `blunderdb search` means by its flags: an
// empty board narrowed by whatever was asked for.
func (f *searchFlags) toFilters() (SearchFilters, error) {
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
	if *f.decisionType != "" {
		decisionTypeFilter = true
		switch strings.ToLower(*f.decisionType) {
		case "checker":
			filter.DecisionType = CheckerAction
		case "cube":
			filter.DecisionType = CubeAction
		default:
			return SearchFilters{}, fmt.Errorf("invalid decision type: %s (must be 'checker' or 'cube')", *f.decisionType)
		}
	}

	// Set dice roll filter
	diceRollFilter := false
	diceRollMode := ""
	if *f.diceFlag != "" {
		diceRollFilter = true
		parts := strings.Split(*f.diceFlag, ",")
		switch len(parts) {
		case 1:
			d1, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil || d1 < 1 || d1 > 6 {
				return SearchFilters{}, fmt.Errorf("invalid --dice value %q: die must be 1-6", *f.diceFlag)
			}
			diceRollMode = "first"
			filter.Dice[0] = d1
		case 2:
			d1, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			d2, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 != nil || err2 != nil || d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
				return SearchFilters{}, fmt.Errorf("invalid --dice value %q: each die must be 1-6", *f.diceFlag)
			}
			diceRollMode = "both"
			filter.Dice[0] = d1
			filter.Dice[1] = d2
		default:
			return SearchFilters{}, fmt.Errorf("invalid --dice value %q: expected '5' or '5,3'", *f.diceFlag)
		}
		// The dice filter also constrains decision_type; default to CheckerAction
		// when --decision was not given (cube actions do not have a roll).
		if !decisionTypeFilter {
			decisionTypeFilter = true
			filter.DecisionType = CheckerAction
		}
	}

	// Build filter strings for the search function
	var pipCountFilter string
	if *f.pipMin > 0 || *f.pipMax > 0 {
		if *f.pipMin > 0 && *f.pipMax > 0 {
			pipCountFilter = fmt.Sprintf("p%d,%d", *f.pipMin, *f.pipMax)
		} else if *f.pipMin > 0 {
			pipCountFilter = fmt.Sprintf("p>%d", *f.pipMin)
		} else {
			pipCountFilter = fmt.Sprintf("p<%d", *f.pipMax)
		}
	}

	var winRateFilter string
	if *f.winRateMin > 0 || *f.winRateMax > 0 {
		if *f.winRateMin > 0 && *f.winRateMax > 0 {
			winRateFilter = fmt.Sprintf("w%f,%f", *f.winRateMin, *f.winRateMax)
		} else if *f.winRateMin > 0 {
			winRateFilter = fmt.Sprintf("w>%f", *f.winRateMin)
		} else {
			winRateFilter = fmt.Sprintf("w<%f", *f.winRateMax)
		}
	}

	var moveErrorFilter string
	if *f.moveErrorMin > 0 || *f.moveErrorMax > 0 {
		if *f.moveErrorMin > 0 && *f.moveErrorMax > 0 {
			moveErrorFilter = fmt.Sprintf("E%f,%f", *f.moveErrorMin, *f.moveErrorMax)
		} else if *f.moveErrorMin > 0 {
			moveErrorFilter = fmt.Sprintf("E>%f", *f.moveErrorMin)
		} else {
			moveErrorFilter = fmt.Sprintf("E<%f", *f.moveErrorMax)
		}
	}

	var player1CheckerOffFilter string
	if *f.checkerOff1Min > 0 {
		player1CheckerOffFilter = fmt.Sprintf("o>%d", *f.checkerOff1Min-1)
	}

	var player2CheckerOffFilter string
	if *f.checkerOff2Min > 0 {
		player2CheckerOffFilter = fmt.Sprintf("O>%d", *f.checkerOff2Min-1)
	}

	// Set cube value filter
	includeCube := false
	if *f.cubeValue > 0 {
		includeCube = true
		filter.Cube.Value = *f.cubeValue
	}

	// Set score filter
	includeScore := false
	if *f.score1 >= 0 || *f.score2 >= 0 || *f.matchLength > 0 {
		includeScore = true
		if *f.score1 >= 0 {
			filter.Score[0] = *f.score1
		}
		if *f.score2 >= 0 {
			filter.Score[1] = *f.score2
		}
	}

	// Comment-presence filter. The two flags are the CLI spelling of one
	// tri-state, so asking for both at once is a user error worth naming rather
	// than an empty result set to puzzle over.
	commentFilter := ""
	switch {
	case *f.hasComment && *f.noComment:
		return SearchFilters{}, fmt.Errorf("--has-comment and --no-comment are mutually exclusive")
	case *f.hasComment:
		commentFilter = "has"
	case *f.noComment:
		commentFilter = "none"
	}

	return SearchFilters{
		Filter:                  filter,
		IncludeCube:             includeCube,
		IncludeScore:            includeScore,
		PipCountFilter:          pipCountFilter,
		WinRateFilter:           winRateFilter,
		MoveErrorFilter:         moveErrorFilter,
		Player1CheckerOffFilter: player1CheckerOffFilter,
		Player2CheckerOffFilter: player2CheckerOffFilter,
		DecisionTypeFilter:      decisionTypeFilter,
		DiceRollFilter:          diceRollFilter,
		DiceRollMode:            diceRollMode,
		MatchIDsFilter:          *f.matchIDsFlag,
		TournamentIDsFilter:     *f.tournamentIDsFlag,
		PositionIDsFilter:       *f.positionIDsFlag,

		IndividuallyImportedFilter: *f.individual,
		FlaggedFilter:              *f.flagged,
		CommentFilter:              commentFilter,
	}, nil
}
