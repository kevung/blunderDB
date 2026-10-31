package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// runCreate handles the create command
func (cli *CLI) runCreate(args []string) error {
	createCmd := flag.NewFlagSet("create", flag.ContinueOnError)

	// Define flags
	dbPath := createCmd.String("db", "", "Path to the database file to create (required)")
	force := createCmd.Bool("force", false, "Overwrite existing database if it exists")
	user := createCmd.String("user", "", "User name (owner of the database)")
	description := createCmd.String("description", "", "Description of the database")
	format := createCmd.String("format", "text", "Output format: text or json")

	createCmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb create [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Create a new database with the required schema and optional metadata.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		createCmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  # Create a new database")
		fmt.Fprintln(cli.stdout(), "  blunderdb create --db mydb.db")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Create with metadata")
		fmt.Fprintln(cli.stdout(), "  blunderdb create --db mydb.db --user \"John Doe\" --description \"My backgammon positions\"")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Force overwrite an existing database")
		fmt.Fprintln(cli.stdout(), "  blunderdb create --db mydb.db --force")
	}

	if err := createCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		createCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

	if !strings.HasSuffix(strings.ToLower(*dbPath), ".db") {
		*dbPath += ".db"
	}

	existing := true
	if _, err := os.Stat(*dbPath); err != nil {
		existing = false
	} else if !*force {
		return fmt.Errorf("database already exists: %s (use --force to overwrite)", *dbPath)
	}

	dir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// --force replaces any file at dbPath, including a corrupt one SQLite
	// cannot open ("file is not a database"), so remove it first rather than
	// let SetupDatabase try to erase it.
	if existing {
		if err := os.Remove(*dbPath); err != nil {
			return fmt.Errorf("failed to remove existing database: %w", err)
		}
	}

	// Create the database
	if text {
		fmt.Fprintf(cli.stdout(), "Creating database: %s\n", *dbPath)
	}
	err := cli.db.SetupDatabase(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Save metadata if provided
	metadata := make(map[string]string)
	if *user != "" {
		metadata["user"] = *user
	}
	if *description != "" {
		metadata["description"] = *description
	}
	metadata["dateOfCreation"] = time.Now().Format("2006-01-02 15:04:05")

	if len(metadata) > 0 {
		err = cli.db.SaveMetadata(metadata)
		if err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	if !text {
		return cli.printJSON(struct {
			Path        string `json:"path"`
			Version     string `json:"version"`
			User        string `json:"user,omitempty"`
			Description string `json:"description,omitempty"`
			Created     string `json:"created"`
		}{
			Path:        *dbPath,
			Version:     DatabaseVersion,
			User:        *user,
			Description: *description,
			Created:     metadata["dateOfCreation"],
		})
	}

	fmt.Fprintf(cli.stdout(), "Successfully created database with schema version %s\n", DatabaseVersion)

	// Show database info
	fmt.Fprintln(cli.stdout(), "\nDatabase Information:")
	fmt.Fprintf(cli.stdout(), "  Version: %s\n", DatabaseVersion)
	if *user != "" {
		fmt.Fprintf(cli.stdout(), "  User: %s\n", *user)
	}
	if *description != "" {
		fmt.Fprintf(cli.stdout(), "  Description: %s\n", *description)
	}
	fmt.Fprintf(cli.stdout(), "  Created: %s\n", metadata["dateOfCreation"])

	return nil
}
