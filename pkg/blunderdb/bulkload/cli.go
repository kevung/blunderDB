package bulkload

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const usage = `blunderdb load — load match files into the public library, offline

  load --dsn postgres://blunderdb_loader@HOST:5432/DB --source DIR|FILE.zip
       [--password-file FILE] [--batch N] [--limit N]

The password is read from --password-file (default
$BLUNDERDB_LOADER_PASSWORD_FILE, then ` + DefaultPasswordFile + `), never from the
command line. An interrupted run resumes with --batch and the id it printed.
`

// RunCLI parses args and runs Load, printing progress to stderr.
func RunCLI(args []string) error {
	fs := flag.NewFlagSet("load", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(fs.Output(), usage) }
	var opts Options
	pwFile := fs.String("password-file", envOr("BLUNDERDB_LOADER_PASSWORD_FILE", DefaultPasswordFile), "file holding the loader password")
	fs.StringVar(&opts.DSN, "dsn", os.Getenv("BLUNDERDB_LOADER_DSN"), "postgres:// URL of the loader role, without password")
	fs.StringVar(&opts.Source, "source", "", "directory of .xg files, or a .zip of them")
	fs.Int64Var(&opts.Batch, "batch", 0, "resume this import batch")
	fs.IntVar(&opts.Limit, "limit", 0, "load only the first N files, in name order")
	fs.IntVar(&opts.Workers, "workers", 0, "parsing goroutines (default: one per CPU)")
	fs.IntVar(&opts.FilesPerTx, "files-per-tx", 0, "files per transaction")
	fs.BoolVar(&opts.Unnest, "unnest", false, "write by unnest instead of COPY (comparison)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if opts.DSN == "" || opts.Source == "" {
		fs.Usage()
		return fmt.Errorf("load: --dsn and --source are required")
	}
	pw, err := os.ReadFile(*pwFile)
	if err != nil {
		return fmt.Errorf("load: password file: %w", err)
	}
	opts.Password = strings.TrimSpace(string(pw))
	last := time.Time{}
	opts.Progress = func(r Report) {
		if time.Since(last) < 5*time.Second && r.FilesDone < r.FilesTotal {
			return
		}
		last = time.Now()
		printProgress(os.Stderr, r)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	r, err := Load(ctx, opts)
	if r.Batch != 0 {
		fmt.Fprintf(os.Stderr, "load: batch %d (resume with --batch %d)\n", r.Batch, r.Batch)
	}
	return err
}

func printProgress(w io.Writer, r Report) {
	eta := "?"
	if r.ETA >= 0 {
		eta = r.ETA.Round(time.Second).String()
	}
	secs := r.Elapsed.Seconds()
	fps, pps := 0.0, 0.0
	if secs > 0 {
		fps, pps = float64(r.FilesDone)/secs, float64(r.Positions)/secs
	}
	fmt.Fprintf(w, "load: %d/%d files (%d new, %d duplicate, %d failed), %d positions, %.1f files/s, %.0f positions/s, %s left\n",
		r.FilesDone, r.FilesTotal, r.Imported, r.Duplicates, r.Failed, r.Positions, fps, pps, eta)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
