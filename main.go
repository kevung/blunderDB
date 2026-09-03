// Command blunderdb is the single binary for every mode this project ships:
// with no arguments it launches the Wails desktop GUI (internal/gui); its
// first argument otherwise selects `serve` (the headless HTTP+JSON daemon,
// internal/server), `migrate` (copy a SQLite database into PostgreSQL under
// a tenant, pkg/blunderdb/migrate), or a CLI subcommand (internal/cli,
// dispatched through cli.IsCommand against the single handlers() table it
// owns — never duplicate that list here). The GUI build embeds
// frontend/dist (go:embed all:frontend/dist) and the app icon; cmd/serve is
// the sibling entrypoint that builds the daemon alone, without Wails or the
// embedded frontend, for the container image.
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
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
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
		// Check if first argument is a CLI command. The list of names lives in
		// internal/cli next to the handlers, so a new subcommand cannot be
		// wired there and forgotten here.
		if cli.IsCommand(os.Args[1]) {
			runCLI()
			return
		}
		// Anything else positional is a database file path the OS handed this
		// process — build/linux/blunderdb.desktop's `Exec=blunderDB %f`, or a
		// Windows/macOS file-association double-click (#241) — not a flag
		// (which would start with "-"): GUI mode, opening that file. A
		// leading "-" is left alone rather than treated as a startup path, so
		// an unrecognised flag reports itself sanely instead of being handed
		// to the GUI as if it were a database.
		if !strings.HasPrefix(os.Args[1], "-") {
			runGUI(os.Args[1])
			return
		}
	}

	// Run GUI mode
	runGUI("")
}

func runCLI() {
	initLogging("cli")
	c := cli.NewCLI()
	args := os.Args[1:]

	if err := c.Run(args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runServe() {
	initLogging("serve")
	if err := server.RunServe(os.Args[2:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCall() {
	initLogging("cli")
	if err := server.RunCall(os.Args[2:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMigrate() {
	initLogging("cli")
	if err := migrate.RunCLI(os.Args[2:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runGUI(startupFilePath string) {
	initLogging("gui")
	db := database.NewDatabase()
	cfg := NewConfig()

	// Load the configuration file
	config, err := cfg.LoadConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error loading configuration file:", err)
		os.Exit(1)
	}

	// Apply the persisted two-sided bearoff path to the engine (ADR-0009).
	if p := config.GetBearoffTSPath(); p != "" {
		race.SetExternalPath(p)
	}

	// Set up the in-memory database
	if err := db.SetupDatabase(":memory:"); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error setting up in-memory database:", err)
		os.Exit(1)
	}

	// Bind the database and config alongside the GUI App struct.
	if err := gui.Run(assets, icon, config.WindowWidth, config.WindowHeight, db, []interface{}{db, cfg}, startupFilePath); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
