package database

import (
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ImportOGXMMatch imports a match from a HedgeHog / OpenGammon .ogxm file,
// delegating to the shared ingest pipeline (ingest.MapOGXM ->
// ingest.WriteMatch), the same path the headless server uses.
func (d *Database) ImportOGXMMatch(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graph, err := ingest.MapOGXM(filePath)
	if err != nil {
		return 0, err
	}
	matchID, err := d.writeImportedMatch(ctx, graph)
	if err != nil {
		return 0, err
	}
	slog.Info("imported OGXM match", "matchID", matchID, "file", filePath)
	return matchID, nil
}
