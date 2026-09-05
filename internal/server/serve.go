package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// Rate limiting defaults (BLUNDERDB_RATE_LIMIT_RPS / -BURST override them):
// generous enough to never bother a real client, but present from the first
// run rather than opt-in — see ADR-0005 and the amendment's threat model:
// a bare compose file that forgets to set these no longer ships with no
// throttling at all.
const (
	defaultRateLimitRPS   = 50
	defaultRateLimitBurst = 100
)

const serveUsage = `blunderdb serve — run the engine as an HTTP + JSON daemon.

SECURITY: this daemon performs NO authentication. It trusts the X-Tenant-ID
request header and MUST run behind a reverse-proxy that handles authentication.
Do NOT expose it directly to the public internet.

X-Tenant-ID is the tenant's positive decimal integer (1, 2, 42, …); the proxy
maps the authenticated account to that integer. A name ("alice") is refused
with 400 invalid — it is never mapped to a tenant by the daemon.

Usage:
  blunderdb serve [flags]

Flags:
`

// serveConfig holds the parsed `serve` flags before any storage is opened.
// Split out of RunServe so parsing — the leading-"serve" tolerance, the
// NArg rejection, the BLUNDERDB_* fallbacks — is unit-testable without
// starting a real daemon.
type serveConfig struct {
	backend        string
	dsn            string
	dbPath         string
	addr           string
	opsAddr        string
	logLevel       string
	enableMetrics  bool
	corsOrigin     string
	rateLimitRPS   float64
	rateLimitBurst int
	enableRLS      bool
	tsPath         string
	identityDir    string
	pprofAddr      string
}

// parseServeArgs parses the `serve` subcommand's flags.
//
// A leading "serve" token is tolerated and stripped: the container image's
// ENTRYPOINT is the bare binary, so `docker run image serve --addr :9090`
// hands us exactly that token as args[0]. Anything else positional is an
// error — flag.FlagSet stops parsing at the first non-flag argument, so a
// mistyped or misplaced flag after one would otherwise be silently ignored
// rather than rejected (#230).
func parseServeArgs(args []string) (*serveConfig, error) {
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, serveUsage)
		fs.PrintDefaults()
	}

	var (
		backend       = fs.String("backend", envOr("BLUNDERDB_BACKEND", "sqlite"), "storage backend: sqlite|postgres")
		dsn           = fs.String("dsn", os.Getenv("BLUNDERDB_DSN"), "backend connection string (sqlite path or postgres DSN)")
		dbPath        = fs.String("db", "", "sqlite database file (shorthand for --backend sqlite --dsn <path>)")
		addr          = fs.String("addr", envOr("BLUNDERDB_ADDR", ":8080"), "listen address host:port")
		logLevel      = fs.String("log-level", envOr("BLUNDERDB_LOG_LEVEL", "info"), "log level: debug|info|warn|error")
		enableMetrics = fs.Bool("metrics", envBoolOr("BLUNDERDB_METRICS", true), "expose /metrics (Prometheus)")
		corsOrigin    = fs.String("cors-allow-origin", envOr("BLUNDERDB_CORS_ALLOW_ORIGIN", ""), "enable CORS for this origin, a comma-separated list of origins, or \"*\" (off by default)")
		rateLimitRPS  = fs.Float64("rate-limit-rps", envFloatOr("BLUNDERDB_RATE_LIMIT_RPS", defaultRateLimitRPS),
			fmt.Sprintf("per-tenant sustained requests/second (0 = disabled; default %d, generous headroom for real traffic)", defaultRateLimitRPS))
		rateLimitBurst = fs.Int("rate-limit-burst", envIntOr("BLUNDERDB_RATE_LIMIT_BURST", defaultRateLimitBurst),
			fmt.Sprintf("per-tenant token-bucket burst (default %d)", defaultRateLimitBurst))
		enableRLS   = fs.Bool("rls", envOr("BLUNDERDB_RLS", "") == "true", "PostgreSQL Row-Level Security: install tenant policies and set app.tenant_id per connection (opt-in defence-in-depth; off by default)")
		tsPath      = fs.String("bearoff-ts", os.Getenv("BLUNDERDB_TS_PATH"), "optional two-sided bearoff database (.bd) widening the embedded TS-06-06; the daemon never downloads one")
		identityDir = fs.String("identity-dir", os.Getenv("BLUNDERDB_IDENTITY_DIR"), "directory holding this daemon's watermark signing identity (created on first use); a watermarked export is refused when unset")
		opsAddr     = fs.String("ops-addr", envOr("BLUNDERDB_OPS_ADDR", ""), "optional listener for the /ops/ family (maintenance.vacuum, tenant.purge) on a SEPARATE address, e.g. \"127.0.0.1:8081\"; empty (the default) serves /ops/ on --addr, where the reverse proxy in front is expected to refuse the prefix (#233)")
		pprofAddr   = fs.String("pprof-addr", envOr("BLUNDERDB_PPROF_ADDR", ""), "optional net/http/pprof listener on a SEPARATE address, e.g. \"127.0.0.1:6060\" (debug only; never expose this on the same address as --addr or to the public internet); empty (the default) exposes no pprof endpoint at all (#238)")
	)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("serve: unexpected argument(s) %v — flags are spelled --name value; check for a typo or a stray positional argument", fs.Args())
	}

	cfg := &serveConfig{
		backend:        *backend,
		dsn:            *dsn,
		dbPath:         *dbPath,
		addr:           *addr,
		opsAddr:        *opsAddr,
		logLevel:       *logLevel,
		enableMetrics:  *enableMetrics,
		corsOrigin:     *corsOrigin,
		rateLimitRPS:   *rateLimitRPS,
		rateLimitBurst: *rateLimitBurst,
		enableRLS:      *enableRLS,
		tsPath:         *tsPath,
		identityDir:    *identityDir,
		pprofAddr:      *pprofAddr,
	}
	if cfg.dbPath != "" {
		cfg.backend = "sqlite"
		cfg.dsn = cfg.dbPath
	}
	return cfg, nil
}

