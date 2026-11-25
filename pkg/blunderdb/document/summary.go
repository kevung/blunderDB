package document

import "github.com/kevung/blunderdb/pkg/blunderdb/engine"

// Summary is what a check of a Relevé reports, without a database.
type Summary struct {
	Title     string `json:"title"`
	Entries   int    `json:"entries"`
	Sure      int    `json:"sure"`
	ToVerify  int    `json:"toVerify"`
	Uncertain int    `json:"uncertain"`
	Verified  int    `json:"verified"`
	// Unverified lists the ranks that refuse an import unless it is forced.
	Unverified []int `json:"unverified"`
	// Positions counts the distinct positions, by the Zobrist hash the
	// library identifies them with; Repeated the entries showing one again.
	Positions int `json:"positions"`
	Repeated  int `json:"repeated"`
}

// Summarize counts a valid Relevé's entries.
func Summarize(r *Releve) Summary {
	s := Summary{Title: r.Document.Title, Entries: len(r.Entries), Unverified: Unverified(r)}
	seen := map[uint64]bool{}
	for _, e := range r.Entries {
		switch e.State {
		case Sure:
			s.Sure++
		case ToVerify:
			s.ToVerify++
		case Uncertain:
			s.Uncertain++
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
