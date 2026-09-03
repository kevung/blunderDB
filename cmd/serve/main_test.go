package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStderr redirects os.Stderr for the duration of fn and returns what
// was written to it. Mirrors internal/cli's captureStdout (cli_test.go).
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	outC := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	fn()

	w.Close()
	os.Stderr = old
	return <-outC
}

// TestRunDispatchesOnFirstArgument is this package's one wiring test (#220):
// run is the only decision cmd/serve makes on its own — everything else is
// forwarded verbatim to internal/server — and it was previously untested, so
// a swapped branch or a typo'd EqualFold would only surface at release time.
//
// "healthcheck" (any case) must reach server.RunHealthcheck; anything else
// must reach server.RunServe. Neither call is allowed to touch a real daemon
// or storage backend, so both are probed through paths that return before any
// side effect: RunHealthcheck against a port nothing listens on, with a short
// timeout (its error is wrapped "healthcheck: …" — proof of which function
// ran); RunServe with -h, which flag.ContinueOnError intercepts before
// OpenStorage or ListenAndServe — proof from its own usage banner, captured
// off stderr, which names "serve" and never "healthcheck".
func TestRunDispatchesOnFirstArgument(t *testing.T) {
	t.Run("healthcheck (any case) reaches RunHealthcheck", func(t *testing.T) {
		for _, word := range []string{"healthcheck", "HealthCheck", "HEALTHCHECK"} {
			err := run([]string{word, "--addr", "127.0.0.1:1", "--timeout", "50ms"})
			if err == nil {
				t.Fatalf("run(%q, ...): want an error (nothing listens on port 1), got nil", word)
			}
			if !strings.Contains(err.Error(), "healthcheck") {
				t.Errorf("run(%q, ...): error %q does not name healthcheck — did dispatch reach RunServe instead?", word, err.Error())
			}
		}
	})

	t.Run("anything else reaches RunServe", func(t *testing.T) {
		for _, args := range [][]string{{"-h"}, {"--help"}} {
			var err error
			stderr := captureStderr(t, func() {
				err = run(args)
			})
			if err == nil {
				t.Fatalf("run(%v): want an error (-h prints usage and returns flag.ErrHelp), got nil", args)
			}
			if !strings.Contains(stderr, "blunderdb serve") {
				t.Errorf("run(%v): stderr %q does not carry serve's usage banner — did dispatch reach RunHealthcheck instead?", args, stderr)
			}
			if strings.Contains(stderr, "blunderdb healthcheck") {
				t.Errorf("run(%v): stderr %q carries healthcheck's usage banner — dispatch reached the wrong function", args, stderr)
			}
		}
	})
}
