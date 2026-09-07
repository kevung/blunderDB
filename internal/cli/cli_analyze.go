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

// runAnalyze handles the analyze command: gammonNet's catch-up sweep (#130,
// ADR-0013/ADR-0015) for a library — write an analysis for every position
// that has none — or, with --stale, its re-analysis sweep (#191): every
// position whose stored analysis is entirely gammonNet's own but was written
// at an older EngineVersion or a different depth than --ply now asks for.
// The same query/evaluate/write loop the GUI's auto-after-import trigger
// (#129), its "analyze now" button, and its "re-analyse stale positions"
// button use (Database.AnalyzeMissingWithGammonNet /
// Database.AnalyzeStaleGammonNet); a CLI run has no interactive evaluation
// to yield to, so it passes no yield func and simply runs at full speed, on
// --jobs cores at once (#147).
func (cli *CLI) runAnalyze(args []string) error {
	analyzeCmd := flag.NewFlagSet("analyze", flag.ContinueOnError)

	dbPath := analyzeCmd.String("db", "", "Path to the database file (required)")
	ply := analyzeCmd.Int("ply", 2, "Search depth (canonical: 2, k=12)")
	pruneK := analyzeCmd.Int("prune-k", 12, "Pruning width (canonical: 12)")
	candidates := analyzeCmd.Int("candidates", 10, "Candidate moves kept per checker decision")
	jobs := analyzeCmd.Int("jobs", runtime.NumCPU(), "Positions analysed in parallel (one CPU each)")
	stale := analyzeCmd.Bool("stale", false, "Re-analyse positions whose gammonNet analysis is outdated, instead of filling gaps")
	matchID := analyzeCmd.Int64("match", 0, "Restrict the sweep to one match's positions (0 = the whole library)")
	compare := analyzeCmd.Bool("compare", false, "Compare gammonNet against the imported analyses instead of writing anything")
	limit := analyzeCmd.Int("limit", 0, "With --compare: stop after this many positions (0 = all)")
	format := analyzeCmd.String("format", "text", "Output format: text or json")

	analyzeCmd.Usage = func() { printAnalyzeUsage(analyzeCmd) }

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
				fmt.Println("Nothing to do: no gammonNet analysis is stale at this depth.")
			case *matchID != 0:
				fmt.Printf("Nothing to do: every position of match %d already has an analysis.\n", *matchID)
			default:
				fmt.Println("Nothing to do: every position already has an analysis.")
			}
			return nil
		}
		return printJSON(analyzeResult{Total: 0, Analyzed: 0, Stale: *stale, MatchID: *matchID, Ply: *ply, PruneK: *pruneK, Candidates: *candidates, Jobs: *jobs})
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
		fmt.Printf("%s %d position(s) with gammonNet (%d-ply, k=%d, %d job(s))...\n", verb, total, *ply, *pruneK, *jobs)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		if _, ok := <-sig; ok {
			if text {
				fmt.Println("\nCancelling...")
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
			fmt.Printf("  %d/%d (%d%%)\n", done, total, pct)
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
		return printJSON(analyzeResult{
			Total: total, Analyzed: analyzed, Cancelled: cancelled, Stale: *stale, MatchID: *matchID,
			Evaluated: summary.Evaluated, Refused: summary.Refused, Failed: summary.Failed,
			Ply: *ply, PruneK: *pruneK, Candidates: *candidates, Jobs: *jobs,
		})
	}
	if cancelled {
		fmt.Println("Cancelled.")
	} else {
		fmt.Println("Done.")
	}
	fmt.Printf("evaluated: %d, refused: %d, failed: %d\n", summary.Evaluated, summary.Refused, summary.Failed)
	return nil
}

