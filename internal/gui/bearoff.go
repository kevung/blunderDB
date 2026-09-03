package gui

import (
	"context"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// Optional download of the TS-06-11 two-sided bearoff database (ADR-0009).
// The file is an immutable release asset under the dedicated tag
// bearoff-data-1; it lands in $XDG_DATA_HOME/blunderdb (never the cache dir)
// and is verified against race.DownloadedSHA256 before being moved in place.
// Interrupted or cancelled downloads keep their .part file and RESUME via
// HTTP Range on the next attempt (see resumableDownload). Progress reaches
// the frontend through Wails events:
//
//	bearoff:progress {received, total}   (throttled)
//	bearoff:done     {}
//	bearoff:error    {message}

const bearoffDownloadURL = "https://github.com/kevung/blunderDB/releases/download/bearoff-data-1/" + race.DownloadedFileName

// BearoffStatus is what the config dialog shows.
type BearoffStatus struct {
	Downloaded    bool   `json:"downloaded"`     // TS-06-11 present in the data dir
	Downloading   bool   `json:"downloading"`    // a download is in flight
	Path          string `json:"path"`           // where the download lives/would live
	SizeBytes     int64  `json:"size_bytes"`     // size of the downloaded file (0 if absent)
	ActiveDomain  int    `json:"active_domain"`  // checkers of the currently resolved source
	ActiveOrigin  string `json:"active_origin"`  // origin of the currently resolved source
	ExternalPath  string `json:"external_path"`  // user-configured .bd (may be "")
	ExpectedBytes int64  `json:"expected_bytes"` // published asset size
	// PartialBytes is the size of an interrupted download (.part) that the
	// next attempt will resume from (0 if none).
	PartialBytes int64 `json:"partial_bytes"`
}

const bearoffExpectedBytes = 1_225_323_048

var (
	bearoffMu     sync.Mutex
	bearoffCancel context.CancelFunc
)

// BearoffStatus reports the state of the two-sided bearoff sources.
func (a *App) BearoffStatus() BearoffStatus {
	st := BearoffStatus{
		Path:          race.DownloadedPath(),
		ExpectedBytes: bearoffExpectedBytes,
	}
	if fi, err := os.Stat(st.Path); err == nil {
		st.Downloaded = true
		st.SizeBytes = fi.Size()
	}
	if fi, err := os.Stat(st.Path + ".part"); err == nil {
		st.PartialBytes = fi.Size()
	}
	bearoffMu.Lock()
	st.Downloading = bearoffCancel != nil
	bearoffMu.Unlock()
	src := race.Resolve()
	st.ActiveDomain = src.Checkers()
	st.ActiveOrigin = src.Origin()
	return st
}

// OpenBearoffFileDialog lets the user pick an external .bd database.
func (a *App) OpenBearoffFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select two-sided bearoff database",
		Filters: []runtime.FileFilter{
			{DisplayName: "gnubg bearoff database (*.bd)", Pattern: "*.bd"},
		},
	})
}

// DownloadBearoffDB starts the TS-06-11 download in the background. Progress
// is emitted as Wails events; a second call while one is in flight is a no-op.
func (a *App) DownloadBearoffDB() error {
	bearoffMu.Lock()
	if bearoffCancel != nil {
		bearoffMu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(a.ctx)
	bearoffCancel = cancel
	bearoffMu.Unlock()

	go func() {
		defer recoverBackground(a.ctx, "bearoff download")
		err := a.downloadBearoff(ctx)
		bearoffMu.Lock()
		bearoffCancel = nil
		bearoffMu.Unlock()
		if err != nil {
			if ctx.Err() == nil { // real failure, not a user cancel
				runtime.EventsEmit(a.ctx, "bearoff:error", map[string]string{"message": err.Error()})
			}
			return
		}
		runtime.EventsEmit(a.ctx, "bearoff:done")
	}()
	return nil
}

// CancelBearoffDownload aborts an in-flight download (partial file removed).
func (a *App) CancelBearoffDownload() {
	bearoffMu.Lock()
	if bearoffCancel != nil {
		bearoffCancel()
	}
	bearoffMu.Unlock()
}

// DeleteBearoffDB removes the downloaded database (the embedded TS-06-06 and
// any external file keep working; nothing else is touched).
func (a *App) DeleteBearoffDB() error {
	p := race.DownloadedPath()
	race.Invalidate() // release the cached handle first: Windows cannot remove an open file
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(p + ".part"); err != nil && !os.IsNotExist(err) {
		return err
	}
	race.Resolve() // re-resolve so the panel downgrades immediately
	return nil
}

func (a *App) downloadBearoff(ctx context.Context) error {
	err := resumableDownload(ctx, bearoffDownloadURL, race.DownloadedPath(),
		race.DownloadedSHA256, bearoffExpectedBytes,
		func(received, total int64) {
			runtime.EventsEmit(a.ctx, "bearoff:progress",
				map[string]int64{"received": received, "total": total})
		})
	if err != nil {
		return err
	}
	race.Resolve() // pick the new database up immediately
	return nil
}
