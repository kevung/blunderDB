package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/kevung/blunderdb/internal/applog"
)

// Log-level scale for the whole backend, chosen by who needs to see it:
//
//   - Error: the operation failed and the log is the only place the user can
//     see it (e.g. a stream broken after headers were sent).
//   - Warn: degraded but recoverable, already visible through a normal
//     channel (a count, an error value, a report).
//   - Info: routine events a support request needs (migration steps, import
//     summaries); shown by default in every mode.
//   - Debug: opt-in only (BLUNDERDB_DEBUG=1).

// initLogging sets the default slog logger: stderr in every mode, plus
// applog's rotating file in GUI mode, which may have no terminal.
func initLogging(mode string) {
	// Every mode shares one level policy.
	level := slog.LevelInfo
	if os.Getenv("BLUNDERDB_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	var w io.Writer = os.Stderr
	if mode == "gui" {
		if lw, err := applog.Open(); err != nil {
			// No logger yet: stderr directly.
			_, _ = os.Stderr.WriteString("blunderdb: could not open the log file, logging to stderr only: " + err.Error() + "\n")
		} else {
			w = io.MultiWriter(os.Stderr, lw)
		}
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
