package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runTournament handles the tournament command: the non-interactive side of directing a
// tournament (ADR-0047).
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
		"ranking":   cli.runTournamentRanking,
		"page":      cli.runTournamentPage,
		"export":    cli.runTournamentExport,
		"move":      cli.runTournamentMove,
		"hall":      cli.runTournamentHall,
		"tables":    cli.runTournamentTables,
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
	fmt.Println("interactively is the engine's own console; these sub-commands read, except")
	fmt.Println("`move`, which changes the table of a running match.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  list       List the directed tournaments of the database")
	fmt.Println("  verify     Replay a direction and report any remaining warning")
	fmt.Println("  standings  Print the standings as CSV")
	fmt.Println("  ranking    Print a season ranking over several tournaments (CSV or JSON)")
	fmt.Println("  page       Write the standalone display page, or a Rencontre's wall page")
	fmt.Println("  export     Print the raw event journal, replayable by the engine's tools")
	fmt.Println("  move       Move a running match to another table, swapping with its occupant")
	fmt.Println("  hall       Print a Rencontre's tables, every event together, and its proposals")
	fmt.Println("  tables     Print the table properties (name, room, reserved, kept for) and rooms")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb tournament list --db base.db")
	fmt.Println("  blunderdb tournament verify --db base.db --id 3")
	fmt.Println("  blunderdb tournament standings --db base.db --id 3 > classement.csv")
	fmt.Println("  blunderdb tournament ranking --db base.db --season --from 2026-01-01 --to 2026-12-31")
	fmt.Println("  blunderdb tournament page --db base.db --id 3 --out /tmp/affichage")
	fmt.Println("  blunderdb tournament page --db base.db --rencontre 1 --out /tmp/salle")
	fmt.Println("  blunderdb tournament export --db base.db --id 3 > journal.json")
	fmt.Println("  blunderdb tournament move --db base.db --id 3 --match m4 --table 7")
	fmt.Println("  blunderdb tournament hall --db base.db --rencontre 1")
	fmt.Println("  blunderdb tournament tables --db base.db --rencontre 1")
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
	fmt.Fprintln(w, "ID\tSTATE\tENGINE\tRENCONTRE\tUPDATED")
	for _, r := range rows {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", r.TournamentID, r.State, r.EngineVersion, r.RencontreName, r.UpdatedAt)
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

// ── ranking ──────────────────────────────────────────────────────────────────

// runTournamentRanking prints the season ranking: the places of several finished tournaments
// turned into points and summed per person (ADR-0062).
func (cli *CLI) runTournamentRanking(args []string) error {
	fs, dbPath := tournamentFlagSet("ranking", "Print a season ranking: the finished tournaments of a Rencontre or a period, scored by place.",
		"blunderdb tournament ranking --db base.db --season --from 2026-01-01 --to 2026-12-31",
		"blunderdb tournament ranking --db base.db --season --rencontre 1 --points 10,6,4 --elo --format json")
	season := fs.Bool("season", false, "Rank over several tournaments (required: a single tournament is `standings`)")
	rencontre := fs.Int64("rencontre", 0, "Only the tournaments of this Rencontre")
	from := fs.String("from", "", "First tournament date, YYYY-MM-DD, inclusive")
	to := fs.String("to", "", "Last tournament date, YYYY-MM-DD, inclusive")
	points := fs.String("points", "", "Points by place, comma-separated, winner first (default 25,18,15,12,10,8,6,4,2,1)")
	participation := fs.Float64("participation", 0, "Points added for every finished tournament played")
	elo := fs.Bool("elo", false, "Add a club Elo replayed over the season's matches (FIBS formula, start 1500)")
	format := fs.String("format", "csv", "Output format: csv or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if !*season {
		fs.Usage()
		return fmt.Errorf("missing required flag: --season (one tournament's standings: tournament standings)")
	}
	q := database.SeasonQuery{RencontreID: *rencontre, From: *from, To: *to, Participation: *participation, Elo: *elo}
	if *points != "" {
		for _, f := range strings.Split(*points, ",") {
			p, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
			if err != nil {
				return fmt.Errorf("--points: %q is not a number", f)
			}
			q.Points = append(q.Points, p)
		}
	}
	switch *format {
	case "csv":
		body, err := cli.db.SeasonCSV(q)
		if err != nil {
			return err
		}
		_, err = os.Stdout.WriteString(body)
		return err
	case "json":
		v, err := cli.db.SeasonRanking(q)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	default:
		return fmt.Errorf("--format: %q is neither csv nor json", *format)
	}
}

// ── page ─────────────────────────────────────────────────────────────────────

// runTournamentPage writes the standalone display page. With --out it writes there and
// remembers the folder, which is what the panel does; without it, the page goes to stdout.
//
// --rencontre writes the room's wall page instead (ADR-0056 §6): --id and --rencontre name two
// different rows, one Tournament and one Rencontre, and are mutually exclusive.
func (cli *CLI) runTournamentPage(args []string) error {
	fs, dbPath := tournamentFlagSet("page", "Write the standalone display page of a direction, or a Rencontre's wall page.",
		"blunderdb tournament page --db base.db --id 3 > affichage.html",
		"blunderdb tournament page --db base.db --id 3 --out /tmp/affichage",
		"blunderdb tournament page --db base.db --rencontre 1 --out /tmp/salle")
	id := fs.Int64("id", 0, "Tournament ID")
	rencontre := fs.Int64("rencontre", 0, "Rencontre ID: write its wall page instead of one tournament's")
	out := fs.String("out", "", "Folder to write the page into (default: standard output)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if (*id == 0) == (*rencontre == 0) {
		fs.Usage()
		return fmt.Errorf("give exactly one of --id or --rencontre")
	}
	if *rencontre != 0 {
		return cli.writeTournamentOrRencontrePage(*out, func() (string, error) { return cli.db.RencontrePageHTML(*rencontre) },
			func(dir string) (*string, error) {
				if _, err := cli.db.SetRencontreOutputDir(*rencontre, dir); err != nil {
					return nil, err
				}
				path, err := cli.db.WriteRencontrePage(*rencontre)
				return &path, err
			})
	}
	return cli.writeTournamentOrRencontrePage(*out, func() (string, error) { return cli.db.DirectionPageHTML(*id) },
		func(dir string) (*string, error) {
			if err := cli.db.SetDirectionOutputDir(*id, dir); err != nil {
				return nil, err
			}
			path, err := cli.db.WriteDirectionPage(*id)
			return &path, err
		})
}

// writeTournamentOrRencontrePage is the --out/stdout choice shared by a Tournament's own page and
// the Rencontre's wall page: render to stdout, or set the folder and write it.
func (cli *CLI) writeTournamentOrRencontrePage(out string, render func() (string, error), write func(dir string) (*string, error)) error {
	if out == "" {
		body, err := render()
		if err != nil {
			return err
		}
		_, err = os.Stdout.WriteString(body)
		return err
	}
	path, err := write(out)
	if err != nil {
		return err
	}
	fmt.Println(*path)
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

// ── move ─────────────────────────────────────────────────────────────────────

// runTournamentMove is the drag-and-drop of the table grid: a running match goes to another
// table, and when that table is taken the two matches swap. The rule (out-of-service table
// refused, both moves in one transaction) is the service's; this only names the gesture.
func (cli *CLI) runTournamentMove(args []string) error {
	fs, dbPath := tournamentFlagSet("move", "Move a running match to another table; a taken table swaps the two matches.",
		"blunderdb tournament move --db base.db --id 3 --match m4 --table 7",
		"blunderdb tournament move --db base.db --id 3 --match m4 --table 2 --format json")
	id := fs.Int64("id", 0, "Tournament ID (required)")
	match := fs.String("match", "", "Running match ID (required)")
	table := fs.Int("table", 0, "Destination table number (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 || *match == "" || *table <= 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id, --match and --table are all required")
	}
	view, err := cli.db.MoveMatchToTable(*id, *match, *table)
	if err != nil {
		return err
	}
	rows := make([]map[string]any, 0, len(view.Running))
	for _, m := range view.Running {
		rows = append(rows, map[string]any{"match": string(m.ID), "table": m.Table})
	}
	// In a Rencontre a swap may have moved a sister event's match too: its matches follow,
	// named with their event.
	if view.RencontreID != 0 {
		h, err := cli.db.RencontreTableGrid(view.RencontreID)
		if err != nil {
			return err
		}
		for _, c := range h.Cells {
			if c.MatchID != "" && c.TournamentID != *id {
				rows = append(rows, map[string]any{"event": c.Event, "match": c.MatchID, "table": c.Table})
			}
		}
	}
	if strings.ToLower(*format) == "json" {
		return printJSON(rows)
	}
	for _, r := range rows {
		if ev, ok := r["event"]; ok {
			fmt.Printf("%s\t%s\ttable %d\n", ev, r["match"], r["table"])
			continue
		}
		fmt.Printf("%s\ttable %d\n", r["match"], r["table"])
	}
	return nil
}

// ── hall ─────────────────────────────────────────────────────────────────────

// runTournamentHall prints the Hall of a Rencontre, the grid the GUI shows: one line per table
// of the room whatever the event, then the proposals of each event. The merge is the service's.
func (cli *CLI) runTournamentHall(args []string) error {
	fs, dbPath := tournamentFlagSet("hall", "Print a Rencontre's tables, every event together, and the proposals of each event.",
		"blunderdb tournament hall --db base.db --rencontre 1",
		"blunderdb tournament hall --db base.db --rencontre 1 --format json")
	rencontre := fs.Int64("rencontre", 0, "Rencontre ID (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *rencontre == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --rencontre")
	}
	h, err := cli.db.RencontreTableGrid(*rencontre)
	if err != nil {
		return err
	}
	if strings.ToLower(*format) == "json" {
		return printJSON(h)
	}
	room := ""
	for _, c := range hallByRoom(h) {
		if c.Room != "" && c.Room != room {
			room = c.Room
			fmt.Printf("room\t%s\n", room)
		}
		table := strconv.Itoa(c.Table)
		if c.NoTable {
			table = "-"
		}
		if c.Name != "" {
			table += " (" + c.Name + ")"
		}
		switch {
		case c.MatchID != "":
			shared := ""
			if c.Shared {
				shared = "\tshared"
			}
			fmt.Printf("%s\t%s\t%s\t%s - %s%s\n", table, c.Event, c.MatchID, c.AName, c.BName, shared)
		case c.Unavailable:
			fmt.Printf("%s\tout of service\n", table)
		case c.Reserved:
			fmt.Printf("%s\treserved\n", table)
		default:
			fmt.Printf("%s\tfree\n", table)
		}
	}
	for _, ev := range h.Events {
		for _, a := range ev.Proposals {
			if a.Kind == tournoi.ActWait {
				continue
			}
			name := func(id tournoi.PlayerID) string {
				if n := ev.Names[string(id)]; n != "" {
					return n
				}
				return string(id)
			}
			if a.A == "" {
				fmt.Printf("proposal\t%s\t%s\n", ev.Name, a.Kind)
				continue
			}
			fmt.Printf("proposal\t%s\t%s\t%s - %s\ttable %d\n", ev.Name, a.Kind, name(a.A), name(a.B), a.Table)
		}
	}
	return nil
}

