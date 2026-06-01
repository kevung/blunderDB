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

	imports *importRegistry
	rl      *middleware.RateLimiter // nil when rate limiting is disabled
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
		imports: newImportRegistry(),
	}
	if opts.RateLimitRPS > 0 {
		s.rl = middleware.NewRateLimiter(opts.RateLimitRPS, opts.RateLimitBurst, opts.now)
	}

	mux := http.NewServeMux()
	s.knownPaths = make(map[string]bool)
	for _, rt := range s.routes() {
		mux.HandleFunc(rt.method+" "+rt.pattern, rt.handler)
		s.knownPaths[rt.pattern] = true
	}
	// Catch-all: any unmatched path returns the JSON error envelope.
	mux.HandleFunc("/", s.notFound)

	s.http = &http.Server{
		Addr:              opts.Addr,
		Handler:           s.chain(mux),
		ReadHeaderTimeout: opts.ReadHeaderTimeout,
	}
	return s, nil
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
	h = middleware.Tenant(func(w http.ResponseWriter, _ *http.Request, msg string) {
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

// limitBody caps request bodies to guard against OOM from a malicious client.
// Import endpoints are exempt from the small default cap: they carry uploaded
// match files and apply their own (larger) limit while spooling.
func (s *Server) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && !strings.HasPrefix(r.URL.Path, "/v1/imports.") {
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

// Run starts the server and blocks until ctx is cancelled, then shuts down
// gracefully within ShutdownTimeout. It returns the listener/serve error, or
// nil on a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return fmt.Errorf("server: listen %s: %w", s.opts.Addr, err)
	}

	if s.rl != nil {
		go s.sweepRateLimiter(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		s.opts.Logger.Info("serving", "addr", ln.Addr().String())
		errCh <- s.http.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		s.opts.Logger.Info("shutting down")
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
