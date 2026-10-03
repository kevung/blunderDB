package storage

import "context"

// Counts holds the headline row counts of a database.
type Counts struct {
	Positions int `json:"positions"`
	Analyses  int `json:"analyses"`
	Matches   int `json:"matches"`
	Games     int `json:"games"`
	Moves     int `json:"moves"`

	// IndividualPositions is the subset of Positions carrying the sticky
	// individually_imported provenance flag (ADR-0001).
	IndividualPositions int `json:"individual_positions"`
	// AnkiCards is the number of study cards across every deck.
	AnkiCards int `json:"anki_cards"`

	// Blunders is the number of Positions the library calls Blunders: those
	// whose largest recorded cost reaches its blunder threshold
	// (LibrarySettings, ADR-0046). It is the number the status bar's library
	// counter shows, and it is deliberately the set the search behind that
	// counter returns — a Position played several ways is scored by the
	// largest of its plays, not by the denormalised first one.
	Blunders int `json:"blunders"`
}

// CountsEstimate is the library counter read without scanning the tables: a
// table whose highest id is under the exact threshold is counted, a larger one
// is estimated by that highest id (an upper bound: deleted rows leave gaps).
// The estimate is never presented as exact: Approximate names what it covers.
type CountsEstimate struct {
	Counts
	// Approximate lists the JSON names of the fields that are estimates.
	Approximate []string `json:"approximate"`
	// BlundersKnown is false when Blunders was not computed: no honest
	// estimate of it exists, only the exact count, which scans every analysis.
	BlundersKnown bool `json:"blunders_known"`
}

// MetadataStore persists the database-level key/value metadata: schema
// version, match-equity-table id, and the headline counts.
type MetadataStore interface {
	// Version returns the recorded schema version.
	Version(ctx context.Context, scope string) (string, error)

	// SetVersion records the schema version.
	SetVersion(ctx context.Context, scope string, version string) error

	// Load returns every metadata key/value pair.
	Load(ctx context.Context, scope string) (map[string]string, error)

	// Save writes the given metadata key/value pairs.
	Save(ctx context.Context, scope string, metadata map[string]string) error

	// Counts returns the headline row counts.
	Counts(ctx context.Context, scope string) (Counts, error)

	// EstimatedCounts returns the headline counts without a full scan: exact
	// while every table's highest id stays under exactBelow, estimated beyond.
	// IndividualPositions and AnkiCards are not filled. Blunders is filled
	// only when Positions is exact.
	EstimatedCounts(ctx context.Context, scope string, exactBelow int) (CountsEstimate, error)
}
