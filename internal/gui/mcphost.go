package gui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultMCPPort is where the GUI serves its MCP tools unless told otherwise.
const DefaultMCPPort = 8765

// Events the display tools send the frontend; it acts on its own state.
const (
	mcpOpenViewEvent     = "mcp:open-view"
	mcpShowPositionEvent = "mcp:show-position"
)

var errNoDatabase = errors.New("no database is open in blunderDB")

// liveEngine is the engine handler over whatever file the GUI has open: the
// daemon's own handler chain, rebuilt when the open file changes, on the
// Storage that borrows the GUI's connection. The tools thus reach the same
// rows the window shows, through /v1 like every other MCP transport.
type liveEngine struct {
	db  *database.Database
	mu  sync.Mutex
	gen uint64
	srv *server.Server
}

func (e *liveEngine) handler() (http.Handler, error) {
	if e.db == nil {
		return nil, errNoDatabase
	}
	st, gen := database.CurrentStore(e.db)
	if st == nil {
		return nil, errNoDatabase
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.srv != nil && e.gen == gen {
		return e.srv.Handler(), nil
	}
	if e.srv != nil {
		e.srv.Close()
		e.srv = nil
	}
	srv, err := server.New(server.Options{Storage: st, SingleTenant: true, SessionPerCall: true,
		Logger: slog.Default().With("component", "mcp-gui")})
	if err != nil {
		return nil, err
	}
	e.srv, e.gen = srv, gen
	return srv.Handler(), nil
}

func (e *liveEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, err := e.handler()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"code":"unavailable","message":%q}`, err.Error())
		return
	}
	h.ServeHTTP(w, r)
}

func (e *liveEngine) close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.srv != nil {
		e.srv.Close()
		e.srv = nil
	}
}

// guiDisplay drives the window from the display tools, by Wails event.
type guiDisplay struct {
	a *App
	// viewName, when set and non-empty, names every view opened: the
	// assistant names its views after the user's sentence, not after the
	// model's label.
	viewName func() string
}

func (d guiDisplay) OpenView(name, query string) error {
	if d.a.ctx == nil {
		return errors.New("the blunderDB window is not ready")
	}
	if d.viewName != nil {
		if n := mcp.ViewName(d.viewName()); n != "" {
			name = n
		}
	}
	runtime.EventsEmit(d.a.ctx, mcpOpenViewEvent, map[string]string{"name": name, "query": query})
	return nil
}

func (d guiDisplay) ShowPosition(id int64) error {
	if d.a.ctx == nil {
		return errors.New("the blunderDB window is not ready")
	}
	runtime.EventsEmit(d.a.ctx, mcpShowPositionEvent, map[string]int64{"id": id})
	return nil
}

// MCPHostConfig is the localhost MCP server's setting, kept by the frontend.
type MCPHostConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
	// Write offers the tools that change the database, as `mcp --write`.
	Write bool `json:"write"`
}

// MCPHostStatus says whether the server listens, and where.
type MCPHostStatus struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
	Write   bool   `json:"write"`
	Error   string `json:"error"`
}

// mcpHost is the MCP server the GUI serves on localhost while it runs.
type mcpHost struct {
	mu     sync.Mutex
	engine liveEngine
	cfg    MCPHostConfig
	http   *http.Server
	url    string
	err    string
}

// newMCPServer builds the GUI's tool set: the built-in tools over the live
// engine plus the display tools. Tenant 1: the GUI holds one local file.
func (a *App) newMCPServer(write bool, display guiDisplay) *sdk.Server {
	return mcp.NewServer(&a.mcp.engine, mcp.Options{Write: write, Tenant: "1", Version: "gui",
		Extensions: []mcp.Extension{mcp.DisplayTools(display)}})
}

// mcpHTTPHandler serves the tools at /mcp. The SDK's anti-rebinding guard
// stays on (a page the user visits must not reach localhost through a DNS
// name it controls) and the cross-origin protection wrapped around it refuses a
// browser's cross-origin POST: on localhost the attacker is a web page (ADR-0059).
func mcpHTTPHandler(srv *sdk.Server) http.Handler {
	h := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	mux := http.NewServeMux()
	mux.Handle("/mcp", http.NewCrossOriginProtection().Handler(h))
	return mux
}

// ConfigureMCPHost starts, restarts or stops the localhost MCP server. Off
// unless enabled: listening on a port is the user's decision.
func (a *App) ConfigureMCPHost(cfg MCPHostConfig) MCPHostStatus {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		cfg.Port = DefaultMCPPort
	}
	a.mcp.mu.Lock()
	defer a.mcp.mu.Unlock()
	if cfg == a.mcp.cfg && (a.mcp.http != nil) == cfg.Enabled {
		return a.mcp.statusLocked()
	}
	a.mcp.stopLocked()
	a.mcp.cfg = cfg
	if !cfg.Enabled {
		return a.mcp.statusLocked()
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		a.mcp.err = err.Error()
		return a.mcp.statusLocked()
	}
	srv := &http.Server{Handler: mcpHTTPHandler(a.newMCPServer(cfg.Write, guiDisplay{a: a})),
		ReadHeaderTimeout: 10 * time.Second}
	a.mcp.http = srv
	a.mcp.url = "http://" + addr + "/mcp"
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("mcp host stopped", "err", err)
		}
	}()
	return a.mcp.statusLocked()
}

// GetMCPHostStatus reports the localhost MCP server's state.
func (a *App) GetMCPHostStatus() MCPHostStatus {
	a.mcp.mu.Lock()
	defer a.mcp.mu.Unlock()
	return a.mcp.statusLocked()
}

func (h *mcpHost) statusLocked() MCPHostStatus {
	st := MCPHostStatus{Running: h.http != nil, Write: h.cfg.Write, Error: h.err}
	if st.Running {
		st.URL = h.url
	}
	return st
}

func (h *mcpHost) stopLocked() {
	h.err = ""
	if h.http == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = h.http.Shutdown(ctx)
	h.http = nil
	h.url = ""
}

// stopMCP stops the localhost server and drops the engine, at shutdown.
func (a *App) stopMCP() {
	a.mcp.mu.Lock()
	a.mcp.stopLocked()
	a.mcp.cfg = MCPHostConfig{}
	a.mcp.mu.Unlock()
	a.mcp.engine.close()
}
