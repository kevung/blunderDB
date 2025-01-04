package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	// Create an instance of the app structure
	app := NewApp()
	db := NewDatabase()

	// Create a temporary database file
	tempDir := os.TempDir()
	tempDBPath := filepath.Join(tempDir, "temp_blunderDB.db")
	file, err := os.Create(tempDBPath)
	if err != nil {
		fmt.Println("Error creating temporary database file:", err)
		return
	}
	file.Close()

	// Set up the temporary database
	err = db.SetupDatabase(tempDBPath)
	if err != nil {
		fmt.Println("Error setting up temporary database:", err)
		return
	}

	// Open the temporary database
	err = db.OpenDatabase(tempDBPath)
	if err != nil {
		fmt.Println("Error opening temporary database:", err)
		return
	}

	// Calculate the initial height based on the aspect factor
	initialWidth := 960
	aspectFactor := 0.7815
	initialHeight := int(float64(initialWidth) * aspectFactor) // Adjust to have equal space above and below

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
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}

	// Clean up the temporary database file
	os.Remove(tempDBPath)
}
