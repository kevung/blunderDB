package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// searchParams is what parseSearchFlags extracts from the command line: the
// query itself (a SearchFilters ready for LoadPositionsByFiltersCore, plus
// the two client-side filters the query can't express — errorMin and
// hasAnalysis need the analysis payload the query already returns) and the
// output options (--format, --limit/--offset, --export). Splitting parsing
// from querying and rendering (B.8, #176) makes each independently testable:
// parseSearchFlags never touches a database, and renderResults never touches
// a flag.
type searchParams struct {
	// queryHelp short-circuits everything else: --query-help prints the token
	// list and exits, without needing a database.
	queryHelp bool
	// diags carries what --query understood but could not act on, so runSearch
	// can say so rather than filtering silently.
	diags       []searchquery.Diag
	filters     SearchFilters
	errorMin    float64
	hasAnalysis bool
	limit       int
	offset      int
	format      string
	outputDB    string
}

// parseSearchFlags defines and parses the `search` command's flags, and
// translates them into a searchParams ready for the database query. It calls
// the FlagSet's Usage() itself on a validation error, matching every other
// CLI command's convention of "print usage, then return the error".
func parseSearchFlags(args []string) (*searchParams, string, error) {
	searchCmd := flag.NewFlagSet("search", flag.ContinueOnError)

	// Define flags
	dbPath := searchCmd.String("db", "", "Path to the database file (required)")
	outputDB := searchCmd.String("export", "", "Export results to a new database file")
	limit := searchCmd.Int("limit", 0, "Maximum number of results (0 = no limit)")
	offset := searchCmd.Int("offset", 0, "Skip this many results before the first one returned (paging, with --limit)")
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
	matchIDsFlag := searchCmd.String("match-ids", "", "Filter by match IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'")
	tournamentIDsFlag := searchCmd.String("tournament-ids", "", "Filter by tournament IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'")
	positionIDsFlag := searchCmd.String("position-ids", "", "Filter by position IDs (range '2,7' or explicit list '5;10;15')")
	diceFlag := searchCmd.String("dice", "", "Filter by dice roll: '5,3' matches both dice (any order); '5' matches positions where 5 was rolled on either die")
	individual := searchCmd.Bool("individual", false, "Only positions imported on their own, not as part of a match")
	flagged := searchCmd.Bool("flagged", false, "Only positions you marked for study in the source tool (eXtreme Gammon flags)")
	hasComment := searchCmd.Bool("has-comment", false, "Only positions carrying a comment (whatever its origin — yours or an imported note)")
	noComment := searchCmd.Bool("no-comment", false, "Only positions carrying no comment")
	query := searchCmd.String("query", "", "Search with the interface's own query language, e.g. 's cube p>30 E>0.05' (see --query-help); exclusive with the filter flags")
	queryHelp := searchCmd.Bool("query-help", false, "List the tokens --query understands, and exit")

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

	if err := searchCmd.Parse(args); err != nil {
		return nil, "", err
	}

	if *queryHelp {
		return &searchParams{queryHelp: true}, "", nil
	}

	// Validate required flags
	if *dbPath == "" {
		searchCmd.Usage()
		return nil, "", fmt.Errorf("missing required flag: --db")
	}

	// --query is the interface's own query language, parsed by the one grammar
	// the command bar uses (pkg/blunderdb/searchquery). It reaches the twenty-odd
	// filters no flag exposes — board-free ones at least: patterns, move
	// patterns, dates, equity, comment text, excluded dice, zones and blots.
	//
	// It replaces the filter flags rather than merging with them: a query and a
	// flag setting the same filter would need a precedence rule nobody could
	// remember, and one that quietly loses a filter is worse than a refusal.
	// The flags that say where to search and how to print stay valid.
	if *query != "" {
		if named := filterFlagsSet(searchCmd); len(named) > 0 {
			return nil, "", fmt.Errorf("--query cannot be combined with the filter flags (%s): put every filter in the query, or use the flags alone", strings.Join(named, ", "))
		}
		filters, diags := searchquery.Parse(*query)
		var unknown []string
		for _, d := range diags {
			if d.Kind == searchquery.DiagUnknown {
				unknown = append(unknown, d.Token)
			}
		}
		if len(unknown) > 0 {
			return nil, "", fmt.Errorf("unknown token(s) in --query: %s (see --query-help)", strings.Join(unknown, ", "))
		}
		return &searchParams{
			filters:  filters,
			diags:    diags,
			limit:    *limit,
			offset:   *offset,
			format:   strings.ToLower(*format),
			outputDB: *outputDB,
		}, *dbPath, nil
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
			return nil, "", fmt.Errorf("invalid decision type: %s (must be 'checker' or 'cube')", *decisionType)
		}
	}

	// Set dice roll filter
	diceRollFilter := false
	diceRollMode := ""
	if *diceFlag != "" {
		diceRollFilter = true
		parts := strings.Split(*diceFlag, ",")
		switch len(parts) {
		case 1:
			d1, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil || d1 < 1 || d1 > 6 {
				return nil, "", fmt.Errorf("invalid --dice value %q: die must be 1-6", *diceFlag)
			}
			diceRollMode = "first"
			filter.Dice[0] = d1
		case 2:
			d1, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			d2, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 != nil || err2 != nil || d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
				return nil, "", fmt.Errorf("invalid --dice value %q: each die must be 1-6", *diceFlag)
			}
			diceRollMode = "both"
			filter.Dice[0] = d1
			filter.Dice[1] = d2
		default:
			return nil, "", fmt.Errorf("invalid --dice value %q: expected '5' or '5,3'", *diceFlag)
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

	// Comment-presence filter. The two flags are the CLI spelling of one
	// tri-state, so asking for both at once is a user error worth naming rather
	// than an empty result set to puzzle over.
	commentFilter := ""
	switch {
	case *hasComment && *noComment:
		return nil, "", fmt.Errorf("--has-comment and --no-comment are mutually exclusive")
	case *hasComment:
		commentFilter = "has"
	case *noComment:
		commentFilter = "none"
	}

	formatLower := strings.ToLower(*format)

	return &searchParams{
		filters: SearchFilters{
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
			MatchIDsFilter:          *matchIDsFlag,
			TournamentIDsFilter:     *tournamentIDsFlag,
			PositionIDsFilter:       *positionIDsFlag,

			IndividuallyImportedFilter: *individual,
			FlaggedFilter:              *flagged,
			CommentFilter:              commentFilter,
		},
		errorMin:    *errorMin,
		hasAnalysis: *hasAnalysis,
		limit:       *limit,
		offset:      *offset,
		format:      formatLower,
		outputDB:    *outputDB,
	}, *dbPath, nil
}

