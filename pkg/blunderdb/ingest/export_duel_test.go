package ingest

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestSQLiteExportCarriesTheDuel: the origin of a Match played here travels
// with the Match, whatever the selection, so the copy refuses the .mat of a
// Match with a Start. A Duel in suspense never travels, a whole export
// included: its seed is every roll to come, and nothing of it may be found in
// the file (ADR-0072 rule 11).
func TestSQLiteExportCarriesTheDuel(t *testing.T) {
	const pendingSeed = "5eedc0ffee5eedc0ffee5eedc0ffee5eed"
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
	origin := storage.MatchOrigin{MatchID: matchID, Start: "-b----E-C---eE---c-e----B-:1:1:1:00:0:0:0:3:10", DiceSeed: "ab",
		StoppedEarly: true, OverTime: 2, Cadence: `{"reserve":180,"delay":12,"timeOut":"lose_match"}`,
		DeclaredBots: []storage.DeclaredBot{{Player: 1, Configuration: "normal", Engine: "v1.6.0"}}}
	gameID, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	decision, cube := int64(4200), int64(1300)
	for i, mv := range []domain.Move{
		{MoveType: "checker", Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5", DecisionMS: &decision, CubeDecisionMS: &cube},
		{MoveType: "checker", Player: -1, Dice: [2]int32{6, 5}, CheckerMove: "24/13"},
	} {
		mv.GameID, mv.MoveNumber = gameID, int32(i)
		if _, err := src.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatal(err)
		}
	}
	if err := src.Duels().SetOrigin(ctx, "", &origin); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Duels().Save(ctx, "", &storage.Duel{FormatVersion: "1", Label: "C — D", Document: "{}", DiceSeed: pendingSeed}); err != nil {
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
		raw, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(raw, []byte(pendingSeed)) {
			t.Errorf("the seed of a Duel in suspense is in the exported file")
		}
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
	duels := func(dst storage.Storage) []*storage.DuelEntry {
		var out []*storage.DuelEntry
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
		if err != nil || !reflect.DeepEqual(*got, want) {
			t.Errorf("origin = %+v, %v; want %+v", got, err, want)
		}
		if _, _, _, err := ReadMatchForMAT(ctx, dst, "", id); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf(".mat of the copy: got %v, want ErrInvalid", err)
		}
		if d := duels(dst); len(d) != 0 {
			t.Errorf("a whole export carries no Duel in suspense: %+v", d)
		}
		// The durations of the decisions travel with the moves; an unknown
		// one stays unknown.
		var moves []*domain.Move
		for mv, err := range dst.Matches().MovesByMatch(ctx, "", id) {
			if err != nil {
				t.Fatal(err)
			}
			moves = append(moves, mv)
		}
		if len(moves) != 2 || moves[0].DecisionMS == nil || *moves[0].DecisionMS != decision ||
			moves[0].CubeDecisionMS == nil || *moves[0].CubeDecisionMS != cube ||
			moves[1].DecisionMS != nil || moves[1].CubeDecisionMS != nil {
			t.Errorf("exported moves = %+v", moves)
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
