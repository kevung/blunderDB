package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/internal/cli"
	"github.com/kevung/blunderdb/internal/gui"
	"github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/migrate"
)

//go:embed build/appicon.png
var icon []byte

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Check if running in CLI mode
	if len(os.Args) > 1 {
		// `serve` runs the HTTP + JSON daemon (its own arg parsing).
		if strings.ToLower(os.Args[1]) == "serve" {
			runServe()
			return
		}
		// `call` invokes a Storage method via the same handlers, in-process.
		if strings.ToLower(os.Args[1]) == "call" {
			runCall()
			return
		}
		// `migrate` copies a SQLite database into PostgreSQL under a tenant.
		if strings.ToLower(os.Args[1]) == "migrate" {
			runMigrate()
			return
		}
		// Check if first argument is a CLI command
		cliCommands := []string{"create", "import", "export", "list", "match", "verify", "delete", "help", "version", "info", "edit", "search"}
		for _, cmd := range cliCommands {
			if strings.ToLower(os.Args[1]) == cmd {
				runCLI()
				return
			}
		}
	}

	// Run GUI mode
	runGUI()
}

func runCLI() {
	initLogging("cli")
	c := cli.NewCLI()
	args := os.Args[1:]

	if err := c.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runServe() {
	initLogging("serve")
	if err := server.RunServe(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCall() {
	initLogging("cli")
	if err := server.RunCall(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMigrate() {
	initLogging("cli")
	if err := migrate.RunCLI(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runGUI() {
	initLogging("gui")
	db := database.NewDatabase()
	cfg := NewConfig()

	// Load the configuration file
	config, err := cfg.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading configuration file:", err)
		os.Exit(1)
	}

	// Set up the in-memory database
	if err := db.SetupDatabase(":memory:"); err != nil {
		fmt.Fprintln(os.Stderr, "Error setting up in-memory database:", err)
		os.Exit(1)
	}

	// Bind the database and config alongside the GUI App struct.
	if err := gui.Run(assets, icon, config.WindowWidth, config.WindowHeight, []interface{}{db, cfg}); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
