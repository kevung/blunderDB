package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
)

// runAnalyze handles the analyze command: gammonNet's catch-up sweep (#130,
// ADR-0013/ADR-0015) for a library — write an analysis for every position
// that has none. The same query/evaluate/write loop the GUI's auto-after-
// import trigger (#129) and its explicit "analyze now" button use
// (Database.AnalyzeMissingWithGammonNet); a CLI run has no interactive
// evaluation to yield to, so it passes no yield func and simply runs at full
// speed, on --jobs cores at once (#147).
func (cli *CLI) runAnalyze(args []string) error {
	analyzeCmd := flag.NewFlagSet("analyze", flag.ContinueOnError)

	dbPath := analyzeCmd.String("db", "", "Path to the database file (required)")
	ply := analyzeCmd.Int("ply", 2, "Search depth (canonical: 2, k=12)")
	pruneK := analyzeCmd.Int("prune-k", 12, "Pruning width (canonical: 12)")
	candidates := analyzeCmd.Int("candidates", 10, "Candidate moves kept per checker decision")
	jobs := analyzeCmd.Int("jobs", runtime.NumCPU(), "Positions analysed in parallel (one CPU each)")
	format := analyzeCmd.String("format", "text", "Output format: text or json")

	analyzeCmd.Usage = func() {
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
		fmt.Println("Positions are analysed --jobs at a time, on that many cores: the")
		fmt.Println("positions of a batch are independent, so the result is the same")
		fmt.Println("whatever --jobs says. Use --jobs 1 to leave the machine free.")
		fmt.Println()
		fmt.Println("Options:")
		analyzeCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb analyze --db database.db")
		fmt.Println("  blunderdb analyze --db database.db --jobs 1")
		fmt.Println("  blunderdb analyze --db database.db --format json")
	}

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

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	total, err := cli.db.CountPositionsWithoutAnalysis()
	if err != nil {
		return fmt.Errorf("counting positions without analysis: %w", err)
	}
	if total == 0 {
		if text {
			fmt.Println("Nothing to do: every position already has an analysis.")
			return nil
		}
		return printJSON(analyzeResult{Total: 0, Analyzed: 0, Ply: *ply, PruneK: *pruneK, Candidates: *candidates, Jobs: *jobs})
	}
	if *jobs < 1 {
		*jobs = 1
	}
	if text {
		fmt.Printf("Analyzing %d position(s) with gammonNet (%d-ply, k=%d, %d job(s))...\n", total, *ply, *pruneK, *jobs)
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

	err = cli.db.AnalyzeMissingWithGammonNet(ctx, *ply, *pruneK, *candidates, *jobs, nil, onProgress)
	cancelled := ctx.Err() != nil && err != nil
	if err != nil && !cancelled {
		return fmt.Errorf("analyze failed: %w", err)
	}

	if !text {
		return printJSON(analyzeResult{
			Total: total, Analyzed: analyzed, Cancelled: cancelled,
			Ply: *ply, PruneK: *pruneK, Candidates: *candidates, Jobs: *jobs,
		})
	}
	if cancelled {
		fmt.Println("Cancelled.")
		return nil
	}
	fmt.Println("Done.")
	return nil
}

// analyzeResult is the --format json shape for `analyze`.
type analyzeResult struct {
	Total      int  `json:"total"`
	Analyzed   int  `json:"analyzed"`
	Cancelled  bool `json:"cancelled,omitempty"`
	Ply        int  `json:"ply"`
	PruneK     int  `json:"prune_k"`
	Candidates int  `json:"candidates"`
	Jobs       int  `json:"jobs"`
}
