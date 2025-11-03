package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed build/appicon.png
var icon []byte

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Check if running in CLI mode
	if len(os.Args) > 1 {
		// Check if first argument is a CLI command
		cliCommands := []string{"import", "export", "list", "delete", "help", "version"}
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
	cli := NewCLI()
	args := os.Args[1:]

	if err := cli.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runGUI() {
	app := NewApp()
	db := NewDatabase()
	cfg := NewConfig()

	// Load the configuration file
	config, err := cfg.LoadConfig()
	if err != nil {
		fmt.Println("Error loading configuration file:", err)
		return
	}

	// Set up the in-memory database
	err = db.SetupDatabase(":memory:")
	if err != nil {
		fmt.Println("Error setting up in-memory database:", err)
		return
	}

	// Initialize width and height from config
	initialWidth := config.WindowWidth
	initialHeight := config.WindowHeight
	fmt.Println("Initial dimensions:", initialWidth, "x", initialHeight)
	fmt.Println("Aspect ratio:", float64(initialHeight)/float64(initialWidth))

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "blunderDB",
		Width:  initialWidth,
		Height: initialHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 240, G: 240, B: 240, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
			db,
			cfg,
		},
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyNever,
			ProgramName:         "blunderDB",
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
