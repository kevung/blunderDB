package server

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/kevung/blunderdb/internal/server/middleware"
)

const callUsage = `blunderdb call <family>.<method> — invoke a Storage method directly.

The CLI dispatches in-process through the exact same handlers the HTTP daemon
serves, so behaviour is identical to POST /v1/<family>.<method>. The JSON
response (or NDJSON stream for list endpoints) is written to stdout; the process
exits non-zero on an error response (the {"error":{…}} envelope still prints).

Usage:
  blunderdb call <family>.<method> [flags]
  blunderdb call --list                       # show every available method

Examples:
  blunderdb call metadata.counts --db my.db
  blunderdb call positions.list --db my.db --json '{"limit":10}'
  blunderdb call matches.get --db my.db --json '{"id":1}'

Flags:
`

// RunCall implements the `call` subcommand: a generic dispatcher over the
// server's domain routes, sharing the exact handler functions (CLI/HTTP
// parity). args are the arguments after "call".
func RunCall(args []string) error {
	// The method (first non-flag token) is separated from the flags so that
	// `call positions.list --json …` parses regardless of flag/arg order.
	var method string
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		method = args[0]
		rest = args[1:]
	}

	fs := flag.NewFlagSet("call", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, callUsage)
		fs.PrintDefaults()
	}
	var (
		backend  = fs.String("backend", envOr("BLUNDERDB_BACKEND", "sqlite"), "storage backend: sqlite|postgres")
		dsn      = fs.String("dsn", os.Getenv("BLUNDERDB_DSN"), "backend connection string (sqlite path or postgres DSN)")
		dbPath   = fs.String("db", "", "sqlite database file (shorthand for --backend sqlite --dsn <path>)")
		scope    = fs.String("scope", "default", "tenant scope (sent as X-Tenant-ID; SQLite ignores it for most families)")
		jsonBody = fs.String("json", "{}", "request body as JSON")
		jsonFile = fs.String("json-file", "", "read the request body from a file instead of --json")
		list     = fs.Bool("list", false, "list every available <family>.<method> and exit")
	)
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *dbPath != "" {
		*backend = "sqlite"
		*dsn = *dbPath
	}

	ctx := context.Background()

	// --list only needs the route table; an in-memory SQLite suffices when no
	// database was given.
	effDSN := *dsn
	if *list && effDSN == "" {
		*backend = "sqlite"
		effDSN = ":memory:"
	}

	st, err := OpenStorage(ctx, *backend, effDSN, false)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return fmt.Errorf("call: migrate: %w", err)
	}

	srv, err := New(Options{Storage: st})
	if err != nil {
		return err
	}

	if *list {
		for _, p := range srv.Paths() {
			_, _ = fmt.Println(strings.TrimPrefix(p, "/v1/"))
		}
		return nil
	}

	if method == "" {
		fs.Usage()
		return fmt.Errorf("call: missing <family>.<method> (try `call --list`)")
	}

	body := *jsonBody
	if *jsonFile != "" {
		b, err := os.ReadFile(*jsonFile)
		if err != nil {
			return fmt.Errorf("call: read --json-file: %w", err)
		}
		body = string(b)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/v1/"+method, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("call: build request: %w", err)
	}
	req.Header.Set(middleware.TenantHeader, *scope)

	// A minimal ResponseWriter that streams straight to stdout, rather than
	// httptest.NewRecorder buffering the entire response in memory before a
	// single copy at the end: list-style routes and exports stream NDJSON
	// through http.Flusher (see ndjson.go, handlers_imports.go), and a large
	// export or a long-running list previously had to finish (and sit fully
	// in RAM) before the caller saw a single byte.
	w := newStdoutResponseWriter()
	srv.Handler().ServeHTTP(w, req)

	// The response body (JSON or the error envelope) always goes to stdout so
	// it stays parseable; an error status maps to a non-zero exit code.
	if w.status >= 400 {
		return fmt.Errorf("call: %s returned HTTP %d", method, w.status)
	}
	return nil
}

// stdoutResponseWriter is a minimal http.ResponseWriter/http.Flusher that
// writes response bytes straight to os.Stdout as the handler produces them,
// tracking only the status code (needed to decide the process exit code) —
// everything else in the response the handler wrote is already on stdout by
// the time ServeHTTP returns, so there is nothing left to copy.
type stdoutResponseWriter struct {
	header    http.Header
	status    int
	wroteHead bool
}

func newStdoutResponseWriter() *stdoutResponseWriter {
	return &stdoutResponseWriter{header: make(http.Header)}
}

func (w *stdoutResponseWriter) Header() http.Header { return w.header }

func (w *stdoutResponseWriter) WriteHeader(status int) {
	if w.wroteHead {
		return
	}
	w.status = status
	w.wroteHead = true
}

func (w *stdoutResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHead {
		w.WriteHeader(http.StatusOK) // implicit 200, matching net/http's own ResponseWriter
	}
	return os.Stdout.Write(b)
}

// Flush satisfies http.Flusher. Every Write already lands on os.Stdout
// immediately (no intermediate buffer in this type), so there is nothing to
// flush; it exists so handlers that type-assert w.(http.Flusher) before
// streaming NDJSON (ndjson.go, handlers_imports.go) find one, same as
// httptest.ResponseRecorder did.
func (w *stdoutResponseWriter) Flush() {}
