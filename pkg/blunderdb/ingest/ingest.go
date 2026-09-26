// Package ingest holds the backend-agnostic import/export pipeline: the daemon
// calls it directly and the desktop Database wrapper delegates to it.
// Everything writes through storage.Storage / storage.Tx, so it works
// identically on SQLite and PostgreSQL.
package ingest

import (
	"context"
	"io"
)

// Format identifies an import/export wire format.
type Format string

const (
	// FormatJSON is the backend-agnostic blunderDB interchange (NDJSON). It
	// round-trips through Storage without any external parser.
	FormatJSON Format = "json"
	// The parser-backed formats are wired in PR3b/PR3c.
	FormatXG       Format = "xg"
	FormatGnuBG    Format = "gnubg"
	FormatBGF      Format = "bgf"
	FormatNativeDB Format = "db"
	FormatPosition Format = "position"
	// FormatSQLite serializes a tenant into a fresh, valid blunderDB SQLite file
	// (a Desktop-openable export / backup). Export-only.
	FormatSQLite Format = "sqlite"
)

// Source is the input to an import. Reader is set for streaming formats
// (JSON); Path is set when the daemon has spooled the upload to a temp file
// for parsers that need random access. At least one is non-zero.
type Source struct {
	Format Format
	Reader io.Reader
	Path   string
	// BatchID stamps every match this import writes with its batch, 0 for
	// none. Set by the caller, the only one that knows how many files the user
	// meant as one import.
	BatchID int64
}

// Progress is reported incrementally during an import.
type Progress struct {
	Matches   int `json:"matches"`
	Games     int `json:"games"`
	Positions int `json:"positions"`
}

// Summary is the terminal result of an import.
type Summary struct {
	SavedPositions    int   `json:"savedPositions"`
	SkippedDuplicates int   `json:"skippedDuplicates"`
	Matches           int   `json:"matches"`
	MatchID           int64 `json:"matchId,omitempty"`
	// Enriched counts the cross-format duplicates whose analyses and comments
	// were merged into a match already stored — neither new nor skipped.
	Enriched int `json:"enriched,omitempty"`
	// BatchID is the import batch these figures belong to, 0 when the caller
	// opened none. /v1/imports.* fills it so a client can ask for the full
	// end-of-import report afterwards.
	BatchID int64 `json:"batchId,omitempty"`
}

// ExportOptions says how an export is made. Format picks the exporter; the
// rest is read by the SQLite export (ExportSQLite) — JSONExporter streams the
// whole position library and ignores it.
type ExportOptions struct {
	Format Format

	// Selection says what leaves; the zero value selects nothing. WholeTenant
	// is the everything-on preset.
	Selection Selection

	// Analysis, Comments and PlayedMoves govern what travels with a position;
	// PlayedMoves matters only with Analysis. FilterLibrary and AnkiDecks add
	// those families whole.
	Analysis      bool
	Comments      bool
	PlayedMoves   bool
	FilterLibrary bool
	AnkiDecks     bool

	// Metadata is copied by allow-list (issuance.Carried). Watermark is the
	// sealed document to write verbatim (see SealWatermark), "" for none.
	// Password wraps the finished file in an encrypted container.
	Metadata  map[string]string
	Watermark string
	Password  string

	// AfterWrite runs on the freshly written file, BEFORE it is sealed into a protected
	// container and while it is still a plain SQLite database.
	//
	// It carries what the storage contract deliberately does not: a tournament's Direction
	// (ADR-0047), whose tables live on the desktop wrapper. A post-pass on the finished file
	// would come too late for a protected export, whose plain intermediate is removed.
	AfterWrite func(ctx context.Context, path string, report ExportReport) error
}

// Importer reads a Source and writes its contents through Storage, emitting
// progress and honouring ctx cancellation (a cancelled import rolls back).
type Importer interface {
	Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error)
}

// Exporter streams stored data out in a chosen format.
type Exporter interface {
	Export(ctx context.Context, scope string, w io.Writer, opts ExportOptions) error
}
