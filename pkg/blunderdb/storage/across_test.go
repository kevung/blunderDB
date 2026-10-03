package storage_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func TestNewReadTenants(t *testing.T) {
	for _, tc := range []struct {
		writer string
		listed []string
		want   []string
	}{
		{"1", nil, []string{"1"}},
		{"1", []string{"2", "3"}, []string{"1", "2", "3"}},
		{"2", []string{"3", "2", "3", "1"}, []string{"2", "3", "1"}},
		{"", nil, []string{""}},
	} {
		got, err := storage.NewReadTenants(tc.writer, tc.listed)
		if err != nil {
			t.Errorf("NewReadTenants(%q, %v): %v", tc.writer, tc.listed, err)
			continue
		}
		if !slices.Equal([]string(got), tc.want) {
			t.Errorf("NewReadTenants(%q, %v) = %v, want %v", tc.writer, tc.listed, got, tc.want)
		}
	}
}

func TestNewReadTenants_Refuses(t *testing.T) {
	for _, listed := range [][]string{{""}, {"alice"}, {"0"}, {"007"}, {" 2"}} {
		if _, err := storage.NewReadTenants("1", listed); !errors.Is(err, storage.ErrInvalidTenant) {
			t.Errorf("NewReadTenants(1, %q): err = %v, want ErrInvalidTenant", listed, err)
		}
	}
	if _, err := storage.NewReadTenants("alice", nil); !errors.Is(err, storage.ErrInvalidTenant) {
		t.Errorf("writer alice: err = %v, want ErrInvalidTenant", err)
	}
	many := make([]string, storage.MaxReadTenants)
	for i := range many {
		many[i] = strconv.Itoa(i + 2)
	}
	if _, err := storage.NewReadTenants("1", many); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("%d read tenants: err = %v, want ErrInvalid", len(many)+1, err)
	}
	if _, err := storage.NewReadTenants("1", many[:storage.MaxReadTenants-1]); err != nil {
		t.Errorf("exactly %d read tenants: %v", storage.MaxReadTenants, err)
	}
}

// TestReadAcross_ScopesTheContext: each call runs under a context carrying its
// own tenant, so PostgreSQL's row-level security filters it as any other read.
func TestReadAcross_ScopesTheContext(t *testing.T) {
	set, err := storage.NewReadTenants("4", []string{"7"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := storage.ReadAcross(context.Background(), set, func(ctx context.Context, scope string) (int64, error) {
		n, _ := storage.TenantFromContext(ctx)
		if strconv.FormatInt(n, 10) != scope {
			t.Errorf("read of %s ran under context tenant %d", scope, n)
		}
		return n, nil
	})
	if err != nil || len(got) != 2 || got[0].Tenant != "4" || got[1].Tenant != "7" {
		t.Fatalf("ReadAcross = %v, %v", got, err)
	}
}

func TestReadAcross_StopsAtTheFirstError(t *testing.T) {
	set, _ := storage.NewReadTenants("1", []string{"2", "3"})
	var read []string
	boom := errors.New("boom")
	_, err := storage.ReadAcross(context.Background(), set, func(_ context.Context, scope string) (int, error) {
		read = append(read, scope)
		if scope == "2" {
			return 0, boom
		}
		return 0, nil
	})
	if !errors.Is(err, boom) || !slices.Equal(read, []string{"1", "2"}) {
		t.Errorf("err = %v, read %v; want boom after 1, 2", err, read)
	}
}
