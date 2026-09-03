// Package server implements the `blunderdb serve` HTTP + JSON daemon. It
// exposes the engine over RPC-style routes (POST /v1/<family>.<method>) backed
// by a storage.Storage value.
//
// Security: this daemon performs NO authentication. It trusts the
// X-Tenant-ID header injected by an upstream reverse-proxy and MUST NOT be
// exposed directly to the public internet. See tasks/headless/06-serve-http.md.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/netutil"

	"github.com/kevung/blunderdb/internal/server/handlers"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Server is the HTTP daemon. Construct it with New and run it with Run.
type Server struct {
	opts       Options
	health     *handlers.Health
	http       *http.Server
	knownPaths map[string]bool
	// uploadPaths is the exact set of routes limitBody exempts from
	// MaxBodyBytes — the multipart import uploads, which cap themselves at
	// ImportMaxBodyBytes (see uploadRoutes in handlers_imports.go).
	uploadPaths map[string]bool
	// allowedMethod maps every registered pattern to the one HTTP method it
	// accepts, so a request to a known path with the wrong method gets the
	// API's own JSON error envelope (405 + Allow) instead of net/http's
	// automatic text/plain response — see methodNotAllowed (#232).
	allowedMethod map[string]string

	imports *importRegistry
	// gammonnetJobs tracks in-flight gammonNet catch-up sweeps (#130), kept
	// separate from imports so cancelling one can never be confused with the
	// other — reuses importRegistry's scope-keyed cancel bookkeeping under
	// its own instance rather than a new type for the same three methods.
	gammonnetJobs *importRegistry
	rl            *middleware.RateLimiter // nil when rate limiting is disabled
	// spool bounds the total bytes concurrently in-flight imports may hold
	// spooled to $TMPDIR — see handleImport and Options.MaxSpoolBytes (#234).
	spool *spoolQuota
}

// New builds a Server from opts. It returns an error if no Storage is set.
func New(opts Options) (*Server, error) {
	if opts.Storage == nil {
		return nil, errors.New("server: Options.Storage is required")
	}
	opts.applyDefaults()

	s := &Server{
		opts: opts,
		health: &handlers.Health{
			Storage:         opts.Storage,
			Metrics:         opts.Metrics,
			ExpectedVersion: domain.DatabaseVersion,
		},
		imports:       newImportRegistry(),
		gammonnetJobs: newImportRegistry(),
		spool:         newSpoolQuota(opts.MaxSpoolBytes),
	}
	if opts.RateLimitRPS > 0 {
		s.rl = middleware.NewRateLimiter(opts.RateLimitRPS, opts.RateLimitBurst, opts.now)
	}

	mux := http.NewServeMux()
	s.knownPaths = make(map[string]bool)
	s.uploadPaths = uploadPaths()
	s.allowedMethod = make(map[string]string)
	for _, rt := range s.routes() {
		mux.HandleFunc(rt.method+" "+rt.pattern, rt.handler)
		s.knownPaths[rt.pattern] = true
		s.allowedMethod[rt.pattern] = rt.method
	}
	// Catch-all: any unmatched path returns the JSON error envelope.
	mux.HandleFunc("/", s.notFound)

	s.http = &http.Server{
		Addr:              opts.Addr,
		Handler:           s.chain(s.withDeadlines(s.methodNotAllowed(mux))),
		ReadHeaderTimeout: opts.ReadHeaderTimeout,
		// Read/WriteTimeout stay unset — see Options.IdleTimeout's doc
		// comment for why; withDeadlines is the per-request replacement.
		IdleTimeout: opts.IdleTimeout,
	}
	return s, nil
}

// withDeadlines sets a per-request read/write deadline via
// http.ResponseController before dispatching to next: RequestTimeout for an
// ordinary call, the far more generous StreamTimeout for a route
// streamingPaths names (#234). Errors from SetReadDeadline/SetWriteDeadline
// are ignored — they fail only when the underlying connection genuinely
// cannot support a deadline (e.g. it has been hijacked already), in which
// case there is nothing more useful to do than proceed without one.
func (s *Server) withDeadlines(next http.Handler) http.Handler {
	streaming := s.streamingPaths()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeout := s.opts.RequestTimeout
		if streaming[r.URL.Path] {
			timeout = s.opts.StreamTimeout
		}
		if timeout > 0 {
			deadline := time.Now().Add(timeout)
			rc := http.NewResponseController(w)
			_ = rc.SetReadDeadline(deadline)
			_ = rc.SetWriteDeadline(deadline)
		}
		next.ServeHTTP(w, r)
	})
}

// chain wraps the mux with the middleware stack. Order (outermost first):
// recover → metrics → logging → cors → tenant → mux. recover is outermost so
// it catches panics from every layer; tenant is innermost so r.Pattern is set
// by the mux for the metrics/logging labels read after next returns.
func (s *Server) chain(mux http.Handler) http.Handler {
	h := mux
	// Rate limiting sits just inside Tenant so it can read the tenant from the
	// context; it is only mounted when enabled (zero overhead otherwise).
	if s.rl != nil {
		h = middleware.RateLimit(s.rl, func(w http.ResponseWriter, _ *http.Request) {
			s.opts.Metrics.IncRateLimitRejected()
			writeErrorCode(w, CodeRateLimited, "too many requests")
		})(h)
	}
	h = middleware.Tenant(s.publicPaths(), func(w http.ResponseWriter, _ *http.Request, msg string) {
		writeErrorCode(w, CodeInvalid, msg)
	})(h)
	h = middleware.CORS(s.opts.CORSAllowOrigin)(h)
	h = middleware.Logging(s.opts.Logger, s.knownPaths, s.opts.now)(h)
	if s.opts.EnableMetrics {
		h = middleware.Metrics(s.opts.Metrics, s.knownPaths, s.opts.now)(h)
	}
	h = s.limitBody(h)
	h = middleware.Recover(s.opts.Logger, func(w http.ResponseWriter, _ *http.Request) {
		writeErrorCode(w, CodeInternal, "internal error")
	})(h)
	return h
}

