package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/kevung/blunderdb/internal/applog"
)

// Log-level scale used across the whole backend (database, storage, ingest,
// CLI, server, GUI bootstrap): the level is chosen by asking who needs to see
// the message, not by how urgent the call site feels.
//
//   - Error: the user must see it. The operation they asked for did not
//     produce what they expect, and nothing else already carries that signal
//     to them (a returned error shown in a dialog or printed by the CLI, a
//     report field) — the log is the only place it becomes visible. A
//     streamed HTTP response breaking after headers are already committed is
//     the paradigm case: the client just sees a truncated file, and the log
//     is the only record of why.
//   - Warn: degraded but recoverable. The operation continues (a row is
//     skipped and counted, a fallback estimate is used, an optional index or
//     default could not be added) and the outcome is already visible to the
//     caller through a normal channel — a returned count, an error value, a
//     report struct — so the log is a diagnostic aid, not the only signal.
//   - Info: routine, expected events worth a trace in normal operation
//     (schema migration steps, import summaries). Shown by default in every
//     mode, GUI included — these are exactly the messages a support request
//     needs, and a raised GUI default made them effectively unreachable.
//   - Debug: verbose detail, opt-in only (BLUNDERDB_DEBUG=1).

// initLogging sets the process-wide default slog logger. Every mode logs to
// stderr; GUI mode additionally writes to applog's rotating file
// ($XDG_STATE_HOME/blunderDB/blunderdb.log, see internal/applog), since
// stderr is invisible once the app is launched by a double-click or a
// desktop-file entry with no attached terminal (#241) — `serve`/CLI/`call`/
// `migrate` keep their existing stderr-only behaviour, since they already
// run attached to whatever launched them.
func initLogging(mode string) {
	// mode is no longer consulted: every mode shares one level policy now
	// (see the scale above). Kept as a parameter for call-site symmetry in
	// main.go and because a future mode may yet need to diverge.
	level := slog.LevelInfo
	if os.Getenv("BLUNDERDB_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	var w io.Writer = os.Stderr
	if mode == "gui" {
		if lw, err := applog.Open(); err != nil {
			// No default logger is set up yet at this point, so this can
			// only reach the user via stderr directly.
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
