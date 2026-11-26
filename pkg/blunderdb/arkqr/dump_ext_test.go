package arkqr_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestDumpArchiveCorpus writes the matches of a database as the archive
// reads them, for the measures of external_test.go. The database is opened
// read-only; give it a copy all the same.
func TestDumpArchiveCorpus(t *testing.T) {
	db, out := os.Getenv("BLUNDERDB_ARKQR_DUMP"), os.Getenv("BLUNDERDB_ARKQR_CORPUS")
	if db == "" || out == "" {
		t.Skip("BLUNDERDB_ARKQR_DUMP and BLUNDERDB_ARKQR_CORPUS unset")
	}
	ctx := context.Background()
	s, err := sqlite.OpenReadOnly(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var ids []int64
	for m, err := range s.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, m.ID)
	}
	var ms []arkqr.Match
	for _, id := range ids {
		am, _, err := ingest.ArchiveMatch(ctx, s, "", id)
		if err != nil {
			t.Fatalf("match %d: %v", id, err)
		}
		ms = append(ms, am)
	}
	b, err := json.Marshal(ms)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d matches", len(ms))
}
