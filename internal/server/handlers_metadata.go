package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type versionResp struct {
	Version string `json:"version"`
}

// metadataRoutes exposes the metadata table read-only, schema version only:
// it is global infrastructure outside Row-Level Security, and a write route
// would let one tenant break every other (ADR-0005: no privileged tenant).
// The library settings (ADR-0046) ARE writable: they live in a per-tenant,
// RLS-scoped table.
func (s *Server) metadataRoutes() []route {
	ms := func() storage.MetadataStore { return s.opts.Storage.Metadata() }
	ls := func() storage.LibrarySettingsStore { return s.opts.Storage.LibrarySettings() }
	return []route{
		{http.MethodPost, "/v1/metadata.version", rpc(func(ctx context.Context, scope string, _ struct{}) (versionResp, error) {
			v, err := ms().Version(ctx, scope)
			return versionResp{Version: v}, err
		})},
		{http.MethodPost, "/v1/metadata.counts", rpc(func(ctx context.Context, scope string, _ struct{}) (storage.Counts, error) {
			return ms().Counts(ctx, scope)
		})},
		{http.MethodPost, "/v1/librarySettings.load", rpc(func(ctx context.Context, scope string, _ struct{}) (storage.LibrarySettings, error) {
			return ls().Load(ctx, scope)
		})},
		{http.MethodPost, "/v1/librarySettings.save", rpc(func(ctx context.Context, scope string, req storage.LibrarySettings) (storage.LibrarySettings, error) {
			if err := ls().Save(ctx, scope, req); err != nil {
				return storage.LibrarySettings{}, err
			}
			return ls().Load(ctx, scope)
		})},
	}
}
