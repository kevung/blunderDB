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
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrUnavailable is a call that did not get an answer from the daemon: the
// network failed, or what came back is not the /v1 error envelope. The
// operation may or may not have been applied — the caller re-reads before it
// retries a write.
var ErrUnavailable = errors.New("remote: daemon unavailable")

// ErrRefused is a daemon or a proxy in front of it that refuses the caller
// (HTTP 401 or 403): the credentials it expects are missing or wrong, which is
// no outage.
var ErrRefused = errors.New("remote: refused by the daemon or its proxy")

// ErrTooLarge is a request the daemon or its proxy refuses for its size (HTTP 413).
var ErrTooLarge = errors.New("remote: request too large")

// ErrRateLimited is the daemon's request rate or quota reached (HTTP 429, or
// the rate_limited and quota_exceeded codes).
var ErrRateLimited = errors.New("remote: daemon limit reached")

// visibleError is a transport-level failure whose text holds nothing of a
// backend: the daemon that serves a remote backend shows it as is rather than
// as "internal error".
type visibleError struct{ err error }

func (e visibleError) Error() string         { return e.err.Error() }
func (e visibleError) Unwrap() error         { return e.err }
func (e visibleError) ClientMessage() string { return e.err.Error() }
func (e visibleError) Is(target error) bool  { return target == storage.ErrInternal }

func visible(err error) error { return visibleError{err} }

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
	return fmt.Sprintf("remote: %s is not served through the remote backend: the route you called needs it, and it has no /v1 route of its own; send the request to the daemon itself (POST <daemon>/v1/<route>)", e.Method)
}

// ClientMessage lets the daemon show this message although the error is
// internal: it names a contract method, nothing of the backend behind it.
func (e *UnsupportedError) ClientMessage() string { return e.Error() }

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
	// HTTPClient replaces the default client; Timeout is then not applied.
	HTTPClient *http.Client
	// Timeout bounds the connection and the wait for a response's headers
	// (never its body, which a stream keeps open). Zero means
	// DefaultTimeout; a negative value, no bound.
	Timeout time.Duration
}

// DefaultTimeout bounds the connection and the wait for response headers when
// Options.Timeout is zero.
const DefaultTimeout = 60 * time.Second

func defaultHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	if timeout < 0 {
		return http.DefaultClient
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = (&net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}).DialContext
	tr.TLSHandshakeTimeout = timeout
	tr.ResponseHeaderTimeout = timeout
	return &http.Client{Transport: tr}
}

type client struct {
	base   string
	tenant string
	header http.Header
	http   *http.Client
	// idle is how long a body may stay silent; zero means no bound.
	idle time.Duration
}

// idleBody cancels its request when no byte arrives for d, so a daemon that
// goes silent after its headers fails the read instead of blocking it.
type idleBody struct {
	io.ReadCloser
	timer  *time.Timer
	d      time.Duration
	cancel context.CancelFunc
}

func (b *idleBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 && b.timer != nil {
		b.timer.Reset(b.d)
	}
	return n, err
}

func (b *idleBody) Close() error {
	if b.timer != nil {
		b.timer.Stop()
	}
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

func newClient(o Options) (*client, error) {
	if o.BaseURL == "" {
		return nil, fmt.Errorf("%w: remote: the daemon's URL is required (--dsn http://host:port, or $BLUNDERDB_DSN)", storage.ErrInvalid)
	}
	if u, err := url.Parse(o.BaseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%w: remote: --dsn must be the daemon's URL, http://host:port or https://host:port, got %q", storage.ErrInvalid, o.BaseURL)
	}
	hc := o.HTTPClient
	if hc == nil {
		hc = defaultHTTPClient(o.Timeout)
	}
	o.BaseURL = strings.TrimSuffix(strings.TrimRight(o.BaseURL, "/"), "/v1")
	c := &client{base: strings.TrimRight(o.BaseURL, "/"), tenant: o.Tenant, header: o.Header.Clone(), http: hc}
	if o.HTTPClient == nil && o.Timeout >= 0 {
		c.idle = o.Timeout
		if c.idle == 0 {
			c.idle = DefaultTimeout
		}
	}
	return c, nil
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
	rctx, cancel := context.WithCancel(ctx)
	hr, err := http.NewRequestWithContext(rctx, http.MethodPost, c.base+"/v1/"+method, body)
	if err != nil {
		cancel()
		return nil, visible(fmt.Errorf("%w: %s: %w", ErrUnavailable, method, err))
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
		cancel()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, visible(fmt.Errorf("%w: cannot reach %s (%s): %w", ErrUnavailable, c.base, method, err))
	}
	// The request context is released with the body; the timer, when there
	// is an idle bound, cancels it on silence.
	ib := &idleBody{ReadCloser: resp.Body, d: c.idle, cancel: cancel}
	if c.idle > 0 {
		ib.timer = time.AfterFunc(c.idle, cancel)
	}
	resp.Body = ib
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
		return statusError(method, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return refused(method, resp.StatusCode, env.Error.Message)
	}
	return envelopeError(method, env.Error.Code, env.Error.Message, env.Error.Details)
}

// statusError reads an answer that carries no error envelope: a proxy's page,
// most often. A refusal or a limit says so; anything else is an outage.
func statusError(method string, status int) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return refused(method, status, "")
	case http.StatusTooManyRequests:
		return visible(fmt.Errorf("%w: %s: HTTP 429; wait and retry, or ask for a higher --rate-limit-rps/--rate-limit-burst on the daemon", ErrRateLimited, method))
	case http.StatusRequestEntityTooLarge:
		return visible(fmt.Errorf("%w: %s: HTTP 413, the request is too large for the daemon or its proxy; send fewer items at a time", ErrTooLarge, method))
	}
	return visible(fmt.Errorf("%w: %s: HTTP %d without an error envelope", ErrUnavailable, method, status))
}

func refused(method string, status int, message string) error {
	if message != "" {
		message = ": " + message
	}
	return visible(fmt.Errorf("%w: %s: HTTP %d%s; check the credentials the proxy expects (--header \"Authorization: Bearer …\") and the tenant (--scope)", ErrRefused, method, status, message))
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
	case "rate_limited", "quota_exceeded":
		return visible(fmt.Errorf("%w: %s: %s: %s; wait and retry, or raise --rate-limit-rps/--rate-limit-burst on the daemon", ErrRateLimited, method, code, message))
	case "forbidden", "unauthorized":
		return refused(method, http.StatusForbidden, message)
	default:
		return visible(fmt.Errorf("%w: %s: %s: %s", ErrUnavailable, method, code, message))
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
		return visible(fmt.Errorf("%w: %s: decode answer: %w", ErrUnavailable, method, err))
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
				yield(zero, visible(fmt.Errorf("%w: %s: decode line: %w", ErrUnavailable, method, err)))
				return
			}
			if !yield(v, nil) {
				return
			}
		}
		if err := sc.Err(); err != nil {
			yield(zero, visible(fmt.Errorf("%w: %s: %w", ErrUnavailable, method, err)))
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
