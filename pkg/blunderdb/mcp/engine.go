package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TenantHeader is the header the engine reads its tenant from — the same one
// /v1 trusts (ADR-0005). It is restated rather than imported so this package
// stays free of internal/server, which mounts it.
const TenantHeader = "X-Tenant-ID"

// UserHeader names the person behind a request; it signs the comments written
// through it. Forwarded as received, like TenantHeader.
const UserHeader = "X-User-Name"

// ReadTenantsHeader lists the tenants an across.* read spans besides
// TenantHeader (ADR-0063). The engine forwards the value the MCP request
// carries, and only on across.* calls: the read set is the proxy's, never a
// choice of the model, and no other route looks at it.
const ReadTenantsHeader = "X-Read-Tenants"

// Engine is how a tool reaches blunderDB: one /v1 call, dispatched in-process
// through the very handler the daemon serves. Going through the handler rather
// than the storage contract keeps one tenant check, one rate limit, one error
// envelope and one set of business rules for HTTP, `call` and MCP alike.
type Engine struct {
	handler http.Handler
	tenant  string
}

// NewEngine wraps an engine handler (internal/server's fully chained one).
// tenant is the scope used when the MCP request carries no X-Tenant-ID: "1" for
// a local file over stdio; empty means the header is mandatory, which is what an
// HTTP mount wants — a missing tenant must never fall back to someone's data.
func NewEngine(handler http.Handler, tenant string) *Engine {
	return &Engine{handler: handler, tenant: tenant}
}

// APIError is a /v1 error envelope, surfaced to the model as a tool error.
type APIError struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Engine) tenantOf(req *sdk.CallToolRequest) (string, error) {
	if req != nil && req.Extra != nil && req.Extra.Header != nil {
		if t := req.Extra.Header.Get(TenantHeader); t != "" {
			return t, nil
		}
	}
	if e.tenant == "" {
		return "", errors.New("no tenant: the request carries no " + TenantHeader)
	}
	return e.tenant, nil
}

func (e *Engine) do(ctx context.Context, req *sdk.CallToolRequest, method string, in any) (*recorder, error) {
	if in == nil {
		in = struct{}{}
	}
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return e.send(ctx, req, method, "application/json", body)
}

func (e *Engine) send(ctx context.Context, req *sdk.CallToolRequest, method, contentType string, body []byte) (*recorder, error) {
	tenant, err := e.tenantOf(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "/v1/"+method, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", contentType)
	r.Header.Set(TenantHeader, tenant)
	if req != nil && req.Extra != nil && req.Extra.Header != nil {
		// The person the proxy names signs what is written, as on a direct call.
		if u := req.Extra.Header.Get(UserHeader); u != "" {
			r.Header.Set(UserHeader, u)
		}
	}
	if strings.HasPrefix(method, "across.") && req != nil && req.Extra != nil {
		// Every line is forwarded as received, so the /v1 gate refuses a
		// repeated header here as it does on a direct call.
		for _, v := range req.Extra.Header.Values(ReadTenantsHeader) {
			r.Header.Add(ReadTenantsHeader, v)
		}
	}
	w := &recorder{header: http.Header{}, status: http.StatusOK}
	e.handler.ServeHTTP(w, r)
	if w.status >= 400 {
		return nil, decodeError(w.status, w.body.Bytes())
	}
	return w, nil
}

// Call posts in to /v1/<method> and decodes the JSON answer into out (nil
// discards it).
func (e *Engine) Call(ctx context.Context, req *sdk.CallToolRequest, method string, in, out any) error {
	w, err := e.do(ctx, req, method, in)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(w.body.Bytes(), out); err != nil {
		return fmt.Errorf("%s: decode answer: %w", method, err)
	}
	return nil
}

// Stream posts in to an NDJSON route and decodes at most limit rows (0: all).
// An error line inside the stream fails the call.
func Stream[T any](ctx context.Context, e *Engine, req *sdk.CallToolRequest, method string, in any, limit int) ([]T, error) {
	w, err := e.do(ctx, req, method, in)
	if err != nil {
		return nil, err
	}
	var out []T
	sc := bufio.NewScanner(&w.body)
	sc.Buffer(make([]byte, 0, 64*1024), 64*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte(`{"error":`)) {
			return nil, decodeError(http.StatusInternalServerError, line)
		}
		var v T
		if err := json.Unmarshal(line, &v); err != nil {
			return nil, fmt.Errorf("%s: decode row: %w", method, err)
		}
		out = append(out, v)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, sc.Err()
}

func decodeError(status int, body []byte) error {
	var env struct {
		Error APIError `json:"error"`
	}
	if json.Unmarshal(body, &env) == nil && env.Error.Message != "" {
		env.Error.Status = status
		return &env.Error
	}
	return &APIError{Status: status, Message: strings.TrimSpace(string(body))}
}

// recorder is the in-process ResponseWriter: the answer is small (tools page
// their lists), so it is buffered whole.
type recorder struct {
	header http.Header
	status int
	wrote  bool
	body   bytes.Buffer
}

func (w *recorder) Header() http.Header { return w.header }

func (w *recorder) WriteHeader(status int) {
	if !w.wrote {
		w.status, w.wrote = status, true
	}
}

func (w *recorder) Write(b []byte) (int, error) {
	w.wrote = true
	return w.body.Write(b)
}

func (w *recorder) Flush() {}

var _ io.Writer = (*recorder)(nil)
