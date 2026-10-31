package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runMET handles the met command: the library's match equity tables
// (ADR-0068). The mechanics live on Database (package mets), shared with the
// GUI and the daemon.
func (cli *CLI) runMET(args []string) error {
	cmd := flag.NewFlagSet("met", flag.ContinueOnError)

	dbPath := cmd.String("db", "", "Path to the database file (required)")
	importPath := cmd.String("import", "", "Import a gnubg match equity table (.xml)")
	makeCurrent := cmd.Bool("current", false, "With --import: make the imported table current")
	use := cmd.Int64("use", -1, "Make table ID current (0: the built-in Kazaross-XG2)")
	format := cmd.String("format", "text", "Output format: text or json")

	cmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb met [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "List, import or choose the match equity table (MET) of a database.")
		fmt.Fprintln(cli.stdout(), "Each database has its own table, Kazaross-XG2 by default. gammonNet")
		fmt.Fprintln(cli.stdout(), "values match scores with the current table, and every analysis it")
		fmt.Fprintln(cli.stdout(), "computes records that table. An analysis at a match score valued with")
		fmt.Fprintln(cli.stdout(), "another table is shown as \"different MET\" and left out of the")
		fmt.Fprintln(cli.stdout(), "statistics. Changing the current table rewrites no analysis.")
		fmt.Fprintln(cli.stdout(), "Only explicit gnubg tables are read (not the parametric zadeh or mec).")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		cmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb met --db database.db")
		fmt.Fprintln(cli.stdout(), "  blunderdb met --db database.db --import Rockwell-Kazaross.xml --current")
		fmt.Fprintln(cli.stdout(), "  blunderdb met --db database.db --use 0")
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
	if *makeCurrent && *importPath == "" {
		return fmt.Errorf("--current needs --import")
	}
	if *importPath != "" && *use >= 0 {
		return fmt.Errorf("--import and --use are exclusive")
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	if *importPath != "" {
		t, err := cli.db.ImportMET(*importPath)
		if err != nil {
			return fmt.Errorf("met import failed: %w", err)
		}
		if formatLower == "text" {
			if t.AlreadyHeld {
				fmt.Fprintf(cli.stdout(), "Table already held: #%d %s (same values)\n", t.ID, t.Name)
			} else {
				fmt.Fprintf(cli.stdout(), "Imported: #%d %s\n", t.ID, t.Name)
			}
		}
		if *makeCurrent {
			if err := cli.db.SetCurrentMET(t.ID); err != nil {
				return fmt.Errorf("met import failed: %w", err)
			}
		}
	} else if *use >= 0 {
		if err := cli.db.SetCurrentMET(*use); err != nil {
			return fmt.Errorf("met --use %d failed: %w", *use, err)
		}
	}

	tables, err := cli.db.ListMETs()
	if err != nil {
		return err
	}
	if formatLower == "json" {
		if tables == nil {
			tables = []*domain.MatchEquityTable{}
		}
		return cli.printJSON(tables)
	}
	for _, t := range tables {
		mark := " "
		if t.Current {
			mark = "*"
		}
		fmt.Fprintf(cli.stdout(), "%s %3d  %s\n", mark, t.ID, t.Name)
	}
	return nil
}
