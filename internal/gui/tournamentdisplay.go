package gui

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/wailsapp/wails/v2/pkg/runtime"

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

// TournamentDisplayConfig names the tournament shown, the port it is shown on, and the
// address the QR code points to — one of TournamentDisplayStatus.Addresses, empty for the
// default one.
type TournamentDisplayConfig struct {
	TournamentID int64  `json:"tournamentId"`
	Port         int    `json:"port"`
	Address      string `json:"address"`
}

// TournamentDisplayStatus says whether the display listens, for which
// tournament, and where: URL is the address phones on the club network open
// (also drawn as a QR code, a PNG data URI), LocalURL the same page on this
// machine. Addresses are the machine's local-network IPv4 addresses, Address
// the one URL uses; with none, URL and QR are empty and the page is reachable
// on this machine only. PublicAddresses are the Addresses outside the private
// ranges: a page served there may be reachable beyond the club.
//
// Error is a code, never a system message: the panel words it (Display*Err),
// the raw cause goes to the log. ErrorPort is the port a "port_in_use" is about.
type TournamentDisplayStatus struct {
	Running         bool     `json:"running"`
	TournamentID    int64    `json:"tournamentId"`
	Addresses       []string `json:"addresses"`
	PublicAddresses []string `json:"publicAddresses"`
	Address         string   `json:"address"`
	URL             string   `json:"url"`
	LocalURL        string   `json:"localUrl"`
	QR              string   `json:"qr"`
	Error           string   `json:"error"`
	ErrorPort       int      `json:"errorPort"`
}

// The codes TournamentDisplayStatus.Error takes.
const (
	DisplayNoTournamentErr = "no_tournament"
	DisplayPortInUseErr    = "port_in_use"
	DisplayFailedErr       = "failed"
)

// displayHost is the tournament display the desktop serves on the local
// network while the director asks for it.
type displayHost struct {
	mu     sync.Mutex
	engine liveEngine
	http   *http.Server
	// served is closed once every Serve loop of http has returned, so a stop that is followed by
	// a listen on the same port finds the port free.
	served chan struct{}
	port   int
	status TournamentDisplayStatus
}

// displayHandler serves the web page and the display route of one tournament.
// The tenant is set here, never taken from the request: the desktop holds one
// file, and a visitor must not choose what he reads. A request for another
// tournament, or for any other route, is a 404 as if the route did not exist.
func displayHandler(engine http.Handler, tournamentID int64) http.Handler {
	mux := http.NewServeMux()
	page := webui.Handler()
	mux.HandleFunc("GET "+webui.Prefix, func(w http.ResponseWriter, r *http.Request) {
		// Files of the page are served as they are; every address that would open the page
		// itself goes to this tournament's, so the club network never reaches the full
		// application, whose every button would answer 404 here.
		if !strings.HasPrefix(r.URL.Path, webui.Prefix+"assets/") && r.URL.Query().Get("tournoi") != strconv.FormatInt(tournamentID, 10) {
			http.Redirect(w, r, displayPath(tournamentID), http.StatusFound)
			return
		}
		page.ServeHTTP(w, r)
	})
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

// virtualPrefixes name the interfaces a phone on the club network never reaches: container
// bridges, virtual machines' networks, tunnels and VPNs. Their addresses are still offered,
// last, since a machine is free to name its interfaces otherwise.
var virtualPrefixes = []string{"docker", "br-", "virbr", "veth", "vboxnet", "vmnet", "tun", "tap", "wg", "zt", "tailscale", "lxc", "lxd", "cni", "flannel"}

func isVirtual(name string) bool {
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// lanAddresses lists the IPv4 addresses of the interfaces that are up and not loopback. The
// first is the default the QR code points to: a private address on a physical interface, then
// a private one on a virtual interface, then the others — an address outside the private ranges
// is offered only when the machine has no other, since a page served there may be reachable
// beyond the club. Empty when the machine has no network, and the page is then reachable here
// only.
func lanAddresses() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var physPriv, virtPriv, physPub, virtPub []string
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
			ip4 := ipnet.IP.To4()
			if ip4 == nil || ip4.IsLinkLocalUnicast() || ip4.IsLoopback() {
				continue
			}
			switch virtual, private := isVirtual(iface.Name), ip4.IsPrivate(); {
			case private && !virtual:
				physPriv = append(physPriv, ip4.String())
			case private:
				virtPriv = append(virtPriv, ip4.String())
			case !virtual:
				physPub = append(physPub, ip4.String())
			default:
				virtPub = append(virtPub, ip4.String())
			}
		}
	}
	return slices.Concat(physPriv, virtPriv, physPub, virtPub)
}

