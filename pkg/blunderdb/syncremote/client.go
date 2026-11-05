package syncremote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The reasons a call fails, for the window to word each one: none carries the
// token, which only ever travels in the Authorization header.
var (
	// ErrRefused: the server rejects the token (401, 403): missing, expired or revoked.
	ErrRefused = errors.New("syncremote: the server refuses the token")
	// ErrUnavailable: no usable answer (network, timeout, 5xx, not the /v1 envelope).
	ErrUnavailable = errors.New("syncremote: the server does not answer")
	// ErrRateLimited: the server's limit is reached (429); retry later.
	ErrRateLimited = errors.New("syncremote: the server's limit is reached")
	// ErrTooLarge: the server itself refuses the page for its size (413 with the
	// code payload_too_large or record_too_large).
	ErrTooLarge = errors.New("syncremote: the page is too large for the server")
	// ErrInterrupted: the connection broke or timed out while the request was
	// being sent, after it was made; a smaller page may get through. It is also
	// an ErrUnavailable.
	ErrInterrupted = errors.New("syncremote: the request was interrupted while being sent")
	// ErrBlocked: a 413 that does not carry the server's own code, so something
	// between (a proxy, a firewall) limits the size of a request; no record is
	// to blame, the run stops.
	ErrBlocked = errors.New("syncremote: something between this machine and the server refuses a request for its size")
	// ErrAllowance: the account's allowance is used up (402); retrying cannot help.
	ErrAllowance = errors.New("syncremote: the allowance is exceeded")
	// ErrToolLocked: the server holds this library's tools while a match is played (409); retry after it.
	ErrToolLocked = errors.New("syncremote: the server is locked during a match")
	// ErrInvalidRecords: the server refuses the content of the records (422
	// invalid_records), not their size; a record alone that it refuses is set aside.
	ErrInvalidRecords = errors.New("syncremote: the server refuses the content of the records")
	// ErrServerRefusal: the server declines the call for a reason of its own that
	// is not about the token (a 403 with a code such as a tenant that keeps no
	// journal): its words are shown, and the run does not go on.
	ErrServerRefusal = errors.New("syncremote: the server declines the call")
	// ErrBadAddress: the address is not usable (not http(s), embeds credentials or a query).
	ErrBadAddress = errors.New("syncremote: invalid server address")
)

// Client talks to one server with one token.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// maxAnswer bounds what is read from one answer: a record of the 256 MiB a
// server may serve, and its envelope.
const maxAnswer = 257 << 20

// NewClient checks the address. https is required unless the server is on this
// machine: a bearer token must not cross a network in clear.
func NewClient(address, token string, hc *http.Client) (*Client, error) {
	base, err := NormalizeAddress(address)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("%w: no token", ErrRefused)
	}
	if hc == nil {
		hc = &http.Client{Timeout: 2 * time.Minute}
	}
	// A redirect would carry the token to wherever it points, in clear if it
	// leaves https: it is an answer to report, not to follow.
	own := *hc
	own.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	hc = &own
	return &Client{base: base, token: strings.TrimSpace(token), http: hc}, nil
}

// NormalizeAddress trims the address and its trailing slash, and refuses what
// cannot carry a token safely.
func NormalizeAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	u, err := url.Parse(address)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("%w: an http(s) address is expected", ErrBadAddress)
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%w: no credentials, query or fragment in the address", ErrBadAddress)
	}
	if u.Scheme == "http" && !isLoopback(u.Hostname()) {
		return "", fmt.Errorf("%w: use https (http is for this machine only)", ErrBadAddress)
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Export asks for the page of the remote journal after a cursor.
func (c *Client) Export(ctx context.Context, after storage.Cursor, limit int) (Page, error) {
	var page Page
	err := c.call(ctx, "changeset.export", struct {
		Cursor storage.Cursor `json:"cursor"`
		Limit  int            `json:"limit"`
	}{after, limit}, &page)
	return page, err
}

// Import sends records to be merged and returns the remote's report.
func (c *Client) Import(ctx context.Context, records []changeset.Record) (changeset.Report, error) {
	var rep changeset.Report
	err := c.call(ctx, "changeset.import", struct {
		Records []changeset.Record `json:"records"`
	}{records}, &rep)
	return rep, err
}

func (c *Client) call(ctx context.Context, method string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/v1/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBadAddress, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	connected := false
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{GotConn: func(httptrace.GotConnInfo) { connected = true }}))
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// The transport's error repeats the URL, never a header.
		if connected {
			return fmt.Errorf("%w: %w: %w", ErrUnavailable, ErrInterrupted, err)
		}
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswer))
	if err != nil {
		return fmt.Errorf("%w: reading the answer: %w", ErrUnavailable, err)
	}
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("%w: the answer is not a changeset: %w", ErrUnavailable, err)
		}
		return nil
	}
	return statusError(resp.StatusCode, resp.Header, raw)
}

// rateLimited is ErrRateLimited that remembers how long the server asked to wait.
type rateLimited struct {
	msg        string
	after      time.Duration
	inProgress bool
}

