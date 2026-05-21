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
			return nil, fmt.Errorf("invalid ID %q: %v", p, err)
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

	switch command {
	case "create":
		return cli.runCreate(commandArgs)
	case "import":
		return cli.runImport(commandArgs)
	case "export":
		return cli.runExport(commandArgs)
	case "list":
		return cli.runList(commandArgs)
	case "delete":
		return cli.runDelete(commandArgs)
	case "match":
		return cli.runMatch(commandArgs)
	case "verify":
		return cli.runVerify(commandArgs)
	case "info":
		return cli.runInfo(commandArgs)
	case "edit":
		return cli.runEdit(commandArgs)
	case "search":
		return cli.runSearch(commandArgs)
	case "help":
		cli.printUsage()
		return nil
	case "version":
		cli.printVersion()
		return nil
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		cli.printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
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
	fmt.Println("  list      List database contents")
	fmt.Println("  search    Search positions with filters")
	fmt.Println("  match     Display match positions and analysis")
	fmt.Println("  info      Display database metadata")
	fmt.Println("  edit      Edit database metadata")
	fmt.Println("  verify    Verify database integrity")
	fmt.Println("  delete    Delete data from the database")
	fmt.Println("  help      Show this help message")
	fmt.Println("  version   Show version information")
	fmt.Println()
	fmt.Println("Use 'blunderdb <command> --help' for more information about a command.")
}

// printVersion prints version information
func (cli *CLI) printVersion() {
	fmt.Printf("blunderDB version %s\n", DatabaseVersion)
}

// initDatabase initializes the database connection
func (cli *CLI) initDatabase(dbPath string) error {
	// Check if database file exists
	fileExists := true
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fileExists = false
		fmt.Printf("Database file does not exist, creating new database: %s\n", dbPath)
		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
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
		return fmt.Errorf("failed to open database: %v", err)
	}

	fmt.Printf("Connected to database: %s\n", dbPath)
	return nil
}
