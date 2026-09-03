package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
}

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
		fmt.Printf("Unknown command: %s\n\n", command)
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
		"identity":    cli.runIdentity,
		"open":        cli.runOpen,
		"list":        cli.runList,
		"delete":      cli.runDelete,
		"match":       cli.runMatch,
		"verify":      cli.runVerify,
		"info":        cli.runInfo,
		"edit":        cli.runEdit,
		"epc":         cli.runEpc,
		"search":      cli.runSearch,
		"vacuum":      cli.runVacuum,
		"analyze":     cli.runAnalyze,
		"collection":  cli.runCollection,
		"anki":        cli.runAnki,
		"healthcheck": cli.runHealthcheck,
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

// printUsage prints the usage information
func (cli *CLI) printUsage() {
	fmt.Println("blunderDB CLI - Backgammon Database Management Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  blunderdb <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create    Create a new database with optional metadata")
	fmt.Println("  import    Import data into the database (match, position, batch)")
	fmt.Println("  export    Export data from the database")
	fmt.Println("  identity  Show or move your issuer identity")
	fmt.Println("  open      Open a password-protected copy into an ordinary database")
	fmt.Println("  list      List database contents")
	fmt.Println("  search    Search positions with filters")
	fmt.Println("  match     Display match positions and analysis")
	fmt.Println("  collection  Manage collections (list, show, create, rename, delete, export)")
	fmt.Println("  anki      Spaced-repetition decks (decks, stats, forecast, sync)")
	fmt.Println("  epc       EPC, win probability and money cube verdict (bearoff)")
	fmt.Println("  analyze   Write a gammonNet analysis for every position missing one")
	fmt.Println("  info      Display database metadata")
	fmt.Println("  edit      Edit database metadata")
	fmt.Println("  verify    Verify database integrity")
	fmt.Println("  vacuum    Compact the database file, reclaiming freed space")
	fmt.Println("  delete    Delete data from the database")
	fmt.Println("  completion  Print a shell completion script (bash, zsh, fish)")
	fmt.Println("  help      Show this help message")
	fmt.Println("  version   Show version information")
	fmt.Println()
	fmt.Println("Headless mode (advanced, optional — see the 'headless' chapter in the docs):")
	fmt.Println("  serve     Run the HTTP + JSON daemon (SQLite or multi-tenant PostgreSQL)")
	fmt.Println("  call      Invoke a daemon handler in-process (scripting/tests)")
	fmt.Println("  migrate   Copy a SQLite database into PostgreSQL under a tenant")
	fmt.Println("  healthcheck  Probe a running daemon's /readyz; exit 0 when it is ready")
	fmt.Println()
	fmt.Println("Use 'blunderdb <command> --help' for more information about a command.")
}

// printVersion prints the application version and the database schema
// version it speaks (they change independently: DatabaseVersion only bumps
// when the SQLite schema changes).
func (cli *CLI) printVersion() {
	fmt.Printf("blunderDB version %s (database schema %s)\n", appVersion, DatabaseVersion)
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
