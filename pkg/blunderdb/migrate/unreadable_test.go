package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestRunFailsOnUnreadableAnalysis copies a source one of whose analyses no
// longer decodes. The batch read only warns and leaves it out, so the copy
// must notice the shortfall at the end of the position phase and fail as a
// whole: a migration that silently dropped an analysis would pass for
// complete.
func TestRunFailsOnUnreadableAnalysis(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "src.db")
	src, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = src.Close() })
	if err := src.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var broken int64
	for i := range 3 {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Score = [2]int{i, 0}
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		a := &domain.PositionAnalysis{XGID: fmt.Sprintf("xgid-%d", i), AnalysisType: "XG Roller++"}
		if err := src.Analyses().Save(ctx, "", id, a); err != nil {
			t.Fatal(err)
		}
		broken = id
	}
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.Exec(`UPDATE analysis SET data = ? WHERE position_id = ?`, []byte("\x00not an analysis"), broken); err != nil {
		t.Fatal(err)
	}

	dst := openMemory(t)
	_, err = Run(ctx, src, dst, "1", Options{})
	if err == nil || !strings.Contains(err.Error(), "1 of the source's 3 analyses could not be read") {
		t.Fatalf("Run error = %v, want the unread analysis reported", err)
	}
	counts, err := dst.Metadata().Counts(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if left := nonEmpty(counts); left != "" {
		t.Fatalf("destination holds %s after a failed migration, want nothing", left)
	}
}
