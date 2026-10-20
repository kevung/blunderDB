package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A transfer copies part of one library into another (ADR-0102): the
// selection of an export, read from the source, merged into the destination
// by the native import. It is the export and the import the user could run by
// hand, chained without a file the user handles, so that the three modes copy
// exactly what an export carries and merge exactly as an import of it does:
// positions dedup by Zobrist hash, analyses and comments merge, matches,
// collections, lessons and decks follow MergeMatches / MergeCollections /
// MergeLessons / MergeDecks. A → B then B → A therefore creates nothing the
// first copy did not.
//
// Nothing is written that says where the copy came from: no watermark, no
// import batch, no metadata of the source. The intermediate file holds only
// issuance.CarriedMetadataKeys, and the native import reads none of it. Nor
// does the source receive anything: an incremental export, which advances a
// cursor in its source, is refused.

// TransferEnd is one side of a transfer: a library and the scope within it
// (a tenant on PostgreSQL, "" on a SQLite file).
type TransferEnd struct {
	S     storage.Storage
	Scope string
}

// TransferReport counts what a transfer read and what it merged.
type TransferReport struct {
	// Exported counts what the selection carried out of the source.
	Exported ExportReport `json:"exported"`
	// Imported counts what the destination took of it; a row it already
	// held is counted as a duplicate, not added.
	Imported Summary `json:"imported"`
}

// ErrSelfTransfer refuses a transfer whose source and destination are the
// same library: it would copy nothing and still hold both ends.
var ErrSelfTransfer = errors.New("a library cannot be transferred into itself")

// Transfer copies what opts selects in from into to. opts are the export
// options of the desktop, the CLI and the daemon, read by OptionsFor; their
// watermark, password, incremental and metadata fields are refused, each
// being a statement about a file or a write on the source.
//
// Each side is read or written under a context scoped to its own tenant, so a
// caller holding the destination tenant's context reads the source under the
// source's (row-level security on PostgreSQL).
func Transfer(ctx context.Context, from, to TransferEnd, opts domain.ExportOptions, prog func(Progress)) (TransferReport, error) {
	if from.S == to.S && from.Scope == to.Scope {
		return TransferReport{}, fmt.Errorf("%w: %w", storage.ErrInvalid, ErrSelfTransfer)
	}
	switch {
	case opts.Incremental:
		return TransferReport{}, fmt.Errorf("%w: a transfer is not incremental: it writes no cursor in its source", storage.ErrInvalid)
	case opts.Watermark != "" || opts.WatermarkNote != "":
		return TransferReport{}, fmt.Errorf("%w: a transfer carries no watermark: nothing records where a copy came from", storage.ErrInvalid)
	case opts.Password != "":
		return TransferReport{}, fmt.Errorf("%w: a transfer writes no file to protect", storage.ErrInvalid)
	case len(opts.Metadata) > 0:
		return TransferReport{}, fmt.Errorf("%w: a transfer carries no metadata: the destination keeps its own", storage.ErrInvalid)
	}
	srcCtx, err := scopedContext(ctx, from.Scope)
	if err != nil {
		return TransferReport{}, err
	}
	dstCtx, err := scopedContext(ctx, to.Scope)
	if err != nil {
		return TransferReport{}, err
	}

	plan, err := OptionsFor(srcCtx, from.S, from.Scope, opts)
	if err != nil {
		return TransferReport{}, err
	}

	dir, err := os.MkdirTemp("", "blunderdb-transfer-*")
	if err != nil {
		return TransferReport{}, fmt.Errorf("ingest: transfer: create a working folder: %w", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			slog.Warn("transfer: removing the working folder", "dir", dir, "err", rmErr)
		}
	}()
	path := filepath.Join(dir, "transfer.db")

	var rep TransferReport
	if rep.Exported, err = ExportSQLite(srcCtx, from.S, from.Scope, path, plan); err != nil {
		return rep, fmt.Errorf("ingest: transfer: read the source: %w", err)
	}
	rep.Exported.Path = ""
	if rep.Imported, err = (DBImporter{S: to.S}).Import(dstCtx, to.Scope, Source{Format: FormatNativeDB, Path: path}, prog); err != nil {
		return rep, fmt.Errorf("ingest: transfer: merge into the destination: %w", err)
	}
	return rep, nil
}

// scopedContext scopes ctx to tenant scope; "" (a SQLite file) leaves it as is.
func scopedContext(ctx context.Context, scope string) (context.Context, error) {
	if scope == "" {
		return ctx, nil
	}
	n, err := storage.ParseTenant(scope)
	if err != nil {
		return nil, err
	}
	return storage.WithTenant(ctx, n), nil
}
