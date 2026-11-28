package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/progress"
)

// withTerminalProgress makes ctx's long operation report on stderr, one line
// rewritten in place, when stderr is a terminal: a pipe or a log file gets
// nothing, and stdout (the command's result, JSON included) is never
// touched. The returned function ends the line.
func withTerminalProgress(ctx context.Context, label string) (context.Context, func()) {
	fi, err := os.Stderr.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return ctx, func() {}
	}
	wrote := false
	report := progress.Throttle(progress.Gap, func(done, total int) {
		wrote = true
		if total > 0 {
			fmt.Fprintf(os.Stderr, "\r%s %d/%d", label, done, total)
		} else {
			fmt.Fprintf(os.Stderr, "\r%s %d", label, done)
		}
	})
	return progress.With(ctx, report), func() {
		if wrote {
			fmt.Fprintln(os.Stderr)
		}
	}
}
