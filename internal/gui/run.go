package gui

import (
	"context"
	"embed"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// Run starts the Wails GUI, binding a new App plus extraBinds. startupFilePath
// is the database the OS passed on the command line, "" if none.
func Run(assets embed.FS, icon []byte, width, height int, db *database.Database, extraBinds []interface{}, startupFilePath string) error {
	app := NewApp(db)
	app.startupFilePath = startupFilePath
	for _, bind := range extraBinds {
		if c, ok := bind.(AssistantConsent); ok {
			app.assistant.consent = c.AssistantConsentURL
		}
	}
	return wails.Run(&options.App{
		Title:  "blunderDB",
		Width:  width,
		Height: height,
		// Below this the toolbar wraps and panels lose their minimum size.
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 240, G: 240, B: 240, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       shutdown(app, extraBinds),
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false, // Must be false on Linux: gtk_drag_dest_unset() prevents GTK drag signals from firing (a Wails v2 bug)
		},
		Bind: append([]interface{}{app}, extraBinds...),
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
			// Without the GPU, WebKitGTK's video sink waits on the main thread to draw while a
			// seek on that thread waits on the sink: the whole window freezes. Wails v2 swaps
			// two values in its C switch: OnDemand is what sets WebKit's ALWAYS, and Always sets
			// ON_DEMAND, deprecated since WebKitGTK 2.54. WEBKIT_DISABLE_DMABUF_RENDERER=1 brings
			// the freeze back: it is no workaround.
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
			ProgramName:      "blunderDB",
		},
		// WebView2 already applies OS DPI scaling: pin its zoom to 1.0 so it
		// does not compound with `--ui-scale`.
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

// shutdownJobGrace bounds the wait for a cancelled job. The gammonNet search
// stops at its next roll, so a batch is gone in milliseconds; past the grace, the close goes ahead: refusing to quit is
// worse than a late write's warning.
const shutdownJobGrace = 5 * time.Second

// shutdown stops the background jobs, then closes the bound resources (the database), in
// that order. Closing lets SQLite remove its `-wal`/`-shm` files and drop the writer lock;
// stopping first keeps the close from racing the batch's writes on the single connection.
func shutdown(app *App, binds []interface{}) func(ctx context.Context) {
	return func(context.Context) {
		app.stopAssistant()
		app.stopMCP()
		app.stopDisplay()
		app.stopMedia()
		app.closing.Store(true)
		app.stopBackgroundJobs(shutdownJobGrace)

		for _, bind := range binds {
			closer, ok := bind.(interface{ Close() error })
			if !ok {
				continue
			}
			if err := closer.Close(); err != nil {
				slog.Warn("closing on shutdown", "err", err)
			}
		}
	}
}
