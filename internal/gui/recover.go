package gui

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/internal/applog"
)

// recoverBackground turns a panic on a detached background goroutine (a
// bearoff download, a gammonNet batch or at-rest evaluation run — see
// bearoff.go, gammonnet_batch.go, gammonnet_eval.go) into a logged error and
// a native dialog, instead of letting the panic escape the goroutine and
// crash the whole desktop process (#241).
//
// Wails' own dispatcher already recovers a panic raised directly inside a
// bound method call (its ProcessMessage turns it into a rejected JS promise
// instead of a process crash) — this covers the one class of panic that
// mechanism can never see, because it happens on a goroutine the original
// request handler has already returned from, with no pending promise left
// to reject.
//
// Call it deferred, at the very top of the goroutine: `defer
// recoverBackground(a.ctx, "bearoff download")`. ctx may be nil (a unit test
// constructing an App without startup()); the dialog is then simply skipped
// and only the log line fires.
func recoverBackground(ctx context.Context, label string) {
	r := recover()
	if r == nil {
		return
	}
	slog.Error("panic recovered in background job",
		"job", label,
		"panic", r,
		"stack", string(debug.Stack()),
	)
	if ctx == nil {
		return
	}
	msg := fmt.Sprintf("%s ran into an unexpected internal error and stopped.\n\nDetails were written to the log file:\n%s\n\n%v", label, applog.Path(), r)
	go func() {
		_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Unexpected error",
			Message: msg,
		})
	}()
}