// RunServe parses the `serve` subcommand flags, opens the storage backend, and
// runs the server until SIGINT/SIGTERM. args are the arguments after "serve".
func RunServe(args []string) error {
	cfg, err := parseServeArgs(args)
	if err != nil {
		return err
	}

	if cfg.tsPath != "" {
		race.SetExternalPath(cfg.tsPath)
	}

	logger := newLogger(cfg.logLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := OpenStorage(ctx, cfg.backend, cfg.dsn, cfg.enableRLS)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		return fmt.Errorf("serve: migrate: %w", err)
	}

	// Install RLS policies after the schema is in place (opt-in). The pool was
	// already configured to set app.tenant_id per connection by openStorage.
	if cfg.enableRLS {
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

	var identity *issuance.Identity
	if cfg.identityDir != "" {
		identity, err = issuance.LoadOrCreateIdentity(cfg.identityDir, "blunderdb-serve")
		if err != nil {
			return fmt.Errorf("serve: identity: %w", err)
		}
	}

	srv, err := New(Options{
		Addr:            cfg.addr,
		OpsAddr:         cfg.opsAddr,
		Storage:         st,
		Logger:          logger,
		Metrics:         metrics.New(),
		EnableMetrics:   cfg.enableMetrics,
		CORSAllowOrigin: cfg.corsOrigin,
		RateLimitRPS:    cfg.rateLimitRPS,
		RateLimitBurst:  cfg.rateLimitBurst,
		Identity:        identity,
	})
	if err != nil {
		return err
	}

	if cfg.pprofAddr != "" {
		stopPprof := startPprofServer(ctx, logger, cfg.pprofAddr)
		defer stopPprof()
	}

	logger.Warn("authentication is delegated to the reverse-proxy; do not expose this daemon to the public internet")
	return srv.Run(ctx)
}

// startPprofServer starts net/http/pprof on its own listener, entirely
// separate from the domain server's Addr — never register these handlers on
// the same mux as /v1/*, since they let a caller pull heap dumps and CPU
// profiles with no tenant scoping at all (#238). It returns a function that
// shuts the pprof listener down; RunServe also stops it once ctx is
// cancelled, so a caller that forgets to invoke the returned func still sees
// it go away with the rest of the daemon.
func startPprofServer(ctx context.Context, logger *slog.Logger, addr string) (stop func()) {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		logger.Warn("pprof listening — debug only, do not expose this address to the public internet", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("pprof server error", "err", err)
		}
	}()
	// The detached context below is on purpose: this goroutine runs precisely
	// because ctx is already done, and Shutdown given a cancelled context
	// returns at once without draining anything.
	go func() { //nolint:gosec // G118: the request-scoped context is the one that just fired; shutting down needs a live deadline of its own
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
			defer cancel()
			_ = srv.Shutdown(shutCtx)
			<-done
		})
	}
}

// OpenStorage opens the requested backend. enableRLS turns on PostgreSQL
// Row-Level Security enforcement (per-connection app.tenant_id); ignored by SQLite.
func OpenStorage(ctx context.Context, backend, dsn string, enableRLS bool) (storage.Storage, error) {
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

// envBoolOr, envFloatOr and envIntOr give --metrics, --cors-allow-origin's
// numeric siblings (--rate-limit-rps/-burst) the same BLUNDERDB_* fallback
// every other serve flag already has (#230); an unset or unparsable value
// keeps fallback rather than failing the whole command over one bad env var.
func envBoolOr(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envFloatOr(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
