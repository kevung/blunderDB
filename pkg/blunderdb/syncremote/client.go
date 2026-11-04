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
	return statusError(resp.StatusCode, raw)
}

func statusError(status int, raw []byte) error {
	var env struct {
		Error struct {
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
	case status == http.StatusTooManyRequests:
		return fmt.Errorf("%w%s", ErrRateLimited, msg)
	case status == http.StatusRequestEntityTooLarge:
		return fmt.Errorf("%w%s", ErrTooLarge, msg)
	default:
		return fmt.Errorf("%w (HTTP %d)%s", ErrUnavailable, status, msg)
	}
}
