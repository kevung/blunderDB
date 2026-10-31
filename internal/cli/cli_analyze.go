package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// NumCPU is the core count a flag defaults to. cmd/cli-doc-gen replaces it,
// so the generated reference does not depend on the machine it ran on.
var NumCPU = runtime.NumCPU

// runAnalyze handles the analyze command: gammonNet's catch-up sweep
// (ADR-0013/ADR-0015) for a library — write an analysis for every position
// that has none — or, with --stale, its re-analysis sweep: every position
// whose stored analysis is entirely gammonNet's own but was written at an
// older EngineVersion or a different depth than --ply now asks for. The same
// query/evaluate/write loop the GUI's auto-after-import trigger, its
// "analyze now" button, and its "re-analyse stale positions" button use
// (Database.AnalyzeMissingWithGammonNet / Database.AnalyzeStaleGammonNet); a
// CLI run has no interactive evaluation to yield to, so it passes no yield
// func and simply runs at full speed, on --jobs cores at once.
func (cli *CLI) runAnalyze(args []string) error {
	analyzeCmd := flag.NewFlagSet("analyze", flag.ContinueOnError)

	dbPath := analyzeCmd.String("db", "", "Path to the database file (required)")
	ply := analyzeCmd.Int("ply", 2, "Search depth (canonical: 2, k=12)")
	pruneK := analyzeCmd.Int("prune-k", 12, "Pruning width (canonical: 12)")
	candidates := analyzeCmd.Int("candidates", 10, "Candidate moves kept per checker decision")
	jobs := analyzeCmd.Int("jobs", NumCPU(), "Positions analysed in parallel (one CPU each)")
	stale := analyzeCmd.Bool("stale", false, "Re-analyse positions whose gammonNet analysis is outdated, instead of filling gaps")
	matchID := analyzeCmd.Int64("match", 0, "Restrict the sweep to one match's positions (0 = the whole library)")
	compare := analyzeCmd.Bool("compare", false, "Compare gammonNet against the imported analyses instead of writing anything")
	limit := analyzeCmd.Int("limit", 0, "With --compare: stop after this many positions (0 = all)")
	format := analyzeCmd.String("format", "text", "Output format: text or json")
	rolloutSpec := analyzeCmd.String("rollout", "", "Roll out the positions --query selects instead: a preset (fast, standard) or custom settings, e.g. 'standard' or 'games=648,truncation=0,ply=1'")
	query := analyzeCmd.String("query", "", "With --rollout: the positions to roll out, in the search query language (see search --query-help); empty means every position")

	analyzeCmd.Usage = func() { cli.printAnalyzeUsage(analyzeCmd) }

	if err := analyzeCmd.Parse(args); err != nil {
		return err
	}

	if *dbPath == "" {
		analyzeCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

	if *rolloutSpec != "" {
		if *stale || *compare || *matchID != 0 {
			return fmt.Errorf("--rollout is its own sweep, narrowed by --query: it does not combine with --stale, --compare or --match")
		}
		if err := cli.initDatabase(*dbPath); err != nil {
			return err
		}
		return cli.runAnalyzeRollout(*rolloutSpec, *query, *jobs, text)
	}
	if *query != "" {
		return fmt.Errorf("--query narrows a --rollout sweep; give --rollout")
	}

	if *compare && *stale {
		return fmt.Errorf("--compare and --stale ask different questions: --compare writes nothing, --stale rewrites; pick one")
	}
	// --match narrows "positions with no analysis"; --stale and --compare
	// both look at positions that HAVE one, so neither has a narrowing to
	// combine with. Refused rather than silently ignored: a scope the user
	// asked for and did not get is the kind of surprise that ends in a
	// library swept by accident.
	if *matchID != 0 && (*stale || *compare) {
		return fmt.Errorf("--match narrows the gap-filling sweep; it means nothing with --stale or --compare")
	}
	if *matchID < 0 {
		return fmt.Errorf("--match takes a match id, got %d", *matchID)
	}

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	if *compare {
		return cli.runAnalyzeCompare(*ply, *pruneK, *candidates, *jobs, *limit, text)
	}

	var (
		total int
		err   error
	)
	switch {
	case *stale:
		total, err = cli.db.CountPositionsWithStaleGammonNet(*ply)
	case *matchID != 0:
		total, err = cli.db.CountMatchPositionsToAnalyze(*matchID)
	default:
		total, err = cli.db.CountPositionsWithoutAnalysis()
	}
	if err != nil {
		return fmt.Errorf("counting positions to analyze: %w", err)
	}
	if total == 0 {
		if text {
			switch {
			case *stale:
				fmt.Fprintln(cli.stdout(), "Nothing to do: no gammonNet analysis is stale at this depth.")
			case *matchID != 0:
				fmt.Fprintf(cli.stdout(), "Nothing to do: every position of match %d already has an analysis.\n", *matchID)
			default:
				fmt.Fprintln(cli.stdout(), "Nothing to do: every position already has an analysis.")
			}
			return nil
		}
		return cli.printJSON(analyzeResult{Total: 0, Analyzed: 0, Stale: *stale, MatchID: *matchID, Ply: *ply, PruneK: *pruneK, Candidates: *candidates, Jobs: *jobs})
	}
	if *jobs < 1 {
		*jobs = 1
	}
	verb := "Analyzing"
	switch {
	case *stale:
		verb = "Re-analyzing stale"
	case *matchID != 0:
		verb = fmt.Sprintf("Analyzing match %d:", *matchID)
	}
	if text {
		fmt.Fprintf(cli.stdout(), "%s %d position(s) with gammonNet (%d-ply, k=%d, %d job(s))...\n", verb, total, *ply, *pruneK, *jobs)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		if _, ok := <-sig; ok {
			if text {
				fmt.Fprintln(cli.stdout(), "\nCancelling...")
			}
			cancel()
		}
	}()

	lastReported := -1
	analyzed := 0
	onProgress := func(done, total int) {
		analyzed = done
		if !text {
			return
		}
		// A line per position would flood a large library's output; report
		// on the first, the last, and every 5% in between.
		pct := done * 100 / total
		if done == total || done == 1 || pct/5 != lastReported/5 {
			fmt.Fprintf(cli.stdout(), "  %d/%d (%d%%)\n", done, total, pct)
		}
		lastReported = pct
	}

	var summary database.GammonNetBatchSummary
	switch {
	case *stale:
		summary, err = cli.db.AnalyzeStaleGammonNet(ctx, *ply, *pruneK, *candidates, *jobs, nil, onProgress)
	case *matchID != 0:
		summary, err = cli.db.AnalyzeMatchWithGammonNet(ctx, *matchID, *ply, *pruneK, *candidates, *jobs, nil, onProgress)
	default:
		summary, err = cli.db.AnalyzeMissingWithGammonNet(ctx, *ply, *pruneK, *candidates, *jobs, nil, onProgress)
	}
	cancelled := ctx.Err() != nil && err != nil
	if err != nil && !cancelled {
		return fmt.Errorf("analyze failed: %w", err)
	}

	if !text {
		return cli.printJSON(analyzeResult{
			Total: total, Analyzed: analyzed, Cancelled: cancelled, Stale: *stale, MatchID: *matchID,
			Evaluated: summary.Evaluated, Refused: summary.Refused, Failed: summary.Failed,
			Ply: *ply, PruneK: *pruneK, Candidates: *candidates, Jobs: *jobs,
		})
	}
	if cancelled {
		fmt.Fprintln(cli.stdout(), "Cancelled.")
	} else {
		fmt.Fprintln(cli.stdout(), "Done.")
	}
	fmt.Fprintf(cli.stdout(), "evaluated: %d, refused: %d, failed: %d\n", summary.Evaluated, summary.Refused, summary.Failed)
	return nil
}

// analyzeResult is the --format json shape for `analyze`.
type analyzeResult struct {
	Total     int  `json:"total"`
	Analyzed  int  `json:"analyzed"`
	Cancelled bool `json:"cancelled,omitempty"`
	// Stale says which sweep ran: filling gaps, or re-analysing what is
	// outdated. Evaluated/Refused/Failed are the same three counters the text
	// output prints — a position gammonNet declines to judge is not a
	// failure, and the JSON must not flatten the two either.
	Stale bool `json:"stale,omitempty"`
	// MatchID names the match the sweep was restricted to, absent when it
	// ran over the whole library — so a caller reading the JSON back can
	// tell a library figure from a match one.
	MatchID    int64 `json:"match_id,omitempty"`
	Evaluated  int   `json:"evaluated"`
	Refused    int   `json:"refused"`
	Failed     int   `json:"failed"`
	Ply        int   `json:"ply"`
	PruneK     int   `json:"prune_k"`
	Candidates int   `json:"candidates"`
	Jobs       int   `json:"jobs"`
}

// runAnalyzeCompare is `blunderdb analyze --compare`: how does the embedded
// engine differ from the analyses that came in with the
// user's files, on the user's own positions?
//
// It writes nothing. That is not a precaution but the point: ADR-0013
// protects an imported analysis unconditionally, and the value of this command
// is precisely that it can be run on a library one is not willing to have
// rewritten.
func (cli *CLI) runAnalyzeCompare(ply, pruneK, candidates, jobs, limit int, text bool) error {
	total, err := cli.db.CountPositionsWithForeignAnalysis()
	if err != nil {
		return fmt.Errorf("counting positions to compare: %w", err)
	}
	if total == 0 {
		if text {
			fmt.Fprintln(cli.stdout(), "Nothing to compare: no position carries an analysis written by another engine.")
			return nil
		}
		return cli.printJSON(gammonnet.Aggregate(nil))
	}
	if limit > 0 && limit < total {
		total = limit
	}
	if jobs < 1 {
		jobs = 1
	}
	if text {
		fmt.Fprintf(cli.stdout(), "Comparing gammonNet with the imported analyses of %d position(s) (%d-ply, k=%d, %d job(s))...\n",
			total, ply, pruneK, jobs)
		fmt.Fprintln(cli.stdout(), "Nothing is written.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		if _, ok := <-sig; ok {
			if text {
				fmt.Fprintln(cli.stdout(), "\nCancelling...")
			}
			cancel()
		}
	}()

	lastReported := -1
	onProgress := func(done, total int) {
		if !text {
			return
		}
		pct := done * 100 / total
		if done == total || done == 1 || pct/5 != lastReported/5 {
			fmt.Fprintf(cli.stdout(), "  %d/%d (%d%%)\n", done, total, pct)
		}
		lastReported = pct
	}

	cmp, err := cli.db.CompareWithGammonNet(ctx, ply, pruneK, candidates, jobs, limit, onProgress)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("comparison failed: %w", err)
	}
	if !text {
		return cli.printJSON(cmp)
	}
	if ctx.Err() != nil {
		fmt.Fprintln(cli.stdout(), "Cancelled; reporting what was compared.")
	}
	fmt.Fprintln(cli.stdout())
	fmt.Fprint(cli.stdout(), cmp.String())
	return nil
}

