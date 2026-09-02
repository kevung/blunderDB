package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// seedDeck builds a collection of n positions from the XG fixture and a deck
// drawn from it, synced so every position has a card.
func seedDeck(t *testing.T, cli *CLI, n int) (deckID int64, positionIDs []int64) {
	t.Helper()
	colID, ids := seedCollection(t, cli, "Openings", n)
	deckID, err := cli.db.CreateAnkiDeck("Drill", "", AnkiSourceCollection, colID, "")
	if err != nil {
		t.Fatalf("CreateAnkiDeck: %v", err)
	}
	if err := cli.db.SyncAnkiDeck(deckID); err != nil {
		t.Fatalf("SyncAnkiDeck: %v", err)
	}
	return deckID, ids
}

func TestCLI_AnkiDecks(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "decks", "--db", dbPath}); err != nil {
			t.Fatalf("anki decks: %v", err)
		}
	})
	if !strings.Contains(out, "No decks found") {
		t.Errorf("expected empty-DB message, got:\n%s", out)
	}

	deckID, ids := seedDeck(t, cli, 3)

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "decks", "--db", dbPath}); err != nil {
			t.Fatalf("anki decks: %v", err)
		}
	})
	if !strings.Contains(out, "Drill") || !strings.Contains(out, "collection 1") {
		t.Errorf("text listing missing the deck:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "decks", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("anki decks --format json: %v", err)
		}
	})
	var decks []AnkiDeck
	if err := json.Unmarshal([]byte(out), &decks); err != nil {
		t.Fatalf("json output does not parse: %v\n%s", err, out)
	}
	if len(decks) != 1 || decks[0].ID != deckID || decks[0].CardCount != len(ids) || decks[0].NewCount != len(ids) {
		t.Errorf("json decks = %+v, want deck %d with %d new cards", decks, deckID, len(ids))
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "decks", "--db", dbPath, "--format", "csv"}); err != nil {
			t.Fatalf("anki decks --format csv: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || lines[0] != "id,name,source,cards,due,new,request_retention" {
		t.Errorf("csv output = %q, want a header and one row", out)
	}
}

func TestCLI_AnkiStats(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	deckID, ids := seedDeck(t, cli, 4)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "stats", "--db", dbPath, "--deck", strconv.FormatInt(deckID, 10)}); err != nil {
			t.Fatalf("anki stats: %v", err)
		}
	})
	if !strings.Contains(out, "Drill") || !strings.Contains(out, "Total cards:") {
		t.Errorf("text stats missing the deck header:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "stats", "--db", dbPath, "--deck", "1", "--format", "json"}); err != nil {
			t.Fatalf("anki stats --format json: %v", err)
		}
	})
	var got struct {
		Deck  AnkiDeck      `json:"deck"`
		Stats AnkiDeckStats `json:"stats"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json output does not parse: %v\n%s", err, out)
	}
	if got.Deck.ID != deckID || got.Stats.TotalCount != len(ids) || got.Stats.NewCount != len(ids) {
		t.Errorf("json stats = %+v, want %d new cards on deck %d", got, len(ids), deckID)
	}

	if err := cli.Run([]string{"anki", "stats", "--db", dbPath, "--deck", "99"}); err == nil {
		t.Error("expected an error for an unknown deck id")
	}
}

func TestCLI_AnkiForecast(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	_, ids := seedDeck(t, cli, 3)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "forecast", "--db", dbPath, "--deck", "1", "--days", "7", "--format", "json"}); err != nil {
			t.Fatalf("anki forecast: %v", err)
		}
	})
	var days []AnkiForecastDay
	if err := json.Unmarshal([]byte(out), &days); err != nil {
		t.Fatalf("json output does not parse: %v\n%s", err, out)
	}
	if len(days) != 7 {
		t.Fatalf("forecast spans %d days, want 7", len(days))
	}
	// Fresh cards are due now: they all land on day 0.
	if days[0].Due != len(ids) {
		t.Errorf("day 0 due = %d, want %d", days[0].Due, len(ids))
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "forecast", "--db", dbPath, "--days", "7"}); err != nil {
			t.Fatalf("anki forecast (all decks): %v", err)
		}
	})
	if !strings.Contains(out, "3 card(s) due over 7 day(s)") {
		t.Errorf("text forecast total missing:\n%s", out)
	}
}

func TestCLI_AnkiSync(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	colID, ids := seedCollection(t, cli, "Openings", 2)

	// Collection deck: created empty, the sync draws its cards.
	deckID, err := cli.db.CreateAnkiDeck("Drill", "", AnkiSourceCollection, colID, "")
	if err != nil {
		t.Fatalf("CreateAnkiDeck: %v", err)
	}
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "sync", "--db", dbPath, "--deck", strconv.FormatInt(deckID, 10)}); err != nil {
			t.Fatalf("anki sync: %v", err)
		}
	})
	if !strings.Contains(out, "2 card(s), 2 added") {
		t.Errorf("sync report = %q", out)
	}

	// Search deck in the GUI's JSON form: the stored ids are the source.
	positions, _ := cli.db.LoadAllPositions()
	extra := positions[len(positions)-1].ID
	source := `{"command":"s c","position":"{}","ids":[` + strconv.FormatInt(ids[0], 10) + `,` + strconv.FormatInt(extra, 10) + `]}`
	searchDeck, err := cli.db.CreateAnkiDeck("Cube", "", AnkiSourceSearch, 0, source)
	if err != nil {
		t.Fatalf("CreateAnkiDeck(search): %v", err)
	}
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "sync", "--db", dbPath, "--deck", strconv.FormatInt(searchDeck, 10)}); err != nil {
			t.Fatalf("anki sync (search): %v", err)
		}
	})
	if !strings.Contains(out, "2 card(s), 2 added") {
		t.Errorf("search sync report = %q", out)
	}
	deckPositions, err := cli.db.GetAnkiDeckPositions(searchDeck)
	if err != nil {
		t.Fatalf("GetAnkiDeckPositions: %v", err)
	}
	if len(deckPositions) != 2 {
		t.Errorf("search deck holds %d positions, want the 2 stored ids", len(deckPositions))
	}

	// Syncing again adds nothing.
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"anki", "sync", "--db", dbPath, "--deck", strconv.FormatInt(searchDeck, 10)}); err != nil {
			t.Fatalf("anki sync (again): %v", err)
		}
	})
	if !strings.Contains(out, "2 card(s), 0 added") {
		t.Errorf("second sync report = %q", out)
	}
}

func TestCLI_AnkiUsageErrors(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	cases := [][]string{
		{"anki"},
		{"anki", "bogus", "--db", dbPath},
		{"anki", "decks"},
		{"anki", "stats", "--db", dbPath},
		{"anki", "sync", "--db", dbPath},
		{"anki", "forecast", "--db", dbPath, "--deck", "42"},
		{"anki", "decks", "--db", dbPath, "--format", "xml"},
	}
	for _, args := range cases {
		var err error
		captureStdout(t, func() { err = cli.Run(args) })
		if err == nil {
			t.Errorf("%v: expected an error", args)
		}
	}
}