// analyzeResult is the --format json shape for `analyze`.
type analyzeResult struct {
	Total     int  `json:"total"`
	Analyzed  int  `json:"analyzed"`
	Cancelled bool `json:"cancelled,omitempty"`
	// Stale says which sweep ran: filling gaps, or re-analysing what is
	// outdated (#191). Evaluated/Refused/Failed are the same three counters
	// the text output prints — a position gammonNet declines to judge is not
	// a failure, and the JSON must not flatten the two either (C.4).
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

// runAnalyzeCompare is `blunderdb analyze --compare` (issue #270, fiche I.14):
// how does the embedded engine differ from the analyses that came in with the
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
			fmt.Println("Nothing to compare: no position carries an analysis written by another engine.")
			return nil
		}
		return printJSON(gammonnet.Aggregate(nil))
	}
	if limit > 0 && limit < total {
		total = limit
	}
	if jobs < 1 {
		jobs = 1
	}
	if text {
		fmt.Printf("Comparing gammonNet with the imported analyses of %d position(s) (%d-ply, k=%d, %d job(s))...\n",
			total, ply, pruneK, jobs)
		fmt.Println("Nothing is written.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		if _, ok := <-sig; ok {
			if text {
				fmt.Println("\nCancelling...")
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
			fmt.Printf("  %d/%d (%d%%)\n", done, total, pct)
		}
		lastReported = pct
	}

	cmp, err := cli.db.CompareWithGammonNet(ctx, ply, pruneK, candidates, jobs, limit, onProgress)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("comparison failed: %w", err)
	}
	if !text {
		return printJSON(cmp)
	}
	if ctx.Err() != nil {
		fmt.Println("Cancelled; reporting what was compared.")
	}
	fmt.Println()
	fmt.Print(cmp.String())
	return nil
}

// printAnalyzeUsage is `analyze --help`, kept out of runAnalyze: fifty print
// statements of prose say nothing about the command's control flow, and
// leaving them inline buried it (and tripped funlen).
func printAnalyzeUsage(analyzeCmd *flag.FlagSet) {
	fmt.Println("Usage: blunderdb analyze [options]")
	fmt.Println()
	fmt.Println("Write a gammonNet analysis for every position that has none —")
	fmt.Println("catching up a library built before this feature existed. A")
	fmt.Println("Position already carrying any analysis (XG, GNUbg, BGBlitz, or a")
	fmt.Println("prior gammonNet run) is left untouched: this only ever fills a")
	fmt.Println("gap (ADR-0013). Interrupted with Ctrl-C, the run is cancelled")
	fmt.Println("cleanly — nothing is lost, and re-running picks up exactly where")
	fmt.Println("it left off, with no journal needed.")
	fmt.Println()
	fmt.Println("--match restricts the sweep to the positions of a single match,")
	fmt.Println("the id `blunderdb list --type matches` prints. Same gap rule and")
	fmt.Println("same guarantees, narrower scope: a match just imported or")
	fmt.Println("transcribed is analysed without sweeping the whole library, and")
	fmt.Println("a correction re-analysed a second time costs only the positions")
	fmt.Println("the correction created.")
	fmt.Println()
	fmt.Println("--stale switches to the other sweep: every position whose stored")
	fmt.Println("analysis is entirely gammonNet's own (never an XG/GNUbg/BGBlitz")
	fmt.Println("one — ADR-0013 protects those unconditionally) but was written at")
	fmt.Println("an older engine version or a different depth than --ply now asks")
	fmt.Println("for. Use it after an engine upgrade, or after raising --ply for a")
	fmt.Println("library already analysed at a shallower depth.")
	fmt.Println()
	fmt.Println("Positions are analysed --jobs at a time, on that many cores: the")
	fmt.Println("positions of a batch are independent, so the result is the same")
	fmt.Println("whatever --jobs says. Use --jobs 1 to leave the machine free.")
	fmt.Println()
	fmt.Println("--compare answers a different question and WRITES NOTHING: on the")
	fmt.Println("positions carrying an analysis somebody else wrote (XG, GNUbg,")
	fmt.Println("BGBlitz), how often does gammonNet name the same best move or the")
	fmt.Println("same cube action, and what would following it have cost on the")
	fmt.Println("imported analysis's own scale? The answer is broken down by game")
	fmt.Println("phase, which is what says where the disagreements sit. Use --limit")
	fmt.Println("to ask the question of a sample rather than of a whole library.")
	fmt.Println()
	fmt.Println("A position gammonNet declines to evaluate (a match score beyond")
	fmt.Println("its MET, a cube state it refuses) is reported separately at the")
	fmt.Println("end, as \"refused\": not a failure, and not retried to no effect.")
	fmt.Println()
	fmt.Println("Options:")
	analyzeCmd.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb analyze --db database.db")
	fmt.Println("  blunderdb analyze --db database.db --jobs 1")
	fmt.Println("  blunderdb analyze --db database.db --match 12")
	fmt.Println("  blunderdb analyze --db database.db --stale --ply 3")
	fmt.Println("  blunderdb analyze --db database.db --format json")
	fmt.Println("  blunderdb analyze --db database.db --compare --limit 500")
}
