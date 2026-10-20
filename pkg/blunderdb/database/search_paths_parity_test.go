package database

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestSearchUnboundedMatchesPagedOnRealMatches holds the unbounded search
// (the whole result at once: ids, positions, count) to the answer of the
// paged one, page after page, on a library of real matches. The two take
// different SQL for the move-error filter, so on that filter they must still
// agree id for id and in the same order. Some positions are given a second,
// different play by player 1, which is what that filter treats apart.
func TestSearchUnboundedMatchesPagedOnRealMatches(t *testing.T) {
	db := newTestDB(t)
	for _, f := range []string{
		"charlot1-charlot2_7p_2025-11-08-2305.xg",
		"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
		"Kev_GammonNetNormal_2026-10-06_7p.xg",
	} {
		if _, err := db.ImportXGMatch(filepath.Join("testdata", f)); err != nil {
			t.Fatalf("ImportXGMatch %s: %v", f, err)
		}
	}
	// Every 7th checker play of player 1 is played again another way.
	if _, err := db.db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action)
		SELECT game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move || ' 6/5', cube_action
		FROM move WHERE player = 1 AND checker_move <> '' AND id % 7 = 0`); err != nil {
		t.Fatalf("second plays: %v", err)
	}
	var multi int
	if err := db.db.QueryRow(`SELECT COUNT(DISTINCT position_id) FROM move WHERE player = 1 AND checker_move LIKE '% 6/5'`).Scan(&multi); err != nil || multi == 0 {
		t.Fatalf("no position played twice (%d, %v)", multi, err)
	}

	const page = 60
	for _, sortKey := range []string{"", "error", "pip_desc"} {
		for _, expr := range []string{"E>100", "E>20", "E<30", "E20,80", "E>0"} {
			f := SearchFilters{Filter: emptyFilter(), MoveErrorFilter: expr, Sort: sortKey}
			name := expr + "/" + sortKey

			var pagedIDs []int64
			var pagedPositions []Position
			for offset := 0; ; offset += page {
				ids, err := db.SearchPositionIDs(f, offset, page)
				if err != nil {
					t.Fatalf("%s: SearchPositionIDs: %v", name, err)
				}
				positions, _, err := db.LoadPositionsByFiltersCore(f, storage.ListOpts{Offset: offset, Limit: page})
				if err != nil {
					t.Fatalf("%s: LoadPositionsByFiltersCore: %v", name, err)
				}
				pagedIDs = append(pagedIDs, ids...)
				pagedPositions = append(pagedPositions, positions...)
				if len(ids) < page {
					break
				}
			}
			if len(pagedIDs) == 0 {
				t.Fatalf("%s: the paged search finds nothing", name)
			}

			ids, err := db.LoadPositionIDsByFilters(f)
			if err != nil {
				t.Fatalf("%s: LoadPositionIDsByFilters: %v", name, err)
			}
			if !reflect.DeepEqual(ids, pagedIDs) {
				t.Errorf("%s: unbounded ids %v, paged %v", name, ids, pagedIDs)
			}
			positions, err := db.LoadPositionsByFilters(f)
			if err != nil {
				t.Fatalf("%s: LoadPositionsByFilters: %v", name, err)
			}
			if !reflect.DeepEqual(positions, pagedPositions) {
				t.Errorf("%s: unbounded positions differ from the paged ones (%d vs %d)", name, len(positions), len(pagedPositions))
			}
			core, analyses, err := db.LoadPositionsByFiltersCore(f, storage.ListOpts{})
			if err != nil {
				t.Fatalf("%s: LoadPositionsByFiltersCore: %v", name, err)
			}
			if !reflect.DeepEqual(core, pagedPositions) {
				t.Errorf("%s: unbounded core positions differ from the paged ones", name)
			}
			for _, p := range core {
				if _, ok := analyses[p.ID]; !ok {
					t.Errorf("%s: no analysis returned for %d", name, p.ID)
					break
				}
			}
			n, err := db.CountPositionsByFilters(f)
			if err != nil {
				t.Fatalf("%s: CountPositionsByFilters: %v", name, err)
			}
			if n != len(pagedIDs) {
				t.Errorf("%s: count %d, paged %d", name, n, len(pagedIDs))
			}
		}
	}
}
