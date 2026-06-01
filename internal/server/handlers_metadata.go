package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type setVersionReq struct {
	Version string `json:"version"`
}

type metadataSaveReq struct {
	Metadata map[string]string `json:"metadata"`
}

type versionResp struct {
	Version string `json:"version"`
}

func (s *Server) metadataRoutes() []route {
	ms := func() storage.MetadataStore { return s.opts.Storage.Metadata() }
	return []route{
		{http.MethodPost, "/v1/metadata.version", rpc(func(ctx context.Context, scope string, _ struct{}) (versionResp, error) {
			v, err := ms().Version(ctx, scope)
			return versionResp{Version: v}, err
		})},
		{http.MethodPost, "/v1/metadata.setVersion", rpcVoid(func(ctx context.Context, scope string, req setVersionReq) error {
			return ms().SetVersion(ctx, scope, req.Version)
		})},
		{http.MethodPost, "/v1/metadata.load", rpc(func(ctx context.Context, scope string, _ struct{}) (map[string]string, error) {
			return ms().Load(ctx, scope)
		})},
		{http.MethodPost, "/v1/metadata.save", rpcVoid(func(ctx context.Context, scope string, req metadataSaveReq) error {
			return ms().Save(ctx, scope, req.Metadata)
		})},
		{http.MethodPost, "/v1/metadata.counts", rpc(func(ctx context.Context, scope string, _ struct{}) (storage.Counts, error) {
			return ms().Counts(ctx, scope)
		})},
	}
}
