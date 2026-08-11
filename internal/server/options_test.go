package server

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestOptions_applyDefaults_IdleTimeout(t *testing.T) {
	var o Options
	o.applyDefaults()
	if o.IdleTimeout != defaultIdleTimeout {
		t.Errorf("IdleTimeout default = %v, want %v", o.IdleTimeout, defaultIdleTimeout)
	}

	o2 := Options{IdleTimeout: 5 * time.Second}
	o2.applyDefaults()
	if o2.IdleTimeout != 5*time.Second {
		t.Errorf("explicit IdleTimeout overridden: got %v, want 5s", o2.IdleTimeout)
	}
}

// TestServer_HTTPServerCarriesIdleTimeout guards the actual fix: an
// http.Server with no IdleTimeout (and no Read/WriteTimeout — see
// Options.IdleTimeout's doc comment) lets a keep-alive connection sit open
// indefinitely. New() must wire Options.IdleTimeout through to the
// http.Server it builds, not just carry it in Options.
func TestServer_HTTPServerCarriesIdleTimeout(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	srv, err := New(Options{Storage: st})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if srv.http.IdleTimeout != defaultIdleTimeout {
		t.Errorf("http.Server.IdleTimeout = %v, want %v", srv.http.IdleTimeout, defaultIdleTimeout)
	}
	// Read/WriteTimeout intentionally stay zero: they would cap the total
	// duration of an in-flight request, breaking long NDJSON streams
	// (imports/list-style routes).
	if srv.http.ReadTimeout != 0 {
		t.Errorf("http.Server.ReadTimeout = %v, want 0 (unset — would break streaming)", srv.http.ReadTimeout)
	}
	if srv.http.WriteTimeout != 0 {
		t.Errorf("http.Server.WriteTimeout = %v, want 0 (unset — would break streaming)", srv.http.WriteTimeout)
	}
}
