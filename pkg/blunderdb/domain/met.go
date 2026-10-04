package domain

// MatchEquityTable is a match equity table a library imported from a gnubg
// .xml file (ADR-0068). Digest identifies its values, so that two libraries
// holding the same table agree on which analyses are comparable whatever the
// file's name; an analysis records it in analysis.met_digest. The built-in
// Kazaross-XG2 has no row: an empty digest names it.
type MatchEquityTable struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Digest    string `json:"digest"`
	Source    string `json:"source,omitempty"` // the .xml as imported
	Current   bool   `json:"current"`
	CreatedAt string `json:"created_at,omitempty"`
}
