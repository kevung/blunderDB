package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpPath is where the daemon serves its MCP tools. Outside /v1 because it is
// not a family.method of the contract, but tenant-scoped all the same: it is
// not a public path, so the tenant middleware demands X-Tenant-ID as on /v1.
const mcpPath = "/mcp"

func (s *Server) mcpRoutes() []route {
	// The tools call /v1 through the fully chained handler, built after the
	// routes: resolved at request time.
	engine := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.http.Handler.ServeHTTP(w, r) })
	h := mcp.NewHTTPHandler(engine, mcp.Options{Write: s.opts.MCPWrite})
	return []route{{http.MethodPost, mcpPath, h.ServeHTTP}}
}

const mcpUsage = `Usage: blunderdb mcp --db <file> [options]

Serve a database's tools to an AI assistant over the Model Context Protocol,
on stdin/stdout. The assistant starts this command itself; for Claude Code:

  claude mcp add blunderdb -- blunderdb mcp --db /path/to/my.db

The tools search positions with the application's query grammar, read a
position and its analysis, explain an error, compute a player's statistics,
list matches, tournaments and collections, and run a quiz. They only read,
unless --write is given: then they may also save a position, comment one,
create and fill a collection.

Options:
`

// RunMCP implements `blunderdb mcp`: the MCP server over stdio, on a local
// SQLite file. The tools dispatch to the same handlers as `call` and the daemon.
func RunMCP(args []string, version string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Print(mcpUsage)
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb mcp --db my.db")
		fmt.Println("  blunderdb mcp --db my.db --write")
	}
	dbPath := fs.String("db", "", "Path to the database file (required)")
	write := fs.Bool("write", false, "Also offer the tools that change the database")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if _, err := os.Stat(*dbPath); err != nil {
		return fmt.Errorf("mcp: %w", err)
	}
	ctx := context.Background()
	st, err := OpenStorage(ctx, "sqlite", *dbPath, false)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return fmt.Errorf("mcp: migrate: %w", err)
	}
	// stdout carries the protocol: every log line goes to stderr, warnings only.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	srv, err := New(Options{Storage: st, Logger: logger, SessionPerCall: true})
	if err != nil {
		return err
	}
	defer srv.Close()
	return ServeMCP(ctx, srv.Handler(), mcp.Options{Write: *write, Tenant: "1", Version: version}, &sdk.StdioTransport{})
}

// ServeMCP runs the MCP server over one transport until the client leaves.
func ServeMCP(ctx context.Context, engine http.Handler, opts mcp.Options, t sdk.Transport) error {
	err := mcp.NewServer(engine, opts).Run(ctx, t)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}