// printAnalyzeUsage is `analyze --help`, kept out of runAnalyze: fifty print
// statements of prose say nothing about the command's control flow, and
// leaving them inline buried it (and tripped funlen).
func (cli *CLI) printAnalyzeUsage(analyzeCmd *flag.FlagSet) {
	fmt.Fprintln(cli.stdout(), "Usage: blunderdb analyze [options]")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Write a gammonNet analysis for every position that has none —")
	fmt.Fprintln(cli.stdout(), "catching up a library built before this feature existed. A")
	fmt.Fprintln(cli.stdout(), "Position already carrying any analysis (XG, GNUbg, BGBlitz, or a")
	fmt.Fprintln(cli.stdout(), "prior gammonNet run) is left untouched: this only ever fills a")
	fmt.Fprintln(cli.stdout(), "gap (ADR-0013). Interrupted with Ctrl-C, the run is cancelled")
	fmt.Fprintln(cli.stdout(), "cleanly — nothing is lost, and re-running picks up exactly where")
	fmt.Fprintln(cli.stdout(), "it left off, with no journal needed.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "--match restricts the sweep to the positions of a single match,")
	fmt.Fprintln(cli.stdout(), "the id `blunderdb list --type matches` prints. Same gap rule and")
	fmt.Fprintln(cli.stdout(), "same guarantees, narrower scope: a match just imported or")
	fmt.Fprintln(cli.stdout(), "transcribed is analysed without sweeping the whole library, and")
	fmt.Fprintln(cli.stdout(), "a correction re-analysed a second time costs only the positions")
	fmt.Fprintln(cli.stdout(), "the correction created.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "--stale switches to the other sweep: every position whose stored")
	fmt.Fprintln(cli.stdout(), "analysis is entirely gammonNet's own (never an XG/GNUbg/BGBlitz")
	fmt.Fprintln(cli.stdout(), "one — ADR-0013 protects those unconditionally) but was written at")
	fmt.Fprintln(cli.stdout(), "an older engine version or a different depth than --ply now asks")
	fmt.Fprintln(cli.stdout(), "for. Use it after an engine upgrade, or after raising --ply for a")
	fmt.Fprintln(cli.stdout(), "library already analysed at a shallower depth.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Positions are analysed --jobs at a time, on that many cores: the")
	fmt.Fprintln(cli.stdout(), "positions of a batch are independent, so the result is the same")
	fmt.Fprintln(cli.stdout(), "whatever --jobs says. Use --jobs 1 to leave the machine free.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "--compare answers a different question and WRITES NOTHING: on the")
	fmt.Fprintln(cli.stdout(), "positions carrying an analysis somebody else wrote (XG, GNUbg,")
	fmt.Fprintln(cli.stdout(), "BGBlitz), how often does gammonNet name the same best move or the")
	fmt.Fprintln(cli.stdout(), "same cube action, and what would following it have cost on the")
	fmt.Fprintln(cli.stdout(), "imported analysis's own scale? The answer is broken down by game")
	fmt.Fprintln(cli.stdout(), "phase, which is what says where the disagreements sit. Use --limit")
	fmt.Fprintln(cli.stdout(), "to ask the question of a sample rather than of a whole library.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "--rollout switches to rollouts: every position --query selects is")
	fmt.Fprintln(cli.stdout(), "rolled out, one after the other on every core, and the rollout is")
	fmt.Fprintln(cli.stdout(), "written beside its analysis — never in its place. The value is a")
	fmt.Fprintln(cli.stdout(), "preset (fast: 216 games truncated at 7; standard: 1296 games")
	fmt.Fprintln(cli.stdout(), "truncated at 11) or custom settings: an optional preset first, then")
	fmt.Fprintln(cli.stdout(), "games=, min-games=, truncation=, jsd=, ply=, candidates=, seed=,")
	fmt.Fprintln(cli.stdout(), "comma-separated. A position already carrying a rollout with the")
	fmt.Fprintln(cli.stdout(), "same settings is skipped, so an interrupted run resumes; the")
	fmt.Fprintln(cli.stdout(), "position in hand when Ctrl-C arrives is dropped whole.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "A position gammonNet declines to evaluate (a match score beyond")
	fmt.Fprintln(cli.stdout(), "its MET, a cube state it refuses) is reported separately at the")
	fmt.Fprintln(cli.stdout(), "end, as \"refused\": not a failure, and not retried to no effect.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Options:")
	analyzeCmd.PrintDefaults()
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Examples:")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db --jobs 1")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db --match 12")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db --stale --ply 3")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db --format json")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db --compare --limit 500")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db --rollout fast --query 'E>80'")
	fmt.Fprintln(cli.stdout(), "  blunderdb analyze --db database.db --rollout 'standard,ply=1' --query 'c'")
}
