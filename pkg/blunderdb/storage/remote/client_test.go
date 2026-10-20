package remote_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/remote"
)

// fake answers every call with status and body, and records the last request.
func fake(t *testing.T, status int, body string) (*remote.Storage, *http.Request) {
	t.Helper()
	var last http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = *r.Clone(context.Background())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)
	h := http.Header{}
	h.Set("Authorization", "Bearer proxy-token")
	rs, err := remote.Open(remote.Options{BaseURL: ts.URL, Tenant: "7", Header: h})
	if err != nil {
		t.Fatal(err)
	}
	return rs, &last
}

func TestHeadersAreSentNotChecked(t *testing.T) {
	rs, last := fake(t, 200, `3`)
	ctx := storage.WithCommentAuthor(context.Background(), "Ada")
	if _, err := rs.Positions().Count(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if got := last.Header.Get("X-Tenant-ID"); got != "7" {
		t.Errorf("scope \"\" sent X-Tenant-ID %q, want the configured 7", got)
	}
	if got := last.Header.Get("Authorization"); got != "Bearer proxy-token" {
		t.Errorf("configured header not forwarded: %q", got)
	}
	if got := last.Header.Get("X-User-Name"); got != "Ada" {
		t.Errorf("X-User-Name = %q, want the context's author", got)
	}
	if last.URL.Path != "/v1/positions.count" || last.Method != http.MethodPost {
		t.Errorf("request = %s %s", last.Method, last.URL.Path)
	}
	if _, err := rs.Positions().Count(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if got := last.Header.Get("X-Tenant-ID"); got != "12" {
		t.Errorf("explicit scope sent X-Tenant-ID %q, want 12", got)
	}
}

func TestEnvelopeBecomesTheSentinels(t *testing.T) {
	for _, tc := range []struct {
		code string
		want error
	}{
		{"not_found", storage.ErrNotFound},
		{"conflict", storage.ErrConflict},
		{"invalid", storage.ErrInvalid},
		{"internal", storage.ErrInternal},
		{"unknown_route", remote.ErrUnsupported},
		{"rate_limited", remote.ErrUnavailable},
	} {
		rs, _ := fake(t, 400, `{"error":{"code":"`+tc.code+`","message":"m"}}`)
		if _, err := rs.Positions().Load(context.Background(), "", 1); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.code, err, tc.want)
		}
	}
}

func TestDuplicateNamesTheExistingRow(t *testing.T) {
	rs, _ := fake(t, 409, `{"error":{"code":"conflict","message":"m","details":{"existingId":5}}}`)
	err := rs.Positions().Update(context.Background(), "", nil)
	var dup *storage.DuplicatePositionError
	if !errors.As(err, &dup) || dup.ExistingID != 5 {
		t.Fatalf("got %v, want a DuplicatePositionError naming 5", err)
	}
}

func TestNoEnvelopeIsUnavailable(t *testing.T) {
	rs, _ := fake(t, 502, `<html>bad gateway</html>`)
	if _, err := rs.Positions().Count(context.Background(), ""); !errors.Is(err, remote.ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
	dead, _ := remote.Open(remote.Options{BaseURL: "http://127.0.0.1:1"})
	if _, err := dead.Positions().Count(context.Background(), ""); !errors.Is(err, remote.ErrUnavailable) {
		t.Fatalf("unreachable daemon: got %v, want ErrUnavailable", err)
	}
}

func TestNoRouteIsATypedRefusal(t *testing.T) {
	rs, last := fake(t, 200, `{}`)
	last.URL = nil
	if _, err := rs.BeginTx(context.Background()); !errors.Is(err, remote.ErrUnsupported) || !errors.Is(err, storage.ErrInternal) {
		t.Fatalf("BeginTx: got %v, want ErrUnsupported and ErrInternal", err)
	}
	if len(remote.Unsupported) == 0 {
		t.Fatal("Unsupported is empty")
	}
	_, err := rs.Positions().RaiseFlag(context.Background(), "", nil)
	var u *remote.UnsupportedError
	if !errors.As(err, &u) || u.Method != "PositionStore.RaiseFlag" {
		t.Fatalf("RaiseFlag: got %v, want an UnsupportedError naming it", err)
	}
	for _, err := range rs.Analyses().WithEngine(context.Background(), "", "x") {
		if !errors.Is(err, remote.ErrUnsupported) {
			t.Fatalf("WithEngine stream: got %v", err)
		}
	}
	if last.URL != nil {
		t.Fatal("a refused method reached the daemon")
	}
}
