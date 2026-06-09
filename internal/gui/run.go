package gui

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Run starts the Wails GUI. The caller supplies the embedded frontend assets
// and icon, the initial window dimensions, and any extra structs to bind for
// the frontend (the App struct is created and bound here).
func Run(assets embed.FS, icon []byte, width, height int, extraBinds []interface{}) error {
	app := NewApp()
	return wails.Run(&options.App{
		Title:  "blunderDB",
		Width:  width,
		Height: height,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 240, G: 240, B: 240, A: 1},
		OnStartup:        app.startup,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false, // Must be false on Linux: gtk_drag_dest_unset() prevents GTK drag signals from firing (Wails v2 bug #4743)
		},
		Bind: append([]interface{}{app}, extraBinds...),
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyNever,
			ProgramName:         "blunderDB",
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			IsZoomControlEnabled: true,
			ZoomFactor:           1.0,
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
	})
}
