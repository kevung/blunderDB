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
)

// Source is the input to an import. Reader is set for streaming formats
// (JSON); Path is set when the daemon has spooled the upload to a temp file
// for parsers that need random access. At least one is non-zero.
type Source struct {
	Format Format
	Reader io.Reader
	Path   string
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
}

// ExportOptions tunes an export.
type ExportOptions struct {
	Format Format
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
