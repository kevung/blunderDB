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

	/* 	db, errDb := backend.SetupDatabase()
	   	if errDb != nil {
	   		log.Fatal(errDb)
	   	} */

	// Create an instance of the app structure
	app := NewApp()
	db := NewDatabase()

	// errDb := db.SetupDatabase("blunderdb.db")
	// if errDb != nil {
	// 	log.Fatal(errDb)
	// }

	// var position Position = InitializePosition()

	// db.SavePosition(position)

	// position2, _ := db.LoadPosition(1)
	// fmt.Printf("%+v\n", position2)

	// if position == *position2 {
	// 	fmt.Println("The game states are equal.")
	// } else {
	// 	fmt.Println("The game states are not equal.")
	// }

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
