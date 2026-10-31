package cli

import (
	"flag"
	"fmt"
	"strings"
)

// runReencode handles the reencode command: it rewrites the analyses a
// release before the binary format (ADR-0070) stored as JSON. The mechanics
// live on Database.ReencodeAnalyses, shared with the GUI and the daemon.
func (cli *CLI) runReencode(args []string) error {
	cmd := flag.NewFlagSet("reencode", flag.ContinueOnError)

	dbPath := cmd.String("db", "", "Path to the database file (required)")
	format := cmd.String("format", "text", "Output format: text or json")

	cmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb reencode [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Rewrite the analyses an older release stored as JSON in the compact")
		fmt.Fprintln(cli.stdout(), "binary format, which is about half the size and reads several times")
		fmt.Fprintln(cli.stdout(), "faster. Old analyses stay readable without it; vacuum does the same")
		fmt.Fprintln(cli.stdout(), "conversion. It works in batches: interrupted, it resumes where it")
		fmt.Fprintln(cli.stdout(), "stopped on the next run. It never runs automatically.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		cmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb reencode --db database.db")
		fmt.Fprintln(cli.stdout(), "  blunderdb reencode --db database.db --format json")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		cmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	result, err := cli.db.ReencodeAnalyses()
	if err != nil {
		return fmt.Errorf("reencode failed: %w", err)
	}
	if formatLower == "json" {
		return cli.printJSON(struct {
			Rewritten int `json:"rewritten"`
		}{Rewritten: result.Rewritten})
	}
	fmt.Fprintf(cli.stdout(), "  Analyses rewritten: %d\n", result.Rewritten)
	return nil
}
