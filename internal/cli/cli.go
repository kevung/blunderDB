package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// withInterruptibleContext runs fn with a context cancelled by SIGINT
// (Ctrl-C) — the CLI's answer to a long-running Database call that cannot
// otherwise be aborted from a foreground process.
// onInterrupt, if non-nil, runs once when the signal arrives, before ctx is
// cancelled — e.g. to print "Cancelling..." only in text output mode.
//
// A caller that wants to tell a genuine failure from a cancellation checks
// errors.Is(err, context.Canceled) on the error fn returns, exactly as a
// context-cancelled query surfaces it (database/sql, and every Storage
// backend's Query/Exec built on it).
func withInterruptibleContext(onInterrupt func(), fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		if _, ok := <-sig; ok {
			if onInterrupt != nil {
				onInterrupt()
			}
			cancel()
		}
	}()
	return fn(ctx)
}

// parseIDList parses a comma-separated string of int64 IDs.
func parseIDList(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %w", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// CLI represents the command-line interface
type CLI struct {
	db *Database
	// out receives everything the commands print on standard output; nil
	// means os.Stdout, so a literal CLI{} stays usable.
	out io.Writer
}

func (cli *CLI) stdout() io.Writer {
	if cli.out == nil {
		return os.Stdout
	}
	return cli.out
}

// SetOutput redirects the commands' standard output.
func (cli *CLI) SetOutput(w io.Writer) { cli.out = w }

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{
		db: NewDatabase(),
	}
}

// Run executes the CLI
func (cli *CLI) Run(args []string) error {
	if len(args) < 1 {
		cli.printUsage()
		return nil
	}

	// Parse the command
	command := args[0]
	commandArgs := args[1:]

	// Lower-cased to match main.go's mode dispatch, which already accepts any
	// casing: `blunderdb VACUUM` must not be routed to the CLI only to be
	// rejected here as unknown.
	run, ok := cli.handlers()[strings.ToLower(command)]
	if !ok {
		fmt.Fprintf(cli.stdout(), "Unknown command: %s\n\n", command)
		cli.printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
	return run(commandArgs)
}

// handlers returns the subcommand table. It is the single source of truth for
// what counts as a CLI invocation: main.go dispatches through IsCommand rather
// than keeping a second, hand-maintained list of names. `vacuum` shipped once
// with its handler wired here and its name missing there, so `blunderdb vacuum`
// silently opened the GUI instead.
func (cli *CLI) handlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"create":      cli.runCreate,
		"import":      cli.runImport,
		"export":      cli.runExport,
		"transfer":    cli.runTransfer,
		"identity":    cli.runIdentity,
		"open":        cli.runOpen,
		"list":        cli.runList,
		"delete":      cli.runDelete,
		"match":       cli.runMatch,
		"verify":      cli.runVerify,
		"info":        cli.runInfo,
		"edit":        cli.runEdit,
		"cubematrix":  cli.runCubeMatrix,
		"epc":         cli.runEpc,
		"gnubgid":     cli.runGnubgID,
		"capture":     cli.runCapture,
		"rollout":     cli.runRollout,
		"search":      cli.runSearch,
		"trash":       cli.runTrash,
		"comment":     cli.runComment,
		"elsewhere":   cli.runElsewhere,
		"vacuum":      cli.runVacuum,
		"repair":      cli.runRepair,
		"reencode":    cli.runReencode,
		"maintenance": cli.runMaintenance,
		"met":         cli.runMET,
		"analyze":     cli.runAnalyze,
		"transcribe":  cli.runTranscribe,
		"duel":        cli.runDuel,
		"collection":  cli.runCollection,
		"lesson":      cli.runLesson,
		"study":       cli.runStudy,
		"anki":        cli.runAnki,
		"stats":       cli.runStats,
		"training":    cli.runTraining,
		"openingbook": cli.runOpeningBook,
		"tournament":  cli.runTournament,
		"players":     cli.runPlayers,
		"events":      cli.runEvents,
		"bearoff":     cli.runBearoff,
		"healthcheck": cli.runHealthcheck,
		"mcp":         cli.runMCP,
		"completion":  cli.runCompletion,
		"help":        func([]string) error { cli.printUsage(); return nil },
		"version":     func([]string) error { cli.printVersion(); return nil },
	}
}

