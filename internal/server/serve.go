package server

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

const serveUsage = `blunderdb serve — run the engine as an HTTP + JSON daemon.

SECURITY: this daemon performs NO authentication. It trusts the X-Tenant-ID
request header and MUST run behind a reverse-proxy that handles authentication.
Do NOT expose it directly to the public internet.

Usage:
  blunderdb serve [flags]

Flags:
`

// RunServe parses the `serve` subcommand flags, opens the storage backend, and
// runs the server until SIGINT/SIGTERM. args are the arguments after "serve".
func RunServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, serveUsage)
		fs.PrintDefaults()
	}

	var (
		backend        = fs.String("backend", envOr("BLUNDERDB_BACKEND", "sqlite"), "storage backend: sqlite|postgres")
		dsn            = fs.String("dsn", os.Getenv("BLUNDERDB_DSN"), "backend connection string (sqlite path or postgres DSN)")
		dbPath         = fs.String("db", "", "sqlite database file (shorthand for --backend sqlite --dsn <path>)")
		addr           = fs.String("addr", envOr("BLUNDERDB_ADDR", ":8080"), "listen address host:port")
		logLevel       = fs.String("log-level", envOr("BLUNDERDB_LOG_LEVEL", "info"), "log level: debug|info|warn|error")
		enableMetrics  = fs.Bool("metrics", true, "expose /metrics (Prometheus)")
		corsOrigin     = fs.String("cors-allow-origin", "", "enable CORS for this origin (off by default)")
		rateLimitRPS   = fs.Float64("rate-limit-rps", 0, "per-tenant sustained requests/second (0 = disabled)")
		rateLimitBurst = fs.Int("rate-limit-burst", 0, "per-tenant token-bucket burst (default 2×rps)")
		enableRLS      = fs.Bool("rls", envOr("BLUNDERDB_RLS", "") == "true", "PostgreSQL Row-Level Security: install tenant policies and set app.tenant_id per connection (opt-in defence-in-depth; off by default)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dbPath != "" {
		*backend = "sqlite"
		*dsn = *dbPath
	}

	logger := newLogger(*logLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := openStorage(ctx, *backend, *dsn, *enableRLS)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		return fmt.Errorf("serve: migrate: %w", err)
	}

	// Install RLS policies after the schema is in place (opt-in). The pool was
	// already configured to set app.tenant_id per connection by openStorage.
	if *enableRLS {
		applier, ok := st.(interface {
			ApplyRLS(context.Context) error
		})
		if !ok {
			return fmt.Errorf("serve: --rls is only supported by the postgres backend")
		}
		if err := applier.ApplyRLS(ctx); err != nil {
			return fmt.Errorf("serve: apply RLS: %w", err)
		}
		logger.Info("Row-Level Security enabled (per-tenant policies installed)")
	}

	srv, err := New(Options{
		Addr:            *addr,
		Storage:         st,
		Logger:          logger,
		Metrics:         metrics.New(),
		EnableMetrics:   *enableMetrics,
		CORSAllowOrigin: *corsOrigin,
		RateLimitRPS:    *rateLimitRPS,
		RateLimitBurst:  *rateLimitBurst,
	})
	if err != nil {
		return err
	}

	logger.Warn("authentication is delegated to the reverse-proxy; do not expose this daemon to the public internet")
	return srv.Run(ctx)
}

// openStorage opens the requested backend. enableRLS turns on PostgreSQL
// Row-Level Security enforcement (per-connection app.tenant_id); it is ignored
// by the SQLite backend.
func openStorage(ctx context.Context, backend, dsn string, enableRLS bool) (storage.Storage, error) {
	opts := &storage.Options{EnableRLS: enableRLS}
	switch strings.ToLower(backend) {
	case "sqlite", "":
		if dsn == "" {
			return nil, fmt.Errorf("serve: sqlite backend requires --db or --dsn (path to the .db file)")
		}
		return sqlite.Open(ctx, dsn, opts)
	case "postgres", "postgresql", "pg":
		if dsn == "" {
			return nil, fmt.Errorf("serve: postgres backend requires --dsn (or BLUNDERDB_DSN)")
		}
		return postgres.Open(ctx, dsn, opts)
	default:
		return nil, fmt.Errorf("serve: unknown backend %q (want sqlite|postgres)", backend)
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
