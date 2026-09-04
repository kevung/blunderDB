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
	f := defineSearchFlags(searchCmd)
	searchCmd.Usage = func() { printSearchUsage(searchCmd) }

	if err := searchCmd.Parse(args); err != nil {
		return nil, "", err
	}

	if *f.queryHelp {
		return &searchParams{queryHelp: true}, "", nil
	}

	// Validate required flags
	if *f.dbPath == "" {
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
	if *f.query != "" {
		if named := filterFlagsSet(searchCmd); len(named) > 0 {
			return nil, "", fmt.Errorf("--query cannot be combined with the filter flags (%s): put every filter in the query, or use the flags alone", strings.Join(named, ", "))
		}
		filters, diags := searchquery.Parse(*f.query)
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
			limit:    *f.limit,
			offset:   *f.offset,
			format:   strings.ToLower(*f.format),
			outputDB: *f.outputDB,
		}, *f.dbPath, nil
	}

	filters, err := f.toFilters()
	if err != nil {
		return nil, "", err
	}
	return &searchParams{
		filters:     filters,
		errorMin:    *f.errorMin,
		hasAnalysis: *f.hasAnalysis,
		limit:       *f.limit,
		offset:      *f.offset,
		format:      strings.ToLower(*f.format),
		outputDB:    *f.outputDB,
	}, *f.dbPath, nil
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
