// Command openingbook-gen computes the book of the first two moves at a
// match score (pkg/blunderdb/openingbook) and compiles it into the file
// the binary embeds. It is a maintainer's tool, like cli-doc-gen: the book
// ships as data, users never run it.
//
// Usage — from the repo root:
//
//	go build -o openingbook-gen ./cmd/openingbook-gen
//	./openingbook-gen run --out openingbook-run.jsonl --threads 16
//	./openingbook-gen compile --in openingbook-run.jsonl
//
// run rolls phase 1, the 15 opening rolls at each score, then phase 2, the
// 21 replies after the best opening play of each score (and the plays tied
// with it, --after). It appends one line per finished job to --out and syncs it; Ctrl-C, a
// kill or a reboot loses at most the jobs in flight, and the same command
// resumes where it stopped. It lowers its own priority (nice 19) unless
// --nice=false, and runs under SCHED_IDLE with --idle. --openings, --replies
// and --scores roll a chosen subset, to check it against another engine. compile sorts the run's lines into
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
	"strings"
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

// presets names the rollout settings run accepts: fast for a first book in
// hours, full for the 2-ply book that corrects it.
func preset(name string) (rollout.Settings, bool) {
	if name == "full" {
		return openingbook.FullSettings(), true
	}
	s, ok := rollout.Preset(name)
	if ok {
		s.Candidates = 4
	}
	return s, ok
}

func parseRolls(spec string) ([]openingbook.Roll, error) {
	var out []openingbook.Roll
	for _, p := range strings.Split(spec, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		r, err := openingbook.ParseRoll(p)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func run(args []string) error {
	cmd := flag.NewFlagSet("run", flag.ContinueOnError)
	out := cmd.String("out", "openingbook-run.jsonl", "Run file: one line per finished job, read back to resume")
	scores := cmd.String("scores", "grid", "Scores: grid (20), full (48), xg, match5 (25), or a list of replier-opener away scores (4-2,1-4,money)")
	openings := cmd.String("openings", "", "Only these opening rolls, comma-separated (31,65): a sample")
	replies := cmd.String("replies", "", "Only these reply rolls, comma-separated (21,66): a sample")
	after := cmd.Int("after", 2, "Reply jobs follow the best opening play and the plays tied with it, at most this many (1 = the best only)")
	threads := cmd.Int("threads", runtime.NumCPU(), "Jobs rolled out at once, one core each")
	presetName := cmd.String("preset", "fast", "Rollout settings: fast (truncated at 7, 108-216 games, 0 ply), standard (truncated at 11, 324-1296 games, 0 ply) or full (untruncated, 324-1296 games, 2 ply)")
	candidates := cmd.Int("candidates", 4, "Plays rolled out per job: the best at 2 ply at the job's score")
	ply := cmd.Int("ply", -1, "Override the preset's ply (-1 keeps it)")
	truncation := cmd.Int("truncation", -1, "Override the preset's truncation in half-moves, 0 = none (-1 keeps it)")
	minGames := cmd.Int("min-games", -1, "Override the preset's minimum games, a multiple of 36 (-1 keeps it)")
	maxGames := cmd.Int("max-games", -1, "Override the preset's maximum games, a multiple of 36 (-1 keeps it)")
	jsd := cmd.Float64("jsd", -1, "Override the preset's JSD stop, 0 = never (-1 keeps it)")
	limit := cmd.Int("limit", 0, "Stop after this many jobs of each phase (0 = all): a sample")
	nice := cmd.Bool("nice", true, "Lower this process's priority to nice 19")
	idle := cmd.Bool("idle", false, "Run under SCHED_IDLE: only on cores nothing else wants (Linux)")
	dry := cmd.Bool("dry-run", false, "Count the jobs done and left, roll nothing")
	if err := cmd.Parse(args); err != nil {
		return err
	}
	sc, err := openingbook.ParseScores(*scores)
	if err != nil {
		return err
	}
	var f openingbook.Filter
	if f.Openings, err = parseRolls(*openings); err != nil {
		return err
	}
	if f.Replies, err = parseRolls(*replies); err != nil {
		return err
	}
	s, ok := preset(*presetName)
	if !ok {
		return fmt.Errorf("unknown preset %q (fast, standard, full)", *presetName)
	}
	s.Candidates = *candidates
	if *ply >= 0 {
		s.Ply = *ply
	}
	if *truncation >= 0 {
		s.Truncation = *truncation
	}
	if *minGames >= 0 {
		s.MinGames = *minGames
	}
	if *maxGames >= 0 {
		s.MaxGames = *maxGames
	}
	if *jsd >= 0 {
		s.JSDLimit = *jsd
	}
	if err := s.Validate(); err != nil {
		return err
	}
	cut := func(jobs []openingbook.Key) []openingbook.Key {
		if *limit > 0 && *limit < len(jobs) {
			return jobs[:*limit]
		}
		return jobs
	}
	replyJobs := func() ([]openingbook.Key, error) {
		b, err := openingbook.ReadRun(*out)
		if err != nil {
			return nil, err
		}
		return cut(openingbook.ReplyJobs(b, sc, *after, f)), nil
	}
	if *dry {
		for i, jobs := range [][]openingbook.Key{cut(openingbook.OpeningJobs(sc, f)), nil} {
			if i == 1 {
				if jobs, err = replyJobs(); err != nil {
					return err
				}
			}
			rep, err := openingbook.Generate(context.Background(), *out, jobs, openingbook.Options{Settings: s, DryRun: true})
			if err != nil {
				return err
			}
			fmt.Printf("phase %d (%s): %d jobs, %d done, %d left\n", i+1, []string{"openings", "replies after the openings rolled so far"}[i], rep.Total, rep.Skipped, rep.Left)
		}
		return nil
	}
	if *nice {
		if err := lowerPriority(); err != nil {
			fmt.Fprintln(os.Stderr, "openingbook-gen: priority unchanged:", err)
		}
	}
	if *idle {
		if err := idlePriority(); err != nil {
			fmt.Fprintln(os.Stderr, "openingbook-gen: SCHED_IDLE refused:", err)
		}
	}
	// The 2-ply choice of candidates forks inside the process's core
	// budget (ADR-0093): GOMAXPROCS is what keeps the run on --threads cores.
	runtime.GOMAXPROCS(max(*threads, 1))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for phase := 1; phase <= 2; phase++ {
		jobs := cut(openingbook.OpeningJobs(sc, f))
		if phase == 2 {
			if jobs, err = replyJobs(); err != nil {
				return err
			}
		}
		start := time.Now()
		rep, err := openingbook.Generate(ctx, *out, jobs, openingbook.Options{
			Settings: s,
			Threads:  *threads,
			Done: func(l openingbook.Line, r openingbook.Report) {
				el := time.Since(start)
				eta := time.Duration(float64(el) / float64(r.Written) * float64(r.Left))
				k, _ := l.Key()
				fmt.Fprintf(os.Stderr, "phase %d  %d/%d  %s  %d games  elapsed %s  left ~%s\n",
					phase, r.Skipped+r.Written, r.Total, k, l.Games,
					el.Round(time.Second), eta.Round(time.Minute))
			},
		})
		fmt.Fprintf(os.Stderr, "phase %d: %d jobs, %d already done, %d written now, %d left\n", phase, rep.Total, rep.Skipped, rep.Written, rep.Left)
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "Interrupted: run the same command again to resume.")
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
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
