package server

import (
	"log/slog"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
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

	// CORSAllowOrigin enables CORS for the given origin(s): "*", or a
	// comma-separated list of exact origins (each one echoed back only to a
	// request whose Origin header matches it — see middleware.CORS). Empty
	// (the default) keeps CORS off — the daemon is internal-only.
	CORSAllowOrigin string

	// MaxBodyBytes caps the size of a request body to guard against OOM.
	// Defaults to defaultMaxBodyBytes when zero.
	MaxBodyBytes int64

	// ImportMaxBodyBytes caps an uploaded import file. Defaults to
	// defaultImportMaxBodyBytes when zero.
	ImportMaxBodyBytes int64

	// MaxSpoolBytes bounds the total bytes concurrently in-flight imports may
	// hold spooled to $TMPDIR at once, across every tenant — see spoolQuota
	// (handlers_imports.go). Defaults to 4×ImportMaxBodyBytes when zero: N
	// concurrent imports each up to ImportMaxBodyBytes would otherwise have
	// no ceiling on disk usage (#234).
	MaxSpoolBytes int64

	// ReadHeaderTimeout bounds the time to read request headers. Defaults to
	// defaultReadHeaderTimeout.
	ReadHeaderTimeout time.Duration

	// IdleTimeout bounds how long a keep-alive connection may sit idle between
	// requests before the server closes it. Defaults to defaultIdleTimeout.
	// Deliberately the only *whole-connection* timeout set on http.Server:
	// its Read/WriteTimeout stay unset because they would bound every
	// request on the connection by the same fixed budget, and
	// imports/NDJSON list-style routes legitimately run far longer than an
	// ordinary call (see streamSeq2 / handlers_imports.go). IdleTimeout only
	// ever fires between requests, so it cannot cut a stream short.
	// RequestTimeout/StreamTimeout below bound an individual request
	// instead, at two different budgets by route shape — the per-request
	// mechanism this comment used to say the daemon had none of (#234).
	IdleTimeout time.Duration

	// RequestTimeout bounds an ordinary (non-streaming) request's total
	// read+write time, applied per request via
	// http.ResponseController.SetReadDeadline/SetWriteDeadline rather than
	// http.Server's whole-connection Read/WriteTimeout (see IdleTimeout's
	// comment for why those stay unset). Defaults to defaultRequestTimeout.
	RequestTimeout time.Duration

	// StreamTimeout is RequestTimeout's counterpart for a streaming route —
	// an rpcStream list, an import/export, a gammonNet sweep (see
	// streamingPaths in routes.go) — which can legitimately run far longer
	// than an ordinary call. Defaults to defaultStreamTimeout.
	StreamTimeout time.Duration

	// MaxConnections caps concurrently accepted TCP connections via
	// netutil.LimitListener: past this many, Accept blocks a new connection
	// until one of the existing ones closes, rather than handing every
	// connection a client can open its own goroutine and file descriptor
	// unconditionally. Defaults to defaultMaxConnections.
	MaxConnections int

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

	// Identity signs the watermark of an exports.sqlite response that asked
	// for one — "the daemon's own" identity, as opposed to the desktop's
	// per-person key (see ingest.SealWatermark). nil (the default) means
	// this daemon cannot watermark; a request that asks for one anyway fails
	// with CodeInvalid rather than silently exporting unmarked. RunServe
	// loads or creates it from --identity-dir.
	Identity *issuance.Identity

	// now is an injectable clock for deterministic tests. Defaults to
	// time.Now.
	now func() time.Time
}

const (
	defaultAddr               = ":8080"
	defaultMaxBodyBytes       = 32 << 20  // 32 MiB; import endpoints raise this.
	defaultImportMaxBodyBytes = 512 << 20 // 512 MiB for uploaded match files.
	defaultReadHeaderTimeout  = 10 * time.Second
	defaultIdleTimeout        = 120 * time.Second
	defaultRequestTimeout     = 30 * time.Second
	defaultStreamTimeout      = 30 * time.Minute
	defaultMaxConnections     = 4096
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
	if o.MaxSpoolBytes == 0 {
		o.MaxSpoolBytes = 4 * o.ImportMaxBodyBytes
	}
	if o.ReadHeaderTimeout == 0 {
		o.ReadHeaderTimeout = defaultReadHeaderTimeout
	}
	if o.IdleTimeout == 0 {
		o.IdleTimeout = defaultIdleTimeout
	}
	if o.RequestTimeout == 0 {
		o.RequestTimeout = defaultRequestTimeout
	}
	if o.StreamTimeout == 0 {
		o.StreamTimeout = defaultStreamTimeout
	}
	if o.MaxConnections == 0 {
		o.MaxConnections = defaultMaxConnections
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