// filterFlagsSet names the filter flags the user actually passed. Only the
// flags that select positions count: --db, --format, --limit, --offset and
// --export say where to look and how to print, and stay compatible with
// --query.
func filterFlagsSet(fs *flag.FlagSet) []string {
	passthrough := map[string]bool{
		"db": true, "format": true, "limit": true, "offset": true,
		"export": true, "query": true, "query-help": true,
	}
	var named []string
	fs.Visit(func(f *flag.Flag) {
		if !passthrough[f.Name] {
			named = append(named, "--"+f.Name)
		}
	})
	return named
}

// printQueryHelp lists the query language's tokens. It is deliberately terse
// and points at the manual: the grammar's reference is doc/source/cmd_mode.rst,
// and duplicating it here would be a second thing to keep in step.
func printQueryHelp(w io.Writer) {
	fmt.Fprintln(w, "blunderdb search --query — the interface's query language")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "A query is the same text the application's command bar takes:")
	fmt.Fprintln(w, "  s cube p>30 E>50        cube decisions, 30+ pips behind, 50+ millipoints of error")
	fmt.Fprintln(w, "  s m\"13/11\" T>2026/01/01 played 13/11, imported this year")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags (no value):")
	fmt.Fprintln(w, "  cube score   match the cube / the score of the position on the board")
	fmt.Fprintln(w, "  d            match the decision type (checker or cube)")
	fmt.Fprintln(w, "  D  D1        match both dice / the first die")
	fmt.Fprintln(w, "  nc           no contact")
	fmt.Fprintln(w, "  M            search the mirrored position too")
	fmt.Fprintln(w, "  i            imported on its own, not inside a match")
	fmt.Fprintln(w, "  fl           flagged for study in the source tool")
	fmt.Fprintln(w, "  co  xco      carries a comment / carries none")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Ranges — each takes x>n, x<n or xa,b (lower-case: you; upper-case: the opponent):")
	fmt.Fprintln(w, "  p P          pip count difference / absolute pip count")
	fmt.Fprintln(w, "  w W  g G  b B   win / gammon / backgammon rate")
	fmt.Fprintln(w, "  o O          checkers borne off        k K   checkers back")
	fmt.Fprintln(w, "  z Z          checkers in the zone      bo BO  outfield blots")
	fmt.Fprintln(w, "  bj BJ        blots in the jan          e      equity")
	fmt.Fprintln(w, "  E            error of the played move, in millipoints")
	fmt.Fprintln(w, "  T            creation date, T>2026/01/01")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Values:")
	fmt.Fprintln(w, "  t\"tag\"       comment text (\";\" separates alternatives)")
	fmt.Fprintln(w, "  m\"13/11\"     best move or cube decision")
	fmt.Fprintln(w, "  pl\"Name\"     a player, at either seat")
	fmt.Fprintln(w, "  xD65         exclude the 6-5 roll (repeatable)")
	fmt.Fprintln(w, "  ma1 tn2 id7  match / tournament / position ids (repeatable)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The board pattern itself cannot be typed: it is drawn in the application.")
	fmt.Fprintln(w, "For the same reason `cube`, `score` and `D` compare against the search board,")
	fmt.Fprintln(w, "which on the command line is empty — they match positions with no cube, no")
	fmt.Fprintln(w, "score and no dice. Use --dice, --cube, --score1/--score2 for those.")
	fmt.Fprintln(w, "Full reference: doc/source/cmd_mode.rst, or the in-app help (?).")
}

