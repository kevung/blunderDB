package database

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// CheckMatchExists answers on match_hash alone, 0 when no match holds it.
func TestCheckMatchExists(t *testing.T) {
	d := newTestDB(t)
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5,
		MatchHash: "exact", CanonicalHash: "canonical"}
	id, err := d.store.Matches().Save(context.Background(), "", &m)
	if err != nil {
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
