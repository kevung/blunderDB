package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"
)

// TestStartPprofServer_ServesAndStopsIndependently guards #238: --pprof-addr
// runs on its own listener, separate from the domain server, and its stop
// func actually shuts it down (a caller must be able to tear it down inside
// a test, or as part of ctx cancellation, without leaking a goroutine or a
// listening socket).
func TestStartPprofServer_ServesAndStopsIndependently(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // free it right back up for startPprofServer to bind

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	stop := startPprofServer(ctx, logger, addr)
	defer stop()

	url := "http://" + addr + "/debug/pprof/"
	var resp *http.Response
	for i := 0; i < 50; i++ {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	stop()

	if resp, err := http.Get(url); err == nil {
		resp.Body.Close()
		t.Error("expected the pprof listener to be down after stop(), got a successful response")
	}
}

// TestStartPprofServer_StopsOnContextCancel: RunServe relies on ctx
// cancellation (not just the returned stop func) to bring the pprof
// listener down alongside the rest of the daemon on SIGINT/SIGTERM.
func TestStartPprofServer_StopsOnContextCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	startPprofServer(ctx, logger, addr)

	url := "http://" + addr + "/debug/pprof/"
	var resp *http.Response
	for i := 0; i < 50; i++ {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	resp.Body.Close()

	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			return // down, as expected
		}
		resp.Body.Close()
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("pprof listener still answering 2s after context cancellation")
}