// IsCommand reports whether name is a CLI subcommand, case-insensitively. The
// mode dispatch in main.go asks this before falling through to the GUI.
func IsCommand(name string) bool {
	_, ok := (&CLI{}).handlers()[strings.ToLower(name)]
	return ok
}

// CommandNames returns every top-level CLI subcommand name, sorted. It is the
// exported view of handlers() for callers outside the package that need the
// list without a second hand-maintained copy — currently cmd/cli-doc-gen,
// which walks it to capture each subcommand's --help text into CLI_USAGE.md.
func CommandNames() []string {
	h := (&CLI{}).handlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// printUsage prints the usage information
func (cli *CLI) printUsage() {
	fmt.Fprintln(cli.stdout(), "blunderDB CLI - Backgammon Database Management Tool")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Usage:")
	fmt.Fprintln(cli.stdout(), "  blunderdb <command> [options]")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Commands:")
	fmt.Fprintln(cli.stdout(), "  create    Create a new database with optional metadata")
	fmt.Fprintln(cli.stdout(), "  import    Import data into the database (match, position, batch)")
	fmt.Fprintln(cli.stdout(), "  export    Export data from the database")
	fmt.Fprintln(cli.stdout(), "  transfer  Copy part of one library into another")
	fmt.Fprintln(cli.stdout(), "  identity  Show or move your issuer identity")
	fmt.Fprintln(cli.stdout(), "  open      Open a password-protected copy into an ordinary database")
	fmt.Fprintln(cli.stdout(), "  list      List database contents")
	fmt.Fprintln(cli.stdout(), "  search    Search positions with filters")
	fmt.Fprintln(cli.stdout(), "  match     Display match positions and analysis")
	fmt.Fprintln(cli.stdout(), "  lesson      Manage lessons (ordered steps showing collections and positions; export them)")
	fmt.Fprintln(cli.stdout(), "  study       The study backlog: your unhandled blunders across every import, and the \"studied\" mark")
	fmt.Fprintln(cli.stdout(), "  collection  Manage collections (list, show, create, rename, delete, export)")
	fmt.Fprintln(cli.stdout(), "  anki      Spaced-repetition decks (decks, stats, forecast, sync)")
	fmt.Fprintln(cli.stdout(), "  stats     Statistics computed apart from list --type stats (recurring)")
	fmt.Fprintln(cli.stdout(), "  training  The Training journal (sessions, missed positions)")
	fmt.Fprintln(cli.stdout(), "  openingbook  Which play at this score? Replies to the opening, judged by the book")
	fmt.Fprintln(cli.stdout(), "  cubematrix  Cube verdict at every score of a match, for one position")
	fmt.Fprintln(cli.stdout(), "  epc       EPC, win probability and money cube verdict (bearoff)")
	fmt.Fprintln(cli.stdout(), "  gnubgid   The GNUbg Position ID and Match ID of a position")
	fmt.Fprintln(cli.stdout(), "  capture   Read a position from a screen capture of an XG board")
	fmt.Fprintln(cli.stdout(), "  rollout   Roll out a position's plays or cube decision (gammonNet)")
	fmt.Fprintln(cli.stdout(), "  bearoff   Generate, list, verify and delete the bearoff tables")
	fmt.Fprintln(cli.stdout(), "  analyze   Write a gammonNet analysis for every position missing one")
	fmt.Fprintln(cli.stdout(), "  duel      Play a Duel one Action per call (create, show, move, double, take, pass, stop)")
	fmt.Fprintln(cli.stdout(), "  transcribe  Replay a .mat, a match or a draft and report its inconsistencies")
	fmt.Fprintln(cli.stdout(), "  tournament  Read a directed tournament (list, verify, standings, page, export)")
	fmt.Fprintln(cli.stdout(), "  players   Other spellings of a player (alias add, list, remove, suggest)")
	fmt.Fprintln(cli.stdout(), "  events    Other spellings of an event (alias add, list, remove, suggest)")
	fmt.Fprintln(cli.stdout(), "  info      Display database metadata")
	fmt.Fprintln(cli.stdout(), "  edit      Edit database metadata")
	fmt.Fprintln(cli.stdout(), "  verify    Verify database integrity")
	fmt.Fprintln(cli.stdout(), "  vacuum    Compact the database file, reclaiming freed space")
	fmt.Fprintln(cli.stdout(), "  repair    Recompute the analysis columns from the analyses themselves")
	fmt.Fprintln(cli.stdout(), "  met       List, import or choose the database's match equity table")
	fmt.Fprintln(cli.stdout(), "  reencode  Rewrite analyses stored by older releases in the compact format")
	fmt.Fprintln(cli.stdout(), "  maintenance  State of health: counters, actions and last runs (status, count, ids, run)")
	fmt.Fprintln(cli.stdout(), "  delete    Delete data from the database")
	fmt.Fprintln(cli.stdout(), "  comment   Comments on a position, attributed to their author (add, list)")
	fmt.Fprintln(cli.stdout(), "  elsewhere What other database files know of one board, and its neighbours there")
	fmt.Fprintln(cli.stdout(), "  trash     What was deleted through the trash, and how to put it back")
	fmt.Fprintln(cli.stdout(), "  completion  Print a shell completion script (bash, zsh, fish)")
	fmt.Fprintln(cli.stdout(), "  help      Show this help message")
	fmt.Fprintln(cli.stdout(), "  version   Show version information")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Headless mode (advanced, optional — see the 'headless' chapter in the docs):")
	fmt.Fprintln(cli.stdout(), "  serve     Run the HTTP + JSON daemon (SQLite or multi-tenant PostgreSQL)")
	fmt.Fprintln(cli.stdout(), "  call      Invoke a daemon handler in-process (scripting/tests)")
	fmt.Fprintln(cli.stdout(), "  migrate   Copy a SQLite database into PostgreSQL under a tenant")
	fmt.Fprintln(cli.stdout(), "  healthcheck  Probe a running daemon's /readyz; exit 0 when it is ready")
	fmt.Fprintln(cli.stdout(), "  mcp          Serve the database's tools to an AI assistant (Model Context Protocol, stdio)")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Use 'blunderdb <command> --help' for more information about a command.")
}

// printVersion prints the application version and the database schema
// version it speaks (they change independently: DatabaseVersion only bumps
// when the SQLite schema changes).
func (cli *CLI) printVersion() {
	fmt.Fprintf(cli.stdout(), "blunderDB version %s (database schema %s)\n", appVersion, DatabaseVersion)
}

// initDatabase initializes the database connection
func (cli *CLI) initDatabase(dbPath string) error {
	// ":memory:" is SQLite's special in-memory DSN, never a real path on
	// disk: always fresh, and there is nothing to os.Stat. That distinction
	// matters on Windows, where the leading/trailing colons make ":memory:"
	// a syntactically invalid filename (ERROR_INVALID_NAME, syscall errno
	// 123) rather than "not found" — os.IsNotExist doesn't recognise that as
	// absence (unlike ERROR_FILE_NOT_FOUND/ERROR_PATH_NOT_FOUND), so the
	// generic path below left fileExists at its true default and routed to
	// OpenDatabase, which fails on a schema that SetupDatabase was never
	// given a chance to create ("no such table: metadata").
	if dbPath == ":memory:" {
		if err := cli.db.SetupDatabase(dbPath); err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Connected to database: %s\n", dbPath)
		return nil
	}

	// Check if database file exists
	fileExists := true
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fileExists = false
		fmt.Fprintf(os.Stderr, "Database file does not exist, creating new database: %s\n", dbPath)
		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// For new databases, use SetupDatabase to create the schema
	// For existing databases, use OpenDatabase
	var err error
	if !fileExists {
		err = cli.db.SetupDatabase(dbPath)
	} else {
		err = cli.db.OpenDatabase(dbPath)
	}

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Connected to database: %s\n", dbPath)
	return nil
}
