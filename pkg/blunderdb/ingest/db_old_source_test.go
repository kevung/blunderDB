package ingest

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestDBImportReadsASourceWithoutLessonTables: a database from before Lessons
// existed has no lesson table; it is a source without lessons, not an error.
func TestDBImportReadsASourceWithoutLessonTables(t *testing.T) {
	ctx := context.Background()
	srcPath := makeSourceDB(t)

	raw, err := sql.Open("sqlite", srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`DROP TABLE lesson_step; DROP TABLE lesson;`); err != nil {
		t.Fatalf("drop lesson tables: %v", err)
	}
	raw.Close()

	target, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	sum, err := DBImporter{S: target}.Import(ctx, "", Source{Format: FormatNativeDB, Path: srcPath}, nil)
	if err != nil {
		t.Fatalf("import of a source without lesson tables: %v", err)
	}
	if sum.SavedPositions == 0 {
		t.Fatalf("nothing imported: %+v", sum)
	}
}
