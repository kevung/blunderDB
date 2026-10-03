package mcp

import (
	"context"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options configures a blunderDB MCP server.
type Options struct {
	// Write offers the tools that change the database. Off by default: a
	// model reads freely, and writes only where its operator said so, as
	// `serve --direction` gates the gestures of a Direction.
	Write bool
	// Tenant is the scope used when a call carries no X-Tenant-ID ("1" for a
	// local file). Empty makes the header mandatory — the HTTP mount.
	Tenant string
	// Version is reported to the client in the initialize handshake.
	Version string
	// Extensions add tools beyond the built-in set, e.g. the display tools a
	// GUI-hosted server offers (open a view, show a position). Each receives
	// the Toolbox, so its tools obey the same write gate.
	Extensions []Extension
}

// Extension registers further tools on a server.
type Extension func(tb *Toolbox)

// Toolbox is what a tool set is registered against: the SDK server, the engine
// the tools call, and the write gate.
type Toolbox struct {
	Server *sdk.Server
	Engine *Engine
	write  bool
}

// Kind says whether a tool only reads or also writes; a write tool is
// registered only when Options.Write is set.
type Kind int

const (
	Reads Kind = iota
	Writes
)

// Add registers one typed tool, filling its annotations from kind. It reports
// whether the tool was registered (a write tool on a read-only server is not).
func Add[In any](tb *Toolbox, kind Kind, t *sdk.Tool, h func(ctx context.Context, req *sdk.CallToolRequest, in In) (any, error)) bool {
	if kind == Writes && !tb.write {
		return false
	}
	if t.Annotations == nil {
		t.Annotations = &sdk.ToolAnnotations{}
	}
	t.Annotations.ReadOnlyHint = kind == Reads
	t.Annotations.IdempotentHint = kind == Reads
	closed := false
	t.Annotations.OpenWorldHint = &closed
	if kind == Writes {
		notDestructive := false
		t.Annotations.DestructiveHint = &notDestructive
	}
	sdk.AddTool(tb.Server, t, func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, any, error) {
		out, err := h(ctx, req, in)
		return nil, out, err
	})
	return true
}

// NewServer builds the MCP server over an engine handler: the built-in tools,
// then each extension.
func NewServer(handler http.Handler, opts Options) *sdk.Server {
	version := opts.Version
	if version == "" {
		version = "dev"
	}
	srv := sdk.NewServer(&sdk.Implementation{Name: "blunderdb", Title: "blunderDB", Version: version},
		&sdk.ServerOptions{Instructions: instructions})
	tb := &Toolbox{Server: srv, Engine: NewEngine(handler, opts.Tenant), write: opts.Write}
	registerBuiltins(tb)
	for _, ext := range opts.Extensions {
		ext(tb)
	}
	return srv
}

// NewHTTPHandler serves the MCP server over streamable HTTP, stateless: every
// POST is self-contained, so no session pins a client to one daemon instance
// and the tenant is read from each request's own X-Tenant-ID.
//
// The SDK's DNS-rebinding guard is off: it refuses a loopback connection whose
// Host is not loopback, which is exactly what an authenticating reverse proxy
// on the same machine sends. The daemon's trust model is that proxy (ADR-0005).
func NewHTTPHandler(handler http.Handler, opts Options) http.Handler {
	srv := NewServer(handler, opts)
	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true, DisableLocalhostProtection: true})
}

const instructions = `blunderDB is a backgammon blunder database: positions from imported matches,
each with its engine analysis, grouped by match, tournament and collection.
Start with database_overview, find positions with search_positions (its description
holds the query grammar), then read one with get_position and explain_error.
Errors are in millipoints (mp) of normalised equity; PR is the performance rating
(lower is better).`
