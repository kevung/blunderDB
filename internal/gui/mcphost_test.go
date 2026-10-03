package gui

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`

func post(t *testing.T, url, origin string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(initialize))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if origin != "" {
		req.Header.Set("Origin", origin)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestMCPHostIsOptInLocalAndRefusesCrossOrigin(t *testing.T) {
	a := NewApp(database.NewDatabase())
	t.Cleanup(a.stopMCP)
	if st := a.GetMCPHostStatus(); st.Running {
		t.Fatal("the host must not listen before it is enabled")
	}
	port := freePort(t)
	st := a.ConfigureMCPHost(MCPHostConfig{Enabled: true, Port: port})
	if !st.Running || st.Write || !strings.HasPrefix(st.URL, "http://127.0.0.1:") {
		t.Fatalf("status %+v", st)
	}
	if code := post(t, st.URL, ""); code != http.StatusOK {
		t.Fatalf("a local client: HTTP %d", code)
	}
	if code := post(t, st.URL, "https://evil.example"); code != http.StatusForbidden {
		t.Fatalf("a cross-origin page: HTTP %d, want 403", code)
	}
	// A second instance on the same port reports why it cannot listen.
	b := NewApp(database.NewDatabase())
	if st := b.ConfigureMCPHost(MCPHostConfig{Enabled: true, Port: port}); st.Running || st.Error == "" {
		t.Fatalf("busy port: %+v", st)
	}
	if st := a.ConfigureMCPHost(MCPHostConfig{Enabled: false, Port: port}); st.Running {
		t.Fatalf("disabled host still running: %+v", st)
	}
}

func TestLiveEngineRefusesWithoutDatabase(t *testing.T) {
	e := &liveEngine{db: database.NewDatabase()}
	if _, err := e.handler(); err != errNoDatabase {
		t.Fatalf("got %v", err)
	}
}

func TestLiveEngineFollowsTheOpenFile(t *testing.T) {
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(a.stopMCP)
	open := func() {
		path, err := a.PrepareDemoDatabase()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(path) })
		if err := db.OpenDatabase(path); err != nil {
			t.Fatal(err)
		}
	}
	open()
	t.Cleanup(func() { _ = db.Close() })
	if _, err := a.mcp.engine.handler(); err != nil {
		t.Fatal(err)
	}
	first := a.mcp.engine.srv
	if _, _ = a.mcp.engine.handler(); a.mcp.engine.srv != first {
		t.Fatal("the handler must be reused while the same file is open")
	}
	open()
	if _, err := a.mcp.engine.handler(); err != nil || a.mcp.engine.srv == first {
		t.Fatalf("the handler must be rebuilt on the new file (err %v)", err)
	}
}
