package ingest

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// A copy whose deeper analyses replaced the stored ones wrote no match: its
// journal line is a duplicate of the match it deepened.
func TestJournalEntryOfADeepenedCopy(t *testing.T) {
	e := JournalEntry(FileOutcome{Path: "a.xg", Status: FileDuplicate, MatchID: 7, Deepened: 3})
	if e.Outcome != domain.JournalDuplicate || e.MatchID != 7 {
		t.Errorf("deepened copy journaled as %+v, want a duplicate of match 7", e)
	}
}
