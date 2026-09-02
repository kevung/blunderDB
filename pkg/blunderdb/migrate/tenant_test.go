package migrate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestRunRejectsNamedTenant pins that a scope which is not a positive decimal
// integer is refused before a single row is read: "mon-tenant" used to be
// accepted and copied into tenant 0, where every other named tenant already
// lived (ADR-0005, amendment 2026-09-03). Untagged (no PostgreSQL, no
// Docker): the check happens before the destination is touched, and a
// dry run needs no destination at all.
func TestRunRejectsNamedTenant(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	if err := src.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	for _, scope := range []string{"mon-tenant", "default", "0", "-1", "007"} {
		_, err := Run(ctx, src, nil, scope, Options{DryRun: true})
		if !errors.Is(err, storage.ErrInvalidTenant) {
			t.Errorf("Run(scope=%q): err = %v, want ErrInvalidTenant", scope, err)
		}
	}

	// The valid spellings still run (dry, so no destination is needed).
	for _, scope := range []string{"", "1", "42"} {
		if _, err := Run(ctx, src, nil, scope, Options{DryRun: true}); err != nil {
			t.Errorf("Run(scope=%q) dry run: %v", scope, err)
		}
	}
}

// TestRunCLIRejectsNamedTenant covers the flag itself: the rejection names
// --tenant-id and the expected format, and fires before --from is opened (the
// source path below does not exist).
func TestRunCLIRejectsNamedTenant(t *testing.T) {
	err := RunCLI([]string{"--from", "/nonexistent/user.db", "--to", "postgres://x", "--tenant-id", "mon-tenant"})
	if err == nil {
		t.Fatal("RunCLI accepted --tenant-id mon-tenant")
	}
	for _, want := range []string{"--tenant-id", storage.TenantFormat, `"mon-tenant"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}
