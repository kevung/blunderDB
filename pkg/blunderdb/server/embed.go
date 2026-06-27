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

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/internal/server/metrics"
)

// Config configures an embedded engine. Backend is "postgres" in production
// (the only tenant-isolating backend); "sqlite" is for tests only.
type Config struct {
	Backend        string // "postgres" | "sqlite"
	DSN            string
	EnableRLS      bool
	EnableMetrics  bool
	Logger         *slog.Logger
	RateLimitRPS   float64
	RateLimitBurst int
}

// Bootstrap opens the storage backend, runs migrations, installs RLS policies
// when enabled, builds the engine server and returns its http.Handler plus an
// io.Closer for the storage pool. Mount the handler behind your own auth and
// inject X-Tenant-ID per request; the engine performs NO authentication.
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
		Storage:        st,
		Logger:         logger,
		Metrics:        metrics.New(),
		EnableMetrics:  cfg.EnableMetrics,
		RateLimitRPS:   cfg.RateLimitRPS,
		RateLimitBurst: cfg.RateLimitBurst,
	})
	if err != nil {
		st.Close()
		return nil, nil, err
	}
	return srv.Handler(), st, nil
}
