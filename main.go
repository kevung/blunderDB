package main

import (
	"embed"

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

	// Calculate the initial height based on the aspect factor
	initialWidth := 960
	aspectFactor := 0.7815
	initialHeight := int(float64(initialWidth) * aspectFactor) // Adjust to have equal space above and below

	// Create application with options
	err := wails.Run(&options.App{
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
}
