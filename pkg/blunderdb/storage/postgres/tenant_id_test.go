// TestTenantID never touches a database, so it lives untagged (no `postgres`
// build tag, no Docker) in `package postgres` (white-box: tenantID is the
// unexported scope→tenant_id conversion every Store method in this package
// funnels through).
package postgres

import (
	"strings"
	"testing"
)

// TestTenantID pins that the conversion refuses to invent a tenant: a scope
// that is not a positive decimal integer panics instead of collapsing onto
// tenant 0 (which is what "alice", "default" and "mon-tenant" all did before
// ADR-0005's 2026-09-03 amendment, sharing one set of rows).
func TestTenantID(t *testing.T) {
	if got := tenantID(""); got != 0 {
		t.Errorf("tenantID(\"\") = %d, want 0 (the implicit desktop tenant)", got)
	}
	if got := tenantID("42"); got != 42 {
		t.Errorf("tenantID(\"42\") = %d, want 42", got)
	}
	for _, scope := range []string{"alice", "default", "0", "-1", "007"} {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("tenantID(%q) did not panic", scope)
					return
				}
				if msg, _ := r.(string); !strings.Contains(msg, scope) {
					t.Errorf("tenantID(%q) panic %v does not name the scope", scope, r)
				}
			}()
			tenantID(scope)
		}()
	}
}
