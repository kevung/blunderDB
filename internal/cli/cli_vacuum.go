package cli

import (
	"flag"
	"fmt"
	"strings"
)

// runVacuum handles the vacuum command: it compacts the database file,
// reclaiming the space deletions leave. The mechanics live on
// Database.Vacuum, shared with the GUI.
func (cli *CLI) runVacuum(args []string) error {
	vacuumCmd := flag.NewFlagSet("vacuum", flag.ContinueOnError)

	dbPath := vacuumCmd.String("db", "", "Path to the database file (required)")
	format := vacuumCmd.String("format", "text", "Output format: text or json")

	vacuumCmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb vacuum [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Compact the database file, reclaiming space left behind by")
		fmt.Fprintln(cli.stdout(), "deletions (matches, tournaments, purges). SQLite needs roughly")
		fmt.Fprintln(cli.stdout(), "twice the current file size in free disk space to rebuild it;")
		fmt.Fprintln(cli.stdout(), "blunderdb refuses with a clear error rather than risk running out")
		fmt.Fprintln(cli.stdout(), "of room partway through. This never runs automatically — it is")
		fmt.Fprintln(cli.stdout(), "the only way it happens.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		vacuumCmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb vacuum --db database.db")
		fmt.Fprintln(cli.stdout(), "  blunderdb vacuum --db database.db --format json")
	}

	if err := vacuumCmd.Parse(args); err != nil {
		return err
	}

	if *dbPath == "" {
		vacuumCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	if formatLower != "json" {
		fmt.Fprintln(cli.stdout(), "Compacting database...")
	}
	result, err := cli.db.Vacuum()
	if err != nil {
		return fmt.Errorf("vacuum failed: %w", err)
	}

	if formatLower == "json" {
		return cli.printJSON(struct {
			SizeBefore int64 `json:"size_before"`
			SizeAfter  int64 `json:"size_after"`
			Reclaimed  int64 `json:"reclaimed"`
		}{
			SizeBefore: result.SizeBefore,
			SizeAfter:  result.SizeAfter,
			Reclaimed:  result.SizeBefore - result.SizeAfter,
		})
	}

	fmt.Fprintf(cli.stdout(), "  Before: %s\n", vacuumCLIHumanBytes(result.SizeBefore))
	fmt.Fprintf(cli.stdout(), "  After:  %s\n", vacuumCLIHumanBytes(result.SizeAfter))
	if result.SizeBefore > result.SizeAfter {
		fmt.Fprintf(cli.stdout(), "  Reclaimed: %s\n", vacuumCLIHumanBytes(result.SizeBefore-result.SizeAfter))
	} else {
		fmt.Fprintln(cli.stdout(), "  Nothing to reclaim.")
	}

	return nil
}

// vacuumCLIHumanBytes formats a byte count for the CLI report (e.g. "12.3 MiB").
func vacuumCLIHumanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMG"[exp])
}
