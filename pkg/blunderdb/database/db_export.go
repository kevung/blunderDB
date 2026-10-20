package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ExportDatabase writes the selection described by opts into a new database
// file. It is the GUI's export dialog and the CLI's `export`; the work is
// ingest.ExportSQLite; this method only seals the watermark with this
// machine's identity and maps the options to a Selection. Uses
// context.Background(); prefer ExportDatabaseCtx when the caller can cancel.
// The report counts what the file holds, family by family.
func (d *Database) ExportDatabase(opts ExportOptions) (ingest.ExportReport, error) {
	return d.ExportDatabaseCtx(context.Background(), opts)
}

// ExportDatabaseCtx is ExportDatabase with a caller-supplied context that
// aborts an in-flight export.
func (d *Database) ExportDatabaseCtx(ctx context.Context, opts ExportOptions) (ingest.ExportReport, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return ingest.ExportReport{}, fmt.Errorf("no database is currently open")
	}

	// Seal the watermark before writing anything: it is the producer's statement about this
	// file, and a failure to sign must not leave a half-made export behind.
	watermarkDocument, err := sealWatermark(opts.Watermark, opts.WatermarkNote)
	if err != nil {
		return ingest.ExportReport{}, fmt.Errorf("cannot sign the watermark: %w", err)
	}

	if opts.Incremental {
		if opts.ExportPath, err = SyncExportPath(opts.ExportPath, time.Now()); err != nil {
			return ingest.ExportReport{}, err
		}
	}
	plan, err := d.exportPlan(ctx, opts)
	if err != nil {
		return ingest.ExportReport{}, err
	}
	plan.Metadata = opts.Metadata
	plan.Watermark = watermarkDocument
	plan.Password = opts.Password
	return ingest.ExportSQLite(ctx, d.store, "", opts.ExportPath, plan)
}

// PlanExport counts, family by family, what ExportDatabase would write for the same
// options, and names the living collections that would leave frozen. It writes nothing:
// the export dialog shows it before the user commits to a file.
func (d *Database) PlanExport(opts ExportOptions) (ingest.ExportPlan, error) {
	return d.PlanExportCtx(context.Background(), opts)
}

// PlanExportCtx is PlanExport with a caller-supplied context.
func (d *Database) PlanExportCtx(ctx context.Context, opts ExportOptions) (ingest.ExportPlan, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return ingest.ExportPlan{}, fmt.Errorf("no database is currently open")
	}
	plan, err := d.exportPlan(ctx, opts)
	if err != nil {
		return ingest.ExportPlan{}, err
	}
	return ingest.PlanExport(ctx, d.store, "", plan)
}

// PlanExportIDs pages through the source ids behind one count of PlanExport — family is
// one of ingest.Plan* — so the dialog can open what a count counts; a page holds at most
// ingest.MaxPlannedIDs.
func (d *Database) PlanExportIDs(opts ExportOptions, family string, offset, limit int) (ingest.PlannedIDs, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return ingest.PlannedIDs{}, fmt.Errorf("no database is currently open")
	}
	plan, err := d.exportPlan(context.Background(), opts)
	if err != nil {
		return ingest.PlannedIDs{}, err
	}
	return ingest.PlanExportIDs(context.Background(), d.store, "", plan, family, offset, limit)
}

// exportPlan maps the GUI's and the CLI's options onto what ingest exports, through the
// reading the daemon shares (ingest.OptionsFor).
func (d *Database) exportPlan(ctx context.Context, opts ExportOptions) (ingest.ExportOptions, error) {
	opts.SyncCursor = SyncCursorFor(opts)
	return ingest.OptionsFor(ctx, d.store, "", opts)
}

// SyncCursorFor names the cursor an incremental export of the desktop or the CLI reads:
// the caller's when given, else the destination folder, so that each folder another
// machine watches keeps its own "last synchronised" mark (ADR-0096).
// ExportPath may name the folder itself, as it does before SyncExportPath picks the file.
func SyncCursorFor(opts ExportOptions) string {
	if !opts.Incremental || opts.SyncCursor != "" || opts.ExportPath == "" {
		return opts.SyncCursor
	}
	dir := opts.ExportPath
	if !isDir(dir) {
		dir = filepath.Dir(dir)
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return dir
}

// SyncExportPath names the file an incremental export writes when its destination is a
// folder; a file path is kept as given, and refused when a file is already there. The file
// an incremental export would replace holds the previous delta, which the cursor counts as
// sent: if the other machine has not merged it yet, overwriting it loses those rows for
// good, and the producer cannot know whether it has (ADR-0007).
func SyncExportPath(path string, now time.Time) (string, error) {
	if !isDir(path) {
		if _, err := os.Lstat(path); err == nil {
			return "", fmt.Errorf("an incremental export never overwrites a file: %s may hold a delta the other machine has not merged yet; name its folder, and each export gets a new name", path)
		}
		return path, nil
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "blunderdb"
	}
	stem := filepath.Join(path, fmt.Sprintf("sync-%s-%s", host, now.Format("20060102-150405")))
	name := stem + ".db"
	for n := 2; ; n++ {
		if _, err := os.Lstat(name); os.IsNotExist(err) {
			return name, nil
		}
		name = fmt.Sprintf("%s-%d.db", stem, n)
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// MatchIDsReachedBy returns the matches whose moves were played in one of these positions:
// what an export of these positions carries when its matches are left unnamed.
func (d *Database) MatchIDsReachedBy(positionIDs []int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Matches().MatchIDsReachedBy(context.Background(), "", positionIDs)
}

// DeleteFile is a helper function to delete a file
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}
