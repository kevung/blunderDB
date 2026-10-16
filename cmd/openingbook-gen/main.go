// Command openingbook-gen computes the book of replies to the opening move
// at a match score (pkg/blunderdb/openingbook) and compiles it into the file
// the binary embeds. It is a maintainer's tool, like cli-doc-gen: the book
// ships as data, users never run it.
//
// Usage — from the repo root:
//
//	go build -o openingbook-gen ./cmd/openingbook-gen
//	./openingbook-gen run --out openingbook-run.jsonl --threads 16
//	./openingbook-gen compile --in openingbook-run.jsonl
//
// run appends one line per finished job to --out and syncs it; Ctrl-C, a
// kill or a reboot loses at most the jobs in flight, and the same command
// resumes where it stopped. It lowers its own priority (nice 19) unless
// --nice=false. compile sorts the run's lines into
// pkg/blunderdb/openingbook/book.jsonl.gz.
package main

import (
	"compress/gzip"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/openingbook"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	case "compile":
		err = compile(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "openingbook-gen:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: openingbook-gen run [options]      roll the book's jobs out, resumably")
	fmt.Fprintln(os.Stderr, "       openingbook-gen compile [options]  write the embedded book from a run")
}

func run(args []string) error {
	cmd := flag.NewFlagSet("run", flag.ContinueOnError)
	out := cmd.String("out", "openingbook-run.jsonl", "Run file: one line per finished job, read back to resume")
	scores := cmd.String("scores", "grid", "Scores: grid (20), full (48), or a list of replier-opener away scores (4-2,1-4,money)")
	threads := cmd.Int("threads", runtime.NumCPU(), "Jobs rolled out at once, one core each")
	preset := cmd.String("preset", "standard", "Rollout preset: standard (1296 games, truncated at 11, JSD 3) or fast")
	candidates := cmd.Int("candidates", 4, "Plays rolled out per job: the best at 2 ply at the job's score")
	limit := cmd.Int("limit", 0, "Stop after this many jobs of the list (0 = all): a sample")
	nice := cmd.Bool("nice", true, "Lower this process's priority to nice 19")
	dry := cmd.Bool("dry-run", false, "Count the jobs done and left, roll nothing")
	if err := cmd.Parse(args); err != nil {
		return err
	}
	sc, err := openingbook.ParseScores(*scores)
	if err != nil {
		return err
	}
	s, ok := rollout.Preset(*preset)
	if !ok {
		return fmt.Errorf("unknown preset %q (standard, fast)", *preset)
	}
	s.Candidates = *candidates
	if err := s.Validate(); err != nil {
		return err
	}
	jobs := openingbook.Jobs(sc)
	if *limit > 0 && *limit < len(jobs) {
		jobs = jobs[:*limit]
	}
	if *dry {
		rep, err := openingbook.Generate(context.Background(), *out, jobs, openingbook.Options{Settings: s, Threads: 0, DryRun: true})
		if err != nil {
			return err
		}
		fmt.Printf("%d jobs: %d done, %d left\n", rep.Total, rep.Skipped, rep.Left)
		return nil
	}
	if *nice {
		if err := lowerPriority(); err != nil {
			fmt.Fprintln(os.Stderr, "openingbook-gen: priority unchanged:", err)
		}
	}
	// The 2-ply choice of candidates forks inside the process's core
	// budget (ADR-0093): GOMAXPROCS is what keeps the run on --threads cores.
	runtime.GOMAXPROCS(max(*threads, 1))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	start := time.Now()
	rep, err := openingbook.Generate(ctx, *out, jobs, openingbook.Options{
		Settings: s,
		Threads:  *threads,
		Done: func(l openingbook.Line, r openingbook.Report) {
			el := time.Since(start)
			eta := time.Duration(float64(el) / float64(r.Written) * float64(r.Left))
			fmt.Fprintf(os.Stderr, "%d/%d  %s/%s@%s  %d games  elapsed %s  left ~%s\n",
				r.Skipped+r.Written, r.Total, l.Opening, l.Reply, l.Score, l.Games,
				el.Round(time.Second), eta.Round(time.Minute))
		},
	})
	fmt.Fprintf(os.Stderr, "%d jobs: %d already done, %d written now, %d left\n", rep.Total, rep.Skipped, rep.Written, rep.Left)
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "Interrupted: run the same command again to resume.")
		return nil
	}
	return err
}

func compile(args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ContinueOnError)
	in := cmd.String("in", "openingbook-run.jsonl", "Run file written by run")
	out := cmd.String("out", "pkg/blunderdb/openingbook/book.jsonl.gz", "Embedded book to write")
	if err := cmd.Parse(args); err != nil {
		return err
	}
	f, err := os.Open(*in)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	book, err := openingbook.Compile(f)
	if err != nil {
		return err
	}
	w, err := os.Create(*out)
	if err != nil {
		return err
	}
	zw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		_ = w.Close()
		return err
	}
	// No name or time in the header: the same run compiles to the same bytes.
	if err := book.Write(zw); err != nil {
		_ = w.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	fmt.Printf("%d lines → %s\n", book.Len(), *out)
	return nil
}
