package server

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

	st, err := openStorage(ctx, *backend, effDSN)
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
			fmt.Println(strings.TrimPrefix(p, "/v1/"))
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

	req := httptest.NewRequest(http.MethodPost, "/v1/"+method, strings.NewReader(body))
	req.Header.Set(middleware.TenantHeader, *scope)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// The response body (JSON or the error envelope) always goes to stdout so
	// it stays parseable; an error status maps to a non-zero exit code.
	if _, err := io.Copy(os.Stdout, rec.Body); err != nil {
		return err
	}
	if rec.Code >= 400 {
		return fmt.Errorf("call: %s returned HTTP %d", method, rec.Code)
	}
	return nil
}
