package database

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// CheckMatchExists answers on match_hash alone, 0 when no match holds it. An
// empty hash identifies nothing, as on the daemon's /v1/matches.findByHash: a
// match stored without a hash (match_hash = ”) is not found by "".
func TestCheckMatchExists(t *testing.T) {
	t.Parallel()
	d := newTestDB(t)
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5,
		MatchHash: "exact", CanonicalHash: "canonical"}
	id, err := d.store.Matches().Save(context.Background(), "", &m)
	if err != nil {
		t.Fatal(err)
	}
	unhashed := domain.Match{Player1Name: "Carol", Player2Name: "Dan", MatchLength: 3}
	unhashedID, err := d.store.Matches().Save(context.Background(), "", &unhashed)
	if err != nil {
		t.Fatal(err)
	}
	// Save stores NULL; an exported library (ingest's export writes the hash
	// as it is) or the v1 migration (a hash it could not compute) store ''.
	if _, err := d.db.Exec(`UPDATE match SET match_hash = '' WHERE id = ?`, unhashedID); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		hash string
		want int64
	}{{"exact", id}, {"canonical", 0}, {"unknown", 0}, {"", 0}} {
		if got, err := d.CheckMatchExists(c.hash); err != nil || got != c.want {
			t.Errorf("CheckMatchExists(%q) = %d, %v; want %d", c.hash, got, err, c.want)
		}
	}
}
