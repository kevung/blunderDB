package document

import "github.com/kevung/blunderdb/pkg/blunderdb/engine"

// Summary is what a check of a Relevé reports, without a database.
type Summary struct {
	Title     string `json:"title"`
	Entries   int    `json:"entries"`
	Sure      int    `json:"sure"`
	ToVerify  int    `json:"toVerify"`
	Uncertain int    `json:"uncertain"`
	// CheckersConfirmed counts the entries to verify whose checkers the
	// printed pips confirmed, their context alone left to verify.
	CheckersConfirmed int `json:"checkersConfirmed"`
	// Complete counts the entries an import accepts as a whole on request:
	// fifteen checkers a side read with no doubt, the context certified.
	Complete int `json:"complete"`
	// CompleteDirectionAssumed counts the complete entries whose direction
	// of play the reader assumed for want of printed pips: a wrong guess
	// exchanges the players of every one of them.
	CompleteDirectionAssumed int `json:"completeDirectionAssumed"`
	// ContextRead, ContextDefault and ContextToVerify count the entries by
	// how their context is known (uncertain ones apart).
	ContextRead     int `json:"contextRead"`
	ContextDefault  int `json:"contextDefault"`
	ContextToVerify int `json:"contextToVerify"`
	// Defaults is the document's default context, nil when it has none.
	Defaults *Defaults `json:"defaults,omitempty"`
	Verified int       `json:"verified"`
	// Unverified lists the ranks that refuse an import unless it is forced;
	// UnverifiedComplete those that still refuse it with complete entries
	// accepted.
	Unverified         []int `json:"unverified"`
	UnverifiedComplete []int `json:"unverifiedComplete"`
	// Positions counts the distinct positions, by the Zobrist hash the
	// library identifies them with; Repeated the entries showing one again.
	Positions int `json:"positions"`
	Repeated  int `json:"repeated"`
}

// Summarize counts a valid Relevé's entries.
func Summarize(r *Releve) Summary {
	r = r.Upgrade()
	s := Summary{Title: r.Document.Title, Entries: len(r.Entries), Defaults: r.Document.Defaults,
		Unverified: Unverified(r, false), UnverifiedComplete: Unverified(r, true)}
	seen := map[uint64]bool{}
	for _, e := range r.Entries {
		switch r.StateOf(e) {
		case Sure:
			s.Sure++
		case ToVerify:
			s.ToVerify++
			if r.CheckersConfirmed(e) {
				s.CheckersConfirmed++
			}
			if r.CompleteCertified(e) {
				s.Complete++
				if hasDoubt(e.Doubts, AssumedDirectionDoubt) {
					s.CompleteDirectionAssumed++
				}
			}
		case Uncertain:
			s.Uncertain++
		}
		switch e.Context {
		case ContextRead:
			s.ContextRead++
		case ContextDefault:
			s.ContextDefault++
		case ContextToVerify:
			s.ContextToVerify++
		}
		if e.Verified {
			s.Verified++
		}
		pos, err := ParsePosition(e.Position)
		if err != nil {
			continue
		}
		h := engine.ZobristHash(&pos)
		if seen[h] {
			s.Repeated++
		} else {
			seen[h] = true
		}
	}
	s.Positions = len(seen)
	return s
}
