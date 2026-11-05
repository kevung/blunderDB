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
	// ErrTooLarge: the server refuses the page for its size (413).
	ErrTooLarge = errors.New("syncremote: the page is too large for the server")
	// ErrAllowance: the account's allowance is used up (402); retrying cannot help.
	ErrAllowance = errors.New("syncremote: the allowance is exceeded")
	// ErrToolLocked: the server holds this library's tools while a match is played (409); retry after it.
	ErrToolLocked = errors.New("syncremote: the server is locked during a match")
	// ErrBadAddress: the address is not usable (not http(s), embeds credentials or a query).
	ErrBadAddress = errors.New("syncremote: invalid server address")
)

// Client talks to one server with one token.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// maxAnswer bounds what is read from one answer.
const maxAnswer = 256 << 20

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
func (c *Client) Export(ctx context.Context, after storage.Cursor, limit int) (changeset.Set, error) {
	var set changeset.Set
	err := c.call(ctx, "changeset.export", struct {
		Cursor storage.Cursor `json:"cursor"`
		Limit  int            `json:"limit"`
	}{after, limit}, &set)
	return set, err
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
		return fmt.Errorf("%w: %v", ErrBadAddress, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// The transport's error repeats the URL, never a header.
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswer))
	if err != nil {
		return fmt.Errorf("%w: reading the answer: %v", ErrUnavailable, err)
	}
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("%w: the answer is not a changeset: %v", ErrUnavailable, err)
		}
		return nil
	}
	return statusError(resp.StatusCode, resp.Header, raw)
}

// rateLimited is ErrRateLimited that remembers how long the server asked to wait.
type rateLimited struct {
	msg   string
	after time.Duration
}

func (e *rateLimited) Error() string             { return ErrRateLimited.Error() + e.msg }
func (e *rateLimited) Unwrap() error             { return ErrRateLimited }
func (e *rateLimited) RetryAfter() time.Duration { return e.after }

// allowance is ErrAllowance worded by the server: its message is shown as is.
type allowance struct{ msg string }

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

func statusError(status int, h http.Header, raw []byte) error {
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
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
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return fmt.Errorf("%w (HTTP %d)%s", ErrRefused, status, msg)
	case status == http.StatusPaymentRequired:
		return &allowance{msg: env.Error.Message}
	case status == http.StatusConflict && env.Error.Code == "tool_locked_during_match":
		return fmt.Errorf("%w%s", ErrToolLocked, msg)
	case status == http.StatusTooManyRequests:
		return &rateLimited{msg: msg, after: parseRetryAfter(h.Get("Retry-After"))}
	case status == http.StatusRequestEntityTooLarge:
		return fmt.Errorf("%w%s", ErrTooLarge, msg)
	default:
		return fmt.Errorf("%w (HTTP %d)%s", ErrUnavailable, status, msg)
	}
}
