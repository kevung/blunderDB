package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/watch"
)

// importWatch keeps importing a folder as matches appear in it (issue #258,
// fiche I.2) — the headless half of the watched folder the desktop offers in
// its settings, and the form a server, a cron or a shell script can use.
//
// It reports what APPEARS, and never what was already there: pointing a watch
// at a folder holding four years of matches must not import all of them. The
// folder as it stands is `import --type batch`'s job, and the two compose
// exactly as one would hope — batch first, then watch.
//
// Ctrl-C stops it between files, never inside one: the file being imported
// finishes and its report is printed before the loop returns.
func (cli *CLI) importWatch(dirPath, format string, interval time.Duration, failOnError bool) error {
	w, err := watch.New(dirPath)
	if err != nil {
		return err
	}
	interval = watch.ClampInterval(interval)

	text := format != "json"
	if text {
		fmt.Printf("Watching %s for new match files (every %s). Ctrl-C to stop.\n\n", dirPath, interval)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			if text {
				fmt.Println("Stopped watching.")
			}
			return nil
		case <-ticker.C:
		}

		files, err := w.Poll()
		if err != nil {
			// A share that has gone away is not the end of the watch: it
			// comes back, and the watcher's memory survives so its contents
			// are not mistaken for new matches. Say so once per occurrence
			// and keep looking.
			if text {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			}
			continue
		}
		for _, f := range files {
			if err := cli.importOneWatched(f, format); err != nil {
				if failOnError {
					return err
				}
				if text {
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", f, err)
				}
			}
		}
	}
}

// importOneWatched imports a single file that appeared, through the same
// `import --type match` path a user would have used by hand — including the
// duplicate detection, so a file rewritten in place is recognised rather than
// imported twice.
func (cli *CLI) importOneWatched(path, format string) error {
	if format != "json" {
		fmt.Printf("── %s\n", path)
	}
	return cli.importMatch(path, format)
}
