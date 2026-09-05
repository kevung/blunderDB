// Package server exposes blunderdb's HTTP engine handler for in-process
// embedding by a trusted parent process (e.g. gammonGo). It is a thin,
// generic wrapper over the internal serve path — no social logic, no auth.
// The embedder is responsible for authentication and for setting the
// X-Tenant-ID header on every request, exactly as the standalone daemon expects.
package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// Config configures an embedded engine. Backend is "postgres" in production
// (the only tenant-isolating backend); "sqlite" is for tests only.
//
// This only covers the settings that shape the *returned Handler's own
// behaviour* — body-size caps, CORS, per-request deadlines, the watermark
// identity. Settings that belong to a listener or an http.Server the
// standalone daemon owns (ReadHeaderTimeout, IdleTimeout, MaxConnections,
// ShutdownTimeout — see internal/server.Options) have no meaning here: the
// embedder runs its own http.Server (or equivalent) around this Handler and
// owns those decisions itself (#241/#236).
type Config struct {
	Backend        string // "postgres" | "sqlite"
	DSN            string
	EnableRLS      bool
	EnableMetrics  bool
	Logger         *slog.Logger
	RateLimitRPS   float64
	RateLimitBurst int

	// MaxBodyBytes caps an ordinary /v1 request body. Defaults to
	// internal/server's own default (32 MiB) when zero — see
	// internal/server.Options.MaxBodyBytes.
	MaxBodyBytes int64
	// ImportMaxBodyBytes caps an uploaded import file specifically (larger
	// than MaxBodyBytes). Defaults to internal/server's own default
	// (512 MiB) when zero.
	ImportMaxBodyBytes int64
	// MaxSpoolBytes bounds the total bytes concurrently in-flight imports
	// may hold spooled to disk at once, across every tenant. Defaults to
	// 4×ImportMaxBodyBytes when zero.
	MaxSpoolBytes int64

	// CORSAllowOrigin enables CORS for the given origin(s) — see
	// internal/server.Options.CORSAllowOrigin's doc comment for the exact
	// syntax ("*", or a comma-separated list). Empty (the default) keeps
	// CORS off.
	CORSAllowOrigin string

	// RequestTimeout bounds an ordinary (non-streaming) request's total
	// read+write time; StreamTimeout is its counterpart for a streaming
	// route (an NDJSON list, an import/export, a gammonNet sweep). Both
	// default to internal/server's own defaults (30s / 30m) when zero. Set
	// via http.ResponseController per request, so they take effect
	// regardless of which http.Server the embedder wraps this Handler in.
	RequestTimeout time.Duration
	StreamTimeout  time.Duration

	// Identity signs the watermark of an exports.sqlite response that asks
	// for one. nil (the default) means this embedding cannot watermark — a
	// request that asks for one anyway fails with CodeInvalid rather than
	// silently exporting unmarked (ingest.SealWatermark is never called
	// without one). There is no default identity: an embedder that wants
	// watermarking loads or creates one itself (see
	// issuance.LoadOrCreateIdentity, the same helper the standalone daemon's
	// --identity-dir uses) and passes it here.
	Identity *issuance.Identity
}

// Bootstrap opens the storage backend, runs migrations, installs RLS policies
// when enabled, builds the engine server and returns its http.Handler plus an
// io.Closer for the storage pool. Mount the handler behind your own auth and
// inject X-Tenant-ID per request — the tenant's positive decimal integer, a
// name is refused with 400 (ADR-0005); the engine performs NO authentication.
func Bootstrap(ctx context.Context, cfg Config) (http.Handler, io.Closer, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	st, err := internalserver.OpenStorage(ctx, cfg.Backend, cfg.DSN, cfg.EnableRLS)
	if err != nil {
		return nil, nil, err
	}
	if err := st.Migrate(ctx); err != nil {
		st.Close()
		return nil, nil, fmt.Errorf("blunderdb embed: migrate: %w", err)
	}
	if cfg.EnableRLS {
		applier, ok := st.(interface {
			ApplyRLS(context.Context) error
		})
		if !ok {
			st.Close()
			return nil, nil, fmt.Errorf("blunderdb embed: RLS requires the postgres backend")
		}
		if err := applier.ApplyRLS(ctx); err != nil {
			st.Close()
			return nil, nil, fmt.Errorf("blunderdb embed: apply RLS: %w", err)
		}
		logger.Info("blunderdb embed: Row-Level Security enabled")
	}

	srv, err := internalserver.New(internalserver.Options{
		Storage:            st,
		Logger:             logger,
		Metrics:            metrics.New(),
		EnableMetrics:      cfg.EnableMetrics,
		RateLimitRPS:       cfg.RateLimitRPS,
		RateLimitBurst:     cfg.RateLimitBurst,
		MaxBodyBytes:       cfg.MaxBodyBytes,
		ImportMaxBodyBytes: cfg.ImportMaxBodyBytes,
		MaxSpoolBytes:      cfg.MaxSpoolBytes,
		CORSAllowOrigin:    cfg.CORSAllowOrigin,
		RequestTimeout:     cfg.RequestTimeout,
		StreamTimeout:      cfg.StreamTimeout,
		Identity:           cfg.Identity,
	})
	if err != nil {
		st.Close()
		return nil, nil, err
	}
	return srv.Handler(), st, nil
}
