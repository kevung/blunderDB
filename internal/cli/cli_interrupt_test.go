package cli

import (
	"context"
	"errors"
	"testing"
)

// withInterruptibleContext (cli.go) is the shared Ctrl-C plumbing that
// cancels a long Database call. Sending os.Interrupt from a test is not
// portable (Windows needs console process groups a runner lacks), so this
// locks the no-signal path: fn's return passes through and onInterrupt never
// fires.
func TestWithInterruptibleContextPassesThroughWithoutSignal(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	// onInterrupt is optional; nil must not panic when fn returns normally.
	err := withInterruptibleContext(nil, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}