// hallByRoom orders the Hall's cells by room when the Rencontre has rooms — each room's tables
// in the order of the rooms, then the tables in none, the shared tables and the matches with no
// table — and leaves them as they are otherwise (ADR-0058 §12).
func hallByRoom(h *database.HallView) []database.HallCell {
	if len(h.Rooms) == 0 {
		return h.Cells
	}
	out := make([]database.HallCell, 0, len(h.Cells))
	for _, room := range h.Rooms {
		for _, c := range h.Cells {
			if c.Room == room && !c.NoTable && !c.Shared {
				out = append(out, c)
			}
		}
	}
	for _, c := range h.Cells {
		if c.Room == "" || c.NoTable || c.Shared {
			out = append(out, c)
		}
	}
	return out
}

// ── tables ───────────────────────────────────────────────────────────────────

// eventRooms is one event of a Rencontre with the rooms it may play in; none means every table.
type eventRooms struct {
	TournamentID int64    `json:"tournamentId"`
	Name         string   `json:"name"`
	Rooms        []string `json:"rooms"`
}

// rencontreTables is what `tournament tables --rencontre` prints as JSON.
type rencontreTables struct {
	RencontreID   int64                 `json:"rencontreId"`
	Tables        int                   `json:"tables"`
	TableSettings []domain.TableSetting `json:"tableSettings"`
	Events        []eventRooms          `json:"events"`
}

