package assistant_test

import (
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	internalserver "github.com/kevung/blunderdb/internal/server"
)

// demoEngine serves a private copy of the demo database (fictional players)
// behind the /v1 engine the MCP tools call.
func demoEngine(t testing.TB) http.Handler {
	t.Helper()
	src, err := os.Open(filepath.Join("..", "..", "..", "internal", "gui", "demo.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	zr, err := gzip.NewReader(src)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "demo.db")
	dst, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, zr); err != nil {
		t.Fatal(err)
	}
	dst.Close()
	ctx := context.Background()
	st, err := internalserver.OpenStorage(ctx, "sqlite", path, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	srv, err := internalserver.New(internalserver.Options{Storage: st, SingleTenant: true,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	return srv.Handler()
}
