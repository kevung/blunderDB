package gui

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/internal/applog"
)

// recoverBackground turns a panic on a background goroutine into a logged
// error and a native dialog instead of a crash; Wails recovers only panics
// inside the bound call itself. Defer it first in the goroutine. With a nil
// ctx only the log line fires.
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