func (e *rateLimited) Error() string             { return ErrRateLimited.Error() + e.msg }
func (e *rateLimited) Unwrap() error             { return ErrRateLimited }
func (e *rateLimited) RetryAfter() time.Duration { return e.after }

// defaultInProgressWait is the wait when the server says an import is still
// running and does not say for how long.
const defaultInProgressWait = 2 * time.Second

// importInProgress says the server is still applying an earlier import of this
// account, one a broken connection left in flight for up to 15 s.
func importInProgress(err error) bool {
	var r *rateLimited
	return errors.As(err, &r) && r.inProgress
}

// allowance is ErrAllowance with the figures the server gave: the window words
// it in its own language from them, the server's message is only a fallback.
type allowance struct {
	msg string
	AllowanceFigures
}

func (e *allowance) Error() string {
	if e.msg == "" {
		return ErrAllowance.Error()
	}
	return e.msg
}
func (e *allowance) Unwrap() error { return ErrAllowance }

// AllowanceMessage is the server's own wording of an exceeded allowance, without the run's prefixes.
func AllowanceMessage(err error) string {
	var a *allowance
	if errors.As(err, &a) {
		return a.Error()
	}
	return err.Error()
}

// AllowanceFigures are the fields of a 402 allowance_exceeded: the axis that is
// exceeded, what the plan allows, what the account holds and what the call adds.
type AllowanceFigures struct {
	Axis    string `json:"axis"`
	Allowed int64  `json:"allowed"`
	Held    int64  `json:"held"`
	New     int64  `json:"new"`
}

// AllowanceOf returns the figures of an exceeded allowance, false when the
// server gave none.
func AllowanceOf(err error) (AllowanceFigures, bool) {
	var a *allowance
	if errors.As(err, &a) && a.Axis != "" {
		return a.AllowanceFigures, true
	}
	return AllowanceFigures{}, false
}

// RetryAfter is the wait a rate-limited answer asked for.
func RetryAfter(err error) (time.Duration, bool) {
	var r interface{ RetryAfter() time.Duration }
	if errors.As(err, &r) && r.RetryAfter() > 0 {
		return r.RetryAfter(), true
	}
	return 0, false
}

// parseRetryAfter reads delta-seconds or an HTTP date; zero when absent or unusable.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		if n < 0 {
			return 0
		}
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// tokenCodes are the codes of a 403 that mean the token, not a decision of the server.
var tokenCodes = map[string]bool{"forbidden": true, "unauthorized": true, "invalid_token": true, "token_expired": true, "token_revoked": true, "token_invalid": true, "missing_token": true}

func statusError(status int, h http.Header, raw []byte) error {
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			AllowanceFigures
			Details *AllowanceFigures `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &env)
	msg := env.Error.Message
	if msg != "" {
		msg = ": " + msg
	}
	switch {
	case status >= 300 && status < 400:
		return fmt.Errorf("%w (HTTP %d): the server redirects, give its final address", ErrBadAddress, status)
	case status == http.StatusGone:
		return fmt.Errorf("%w%s", storage.ErrCursorStale, msg)
	case status == http.StatusForbidden && env.Error.Code != "" && !tokenCodes[env.Error.Code]:
		return fmt.Errorf("%w (HTTP %d)%s", ErrServerRefusal, status, msg)
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return fmt.Errorf("%w (HTTP %d)%s", ErrRefused, status, msg)
	case status == http.StatusPaymentRequired:
		f := env.Error.AllowanceFigures
		if f.Axis == "" && env.Error.Details != nil {
			f = *env.Error.Details
		}
		return &allowance{msg: env.Error.Message, AllowanceFigures: f}
	case status == http.StatusConflict && env.Error.Code == "tool_locked_during_match":
		return fmt.Errorf("%w%s", ErrToolLocked, msg)
	case status == http.StatusTooManyRequests:
		r := &rateLimited{msg: msg, after: parseRetryAfter(h.Get("Retry-After")), inProgress: env.Error.Code == "import_in_progress"}
		if r.inProgress && r.after == 0 {
			r.after = defaultInProgressWait
		}
		return r
	case status == http.StatusUnprocessableEntity && env.Error.Code == "invalid_records":
		return fmt.Errorf("%w%s", ErrInvalidRecords, msg)
	case status == http.StatusRequestEntityTooLarge && (env.Error.Code == "payload_too_large" || env.Error.Code == "record_too_large"):
		return fmt.Errorf("%w%s", ErrTooLarge, msg)
	case status == http.StatusRequestEntityTooLarge:
		return fmt.Errorf("%w (HTTP %d)", ErrBlocked, status)
	// The server could not read the body it was sent: cut or too slow.
	case status == http.StatusBadRequest && env.Error.Code == "invalid_body":
		return fmt.Errorf("%w (HTTP %d)%s: %w", ErrUnavailable, status, msg, ErrInterrupted)
	default:
		return fmt.Errorf("%w (HTTP %d)%s", ErrUnavailable, status, msg)
	}
}
