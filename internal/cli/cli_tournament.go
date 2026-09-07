package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// runTournament handles the tournament command: the non-interactive side of directing a
// tournament (ADR-0047, issue #395).
//
// What the software can do, it must be able to do without a graphical interface — that is the
// project's parity invariant. What it must NOT do here is a second director's console: Nicomaque
// ships its own interactive one, and this sub-command deliberately answers only the questions a
// script asks after the fact, or between two gestures: what does this journal replay to, what is
// the ranking, write me the page, give me the raw log, what is directed in this database.
//
// Nothing here waits for input.
func (cli *CLI) runTournament(args []string) error {
	if len(args) < 1 {
		cli.printTournamentUsage()
		return fmt.Errorf("missing tournament sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printTournamentUsage()
		return nil
	}
	run, ok := cli.tournamentHandlers()[sub]
	if !ok {
		cli.printTournamentUsage()
		return fmt.Errorf("unknown tournament sub-command: %s", args[0])
	}
	return run(args[1:])
}

// tournamentHandlers returns the sub-command table of `blunderdb tournament`. A map, like
// handlers(), so the doc-sync test and the inventory script can walk it.
func (cli *CLI) tournamentHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"list":      cli.runTournamentList,
		"verify":    cli.runTournamentVerify,
		"standings": cli.runTournamentStandings,
		"page":      cli.runTournamentPage,
		"export":    cli.runTournamentExport,
	}
}

// TournamentSubcommands returns the sub-commands of `blunderdb tournament`, sorted — the
// exported view cmd/cli-doc-gen walks.
func (cli *CLI) TournamentSubcommands() []string {
	h := cli.tournamentHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printTournamentUsage() {
	fmt.Println("Usage: blunderdb tournament <sub-command> [options]")
	fmt.Println()
	fmt.Println("Read a directed tournament without a graphical interface. Directing one")
	fmt.Println("interactively is the engine's own console; these sub-commands only read.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  list       List the directed tournaments of the database")
	fmt.Println("  verify     Replay a direction and report any remaining warning")
	fmt.Println("  standings  Print the standings as CSV")
	fmt.Println("  page       Write the standalone display page")
	fmt.Println("  export     Print the raw event journal, replayable by the engine's tools")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb tournament list --db base.db")
	fmt.Println("  blunderdb tournament verify --db base.db --id 3")
	fmt.Println("  blunderdb tournament standings --db base.db --id 3 > classement.csv")
	fmt.Println("  blunderdb tournament page --db base.db --id 3 --out /tmp/affichage")
	fmt.Println("  blunderdb tournament export --db base.db --id 3 > journal.json")
}

func tournamentFlagSet(sub, summary string, examples ...string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("tournament "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Printf("Usage: blunderdb tournament %s [options]\n\n%s\n\nOptions:\n", sub, summary)
		fs.PrintDefaults()
		if len(examples) > 0 {
			fmt.Println()
			fmt.Println("Examples:")
			for _, ex := range examples {
				fmt.Println("  " + ex)
			}
		}
	}
	return fs, dbPath
}

// ── list ─────────────────────────────────────────────────────────────────────

func (cli *CLI) runTournamentList(args []string) error {
	fs, dbPath := tournamentFlagSet("list", "List the directed tournaments of the database.",
		"blunderdb tournament list --db base.db",
		"blunderdb tournament list --db base.db --format json")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	rows, err := cli.db.ListDirections()
	if err != nil {
		return fmt.Errorf("failed to list directions: %w", err)
	}
	if strings.ToLower(*format) == "json" {
		if rows == nil {
			rows = []database.DirectionSummary{}
		}
		return printJSON(rows)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATE\tENGINE\tUPDATED")
	for _, r := range rows {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", r.TournamentID, r.State, r.EngineVersion, r.UpdatedAt)
	}
	return w.Flush()
}

// ── verify ───────────────────────────────────────────────────────────────────

// runTournamentVerify replays a direction and reports what the engine still complains about.
//
// It EXITS IN ERROR when a warning remains: that is the point of an after-the-fact check, and a
// script that runs it over a season's databases wants a non-zero status, not a line to grep.
func (cli *CLI) runTournamentVerify(args []string) error {
	fs, dbPath := tournamentFlagSet("verify",
		"Replay a direction and report any remaining warning. Exits in error if one remains.",
		"blunderdb tournament verify --db base.db --id 3",
		"blunderdb tournament verify --db base.db --id 3 --format json")
	id := fs.Int64("id", 0, "Tournament ID (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	v, err := cli.db.GetDirection(*id)
	if err != nil {
		return err
	}
	if strings.ToLower(*format) == "json" {
		if err := printJSON(v.Warnings); err != nil {
			return err
		}
	} else {
		for _, w := range v.Warnings {
			// The engine's codes, not sentences: this output is read by a script, and the
			// nine languages live in the interface.
			line, err := json.Marshal(w)
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, string(line))
		}
		if len(v.Warnings) == 0 {
			fmt.Printf("tournament %d: %d events, no warning\n", *id, v.EventCount)
		}
	}
	if n := len(v.Warnings); n > 0 {
		return fmt.Errorf("tournament %d: %d warning(s) remain", *id, n)
	}
	return nil
}

// ── standings ────────────────────────────────────────────────────────────────

func (cli *CLI) runTournamentStandings(args []string) error {
	fs, dbPath := tournamentFlagSet("standings", "Print the standings of a direction as CSV.",
		"blunderdb tournament standings --db base.db --id 3",
		"blunderdb tournament standings --db base.db --id 3 > classement.csv")
	id := fs.Int64("id", 0, "Tournament ID (required)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	body, err := cli.db.StandingsCSV(*id)
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(body)
	return err
}

// ── page ─────────────────────────────────────────────────────────────────────

// runTournamentPage writes the standalone display page. With --out it writes there and
// remembers the folder, which is what the panel does; without it, the page goes to stdout.
func (cli *CLI) runTournamentPage(args []string) error {
	fs, dbPath := tournamentFlagSet("page", "Write the standalone display page of a direction.",
		"blunderdb tournament page --db base.db --id 3 > affichage.html",
		"blunderdb tournament page --db base.db --id 3 --out /tmp/affichage")
	id := fs.Int64("id", 0, "Tournament ID (required)")
	out := fs.String("out", "", "Folder to write the page into (default: standard output)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	if *out == "" {
		body, err := cli.db.DirectionPageHTML(*id)
		if err != nil {
			return err
		}
		_, err = os.Stdout.WriteString(body)
		return err
	}
	if err := cli.db.SetDirectionOutputDir(*id, *out); err != nil {
		return err
	}
	path, err := cli.db.WriteDirectionPage(*id)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// ── export ───────────────────────────────────────────────────────────────────

// runTournamentExport prints the raw journal, replayable by the engine's own tools.
//
// Raw and not derived: the journal is the whole truth of a Direction, and everything else —
// standings, brackets, warnings — is replayed from it. A tool that reads this output needs no
// blunderDB at all.
func (cli *CLI) runTournamentExport(args []string) error {
	fs, dbPath := tournamentFlagSet("export", "Print the raw event journal of a direction.",
		"blunderdb tournament export --db base.db --id 3 > journal.json")
	id := fs.Int64("id", 0, "Tournament ID (required)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	body, err := cli.db.DirectionJournalJSON(*id)
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(body)
	return err
}