// runTournamentTables prints the table properties (ADR-0058): a Rencontre's, with the rooms of
// each of its events, or the ones a Tournament plays under — its Rencontre's when it plays in
// one. It reads only: the properties are written through `call` (ADR-0057).
func (cli *CLI) runTournamentTables(args []string) error {
	fs, dbPath := tournamentFlagSet("tables", "Print the table properties — name, room, reserved, kept for — and the rooms of each event.",
		"blunderdb tournament tables --db base.db --rencontre 1",
		"blunderdb tournament tables --db base.db --tournament 3 --format json")
	rencontre := fs.Int64("rencontre", 0, "Rencontre ID")
	tournament := fs.Int64("tournament", 0, "Tournament ID")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if (*rencontre == 0) == (*tournament == 0) {
		fs.Usage()
		return fmt.Errorf("give exactly one of --rencontre and --tournament")
	}
	asJSON := strings.ToLower(*format) == "json"
	if *tournament != 0 {
		plan, err := cli.db.TablePlan(*tournament)
		if err != nil {
			return err
		}
		if asJSON {
			return printJSON(plan)
		}
		printTableSettings(plan.Settings)
		if len(plan.Rooms) > 0 {
			fmt.Printf("rooms\t%s\n", strings.Join(plan.Rooms, ", "))
		}
		return nil
	}
	r, err := cli.db.GetRencontre(*rencontre)
	if err != nil {
		return err
	}
	out := rencontreTables{RencontreID: r.ID, Tables: r.Tables, TableSettings: r.TableSettings, Events: []eventRooms{}}
	for _, m := range r.Members {
		rooms := r.EventRooms[m.TournamentID]
		if rooms == nil {
			rooms = []string{}
		}
		out.Events = append(out.Events, eventRooms{TournamentID: m.TournamentID, Name: m.Name, Rooms: rooms})
	}
	if asJSON {
		return printJSON(out)
	}
	printTableSettings(out.TableSettings)
	for _, ev := range out.Events {
		rooms := "every table"
		if len(ev.Rooms) > 0 {
			rooms = strings.Join(ev.Rooms, ", ")
		}
		fmt.Printf("event\t%d\t%s\t%s\n", ev.TournamentID, ev.Name, rooms)
	}
	return nil
}

// printTableSettings prints one line per table that has properties: number, name, room,
// reserved, the persons it is kept for.
func printTableSettings(settings []domain.TableSetting) {
	fmt.Println("table\tname\troom\treserved\tkept for")
	for _, s := range settings {
		reserved := "no"
		if s.Reserved {
			reserved = "yes"
		}
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", s.Number, s.Name, s.Room, reserved, strings.Join(s.AssignedTo, ", "))
	}
}
