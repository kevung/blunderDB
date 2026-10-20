package gui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/internal/server/webui"
)

// DefaultTournamentDisplayPort is where the Direction serves its read-only
// display unless told otherwise.
const DefaultTournamentDisplayPort = 8770

// displayRoute is the ONE /v1 route the club network reaches (ADR-0097): the
// Direction's display page, the same page the desktop writes to its folder.
// Everything else the engine serves, reads included, stays off the network.
const displayRoute = "/v1/directions.pageHtml"

// displayBodyLimit bounds what the display route reads before it decodes:
// its request is one integer.
const displayBodyLimit = 4 << 10

// TournamentDisplayConfig names the tournament shown and the port it is shown on.
type TournamentDisplayConfig struct {
	TournamentID int64 `json:"tournamentId"`
	Port         int   `json:"port"`
}

// TournamentDisplayStatus says whether the display listens, for which
// tournament, and where: URL is the address phones on the club network open
// (also drawn as a QR code, a PNG data URI), LocalURL the same page on this
// machine.
type TournamentDisplayStatus struct {
	Running      bool   `json:"running"`
	TournamentID int64  `json:"tournamentId"`
	URL          string `json:"url"`
	LocalURL     string `json:"localUrl"`
	QR           string `json:"qr"`
	Error        string `json:"error"`
}

// displayHost is the tournament display the desktop serves on the local
// network while the director asks for it.
type displayHost struct {
	mu     sync.Mutex
	engine liveEngine
	http   *http.Server
	status TournamentDisplayStatus
}

// displayHandler serves the web page and the display route of one tournament.
// The tenant is set here, never taken from the request: the desktop holds one
// file, and a visitor must not choose what he reads. A request for another
// tournament, or for any other route, is a 404 as if the route did not exist.
func displayHandler(engine http.Handler, tournamentID int64) http.Handler {
	mux := http.NewServeMux()
	page := webui.Handler()
	mux.Handle("GET "+webui.Prefix, page)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, displayPath(tournamentID), http.StatusFound)
	})
	mux.HandleFunc("POST "+displayRoute, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, displayBodyLimit+1))
		if err != nil || len(body) > displayBodyLimit {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		var req struct {
			TournamentID int64 `json:"tournamentId"`
		}
		if json.Unmarshal(body, &req) != nil || req.TournamentID != tournamentID {
			http.NotFound(w, r)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		r.Header.Set(middleware.TenantHeader, middleware.SingleTenantID)
		r.Header.Del(middleware.ReadTenantsHeader)
		engine.ServeHTTP(w, r)
	})
	return mux
}

// displayPath is the page's address for one tournament: the web front shows
// only that tournament when the query names it.
func displayPath(tournamentID int64) string {
	return webui.Prefix + "?" + url.Values{"tournoi": {strconv.FormatInt(tournamentID, 10)}}.Encode()
}

// lanAddress is the first IPv4 address of an interface that is up and not a
// loopback: the address a phone on the same network reaches. Empty when the
// machine has none (no network), and the page is then reachable here only.
func lanAddress() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLinkLocalUnicast() {
				return ip4.String()
			}
		}
	}
	return ""
}

// qrDataURI draws text as a QR code, a PNG data URI the panel shows as is.
func qrDataURI(text string) (string, error) {
	png, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// StartTournamentDisplay serves the display of one tournament on the local
// network, or moves it to another tournament or port. Off until asked:
// opening a port to the club network is the director's decision, and the
// daemon authenticates nobody (ADR-0005), so everyone on that network can
// read the page.
func (a *App) StartTournamentDisplay(cfg TournamentDisplayConfig) TournamentDisplayStatus {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		cfg.Port = DefaultTournamentDisplayPort
	}
	h := &a.display
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopLocked()
	if cfg.TournamentID <= 0 {
		h.status.Error = "no tournament to display"
		return h.status
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(cfg.Port)))
	if err != nil {
		h.status.Error = err.Error()
		return h.status
	}
	path := displayPath(cfg.TournamentID)
	host := lanAddress()
	if host == "" {
		host = "127.0.0.1"
	}
	lan := "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.Port)) + path
	qr, err := qrDataURI(lan)
	if err != nil {
		_ = ln.Close()
		h.status.Error = err.Error()
		return h.status
	}
	srv := &http.Server{Handler: displayHandler(&h.engine, cfg.TournamentID), ReadHeaderTimeout: 10 * time.Second}
	h.http = srv
	h.status = TournamentDisplayStatus{Running: true, TournamentID: cfg.TournamentID, URL: lan, QR: qr,
		LocalURL: fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Port, path)}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("tournament display stopped", "err", err)
		}
	}()
	return h.status
}

// StopTournamentDisplay closes the display; the port is released.
func (a *App) StopTournamentDisplay() TournamentDisplayStatus {
	a.display.mu.Lock()
	defer a.display.mu.Unlock()
	a.display.stopLocked()
	return a.display.status
}

// GetTournamentDisplayStatus reports the display's state.
func (a *App) GetTournamentDisplayStatus() TournamentDisplayStatus {
	a.display.mu.Lock()
	defer a.display.mu.Unlock()
	return a.display.status
}

func (h *displayHost) stopLocked() {
	h.status = TournamentDisplayStatus{}
	if h.http == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = h.http.Shutdown(ctx)
	h.http = nil
}

// stopDisplay stops the display and drops its engine, at shutdown.
func (a *App) stopDisplay() {
	a.display.mu.Lock()
	a.display.stopLocked()
	a.display.mu.Unlock()
	a.display.engine.close()
}
