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
	// largest of its plays (#167), not by the denormalised first one.
	Blunders int `json:"blunders"`
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
}
