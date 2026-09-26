package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/watch"
)

// importWatch keeps importing a folder as matches appear in it — the
// headless half of the desktop's watched folder. It imports only what
// appears, never what was already there (that is `import --type batch`'s
// job: batch first, then watch). Ctrl-C stops between files, never inside
// one.
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
			// A vanished share usually comes back, and the watcher's memory
			// survives so its files are not seen as new: warn and keep going.
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

// importOneWatched imports one file through the `import --type match` path,
// whose duplicate detection recognises a file rewritten in place.
func (cli *CLI) importOneWatched(path, format string) error {
	if format != "json" {
		fmt.Printf("── %s\n", path)
	}
	return cli.importMatch(path, format)
}
