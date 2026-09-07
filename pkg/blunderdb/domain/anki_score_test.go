package domain

import "testing"

// The deck is the 36 unordered scores of 2 to 9 away, and "unordered" is the
// whole of it: 3:5 and 5:3 are one question asked from two sides, so a deck
// that held both would ask the same thing twice.
func TestUnorderedScoreKeys(t *testing.T) {
	keys := UnorderedScoreKeys()
	if len(keys) != 36 {
		t.Fatalf("got %d keys, want 36", len(keys))
	}
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k] {
			t.Errorf("key %q appears twice", k)
		}
		seen[k] = true
		a, b, err := ParseScoreKey(k)
		if err != nil {
			t.Fatalf("key %q: %v", k, err)
		}
		if a > b {
			t.Errorf("key %q is not written smaller away first", k)
		}
		if ScoreKey(b, a) != k {
			t.Errorf("ScoreKey(%d, %d) = %q, want %q — the two faces are one key", b, a, ScoreKey(b, a), k)
		}
	}
	for _, want := range []string{"2:2", "2:9", "9:9"} {
		if !seen[want] {
			t.Errorf("the corner %q is missing from the deck", want)
		}
	}
}

func TestParseScoreKeyRejects(t *testing.T) {
	for _, key := range []string{"", "3", "3:", ":5", "a:5", "3:b", "1:5", "3:10", "0:0"} {
		if _, _, err := ParseScoreKey(key); err == nil {
			t.Errorf("ParseScoreKey(%q): got no error, want one", key)
		}
	}
}
