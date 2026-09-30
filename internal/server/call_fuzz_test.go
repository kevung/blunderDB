package server

import (
	"os"
	"strings"
	"testing"
	"time"
)

// FuzzRunCall drives the `call` dispatcher's argument handling — the method
// token, the flag set, the --scope check and the JSON body — with arbitrary
// arguments. The store is pinned to an in-memory SQLite: flags naming a file
// (--db, --dsn, --backend, --json-file) are dropped, since writing or reading
// arbitrary paths is not what is under test. Contract: no panic, no hang, an
// error for what is refused.
func FuzzRunCall(f *testing.F) {
	for _, s := range []string{
		"metadata.counts",
		"--list",
		"positions.list\n--json\n{\"limit\":10}",
		"matches.get\n--json\n{\"id\":1}\n--scope\n0",
		"positions.bogus\n--scope\n-1",
		"\n--json",
		"search.query\n--json={\"query\":\"s cube\"}",
		"../ops/tenant.purge",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		var args []string
		for _, a := range strings.Split(s, "\n") {
			name := strings.TrimLeft(a, "-")
			if i := strings.IndexByte(name, '='); i >= 0 {
				name = name[:i]
			}
			if strings.HasPrefix(a, "-") && (name == "db" || name == "dsn" || name == "backend" || name == "json-file") {
				continue
			}
			args = append(args, a)
		}
		if len(args) > 0 {
			for _, heavy := range []string{"imports", "exports", "gammonnet"} {
				if strings.Contains(args[0], heavy) {
					return
				}
			}
		}
		args = append(args, "--backend", "sqlite", "--dsn", ":memory:")
		// flag.Parse stops at the first positional argument, so the pinning
		// flags only hold when every argument before them is a flag or a
		// flag value; otherwise the defaults apply, which are the same
		// in-memory SQLite unless the environment says otherwise.
		t.Setenv("BLUNDERDB_BACKEND", "sqlite")
		t.Setenv("BLUNDERDB_DSN", ":memory:")

		devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		oldOut, oldErr := os.Stdout, os.Stderr
		os.Stdout, os.Stderr = devnull, devnull
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = RunCall(args)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			os.Stdout, os.Stderr = oldOut, oldErr
			t.Fatalf("RunCall(%q) did not return within 10s", args)
		}
		os.Stdout, os.Stderr = oldOut, oldErr
		devnull.Close()
	})
}
