package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"iter"
	"net/http"

	"github.com/kevung/blunderdb/internal/server/middleware"
)

// This file holds the generic RPC plumbing shared by every /v1 domain handler.
// A handler is one of three shapes:
//
//   - rpc      decode JSON request → call → encode JSON response
//   - rpcStream decode JSON request → call → stream NDJSON items
//   - rpcVoid  decode JSON request → call → {"ok":true}
//
// Each concrete route is a tiny closure that binds the storage method, so the
// surface stays type-safe and mechanical (see handlers_<family>.go).

// okResp is the body returned by rpcVoid handlers.
type okResp struct {
	OK bool `json:"ok"`
}

// idResp wraps a freshly-created row id.
type idResp struct {
	ID int64 `json:"id"`
}

// idReq is the common "operate on this id" request.
type idReq struct {
	ID int64 `json:"id"`
}

// scopeOf returns the tenant scope for the request (empty if none, e.g. in a
// test that bypasses the tenant middleware).
func scopeOf(r *http.Request) string {
	scope, _ := middleware.TenantFromContext(r.Context())
	return scope
}

// decodeJSON decodes the request body into dst. An empty body is accepted and
// leaves dst at its zero value, so methods with all-optional fields can be
// called with no body.
func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

// writeJSONResp writes v as a 200 JSON response.
func writeJSONResp(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

// rpc builds a handler that decodes Req, invokes fn with the tenant scope, and
// encodes the Resp as JSON.
func rpc[Req any, Resp any](fn func(ctx context.Context, scope string, req Req) (Resp, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := decodeJSON(r, &req); err != nil {
			writeErrorCode(w, CodeInvalid, "invalid JSON body: "+err.Error())
			return
		}
		resp, err := fn(r.Context(), scopeOf(r), req)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		writeJSONResp(w, resp)
	}
}

// rpcVoid builds a handler for a storage method that returns only an error.
func rpcVoid[Req any](fn func(ctx context.Context, scope string, req Req) error) http.HandlerFunc {
	return rpc(func(ctx context.Context, scope string, req Req) (okResp, error) {
		if err := fn(ctx, scope, req); err != nil {
			return okResp{}, err
		}
		return okResp{OK: true}, nil
	})
}

// rpcStream builds a handler that decodes Req and streams the resulting
// iter.Seq2 as NDJSON.
func rpcStream[Req any, T any](fn func(ctx context.Context, scope string, req Req) iter.Seq2[T, error]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := decodeJSON(r, &req); err != nil {
			writeErrorCode(w, CodeInvalid, "invalid JSON body: "+err.Error())
			return
		}
		streamSeq2(w, fn(r.Context(), scopeOf(r), req))
	}
}
