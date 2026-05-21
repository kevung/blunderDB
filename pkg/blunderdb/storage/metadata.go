package storage

import "context"

// Counts holds the headline row counts of a database.
type Counts struct {
	Positions int `json:"positions"`
	Analyses  int `json:"analyses"`
	Matches   int `json:"matches"`
	Games     int `json:"games"`
	Moves     int `json:"moves"`
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
