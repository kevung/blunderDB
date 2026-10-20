package remote

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrUnavailable is a call that did not get an answer from the daemon: the
// network failed, or what came back is not the /v1 error envelope. The
// operation may or may not have been applied — the caller re-reads before it
// retries a write.
var ErrUnavailable = errors.New("remote: daemon unavailable")

// ErrUnsupported is a contract method the /v1 API exposes no route for.
var ErrUnsupported = errors.New("remote: no /v1 route for this storage method")

// UnsupportedError names the contract method that has no /v1 route. It
// matches ErrUnsupported and storage.ErrInternal: a caller that only knows
// the storage sentinels still sees a failure, never a not-found that reads as
// an absent row.
type UnsupportedError struct {
	Method string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("remote: %s has no /v1 route", e.Method)
}

// Is matches ErrUnsupported and storage.ErrInternal.
func (e *UnsupportedError) Is(target error) bool {
	return target == ErrUnsupported || target == storage.ErrInternal
}

func unsupported(method string) error { return &UnsupportedError{Method: method} }

// unsupportedSeq is the stream of a contract method with no /v1 route: one
// error, no element.
func unsupportedSeq[T any](method string) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		yield(zero, unsupported(method))
	}
}

// Options configures the backend.
type Options struct {
	// BaseURL is the daemon's root, e.g. "https://blunderdb.example.org"; the
	// backend appends "/v1/<method>".
	BaseURL string
	// Tenant is the X-Tenant-ID sent when a method's scope is "" — the
	// single implicit tenant the CLI and the desktop app pass. A non-empty
	// scope is sent as is. An authenticating proxy normally overwrites it.
	Tenant string
	// Header is added to every request: the proxy's token, an
	// X-User-Name. The backend reads none of it.
	Header http.Header
	// HTTPClient defaults to http.DefaultClient.
	HTTPClient *http.Client
}

type client struct {
	base   string
	tenant string
	header http.Header
	http   *http.Client
}

func newClient(o Options) (*client, error) {
	if o.BaseURL == "" {
		return nil, fmt.Errorf("%w: remote: BaseURL is required", storage.ErrInvalid)
	}
	hc := o.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	return &client{base: strings.TrimRight(o.BaseURL, "/"), tenant: o.Tenant, header: o.Header.Clone(), http: hc}, nil
}

// post sends one /v1 call and returns the open response; the caller closes
// its body. A non-2xx answer is decoded into the storage sentinels here.
func (c *client) post(ctx context.Context, scope, method string, req any) (*http.Response, error) {
	var body io.Reader = http.NoBody
	if req != nil {
		b, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("%w: remote: encode %s: %w", storage.ErrInvalid, method, err)
		}
		body = bytes.NewReader(b)
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/v1/"+method, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnavailable, method, err)
	}
	for k, vs := range c.header {
		for _, v := range vs {
			hr.Header.Add(k, v)
		}
	}
	hr.Header.Set("Content-Type", "application/json")
	tenant := scope
	if tenant == "" {
		tenant = c.tenant
	}
	if tenant != "" {
		hr.Header.Set("X-Tenant-ID", tenant)
	}
	if author := storage.CommentAuthorFromContext(ctx); author != "" {
		hr.Header.Set("X-User-Name", author)
	}
	resp, err := c.http.Do(hr)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %s: %w", ErrUnavailable, method, err)
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		return nil, decodeError(method, resp)
	}
	return resp, nil
}

// errorEnvelope is the daemon's wire shape for every failure.
type errorEnvelope struct {
	Error *struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func decodeError(method string, resp *http.Response) error {
	var env errorEnvelope
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&env); err != nil || env.Error == nil {
		return fmt.Errorf("%w: %s: HTTP %d without an error envelope", ErrUnavailable, method, resp.StatusCode)
	}
	return envelopeError(method, env.Error.Code, env.Error.Message, env.Error.Details)
}

// namedRefusals are the storage errors that wrap a sentinel with a message of
// their own; the daemon sends that message, so it names the refusal back.
var namedRefusals = []error{storage.ErrNoReviewToUndo, storage.ErrSwapDraftOpen, storage.ErrSwapPlayedHere}

func envelopeError(method, code, message string, details map[string]any) error {
	for _, r := range namedRefusals {
		if strings.Contains(message, r.Error()) {
			return fmt.Errorf("%w (%s)", r, message)
		}
	}
	switch code {
	case "not_found":
		return fmt.Errorf("%w: %s", storage.ErrNotFound, message)
	case "conflict":
		// A duplicate position names the row that already holds the board.
		if id, ok := details["existingId"].(float64); ok {
			return &storage.DuplicatePositionError{ExistingID: int64(id)}
		}
		return fmt.Errorf("%w: %s", storage.ErrConflict, message)
	case "invalid":
		return fmt.Errorf("%w: %s", storage.ErrInvalid, message)
	case "unknown_route":
		// The daemon is older than this backend: the route it expects is
		// missing there, which is no answer about the data.
		return fmt.Errorf("%w: %s", unsupported(method), message)
	case "internal":
		return fmt.Errorf("%w: %s", storage.ErrInternal, message)
	default:
		return fmt.Errorf("%w: %s: %s: %s", ErrUnavailable, method, code, message)
	}
}

// call sends req and decodes the JSON answer into out (nil discards it).
func (c *client) call(ctx context.Context, scope, method string, req, out any) error {
	resp, err := c.post(ctx, scope, method, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %s: decode answer: %w", ErrUnavailable, method, err)
	}
	return nil
}

// stream sends req and yields each NDJSON line of the answer. A line that is
// an error envelope ends the stream with that error, as the daemon writes it
// once it has started answering.
func stream[T any](ctx context.Context, c *client, scope, method string, req any) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		resp, err := c.post(ctx, scope, method, req)
		if err != nil {
			yield(zero, err)
			return
		}
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 64<<20)
		for sc.Scan() {
			line := bytes.TrimSpace(sc.Bytes())
			if len(line) == 0 {
				continue
			}
			if bytes.HasPrefix(line, []byte(`{"error":`)) {
				var env errorEnvelope
				if json.Unmarshal(line, &env) == nil && env.Error != nil {
					yield(zero, envelopeError(method, env.Error.Code, env.Error.Message, env.Error.Details))
					return
				}
			}
			var v T
			if err := json.Unmarshal(line, &v); err != nil {
				yield(zero, fmt.Errorf("%w: %s: decode line: %w", ErrUnavailable, method, err))
				return
			}
			if !yield(v, nil) {
				return
			}
		}
		if err := sc.Err(); err != nil {
			yield(zero, fmt.Errorf("%w: %s: %w", ErrUnavailable, method, err))
		}
	}
}

// collect drains a stream into a slice.
func collect[T any](seq iter.Seq2[T, error]) ([]T, error) {
	var out []T
	for v, err := range seq {
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
