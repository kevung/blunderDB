package main

import (
	"log/slog"
	"os"
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
func initLogging(mode string) {
	// mode is no longer consulted: every mode shares one level policy now
	// (see the scale above). Kept as a parameter for call-site symmetry in
	// main.go and because a future mode may yet need to diverge.
	level := slog.LevelInfo
	if os.Getenv("BLUNDERDB_DEBUG") == "1" {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
