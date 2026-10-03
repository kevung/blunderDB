package mcp_test

import (
	"strings"
	"testing"

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/mcp"
)

type fakeDisplay struct {
	views     [][2]string
	positions []int64
}

func (f *fakeDisplay) OpenView(name, query string) error {
	f.views = append(f.views, [2]string{name, query})
	return nil
}

func (f *fakeDisplay) ShowPosition(id int64) error {
	f.positions = append(f.positions, id)
	return nil
}

func TestDisplayTools(t *testing.T) {
	d := &fakeDisplay{}
	cs := connect(t, demoServer(t, internalserver.Options{}).Handler(),
		mcp.Options{Tenant: "1", Extensions: []mcp.Extension{mcp.DisplayTools(d)}})

	out := call(t, cs, "open_view", obj{"name": "mes erreurs de videau", "query": "cube E>80"})
	if out["opened"] != "mes erreurs de videau" || len(d.views) != 1 || d.views[0][1] != "cube E>80" {
		t.Fatalf("open_view: %v, display got %v", out, d.views)
	}
	if callErr(t, cs, "open_view", obj{"query": "  "}) == "" || len(d.views) != 1 {
		t.Fatal("an empty query must be refused before the screen is asked")
	}

	found := call(t, cs, "search_positions", obj{"query": "E>0", "limit": 1})
	pid := id(t, list(t, found, "positions")[0], "id")
	call(t, cs, "show_position", obj{"positionId": pid})
	if len(d.positions) != 1 || d.positions[0] != int64(pid) {
		t.Fatalf("show_position: display got %v", d.positions)
	}
	if callErr(t, cs, "show_position", obj{"positionId": 99999999}) == "" || len(d.positions) != 1 {
		t.Fatal("an unknown position must be refused before the screen is asked")
	}
}

func TestViewName(t *testing.T) {
	if got := mcp.ViewName("  mes   erreurs\n de videau "); got != "mes erreurs de videau" {
		t.Fatalf("got %q", got)
	}
	long := mcp.ViewName(strings.Repeat("blunder ", 20))
	if len([]rune(long)) > 41 || !strings.HasSuffix(long, "…") {
		t.Fatalf("long name not shortened: %q", long)
	}
}
