package main

import (
	"embed"
    "log"
    "fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

    "blunderdb/backend"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

    db, errDb := backend.SetupDatabase()
    if errDb != nil {
        log.Fatal(errDb)
    }

    var position backend.Position
    position = backend.InitializePosition()

    backend.SavePosition(db, position)

    position2, _ := backend.LoadPosition(db, 1)
    fmt.Printf("%+v\n", position2)

    if position == *position2 {
        fmt.Println("The game states are equal.")
    } else {
        fmt.Println("The game states are not equal.")
    }

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "blunderDB",
		Width:  1024,
		Height: 728,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
