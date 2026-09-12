package database

// search_encounter_test.go — le jeton `n` (rencontres, #282) répond la même
// chose quel que soit le chemin qui le porte (#362) : la ligne de commande
// (`blunderdb search --query 's n>3'`, qui passe par searchquery.Parse), la
// charge utile JSON que l'application envoie à LoadPositionIDsByFilters, et une
// collection vivante dont la requête le porte. Avant #362, l'application
// n'envoyait pas le champ : la recherche partait sans lui et rendait tout.

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func TestEncounterToken_SameSetOnEveryPath(t *testing.T) {
	t.Parallel()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "encounter.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	// Trois positions distinctes : rencontrée quatre fois, deux fois, jamais.
	save := func(checkers int) int64 {
		p := InitializePosition()
		p.Board.Points[6].Checkers = checkers
		id, err := db.SavePosition(&p)
		if err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
		return id
	}
	often, twice, never := save(3), save(4), save(2)

	rawDB := db.db
	matchRes, err := rawDB.Exec(`INSERT INTO match (player1_name, player2_name, match_length, match_date, import_date, game_count, match_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, "Alice", "Bob", 7, time.Now(), time.Now(), 1, "hash-encounter-test")
	if err != nil {
		t.Fatalf("insert match: %v", err)
	}
	matchID, _ := matchRes.LastInsertId()
	gameRes, err := rawDB.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, matchID, 1, 0, 0, 1, 1, 6)
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	gameID, _ := gameRes.LastInsertId()
	moveNumber := 0
	meet := func(positionID int64, times int) {
		for range times {
			moveNumber++
			if _, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, gameID, moveNumber, "checker", positionID, 0, 3, 1, "8/5 6/5"); err != nil {
				t.Fatalf("insert move: %v", err)
			}
		}
	}
	meet(often, 4)
	meet(twice, 2)

	// La ligne de commande : la requête passe par la grammaire Go.
	viaCLI := func(query string) []int64 {
		t.Helper()
		filters, diags := searchquery.Parse(query)
		for _, d := range diags {
			if d.Kind == searchquery.DiagUnknown {
				t.Fatalf("%q: unknown token %q", query, d.Token)
			}
		}
		positions, _, err := db.LoadPositionsByFiltersCore(filters, storage.ListOpts{})
		if err != nil {
			t.Fatalf("LoadPositionsByFiltersCore(%q): %v", query, err)
		}
		return sortedIDs(positions)
	}

	// L'application : la charge utile telle que buildSearchFilterPayload /
	// loadPositionsByFilters la sérialisent (champs vides compris), décodée
	// comme la liaison Wails la décode.
	viaApp := func(encounter string) []int64 {
		t.Helper()
		payload := `{"includeCube":false,"includeScore":false,"pipCountFilter":"","searchText":"",
			"commentFilter":"","gamePhaseFilter":"","gameTypeFilter":"","tagFilter":"",
			"encounterFilter":"` + encounter + `","likeFilter":false,"likeTargetId":0,"likeMaxDistance":0,
			"decisionTypeFilter":false,"cubeResponseFilter":"","diceRollFilter":false,"diceRollMode":"both",
			"exceptDiceFilter":"","matchIDsFilter":"","tournamentIDsFilter":"","playerFilter":"",
			"positionIDsFilter":"","restrictToPositionIDs":""}`
		var f SearchFilters
		if err := json.Unmarshal([]byte(payload), &f); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		ids, err := db.LoadPositionIDsByFilters(f)
		if err != nil {
			t.Fatalf("LoadPositionIDsByFilters: %v", err)
		}
		slices.Sort(ids)
		return ids
	}

	for _, tc := range []struct {
		query, token string
		want         []int64
	}{
		{"s n>3", "n>3", []int64{often}},
		{"s n2,5", "n2,5", []int64{often, twice}},
		{"s n4", "n4,4", []int64{often}},
		{"s n<1", "n<1", []int64{never}},
	} {
		if got := viaCLI(tc.query); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("CLI %q = %v, want %v", tc.query, got, tc.want)
		}
		if got := viaApp(tc.token); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("app payload encounterFilter=%q = %v, want %v", tc.token, got, tc.want)
		}
	}

	// Une collection vivante se recalcule à chaque ouverture, sur le bon
	// ensemble, et suit la base quand un import ajoute des rencontres.
	colID, err := db.CreateCollection("Souvent vues", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := db.SetCollectionFilter(colID, "s n>3"); err != nil {
		t.Fatalf("SetCollectionFilter: %v", err)
	}
	living := func() []int64 {
		t.Helper()
		positions, err := db.GetCollectionPositions(colID)
		if err != nil {
			t.Fatalf("GetCollectionPositions: %v", err)
		}
		return sortedIDs(positions)
	}
	if got, want := living(), []int64{often}; !reflect.DeepEqual(got, want) {
		t.Errorf("living collection `s n>3` = %v, want %v", got, want)
	}
	meet(twice, 2)
	if got, want := living(), []int64{often, twice}; !reflect.DeepEqual(got, want) {
		t.Errorf("living collection `s n>3` after two more encounters = %v, want %v", got, want)
	}
}
