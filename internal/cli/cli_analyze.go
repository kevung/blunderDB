package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
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
	analyzeCmd := flag.NewFlagSet("analyze", flag.ExitOnError)

	dbPath := analyzeCmd.String("db", "", "Path to the database file (required)")
	ply := analyzeCmd.Int("ply", 2, "Search depth (canonical: 2, k=12)")
	pruneK := analyzeCmd.Int("prune-k", 12, "Pruning width (canonical: 12)")
	candidates := analyzeCmd.Int("candidates", 10, "Candidate moves kept per checker decision")
	jobs := analyzeCmd.Int("jobs", runtime.NumCPU(), "Positions analysed in parallel (one CPU each)")
	stale := analyzeCmd.Bool("stale", false, "Re-analyse positions whose gammonNet analysis is outdated, instead of filling gaps")

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
		fmt.Println("  blunderdb analyze --db database.db --stale --ply 3")
	}

	if err := analyzeCmd.Parse(args); err != nil {
		return err
	}

	if *dbPath == "" {
		analyzeCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	var (
		total int
		err   error
	)
	if *stale {
		total, err = cli.db.CountPositionsWithStaleGammonNet(*ply)
	} else {
		total, err = cli.db.CountPositionsWithoutAnalysis()
	}
	if err != nil {
		return fmt.Errorf("counting positions to analyze: %w", err)
	}
	if total == 0 {
		if *stale {
			fmt.Println("Nothing to do: no gammonNet analysis is stale at this depth.")
		} else {
			fmt.Println("Nothing to do: every position already has an analysis.")
		}
		return nil
	}
	if *jobs < 1 {
		*jobs = 1
	}
	verb := "Analyzing"
	if *stale {
		verb = "Re-analyzing stale"
	}
	fmt.Printf("%s %d position(s) with gammonNet (%d-ply, k=%d, %d job(s))...\n", verb, total, *ply, *pruneK, *jobs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		if _, ok := <-sig; ok {
			fmt.Println("\nCancelling...")
			cancel()
		}
	}()

	lastReported := -1
	onProgress := func(done, total int) {
		// A line per position would flood a large library's output; report
		// on the first, the last, and every 5% in between.
		pct := done * 100 / total
		if done == total || done == 1 || pct/5 != lastReported/5 {
			fmt.Printf("  %d/%d (%d%%)\n", done, total, pct)
		}
		lastReported = pct
	}

	var summary database.GammonNetBatchSummary
	if *stale {
		summary, err = cli.db.AnalyzeStaleGammonNet(ctx, *ply, *pruneK, *candidates, *jobs, nil, onProgress)
	} else {
		summary, err = cli.db.AnalyzeMissingWithGammonNet(ctx, *ply, *pruneK, *candidates, *jobs, nil, onProgress)
	}
	if err != nil {
		if ctx.Err() != nil {
			fmt.Println("Cancelled.")
			fmt.Printf("evaluated: %d, refused: %d, failed: %d\n", summary.Evaluated, summary.Refused, summary.Failed)
			return nil
		}
		return fmt.Errorf("analyze failed: %w", err)
	}
	fmt.Println("Done.")
	fmt.Printf("evaluated: %d, refused: %d, failed: %d\n", summary.Evaluated, summary.Refused, summary.Failed)
	return nil
}
