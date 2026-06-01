package server

import (
	"log/slog"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Options configures a Server. Storage is required; the rest have sane
// defaults applied by New.
type Options struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string

	// Storage is the backend the handlers operate on. Required.
	Storage storage.Storage

	// Logger receives structured request and lifecycle logs. Defaults to
	// slog.Default().
	Logger *slog.Logger

	// Metrics is the registry backing /metrics. Defaults to a fresh registry.
	Metrics *metrics.Registry

	// EnableMetrics toggles the /metrics endpoint and the metrics middleware.
	EnableMetrics bool

	// CORSAllowOrigin enables CORS for the given origin ("*" or a URL). Empty
	// (the default) keeps CORS off — the daemon is internal-only.
	CORSAllowOrigin string

	// MaxBodyBytes caps the size of a request body to guard against OOM.
	// Defaults to defaultMaxBodyBytes when zero.
	MaxBodyBytes int64

	// ImportMaxBodyBytes caps an uploaded import file. Defaults to
	// defaultImportMaxBodyBytes when zero.
	ImportMaxBodyBytes int64

	// ReadHeaderTimeout bounds the time to read request headers. Defaults to
	// defaultReadHeaderTimeout.
	ReadHeaderTimeout time.Duration

	// ShutdownTimeout bounds graceful shutdown. Defaults to
	// defaultShutdownTimeout.
	ShutdownTimeout time.Duration

	// RateLimitRPS is the per-tenant sustained request rate. Zero (the
	// default) disables rate limiting entirely (the middleware is not mounted,
	// so there is no overhead).
	RateLimitRPS float64

	// RateLimitBurst is the per-tenant token-bucket size. Defaults to
	// 2×RateLimitRPS (min 1) when zero and rate limiting is enabled.
	RateLimitBurst int

	// now is an injectable clock for deterministic tests. Defaults to
	// time.Now.
	now func() time.Time
}

const (
	defaultAddr               = ":8080"
	defaultMaxBodyBytes       = 32 << 20  // 32 MiB; import endpoints raise this.
	defaultImportMaxBodyBytes = 512 << 20 // 512 MiB for uploaded match files.
	defaultReadHeaderTimeout  = 10 * time.Second
	defaultShutdownTimeout    = 15 * time.Second
)

func (o *Options) applyDefaults() {
	if o.Addr == "" {
		o.Addr = defaultAddr
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Metrics == nil {
		o.Metrics = metrics.New()
	}
	if o.MaxBodyBytes == 0 {
		o.MaxBodyBytes = defaultMaxBodyBytes
	}
	if o.ImportMaxBodyBytes == 0 {
		o.ImportMaxBodyBytes = defaultImportMaxBodyBytes
	}
	if o.ReadHeaderTimeout == 0 {
		o.ReadHeaderTimeout = defaultReadHeaderTimeout
	}
	if o.ShutdownTimeout == 0 {
		o.ShutdownTimeout = defaultShutdownTimeout
	}
	if o.now == nil {
		o.now = time.Now
	}
	if o.RateLimitRPS > 0 && o.RateLimitBurst == 0 {
		o.RateLimitBurst = int(2 * o.RateLimitRPS)
		if o.RateLimitBurst < 1 {
			o.RateLimitBurst = 1
		}
	}
}
