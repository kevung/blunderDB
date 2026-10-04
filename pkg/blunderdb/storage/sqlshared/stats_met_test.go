package sqlshared

import (
	"regexp"
	"strings"
	"testing"
)

// tenantDialect confines every table to a tenant, as PostgreSQL does.
type tenantDialect struct{ fakeDialect }

func (tenantDialect) TenantFilter(alias, scope string) (string, []any) {
	if alias != "" {
		alias += "."
	}
	return alias + "tenant_id = ?", []any{"t:" + scope}
}

// Every subquery of the match-level clause reads analysis rows, which are
// per tenant: one without the tenant filter would let another tenant's tag
// decide whether this tenant's match is comparable.
func TestMETComparableMatchFiltersEveryAnalysisByTenant(t *testing.T) {
	sql, args := metComparableMatch(tenantDialect{}, "club")

	if got, want := strings.Count(sql, "?"), len(args); got != want {
		t.Fatalf("%d placeholders, %d arguments:\n%s", got, want, sql)
	}
	reads := regexp.MustCompile(`(FROM|JOIN) analysis a\b`).FindAllStringIndex(sql, -1)
	filters := strings.Count(sql, "a.tenant_id = ?")
	if len(reads) != 3 || filters != len(reads) {
		t.Errorf("%d analysis reads, %d tenant filters on them; want 3 each:\n%s", len(reads), filters, sql)
	}
	for i, a := range args {
		if a != "t:club" {
			t.Errorf("argument %d = %v, want the scope's tenant", i, a)
		}
	}
}
