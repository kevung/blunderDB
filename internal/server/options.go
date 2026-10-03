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

	// OpsAddr, when non-empty, moves the /ops/ family (maintenance.vacuum,
	// tenant.purge) onto a listener of its own, e.g. "127.0.0.1:8081", and
	// takes it off Addr entirely. Empty (the default) keeps /ops/ on the main
	// listener, where the proxy in front is expected to refuse the prefix —
	// see opsRoutes for why these two calls are not tenant-shaped, and
	// ADR-0005 for why the daemon cannot make that decision itself.
	OpsAddr string

	// SingleTenant refuses any X-Tenant-ID other than "1".
	//
	// The SQLite backend has no tenant column: its TenantFilter is "1=1", so
	// every scope reads and writes the same rows. Serving it with the tenant
	// header accepted but ignored means two tenants share one library while
	// the protocol says otherwise — an isolation that exists in the caller's
	// mind and nowhere else. Refusing is the honest answer: a
	// deployment that genuinely needs tenants needs PostgreSQL.
	//
	// Set by `serve` for the SQLite backend; the default (false) leaves every
	// valid tenant through, which is what PostgreSQL wants.
	SingleTenant bool

	// Storage is the backend the handlers operate on. Required.
	Storage storage.Storage

	// Logger receives structured request and lifecycle logs. Defaults to
	// slog.Default().
	Logger *slog.Logger

	// Metrics is the registry backing /metrics. Defaults to a fresh registry.
	Metrics *metrics.Registry

	// EnableMetrics toggles the /metrics endpoint and the metrics middleware.
	EnableMetrics bool

	// EnableWebUI serves the read-mostly web page under /app/ (ADR-0039). OFF
	// by default, and that default is the decision: the daemon authenticates
	// nobody (ADR-0005), so a browser-reachable interface
	// switched on out of the box would invite exactly the deployment ADR-0005
	// forbids — a daemon exposed with no proxy in front of it.
	EnableWebUI bool

	// EnableDirection serves the gestures of a Direction and of a Rencontre
	// (/v1/directions.*, /v1/rencontres.* that write, ADR-0057 rule 5). OFF by
	// default for the reason EnableWebUI is: the daemon authenticates nobody
	// (ADR-0005), and these routes enter results. Without it they answer 404,
	// as absent; the reads are served either way.
	EnableDirection bool
	// Transcription serves the transcription gestures (create, open, apply,
	// undo, redo, close, finish, abandon, editMatch; ADR-0057 rule 5). OFF by
	// default for the reason EnableWebUI is: writing through a daemon that
	// authenticates nobody is the operator's decision. The reads are served
	// either way.
	Transcription bool
	// MCPWrite offers the write tools of /mcp (save a position, comment it,
	// fill a collection). OFF by default for the reason EnableDirection is;
	// the read tools are served either way.
	MCPWrite bool

	// SessionPerCall lets a transcription gesture name no session: it then
	// uses the draft's live session or opens one. Right for `call`, where
	// every invocation is its own process and no session outlives it; over
	// HTTP a gesture without its session is refused (400), so a client whose
	// session expired is told (410) rather than handed a new one silently.
	SessionPerCall bool

	// TranscriptionTTL closes a transcription session idle for longer; its
	// undo stack goes, nothing typed does. 0 means 30 minutes.
	TranscriptionTTL time.Duration

	// EventsDSN is the PostgreSQL database the daemon shares with other
	// instances: when set, and EnableDirection or Transcription is, the
	// committed gestures travel between instances by LISTEN/NOTIFY
	// (pkg/blunderdb/events/pgnotify) and New fails if it cannot listen.
	// Empty: the in-memory bus alone, right for SQLite, which one instance
	// holds by construction.
	EventsDSN string
	// EventsSendOnly announces the gestures over EventsDSN without listening:
	// a process with no subscriber (`call`). New never fails on the
	// transport then; without it, the gesture is simply not announced.
	EventsSendOnly bool

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
	// no ceiling on disk usage.
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
	// RequestTimeout/StreamTimeout below bound an individual request instead,
	// at two different budgets by route shape.
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

	// Quotas bound each tenant's stored positions, engine time and concurrent
	// imports. The zero value is unlimited.
	Quotas TenantQuotas

	// ImportDir is the one directory of this host from which imports.batch
	// reads match files by path. Empty (the default) refuses every path: a
	// batch then arrives only as an uploaded archive. The daemon authenticates
	// nobody, so naming a directory is the operator's decision to let any
	// caller reach it.
	ImportDir string

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
	// eventsHeartbeat, eventsBuffer and eventsMaxPerTenant tune /v1/events (events.go), for
	// tests.
	eventsHeartbeat    time.Duration
	eventsBuffer       int
	eventsMaxPerTenant int
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
	if o.eventsHeartbeat == 0 {
		o.eventsHeartbeat = defaultEventsHeartbeat
	}
	if o.eventsBuffer == 0 {
		o.eventsBuffer = defaultEventsBuffer
	}
	if o.eventsMaxPerTenant == 0 {
		o.eventsMaxPerTenant = defaultEventsMaxPerTenant
	}
	if o.RateLimitRPS > 0 && o.RateLimitBurst == 0 {
		o.RateLimitBurst = int(2 * o.RateLimitRPS)
		if o.RateLimitBurst < 1 {
			o.RateLimitBurst = 1
		}
	}
}
