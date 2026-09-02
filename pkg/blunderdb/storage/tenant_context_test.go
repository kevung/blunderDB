package storage

import (
	"errors"
	"strings"
	"testing"
)

// TestParseTenant pins the one accepted spelling of a tenant: the canonical
// decimal form of a positive integer, or the empty implicit desktop tenant.
// Before ADR-0005's 2026-09-03 amendment every rejected value below silently
// became tenant 0, so "alice", "bob" and "default" shared one set of rows.
func TestParseTenant(t *testing.T) {
	accepted := map[string]int64{
		"":                    0,
		"1":                   1,
		"42":                  42,
		"9223372036854775807": 1<<63 - 1,
	}
	for scope, want := range accepted {
		got, err := ParseTenant(scope)
		if err != nil || got != want {
			t.Errorf("ParseTenant(%q) = %d, %v; want %d, nil", scope, got, err, want)
		}
	}

	rejected := []string{
		"alice", "default", "mon-tenant", "tenant-a",
		"0", "-1", "+1", "007", "1.0", "1e3", " 7", "7 ", "\t7",
		"9223372036854775808", // int64 overflow
		"١٢",                  // non-ASCII digits
	}
	for _, scope := range rejected {
		got, err := ParseTenant(scope)
		if err == nil {
			t.Errorf("ParseTenant(%q) = %d, nil; want an error", scope, got)
			continue
		}
		if !errors.Is(err, ErrInvalidTenant) {
			t.Errorf("ParseTenant(%q): %v does not wrap ErrInvalidTenant", scope, err)
		}
		if !strings.Contains(err.Error(), TenantFormat) {
			t.Errorf("ParseTenant(%q): %q does not name the expected format", scope, err)
		}
		if got != 0 {
			t.Errorf("ParseTenant(%q) returned %d alongside the error; want 0", scope, got)
		}
	}
}
