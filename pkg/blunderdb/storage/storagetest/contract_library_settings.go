// Contract cases for the library's own settings — the error and blunder
// thresholds of ADR-0043. The two backends store them in different tables
// (metadata on SQLite, the tenant-scoped library_settings on PostgreSQL), so
// what is held identical here is the behaviour: the defaults a library that
// never set them reads, the nesting rule, and the round trip.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testLibrarySettingsDefaults(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	got, err := s.LibrarySettings().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load on a fresh library: %v", err)
	}
	want := storage.DefaultLibrarySettings()
	if got != want {
		t.Fatalf("fresh library: got %+v, want %+v", got, want)
	}
	// The defaults are the values the code used as constants before they
	// became settings; a change here moves every blunder count ever shown.
	if want.ErrorThresholdMP != 50 || want.BlunderThresholdMP != 100 {
		t.Fatalf("defaults drifted: %+v, want {50 100}", want)
	}
}

func testLibrarySettingsRoundTrip(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ls := s.LibrarySettings()

	set := storage.LibrarySettings{ErrorThresholdMP: 20, BlunderThresholdMP: 80}
	if err := ls.Save(ctx, "", set); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := ls.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != set {
		t.Fatalf("round trip: got %+v, want %+v", got, set)
	}

	// Saving again replaces rather than duplicating: the pair is two rows
	// keyed by name, not an append-only log.
	again := storage.LibrarySettings{ErrorThresholdMP: 40, BlunderThresholdMP: 200}
	if err := ls.Save(ctx, "", again); err != nil {
		t.Fatalf("Save again: %v", err)
	}
	got, err = ls.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load again: %v", err)
	}
	if got != again {
		t.Fatalf("second round trip: got %+v, want %+v", got, again)
	}
}

func testLibrarySettingsRejectsInverted(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ls := s.LibrarySettings()

	ok := storage.LibrarySettings{ErrorThresholdMP: 30, BlunderThresholdMP: 90}
	if err := ls.Save(ctx, "", ok); err != nil {
		t.Fatalf("Save valid: %v", err)
	}

	// Every Blunder is an Error, so an error threshold above the blunder
	// threshold names a set that cannot exist.
	bad := storage.LibrarySettings{ErrorThresholdMP: 120, BlunderThresholdMP: 90}
	if err := ls.Save(ctx, "", bad); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("Save inverted: got %v, want ErrInvalid", err)
	}
	// Zero is not "no threshold": it would make a perfect decision an error.
	if err := ls.Save(ctx, "", storage.LibrarySettings{ErrorThresholdMP: 0, BlunderThresholdMP: 90}); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("Save zero: got %v, want ErrInvalid", err)
	}

	// A refused save leaves the stored pair alone.
	got, err := ls.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load after refusal: %v", err)
	}
	if got != ok {
		t.Fatalf("refused save changed the library: got %+v, want %+v", got, ok)
	}
}

// testBlunderCountPromisesTheSearch holds the one property the status bar's
// library counter exists for: the number it shows is the number of Positions
// the search behind its own link returns. A Position played two ways — well
// once, badly once — is a Blunder, because `E>x` scores a deduplicated
// Position by the largest of its recorded plays (#167) and the counter must
// not disagree with the list it opens.
func testBlunderCountPromisesTheSearch(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	matchID, _ := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	game := domain.Game{MatchID: matchID, GameNumber: 2, Winner: 1, PointsWon: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &game)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	// One position, two plays by player 1: the cheap one first, so the
	// denormalised column (which scores the FIRST of the analysis' played
	// moves) says 20 millipoints while the truth about the position is 300.
	pos := statsDecisionPos(t, 4)
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	for i, played := range []string{"13/11 24/23", "8/6 6/4"} {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(20 + i), MoveType: "checker",
			PositionID: posID, Player: 1, CheckerMove: played}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove %q: %v", played, err)
		}
	}
	cheap, dear := 0.020, 0.300
	analysis := domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		PlayedMoves:  []string{"13/11 24/23", "8/6 6/4"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "24/22 13/11", Equity: 0.500},
			{Move: "13/11 24/23", Equity: 0.480, EquityError: &cheap},
			{Move: "8/6 6/4", Equity: 0.200, EquityError: &dear},
		}},
	}
	if err := s.Analyses().Save(ctx, "", posID, &analysis); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	counts, err := s.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	// The fixture match's own two decisions cost 50 mp each — Errors at the
	// default threshold, not Blunders — so the multi-played position is the
	// only Blunder in the library.
	if counts.Blunders != 1 {
		t.Fatalf("Blunders = %d, want 1: the position player 1 played twice cost 0.300 the second time, "+
			"and the column that scores only the first play says 0.020", counts.Blunders)
	}

	// Raise the threshold past that cost and the counter follows the library,
	// because the threshold is the library's and not a constant.
	if err := s.LibrarySettings().Save(ctx, "", storage.LibrarySettings{
		ErrorThresholdMP: 50, BlunderThresholdMP: 500}); err != nil {
		t.Fatalf("Save settings: %v", err)
	}
	counts, err = s.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatalf("Counts after raising the threshold: %v", err)
	}
	if counts.Blunders != 0 {
		t.Fatalf("Blunders = %d at a 500 mp threshold, want 0", counts.Blunders)
	}
}
