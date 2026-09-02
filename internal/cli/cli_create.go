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
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)

	// Define flags
	dbPath := createCmd.String("db", "", "Path to the database file to create (required)")
	force := createCmd.Bool("force", false, "Overwrite existing database if it exists")
	user := createCmd.String("user", "", "User name (owner of the database)")
	description := createCmd.String("description", "", "Description of the database")

	createCmd.Usage = func() {
		fmt.Println("Usage: blunderdb create [options]")
		fmt.Println()
		fmt.Println("Create a new database with the required schema and optional metadata.")
		fmt.Println()
		fmt.Println("Options:")
		createCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Create a new database")
		fmt.Println("  blunderdb create --db mydb.db")
		fmt.Println()
		fmt.Println("  # Create with metadata")
		fmt.Println("  blunderdb create --db mydb.db --user \"John Doe\" --description \"My backgammon positions\"")
		fmt.Println()
		fmt.Println("  # Force overwrite an existing database")
		fmt.Println("  blunderdb create --db mydb.db --force")
	}

	if err := createCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		createCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Ensure .db extension is present
	if !strings.HasSuffix(strings.ToLower(*dbPath), ".db") {
		*dbPath += ".db"
	}

	// Check if database already exists
	existing := true
	if _, err := os.Stat(*dbPath); err != nil {
		existing = false
	} else if !*force {
		return fmt.Errorf("database already exists: %s (use --force to overwrite)", *dbPath)
	}

	// Create directory if needed
	dir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// --force overwrites whatever is at dbPath, not just a database SetupDatabase
	// can open and erase in place: the point of --force is to replace ANY
	// existing file there, including a leftover/corrupt one that fails
	// SQLite's own header check ("file is not a database"). Removing it first
	// guarantees SetupDatabase always starts from nothing rather than an
	// unopenable file it has no chance of erasing content-wise.
	if existing {
		if err := os.Remove(*dbPath); err != nil {
			return fmt.Errorf("failed to remove existing database: %w", err)
		}
	}

	// Create the database
	fmt.Printf("Creating database: %s\n", *dbPath)
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

	fmt.Printf("Successfully created database with schema version %s\n", DatabaseVersion)

	// Show database info
	fmt.Println("\nDatabase Information:")
	fmt.Printf("  Version: %s\n", DatabaseVersion)
	if *user != "" {
		fmt.Printf("  User: %s\n", *user)
	}
	if *description != "" {
		fmt.Printf("  Description: %s\n", *description)
	}
	fmt.Printf("  Created: %s\n", metadata["dateOfCreation"])

	return nil
}
