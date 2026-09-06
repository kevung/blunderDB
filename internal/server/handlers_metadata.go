package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type versionResp struct {
	Version string `json:"version"`
}

// metadataRoutes exposes the metadata table read-only, and only its schema
// version: the table is database infrastructure (schema version, issuance),
// global to every tenant and outside Row-Level Security. Until #156 it also
// served metadata.load, metadata.save and metadata.setVersion — a tenant
// could read every other tenant's session state (then stored there), rewrite
// database_version, and take /readyz down for the whole instance. Those
// routes are gone for good (ADR-0005: the daemon has no privileged tenant),
// and the session state has its own tenant-scoped table.
//
// The library's own settings (ADR-0043) are served here too, and they ARE
// writable: they are per-tenant data in a per-tenant table (library_settings,
// under Row-Level Security), so a tenant setting its own thresholds reaches
// nothing but its own rows — the very property metadata.save lacked.
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
