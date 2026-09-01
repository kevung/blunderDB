package postgres

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
)

// TestRebindIntRange guards the placeholder contract between the shared
// searchfilter builders and this backend: searchfilter.AppendIntRangeSQL emits
// '?' and rebind must turn every one of them into a numbered '$N', in order.
func TestRebindIntRange(t *testing.T) {
	var where strings.Builder
	var args []any
	where.WriteString("SELECT id FROM position WHERE tenant_id = ?")
	searchfilter.AppendIntRangeSQL("pip_diff", 3, 9, true, true, &where, &args)
	searchfilter.AppendIntRangeSQL("off_1", 2, 0, true, false, &where, &args)
	searchfilter.AppendIntRangeSQL("off_2", 4, 4, true, true, &where, &args)

	got := rebind(where.String())
	want := "SELECT id FROM position WHERE tenant_id = $1 AND pip_diff BETWEEN $2 AND $3 AND off_1 >= $4 AND off_2 = $5"
	if got != want {
		t.Fatalf("rebind:\n got %q\nwant %q", got, want)
	}
	if strings.Contains(got, "?") {
		t.Fatalf("rebind left a '?' placeholder in %q", got)
	}
	if len(args) != 4 {
		t.Fatalf("args = %v, want 4 bound values", args)
	}
}
