// Command blunderdb-synthdb builds a SQLite database of synthetic matches for
// scale benchmarks: no committed fixture comes near the million positions an
// imported corpus reaches, and the search, stats and listing paths only show
// their cost at that size.
//
//	go run ./cmd/blunderdb-synthdb -out /path/scale.db -positions 1000000
//
// Matches are the committed testdata .xg fixtures, re-mapped with fictional
// names, events and dates and with shifted away scores so that positions do
// not all deduplicate (see Generate). The same -seed and fixtures always give
// the same database. testdata/test.xg is left out by default: the scale
// benchmarks import it into the full base, and it must arrive as a new match.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func main() {
	out := flag.String("out", "", "SQLite database to create (must not exist)")
	positions := flag.Int("positions", 100_000, "stop once at least this many positions were written")
	seed := flag.Uint64("seed", 1, "seed for names, events and dates")
	fixtures := flag.String("fixtures", "testdata/*.xg,testdata/*/*.xg", "comma-separated globs of .xg fixtures")
	exclude := flag.String("exclude", "testdata/test.xg", "comma-separated fixtures to leave out")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "blunderdb-synthdb: -out is required")
		os.Exit(2)
	}
	if _, err := os.Stat(*out); err == nil {
		fmt.Fprintf(os.Stderr, "blunderdb-synthdb: %s already exists\n", *out)
		os.Exit(2)
	}

	files, err := expand(*fixtures, *exclude)
	if err != nil {
		fmt.Fprintln(os.Stderr, "blunderdb-synthdb:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	store, err := sqlite.Open(ctx, *out, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "blunderdb-synthdb:", err)
		os.Exit(1)
	}
	defer store.Close()

	start := time.Now()
	last := start
	res, err := Generate(ctx, store, Options{
		Fixtures:  files,
		Positions: *positions,
		Seed:      *seed,
		Progress: func(m, p int) {
			if time.Since(last) >= 10*time.Second {
				last = time.Now()
				el := time.Since(start).Seconds()
				fmt.Fprintf(os.Stderr, "%d matches, %d positions, %.0f pos/s\n", m, p, float64(p)/el)
			}
		},
	})
	for _, s := range res.Skipped {
		fmt.Fprintln(os.Stderr, "skipped:", s)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "blunderdb-synthdb:", err)
		os.Exit(1)
	}
	el := time.Since(start)
	fmt.Printf("%d matches, %d positions written in %s (%.0f pos/s)\n",
		res.Matches, res.Positions, el.Round(time.Second), float64(res.Positions)/el.Seconds())
}

func expand(globs, exclude string) ([]string, error) {
	skip := map[string]bool{}
	for _, e := range strings.Split(exclude, ",") {
		if e = strings.TrimSpace(e); e != "" {
			skip[filepath.Clean(e)] = true
		}
	}
	seen := map[string]bool{}
	var files []string
	for _, g := range strings.Split(globs, ",") {
		matches, err := filepath.Glob(strings.TrimSpace(g))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			m = filepath.Clean(m)
			if skip[m] || seen[m] {
				continue
			}
			seen[m] = true
			files = append(files, m)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no fixture matches %q", globs)
	}
	return files, nil
}
