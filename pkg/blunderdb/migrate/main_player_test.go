package migrate

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The Main player follows the library: every import into the destination
// seats them first, as it did in the source.
func TestRunCarriesTheMainPlayer(t *testing.T) {
	ctx := context.Background()
	open := func() storage.Storage {
		s, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		return s
	}
	src, dst := open(), open()
	if err := src.LibrarySettings().SetMainPlayer(ctx, "", "Kevin"); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(ctx, src, dst, "", Options{}); err != nil {
		t.Fatal(err)
	}
	if got, err := dst.LibrarySettings().MainPlayer(ctx, ""); err != nil || got != "Kevin" {
		t.Fatalf("destination Main player = %q, %v; want Kevin", got, err)
	}
}