// publicPaths is the set of routes reachable without a tenant: everything in
// the routing table outside the /v1 domain surface, i.e. the ops endpoints.
// Derived rather than listed so a new ops route is public the day it lands,
// and a new domain route can never be.
func (s *Server) publicPaths() map[string]bool {
	public := make(map[string]bool)
	for _, rt := range s.routes() {
		if !strings.HasPrefix(rt.pattern, "/v1/") {
			public[rt.pattern] = true
		}
	}
	return public
}

// limitBody caps request bodies to guard against OOM from a malicious client.
// The upload endpoints — exactly uploadPaths, not a "/v1/imports." prefix
// that would also cover imports.cancel (#160) — are exempt from the small
// default cap: they carry uploaded match files and apply their own (larger)
// limit while spooling. A declared Content-Length over the cap is refused
// 413 before a byte of it is read; an undeclared one is cut off by
// MaxBytesReader and the handler's decoder reports it (writeDecodeError).
func (s *Server) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && !s.uploadPaths[r.URL.Path] {
			if r.ContentLength > s.opts.MaxBodyBytes {
				writeBodyTooLarge(w, s.opts.MaxBodyBytes)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, s.opts.MaxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// Handler exposes the fully-wired http.Handler for tests (httptest).
func (s *Server) Handler() http.Handler { return s.http.Handler }

// rateLimitSweepInterval and rateLimitMaxIdle govern eviction of idle per-tenant
// buckets so the map doesn't grow without bound.
const (
	rateLimitSweepInterval = 5 * time.Minute
	rateLimitMaxIdle       = 30 * time.Minute
)

// sweepRateLimiter periodically evicts idle tenant buckets and publishes the
// live bucket count to the metrics registry, until ctx is cancelled.
func (s *Server) sweepRateLimiter(ctx context.Context) {
	t := time.NewTicker(rateLimitSweepInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.opts.Metrics.SetRateLimitBuckets(s.rl.Sweep(rateLimitMaxIdle))
		}
	}
}

// poolStatsSweepInterval bounds how stale blunderdb_pg_pool_* can be.
const poolStatsSweepInterval = 15 * time.Second

// poolStatsProvider is implemented by postgres.Storage — duck-typed so this
// package need not import the postgres package just for this (it already
// does, in serve.go, but the interface keeps this specific dependency
// explicit and minimal). The SQLite backend does not implement it, so the
// sweep below is simply never started against it.
type poolStatsProvider interface {
	PoolStats() (acquired, idle, max int32, waitCount int64)
}

// sweepPoolStats periodically publishes the PostgreSQL connection pool's
// state to the metrics registry (#235), until ctx is cancelled.
func (s *Server) sweepPoolStats(ctx context.Context, pool poolStatsProvider) {
	t := time.NewTicker(poolStatsSweepInterval)
	defer t.Stop()
	publish := func() {
		acquired, idle, max, waitCount := pool.PoolStats()
		s.opts.Metrics.SetPoolStats(acquired, idle, max, waitCount)
	}
	publish() // first data point without waiting a full interval
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			publish()
		}
	}
}

// Run starts the server and blocks until ctx is cancelled, then shuts down
// gracefully within ShutdownTimeout. It returns the listener/serve error, or
// nil on a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return fmt.Errorf("server: listen %s: %w", s.opts.Addr, err)
	}
	if s.opts.MaxConnections > 0 {
		// The (n+1)-th concurrent connection blocks in Accept until one of
		// the first n closes, instead of every connection a client opens
		// getting its own goroutine and file descriptor unconditionally
		// (#234).
		ln = netutil.LimitListener(ln, s.opts.MaxConnections)
	}

	if s.rl != nil {
		go s.sweepRateLimiter(ctx)
	}
	if s.opts.EnableMetrics {
		if pool, ok := s.opts.Storage.(poolStatsProvider); ok {
			go s.sweepPoolStats(ctx, pool)
		}
	}

	errCh := make(chan error, 1)
	go func() {
		s.opts.Logger.Info("serving", "addr", ln.Addr().String())
		errCh <- s.http.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		s.opts.Logger.Info("shutting down")
		// Cancel every in-flight import/gammonNet job BEFORE Shutdown: each
		// one's handler is watching its own context and, once cancelled,
		// emits a trailing {"event":"cancelled"} and returns on its own —
		// see runGammonNetSweep and handleImport. Left to Shutdown alone,
		// a streaming handler is either waited on past ShutdownTimeout (an
		// import mid a 512 MiB upload) or has its connection cut the moment
		// the deadline passes, with no chance to tell the client why
		// (#234).
		s.imports.cancelAll()
		s.gammonnetJobs.cancelAll()
		shutCtx, cancel := context.WithTimeout(context.Background(), s.opts.ShutdownTimeout)
		defer cancel()
		if err := s.http.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("server: shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("server: serve: %w", err)
	}
}
