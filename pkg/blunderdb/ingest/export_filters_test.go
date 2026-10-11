package ingest

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A saved filter that names a match by id has it renumbered to the file's;
// one naming a match the export leaves behind does not leave; a whole export
// carries every filter.
func TestExportRewritesSavedFilterIDs(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	left := &domain.Match{Player1Name: "Zed", Player2Name: "Yann", MatchLength: 5}
	kept := &domain.Match{Player1Name: "Ann", Player2Name: "Bob", MatchLength: 5}
	for _, m := range []*domain.Match{left, kept} {
		if _, err := src.Matches().Save(ctx, "", m); err != nil {
			t.Fatal(err)
		}
	}
	keptRef, leftRef := "ma"+strconv.FormatInt(kept.ID, 10), "ma"+strconv.FormatInt(left.ID, 10)
	for name, command := range map[string]string{
		"kept": "s " + keptRef, "narrow": "ss " + keptRef, "left": "s " + leftRef, "plain": "s",
	} {
		if _, err := src.Filters().Save(ctx, "", name, command); err != nil {
			t.Fatal(err)
		}
	}

	export := func(opts ExportOptions) map[string]string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "export.sqlite")
		if _, err := ExportSQLite(ctx, src, "", path, opts); err != nil {
			t.Fatal(err)
		}
		dst, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer dst.Close()
		out := map[string]string{}
		for f, err := range dst.Filters().List(ctx, "") {
			if err != nil {
				t.Fatal(err)
			}
			out[f.Name] = f.Command
		}
		return out
	}

	got := export(ExportOptions{Format: FormatSQLite, FilterLibrary: true,
		Selection: Selection{MatchIDs: []int64{kept.ID}}})
	if _, ok := got["left"]; ok {
		t.Errorf("a filter naming a match left behind was exported: %q", got["left"])
	}
	if got["plain"] != "s" {
		t.Errorf("a filter without ids = %q, want it unchanged", got["plain"])
	}
	for name, verb := range map[string]string{"kept": "s ", "narrow": "ss "} {
		cmd, ok := got[name]
		if !ok {
			t.Fatalf("filter %q missing from %v", name, got)
		}
		if f, _ := searchquery.Parse(cmd); f.MatchIDsFilter != "1" || cmd[:len(verb)] != verb {
			t.Errorf("filter %q = %q, want %s naming the file's match 1", name, cmd, verb)
		}
	}

	if all := export(WholeTenant(FormatSQLite)); len(all) != 4 {
		t.Errorf("whole export carried filters %v, want all four", all)
	}
}
