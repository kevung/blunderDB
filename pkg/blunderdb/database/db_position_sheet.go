package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
)

// PositionSheetSource names the list a position sheet prints: ids, a query,
// or a collection.
type PositionSheetSource = printsheet.PositionSource

// PositionSheetOptions are the choices of the position list's print dialog.
type PositionSheetOptions = printsheet.PositionsOptions

// PositionSheetHTML renders the printable list of every position src names
// — the whole list, at most printsheet.MaxPositions — one line a position in
// ArkID, in one self-contained HTML document. It writes nothing to the
// library (ADR-0007).
func (d *Database) PositionSheetHTML(src PositionSheetSource, opts PositionSheetOptions) (string, error) {
	return PositionSheetHTMLCtx(context.Background(), d, src, opts)
}

// PositionSheetHTMLCtx is PositionSheetHTML under ctx: cancelling it stops
// the reading or, once read, before the codes are drawn. A function, not a
// method: the interface binds every method of Database.
func PositionSheetHTMLCtx(ctx context.Context, d *Database, src PositionSheetSource, opts PositionSheetOptions) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	page, err := printsheet.BuildPositions(ctx, d.store, "", src, opts)
	return string(page), err
}
