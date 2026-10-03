package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// An import stores the canonical names, keeps the file's names in the
// fingerprints, and recognises a stored match under a known spelling.
func TestImportAppliesAliasesAfterTheFingerprints(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	g1 := [][2]int32{{3, 1}, {6, 4}, {2, 2}, {5, 1}}
	g2 := [][2]int32{{4, 2}, {6, 6}, {1, 3}}
	first := writeGraph(t, s, diceGraph("Alice Martin", "Bob Durand", g1, g2))

	for _, a := range []storage.Alias{{Alias: "Martin A.", Canonical: "Alice Martin"}, {Alias: "Durand B.", Canonical: "Bob Durand"}} {
		if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, a.Alias, a.Canonical); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Aliases().Set(ctx, "", storage.AliasEvent, "Open 25", "Autumn Open 2025"); err != nil {
		t.Fatal(err)
	}

	// The same match under the aliases: not a second match, not a suspect.
	again := writeGraph(t, s, diceGraph("Martin A.", "Durand B.", g1, g2))
	if !again.Skipped || again.MatchID != first.MatchID || again.ProbableDuplicate != nil {
		t.Fatalf("a known spelling of a stored match must be found as that match: %+v", again)
	}

	// Another match under the aliases: stored under the canonical names and
	// the canonical event, its fingerprint still the file's.
	other := diceGraph("Martin A.", "Carol Petit", [][2]int32{{1, 2}, {5, 5}})
	other.Match.Event = "Open 25"
	hash := other.Match.MatchHash
	res := writeGraph(t, s, other)
	m, err := s.Matches().Get(ctx, "", res.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Player1Name != "Alice Martin" || m.Event != "Autumn Open 2025" || m.MatchHash != hash {
		t.Fatalf("stored %q / %q / %q, want canonical names and the file's hash %q", m.Player1Name, m.Event, m.MatchHash, hash)
	}
	if res.Tournament != "Autumn Open 2025" {
		t.Fatalf("tournament %q, want the canonical event", res.Tournament)
	}

	// Re-importing the same file finds its own fingerprint.
	dup := diceGraph("Martin A.", "Carol Petit", [][2]int32{{1, 2}, {5, 5}})
	if r := writeGraph(t, s, dup); !r.Skipped || r.MatchID != res.MatchID {
		t.Fatalf("re-import after the alias must be an exact duplicate: %+v", r)
	}
}
