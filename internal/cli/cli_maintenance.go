package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

// runMaintenance handles `blunderdb maintenance <sub>`: the library's state
// of health (ADR-0088), one line per operation, read and run through the
// same maintenance package as the GUI's tab and the daemon's routes.
func (cli *CLI) runMaintenance(args []string) error {
	if len(args) < 1 {
		printMaintenanceUsage()
		return fmt.Errorf("missing maintenance sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		printMaintenanceUsage()
		return nil
	}
	run, ok := cli.maintenanceHandlers()[sub]
	if !ok {
		printMaintenanceUsage()
		return fmt.Errorf("unknown maintenance sub-command: %s", args[0])
	}
	return run(args[1:])
}

// maintenanceHandlers returns the sub-command table of `blunderdb
// maintenance`, a map like handlers() so the parity test can walk it.
func (cli *CLI) maintenanceHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"status": cli.runMaintenanceStatus,
		"count":  cli.runMaintenanceCount,
		"ids":    cli.runMaintenanceIDs,
		"run":    cli.runMaintenanceRun,
	}
}

// MaintenanceSubcommands returns the sub-commands of `blunderdb maintenance`,
// sorted — the exported view cmd/cli-doc-gen walks.
func (cli *CLI) MaintenanceSubcommands() []string {
	h := cli.maintenanceHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func printMaintenanceUsage() {
	fmt.Println("Usage: blunderdb maintenance <status|count|ids|run> [operation] --db <file> [--format text|json]")
	fmt.Println()
	fmt.Println("The database's state of health: one line per maintenance operation,")
	fmt.Println("with its counter, its action and its last run. Nothing runs by itself.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  status            Every operation, its free counts and its last run")
	fmt.Println("  count <operation> Compute one operation's counter, however long it takes")
	fmt.Println("  ids <operation>   List the ids one operation counts")
	fmt.Println("  run <operation>   Run one operation's action (Ctrl-C stops it between batches)")
	fmt.Println()
	fmt.Println("Operations:")
	for _, k := range maintenance.Kinds() {
		fmt.Printf("  %s\n", k)
	}
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb maintenance status --db database.db")
	fmt.Println("  blunderdb maintenance count analysis.missing --db database.db")
	fmt.Println("  blunderdb maintenance run stats.matches --db database.db --format json")
}

// maintenanceArgs parses a sub-command's arguments: the operation, when the
// sub-command takes one, may come before or after the flags.
func (cli *CLI) maintenanceArgs(sub string, args []string, wantKind bool) (kind string, json bool, err error) {
	fs := flag.NewFlagSet("maintenance "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		printMaintenanceUsage()
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}
	if wantKind && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		kind, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return "", false, err
	}
	if wantKind && kind == "" && fs.NArg() > 0 {
		kind = fs.Arg(0)
	}
	if *dbPath == "" {
		fs.Usage()
		return "", false, fmt.Errorf("missing required flag: --db")
	}
	if wantKind && kind == "" {
		return "", false, fmt.Errorf("maintenance %s: missing operation (one of %s)", sub, kindList())
	}
	f := strings.ToLower(*format)
	if f != "text" && f != "json" {
		return "", false, fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return "", false, err
	}
	return kind, f == "json", nil
}

func kindList() string {
	ks := maintenance.Kinds()
	s := make([]string, len(ks))
	for i, k := range ks {
		s[i] = string(k)
	}
	return strings.Join(s, ", ")
}

func (cli *CLI) runMaintenanceStatus(args []string) error {
	_, asJSON, err := cli.maintenanceArgs("status", args, false)
	if err != nil {
		return err
	}
	lines, err := cli.db.MaintenanceStatus()
	if err != nil {
		return fmt.Errorf("maintenance status: %w", err)
	}
	if asJSON {
		return printJSON(lines)
	}
	for _, l := range lines {
		count := "-"
		switch {
		case !l.Applicable:
			count = "n/a"
		case l.Count != nil:
			count = fmt.Sprintf("%d %s", *l.Count, l.Unit)
		case l.Tier == maintenance.TierSeconds || l.Tier == maintenance.TierScan:
			count = "(count on demand)"
		}
		last := "never run"
		if l.Last != nil {
			last = fmt.Sprintf("%s, %d changed", l.Last.Outcome, l.Last.Changed)
		}
		fmt.Printf("  %-18s %-24s %s\n", l.Kind, count, last)
	}
	return nil
}

func (cli *CLI) runMaintenanceCount(args []string) error {
	kind, asJSON, err := cli.maintenanceArgs("count", args, true)
	if err != nil {
		return err
	}
	n, err := cli.db.CountMaintenance(kind)
	if err != nil {
		return fmt.Errorf("maintenance count: %w", err)
	}
	if asJSON {
		return printJSON(struct {
			Kind  string `json:"kind"`
			Count int64  `json:"count"`
		}{kind, n})
	}
	fmt.Printf("  %s: %d\n", kind, n)
	return nil
}

func (cli *CLI) runMaintenanceIDs(args []string) error {
	kind, asJSON, err := cli.maintenanceArgs("ids", args, true)
	if err != nil {
		return err
	}
	ids, err := cli.db.MaintenanceIDs(kind)
	if err != nil {
		return fmt.Errorf("maintenance ids: %w", err)
	}
	if asJSON {
		if ids == nil {
			ids = []int64{}
		}
		return printJSON(ids)
	}
	for _, id := range ids {
		fmt.Println(id)
	}
	return nil
}

func (cli *CLI) runMaintenanceRun(args []string) error {
	kind, asJSON, err := cli.maintenanceArgs("run", args, true)
	if err != nil {
		return err
	}
	var r maintenance.Run
	onInterrupt := func() {
		if !asJSON {
			fmt.Fprintln(os.Stderr, "Cancelling after the current batch...")
		}
		cli.db.CancelMaintenance()
	}
	err = withInterruptibleContext(onInterrupt, func(context.Context) error {
		var progress func(done, total int)
		if !asJSON {
			progress = func(done, total int) {
				if total > 0 {
					fmt.Fprintf(os.Stderr, "\r  %d/%d", done, total)
				} else {
					fmt.Fprintf(os.Stderr, "\r  %d", done)
				}
			}
		}
		var runErr error
		r, runErr = cli.db.RunMaintenanceWithProgress(kind, progress)
		if progress != nil && r.Kind != "" {
			fmt.Fprintln(os.Stderr)
		}
		return runErr
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("maintenance run: %w", err)
	}
	if asJSON {
		return printJSON(r)
	}
	fmt.Printf("  %s: %s, %d changed\n", r.Kind, r.Outcome, r.Changed)
	return nil
}
