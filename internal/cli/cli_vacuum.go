package cli

import (
	"flag"
	"fmt"
)

// runVacuum handles the vacuum command: it compacts the database file,
// reclaiming space left behind by deletions (matches, tournaments, purges)
// that SQLite does not shrink the file for on its own. The mechanics —
// WAL checkpoint, free-space guard, VACUUM outside a transaction, trailing
// ANALYZE — live on Database.Vacuum so the GUI's "Compacter la base" button
// goes through the exact same path.
func (cli *CLI) runVacuum(args []string) error {
	vacuumCmd := flag.NewFlagSet("vacuum", flag.ExitOnError)

	dbPath := vacuumCmd.String("db", "", "Path to the database file (required)")

	vacuumCmd.Usage = func() {
		fmt.Println("Usage: blunderdb vacuum [options]")
		fmt.Println()
		fmt.Println("Compact the database file, reclaiming space left behind by")
		fmt.Println("deletions (matches, tournaments, purges). SQLite needs roughly")
		fmt.Println("twice the current file size in free disk space to rebuild it;")
		fmt.Println("blunderdb refuses with a clear error rather than risk running out")
		fmt.Println("of room partway through. This never runs automatically — it is")
		fmt.Println("the only way it happens.")
		fmt.Println()
		fmt.Println("Options:")
		vacuumCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb vacuum --db database.db")
	}

	if err := vacuumCmd.Parse(args); err != nil {
		return err
	}

	if *dbPath == "" {
		vacuumCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	fmt.Println("Compacting database...")
	result, err := cli.db.Vacuum()
	if err != nil {
		return fmt.Errorf("vacuum failed: %w", err)
	}

	fmt.Printf("  Before: %s\n", vacuumCLIHumanBytes(result.SizeBefore))
	fmt.Printf("  After:  %s\n", vacuumCLIHumanBytes(result.SizeAfter))
	if result.SizeBefore > result.SizeAfter {
		fmt.Printf("  Reclaimed: %s\n", vacuumCLIHumanBytes(result.SizeBefore-result.SizeAfter))
	} else {
		fmt.Println("  Nothing to reclaim.")
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
