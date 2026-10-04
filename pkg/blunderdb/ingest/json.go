package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// positionBundle is one NDJSON record of the JSON interchange. Bundling the
// analysis and comments with the position keeps every record self-contained,
// so the importer never has to remap reassigned position ids.
type positionBundle struct {
	Position *domain.Position         `json:"position"`
	Analysis *domain.PositionAnalysis `json:"analysis,omitempty"`
	Comments []string                 `json:"comments,omitempty"`
	// CommentAuthors[i] signs Comments[i]; absent, or shorter than Comments,
	// for comments nobody signed. A separate list keeps every older record
	// readable, and an import never signs in the importer's name.
	CommentAuthors []string `json:"commentAuthors,omitempty"`
	// MET is the id, within this stream, of the table the analysis was
	// valued with; absent for the built-in one (ADR-0068, rule 5).
	MET int64 `json:"met,omitempty"`
	// Table, on a record of its own without a position, is a table cited by
	// a later record's MET. Readers that predate it skip the record.
	Table *domain.MatchEquityTable `json:"matchEquityTable,omitempty"`
}

// commentAuthor is the author of the bundle's i-th comment, "" when unsigned.
func (b *positionBundle) commentAuthor(i int) string {
	if i < len(b.CommentAuthors) {
		return b.CommentAuthors[i]
	}
	return ""
}

// flusher is implemented by streaming writers (e.g. http.ResponseWriter) so
// the exporter can push records to the client incrementally.
type flusher interface{ Flush() }

// JSONExporter streams the position library (positions + analyses + comments)
// as NDJSON. It is fully backend-agnostic: it only reads through Storage.
//
// Match graphs are added once the Storage-based matchWriter lands (PR3b).
type JSONExporter struct{ S storage.Storage }

func (e JSONExporter) Export(ctx context.Context, scope string, w io.Writer, _ ExportOptions) error {
	// Drain the position list first so its connection is released before we
	// issue the per-position analysis/comment queries. Nesting those queries
	// inside the List iterator would hold one connection while grabbing a
	// second — which breaks under :memory: (separate DB per connection) and
	// risks pool starvation on any backend.
	var positions []*domain.Position
	for p, err := range e.S.Positions().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return err
		}
		pc := *p
		positions = append(positions, &pc)
	}

	enc := json.NewEncoder(w)
	fl, _ := w.(flusher)
	written := map[int64]bool{}
	for _, p := range positions {
		if err := ctx.Err(); err != nil {
			return err
		}
		b := positionBundle{Position: p}
		a, err := e.S.Analyses().Load(ctx, scope, p.ID)
		switch {
		case err == nil:
			b.Analysis = a
			if b.MET, err = e.S.MatchEquityTables().OfAnalysis(ctx, scope, p.ID); err != nil {
				return err
			}
		case errors.Is(err, storage.ErrNotFound):
			// no analysis for this position
		default:
			return err
		}
		if b.MET != 0 && !written[b.MET] {
			t, err := e.S.MatchEquityTables().Load(ctx, scope, b.MET)
			if err != nil {
				return err
			}
			// The receiver keeps its own choice of table (ADR-0007).
			t.Current, t.CreatedAt = false, ""
			if err := enc.Encode(positionBundle{Table: t}); err != nil {
				return err
			}
			written[b.MET] = true
		}
		for c, err := range e.S.Comments().ByPosition(ctx, scope, p.ID) {
			if err != nil {
				return err
			}
			b.Comments = append(b.Comments, c.Text)
			b.CommentAuthors = append(b.CommentAuthors, c.Author)
		}
		if !slices.ContainsFunc(b.CommentAuthors, func(a string) bool { return a != "" }) {
			b.CommentAuthors = nil
		}
		if err := enc.Encode(b); err != nil {
			return err
		}
		if fl != nil {
			fl.Flush()
		}
	}
	return nil
}

// JSONImporter reads the NDJSON interchange and writes it through Storage
// inside a single transaction (atomic, cancellable).
type JSONImporter struct{ S storage.Storage }

func (im JSONImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	r, closeFn, err := openSource(src)
	if err != nil {
		return Summary{}, err
	}
	defer closeFn()

	tx, err := im.S.BeginTx(ctx)
	if err != nil {
		return Summary{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var sum Summary
	tables := mets.NewCarrier(tx.MatchEquityTables(), scope, nil)
	dec := json.NewDecoder(r)
	for dec.More() {
		if err := ctx.Err(); err != nil {
			return sum, err
		}
		var b positionBundle
		if err := dec.Decode(&b); err != nil {
			return sum, fmt.Errorf("ingest: decode bundle: %w", err)
		}
		if b.Table != nil {
			tables.Add(b.Table)
		}
		if b.Position == nil {
			continue
		}
		// NDJSON is a faithful interchange of a whole position library, not a
		// hand-picked position file: provenance is carried in the record and
		// honoured as-is (ADR-0001), exactly like a database-to-database copy.
		// Forcing the flag here would mark every position of a restored backup
		// as individually imported. Records written before the flag existed
		// simply decode to false.
		id, err := tx.Positions().Save(ctx, scope, b.Position)
		if err != nil {
			return sum, err
		}
		if b.Analysis != nil {
			// A record citing a table the stream never carried fails the
			// import rather than read its verdict under another table.
			met, err := tables.Target(ctx, b.MET)
			if err != nil {
				return sum, fmt.Errorf("ingest: position %d: %w", sum.SavedPositions+1, err)
			}
			if err := mergeDBAnalysis(ctx, tx, scope, id, b.Analysis, met); err != nil {
				return sum, err
			}
		}
		for i, txt := range b.Comments {
			if _, err := tx.Comments().Add(storage.WithCommentAuthor(ctx, b.commentAuthor(i)), scope, id, txt); err != nil {
				return sum, err
			}
		}
		sum.SavedPositions++
		if prog != nil {
			prog(Progress{Positions: sum.SavedPositions})
		}
	}
	if err := ctx.Err(); err != nil {
		return sum, err
	}
	if err := tx.Commit(); err != nil {
		return sum, err
	}
	committed = true
	return sum, nil
}

// openSource yields a reader for src, preferring an in-memory Reader and
// falling back to opening Path. The returned closeFn is always safe to call.
func openSource(src Source) (io.Reader, func(), error) {
	if src.Reader != nil {
		return src.Reader, func() {}, nil
	}
	if src.Path != "" {
		f, err := os.Open(src.Path)
		if err != nil {
			return nil, func() {}, fmt.Errorf("ingest: open source: %w", err)
		}
		return f, func() { f.Close() }, nil
	}
	return nil, func() {}, errors.New("ingest: source has neither Reader nor Path")
}