// runSearch handles the search command
func (cli *CLI) runSearch(args []string) error {
	params, dbPath, err := parseSearchFlags(args)
	if err != nil {
		return err
	}

	if params.queryHelp {
		printQueryHelp(os.Stdout)
		return nil
	}

	// A token the query language understands but cannot act on here (`x`, the
	// exclusion structure, which is a board rather than text) is said out loud
	// on stderr: silently narrowing nothing is how a query lies.
	for _, d := range params.diags {
		fmt.Fprintf(os.Stderr, "note: %s\n", d)
	}

	// Initialize database
	if err := cli.initDatabase(dbPath); err != nil {
		return err
	}

	// --error-min/--has-analysis are applied client-side below, on the
	// analysis payload the query already returns (no extra round trip), and
	// can reject a result the SQL scan matched — so --limit/--offset are only
	// pushed into the SQL scan itself (real pagination, B.10 #178) when
	// neither is set; otherwise the scan stays unbounded and --limit/--offset
	// apply after filtering, exactly as before, so a page is never short just
	// because the SQL page it was drawn from happened to filter out rows.
	opts := storage.ListOpts{}
	if params.errorMin <= 0 && !params.hasAnalysis {
		opts = storage.ListOpts{Limit: params.limit, Offset: params.offset}
	}

	// Ctrl-C cancels the scan in flight instead of waiting it out (B.13,
	// #181), the same contract `analyze` already gives a long batch.
	textOutput := strings.ToLower(params.format) != "json"
	var positions []Position
	var analysisMap map[int64]*PositionAnalysis
	err = withInterruptibleContext(func() {
		if textOutput {
			fmt.Println("\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		positions, analysisMap, err = cli.db.LoadPositionsByFiltersCoreCtx(ctx, params.filters, opts)
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("search cancelled")
		}
		return fmt.Errorf("failed to search positions: %w", err)
	}

	// Apply errorMin / hasAnalysis using the analysis map from the JOIN (no extra DB queries).
	var filteredPositions []Position
	for _, pos := range positions {
		if params.errorMin > 0 || params.hasAnalysis {
			analysis := analysisMap[pos.ID]
			if analysis == nil {
				if params.hasAnalysis {
					continue
				}
			} else if params.errorMin > 0 {
				hasError := false
				if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 1 {
					if analysis.CheckerAnalysis.Moves[1].EquityError != nil {
						if math.Round(*analysis.CheckerAnalysis.Moves[1].EquityError*1000)/1000 >= params.errorMin {
							hasError = true
						}
					}
				}
				if analysis.DoublingCubeAnalysis != nil {
					if math.Round(analysis.DoublingCubeAnalysis.CubefulNoDoubleError*1000)/1000 >= params.errorMin ||
						math.Round(analysis.DoublingCubeAnalysis.CubefulDoubleTakeError*1000)/1000 >= params.errorMin ||
						math.Round(analysis.DoublingCubeAnalysis.CubefulDoublePassError*1000)/1000 >= params.errorMin {
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

	// Apply limit/offset client-side only when they were not already pushed
	// into the SQL scan above (opts.Limit/Offset zero): the --error-min/
	// --has-analysis path queries unbounded, so paging still has to happen
	// here, on the filtered set.
	if opts.Limit == 0 && opts.Offset == 0 && (params.limit > 0 || params.offset > 0) {
		if params.offset > 0 {
			if params.offset >= len(filteredPositions) {
				filteredPositions = nil
			} else {
				filteredPositions = filteredPositions[params.offset:]
			}
		}
		if params.limit > 0 && len(filteredPositions) > params.limit {
			filteredPositions = filteredPositions[:params.limit]
		}
	}

	// Output results
	fmt.Printf("Found %d position(s)\n\n", len(filteredPositions))

	if len(filteredPositions) > 0 {
		if err := cli.renderResults(os.Stdout, filteredPositions, params.format); err != nil {
			return err
		}
	}

	// Export to new database if requested
	if params.outputDB != "" {
		fmt.Printf("\nExporting %d positions to: %s\n", len(filteredPositions), params.outputDB)

		// Get metadata from source database
		metadata, _ := cli.db.LoadMetadata()
		metadata["description"] = fmt.Sprintf("Exported from search: %d positions", len(filteredPositions))
		metadata["dateOfCreation"] = time.Now().Format("2006-01-02 15:04:05")

		err = cli.db.ExportDatabase(ExportOptions{
			ExportPath:         params.outputDB,
			Positions:          filteredPositions,
			Metadata:           metadata,
			IncludeAnalysis:    true,
			IncludeComments:    true,
			IncludePlayedMoves: true,
		})
		if err != nil {
			return fmt.Errorf("failed to export database: %w", err)
		}

		fmt.Println("Export completed successfully")
	}

	return nil
}

// searchPositionResult is the --format json shape for one matched position.
type searchPositionResult struct {
	ID           int64   `json:"id"`
	XGID         string  `json:"xgid,omitempty"`
	Score        [2]int  `json:"score"`
	Cube         int     `json:"cube"`
	DecisionType string  `json:"decision_type"`
	Dice         [2]int  `json:"dice"`
	BestMove     string  `json:"best_move,omitempty"`
	Equity       float64 `json:"equity,omitempty"`
}

// renderResults writes already-filtered, already-limited search results to w
// in the requested format (table, json, or xgid — anything else falls back
// to table, matching the flag's own default). It is the second half of the
// split runSearch used to be (B.8, #176): everything here is pure formatting
// over positions already in hand, plus one LoadAnalysis call per position to
// fill in best-move/equity/XGID — the same lookup both the table and json
// paths always made.
func (cli *CLI) renderResults(w io.Writer, positions []Position, format string) error {
	switch format {
	case "json":
		var results []searchPositionResult
		for _, pos := range positions {
			result := searchPositionResult{
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
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Fprintln(w, string(jsonData))

	case "xgid":
		for _, pos := range positions {
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err == nil && analysis != nil && analysis.XGID != "" {
				fmt.Fprintln(w, analysis.XGID)
			}
		}

	default: // table format
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tScore\tCube\tType\tDice\tBest Move\tEquity")
		fmt.Fprintln(tw, "--\t-----\t----\t----\t----\t---------\t------")

		for _, pos := range positions {
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

			fmt.Fprintf(tw, "%d\t%d-%d\t%d\t%s\t%s\t%s\t%s\n",
				pos.ID, pos.Score[0], pos.Score[1], pos.Cube.Value, decType, diceStr, bestMove, equityStr)
		}
		tw.Flush()
	}

	return nil
}
