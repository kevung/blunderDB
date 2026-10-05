package ingest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestSQLiteExportCarriesTheDuel: the origin of a Match played here travels
// with the Match, whatever the selection, so the copy refuses the .mat of a
// Match with a Start; the Duels in suspense travel with a whole export only,
// seed included.
func TestSQLiteExportCarriesTheDuel(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	matchID, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3, MatchHash: "h-duel"})
	if err != nil {
		t.Fatal(err)
	}
	origin := storage.MatchOrigin{MatchID: matchID, Start: "-b----E-C---eE---c-e----B-:1:1:1:00:0:0:0:3:10", DiceSeed: "ab", StoppedEarly: true}
	if err := src.Duels().SetOrigin(ctx, "", &origin); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Duels().Save(ctx, "", &storage.Duel{FormatVersion: "1", Label: "C — D", Document: "{}", DiceSeed: "cd"}); err != nil {
		t.Fatal(err)
	}

	exportTo := func(t *testing.T, opts ExportOptions) storage.Storage {
		t.Helper()
		outPath := filepath.Join(t.TempDir(), "export.sqlite")
		out, err := os.Create(outPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := (SQLiteExporter{S: src}).Export(ctx, "", out, opts); err != nil {
			out.Close()
			t.Fatalf("export: %v", err)
		}
		out.Close()
		dst, err := sqlite.Open(ctx, outPath, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { dst.Close() })
		return dst
	}
	onlyMatch := func(t *testing.T, dst storage.Storage) int64 {
		t.Helper()
		var ids []int64
		for m, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, m.ID)
		}
		if len(ids) != 1 {
			t.Fatalf("exported matches = %v, want one", ids)
		}
		return ids[0]
	}
	duels := func(dst storage.Storage) []*storage.Duel {
		var out []*storage.Duel
		for d, err := range dst.Duels().List(ctx, "") {
			if err == nil {
				out = append(out, d)
			}
		}
		return out
	}

	t.Run("whole export", func(t *testing.T) {
		dst := exportTo(t, WholeTenant(FormatSQLite))
		id := onlyMatch(t, dst)
		got, err := dst.Duels().Origin(ctx, "", id)
		want := origin
		want.MatchID = id
		if err != nil || *got != want {
			t.Errorf("origin = %+v, %v; want %+v", got, err, want)
		}
		if _, _, _, err := ReadMatchForMAT(ctx, dst, "", id); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf(".mat of the copy: got %v, want ErrInvalid", err)
		}
		if d := duels(dst); len(d) != 1 || d[0].DiceSeed != "cd" || d[0].Label != "C — D" {
			t.Errorf("exported duels = %+v", d)
		}
	})

	t.Run("selected match", func(t *testing.T) {
		opts := WholeTenant(FormatSQLite)
		opts.Selection.AllMatches = false
		opts.Selection.MatchIDs = []int64{matchID}
		dst := exportTo(t, opts)
		id := onlyMatch(t, dst)
		if got, err := dst.Duels().Origin(ctx, "", id); err != nil || got.Start != origin.Start {
			t.Errorf("origin of a selected match = %+v, %v", got, err)
		}
		if d := duels(dst); len(d) != 0 {
			t.Errorf("a filtered export carries no Duel in suspense: %+v", d)
		}
	})
}
