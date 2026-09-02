package server

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const healthcheckUsage = `blunderdb healthcheck — ask a running daemon whether it is ready.

Performs one GET on the daemon's /readyz endpoint and exits 0 when the answer
is 200 (storage reachable, schema version as expected), 1 otherwise: the
storage is down, the schema is stale, or nothing listens at the address. It
is what the container image's HEALTHCHECK runs — the image is distroless and
ships no curl or wget — and works just as well from a shell or a systemd unit.

The address defaults to the one the daemon itself would listen on: --addr, or
BLUNDERDB_ADDR, or :8080. A listen address with no host (":8080") or a
wildcard host (0.0.0.0, [::]) is probed on the loopback interface.

Usage:
  blunderdb healthcheck [flags]

Flags:
`

// RunHealthcheck parses the `healthcheck` subcommand flags and probes a
// daemon's /readyz. args are the arguments after "healthcheck". A non-nil
// error means "not ready"; the message says why so an operator running it
// by hand sees the reason (`docker inspect` also records the last output).
func RunHealthcheck(args []string) error {
	fs := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, healthcheckUsage)
		fs.PrintDefaults()
	}
	var (
		addr    = fs.String("addr", envOr("BLUNDERDB_ADDR", defaultAddr), "address the daemon listens on (host:port)")
		timeout = fs.Duration("timeout", 2*time.Second, "give up after this long")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	status, err := Healthcheck(ctx, *addr)
	if err != nil {
		return err
	}
	fmt.Println(status)
	return nil
}

// Healthcheck performs one GET on /readyz at the daemon listening on addr
// and returns the daemon's status word ("ready") on success. It fails when
// the daemon is unreachable, answers anything but 200, or does not answer
// before ctx expires; the error carries the daemon's own status word
// ("down", "version_mismatch", …) when there is one.
func Healthcheck(ctx context.Context, addr string) (string, error) {
	url := ProbeURL(addr) + "/readyz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("healthcheck: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("healthcheck: %w", err)
	}
	defer resp.Body.Close()

	// The body is small JSON ({"status": "..."}); cap the read all the same
	// so a stray server on the port cannot make the probe swallow a stream.
	var body struct {
		Status string `json:"status"`
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = json.Unmarshal(raw, &body)

	if resp.StatusCode != http.StatusOK {
		if body.Status == "" {
			return "", fmt.Errorf("healthcheck: %s answered %s", url, resp.Status)
		}
		return "", fmt.Errorf("healthcheck: %s answered %s (%s)", url, resp.Status, body.Status)
	}
	if body.Status == "" {
		return "", errors.New("healthcheck: " + url + " answered 200 but not with the daemon's JSON — is this a blunderdb daemon?")
	}
	return body.Status, nil
}

// ProbeURL turns a listen address, as `serve --addr` takes it, into the base
// URL a client on the same machine reaches it at. A missing or wildcard host
// (":8080", "0.0.0.0:8080", "[::]:8080") becomes the loopback address of the
// same family: the probe runs in the daemon's own network namespace, and the
// loopback is the one interface a wildcard listener is certain to cover. A
// concrete host is kept as is. An address without a port gets the default
// one, and a value already carrying a scheme is trusted verbatim.
func ProbeURL(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// No port at all (a bare host, or empty): use the default port.
		host, port = strings.Trim(addr, "[]"), strings.TrimPrefix(defaultAddr, ":")
	}
	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}
	return "http://" + net.JoinHostPort(host, port)
}
