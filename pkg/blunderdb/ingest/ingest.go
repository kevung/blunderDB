// Package ingest holds the backend-agnostic import/export pipeline used by the
// `blunderdb serve` daemon. Unlike the legacy import path in package database
// (which is soldered to the SQLite-only *Database wrapper), everything here
// writes through the storage.Storage / storage.Tx interfaces, so it works
// identically on SQLite and PostgreSQL.
//
// See tasks/headless/12-imports-exports-over-storage.md for the full design.
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
	// BatchID stamps every match this import writes with the batch it came in
	// with (issue #257), 0 for an import that opened none. Set by the caller
	// that owns the batch — the daemon's handler, the CLI — because only it
	// knows how many files the user meant as one import.
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
	// were merged into a match already stored — neither a new match nor a
	// skipped one, and invisible in the summary until #257 needed to say so.
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
