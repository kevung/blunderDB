package cli

import (
	"context"
	"errors"
	"testing"
)

// withInterruptibleContext (cli.go) is the shared Ctrl-C plumbing search,
// list --type stats and export now use (B.13, #181) to cancel a long Database
// call in flight, mirroring the pattern cli_analyze.go established for
// `analyze`. Actually delivering os.Interrupt to the process from a test is
// avoided here: it is not portably reliable across this project's CI matrix
// (windows-latest included — Process.Signal(os.Interrupt) there depends on
// console process-group plumbing a test runner does not provide). What is
// safe and worth locking down is the part unique to this helper: fn's return
// value passes through untouched, and onInterrupt never fires, when no
// signal arrives — the two ways a careless refactor of the shared helper
// could silently break every caller at once.
func TestWithInterruptibleContextPassesThroughWithoutSignal(t *testing.T) {
	called := false
	sentinel := errors.New("sentinel")

	err := withInterruptibleContext(func() {
		called = true
	}, func(ctx context.Context) error {
		if ctx.Err() != nil {
			t.Fatalf("ctx already cancelled before fn ran: %v", ctx.Err())
		}
		return sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want sentinel", err)
	}
	if called {
		t.Error("onInterrupt was called, but no signal was sent")
	}
}

func TestWithInterruptibleContextNilOnInterrupt(t *testing.T) {
	// onInterrupt is optional; passing nil must not panic when fn returns
	// normally (the only path this test can exercise without sending a
	// signal).
	err := withInterruptibleContext(nil, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}