// publicAddresses are those of addrs outside the private ranges.
func publicAddresses(addrs []string) []string {
	var out []string
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && !ip.IsPrivate() {
			out = append(out, a)
		}
	}
	return out
}

// chooseAddress is the address asked for when the machine has it, the default otherwise.
func chooseAddress(addrs []string, asked string) string {
	if slices.Contains(addrs, asked) {
		return asked
	}
	if len(addrs) > 0 {
		return addrs[0]
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
	fail := func(code string, cause error) TournamentDisplayStatus {
		if cause != nil {
			slog.Warn("tournament display could not start", "port", cfg.Port, "err", cause)
		}
		st := h.status
		st.Error, st.ErrorPort = code, cfg.Port
		h.status = st
		return st
	}
	if cfg.TournamentID <= 0 {
		return fail(DisplayNoTournamentErr, nil)
	}
	// The new port is taken BEFORE the running display is dropped: moving to a port that is
	// busy must not switch off the phones already looking. On the same port the old listener
	// has to go first, and is waited for.
	if h.http != nil && h.port == cfg.Port {
		h.stopLocked()
	}
	addrs := lanAddresses()
	host := chooseAddress(addrs, cfg.Address)
	// Only the chosen address and this machine listen: not every interface, IPv6 included.
	lns, err := listenDisplay(host, cfg.Port)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) || strings.Contains(err.Error(), "address already in use") ||
			strings.Contains(err.Error(), "Only one usage of each socket address") {
			return fail(DisplayPortInUseErr, err)
		}
		return fail(DisplayFailedErr, err)
	}
	path := displayPath(cfg.TournamentID)
	var lan, qr string
	if host != "" {
		lan = "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.Port)) + path
		if qr, err = qrDataURI(lan); err != nil {
			for _, ln := range lns {
				_ = ln.Close()
			}
			return fail(DisplayFailedErr, err)
		}
	}
	h.stopLocked()
	srv := &http.Server{Handler: displayHandler(&h.engine, cfg.TournamentID), ReadHeaderTimeout: 10 * time.Second}
	h.http, h.port = srv, cfg.Port
	h.served = make(chan struct{})
	h.status = TournamentDisplayStatus{Running: true, TournamentID: cfg.TournamentID,
		Addresses: addrs, PublicAddresses: publicAddresses(addrs), Address: host, URL: lan, QR: qr,
		LocalURL: fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Port, path)}
	var wg sync.WaitGroup
	for _, ln := range lns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Warn("tournament display stopped", "err", err)
			}
		}()
	}
	go func(done chan struct{}) { wg.Wait(); close(done) }(h.served)
	return h.status
}

// listenDisplay listens on this machine and, when the machine has a local address, on that one.
func listenDisplay(host string, port int) ([]net.Listener, error) {
	var lns []net.Listener
	for _, ip := range []string{"127.0.0.1", host} {
		if ip == "" {
			continue
		}
		ln, err := net.Listen("tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
		if err != nil {
			for _, l := range lns {
				_ = l.Close()
			}
			return nil, err
		}
		lns = append(lns, ln)
	}
	return lns, nil
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

// stopLocked closes the listener and drops the connections at once: it also runs while the
// open file is being switched, and a graceful shutdown would wait on requests that wait on
// that very switch.
func (h *displayHost) stopLocked() {
	h.status = TournamentDisplayStatus{}
	if h.http == nil {
		return
	}
	_ = h.http.Close()
	// Close does not wait for handlers, only the Serve loops, which return at once: waiting for
	// them is what frees the port for an immediate restart.
	<-h.served
	h.http = nil
}

// displayStatusEvent tells the Direction panel the display's new state when it changes
// without the panel asking.
const displayStatusEvent = "direction:network-display"

// stopDisplayForSwitch stops the display before the open file changes: it was started on one
// file's tournament, and the same id in the next file is another tournament.
func (a *App) stopDisplayForSwitch() {
	a.display.mu.Lock()
	running := a.display.http != nil
	a.display.stopLocked()
	st := a.display.status
	a.display.mu.Unlock()
	if running && a.ctx != nil {
		runtime.EventsEmit(a.ctx, displayStatusEvent, st)
	}
}

// stopDisplay stops the display and drops its engine, at shutdown.
func (a *App) stopDisplay() {
	a.display.mu.Lock()
	a.display.stopLocked()
	a.display.mu.Unlock()
	a.display.engine.close()
}
