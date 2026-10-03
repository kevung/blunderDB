package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestSearchCorpusTokens runs the tokens that name the match a position was
// met in — players and seat, opponent, tournament, round, length, date, PR —
// and the provenance of its analysis, against a real SQLite file. Two matches:
//
//	A  Alice (seat 1) v Bob   (seat 2), 7 points, 2024-03-15, round 3, "Open Cup"
//	B  Carol (seat 1) v Alice (seat 2), 5 points, 2023-11-02, round Final
//
// Alice plays pos1 in A and pos4 in B; Bob plays pos2; Carol plays pos3.
func TestSearchCorpusTokens(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "corpus.db")
	s, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	mkMatch := func(p1, p2, round string, length int, date time.Time) (matchID, gameID int64) {
		m := domain.Match{Player1Name: p1, Player2Name: p2, Round: round, MatchLength: int32(length), MatchDate: date}
		id, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		g := domain.Game{MatchID: id, GameNumber: 1, Winner: 1, PointsWon: 1}
		gid, err := s.Matches().CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return id, gid
	}
	matchA, gameA := mkMatch("Alice", "Bob", "3", 7, time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC))
	matchB, gameB := mkMatch("Carol", "Alice", "Final", 5, time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC))

	tid, err := s.Tournaments().Create(ctx, "", "Open Cup", "", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tid, matchA); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}

	n := int32(0)
	play := func(gameID int64, player int32, die int, a *domain.PositionAnalysis) int64 {
		p := domain.InitializePosition()
		p.Dice = [2]int{die, 1}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		n++
		mv := domain.Move{GameID: gameID, MoveNumber: n, MoveType: "checker", PositionID: id, Player: player, CheckerMove: "13/11 24/23"}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		if a != nil {
			if err := s.Analyses().Save(ctx, "", id, a); err != nil {
				t.Fatalf("Save analysis: %v", err)
			}
		}
		return id
	}
	analysis := func(engine, depth string) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			PlayedMoves:  []string{"13/11 24/23"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "13/11 24/23", Equity: 0.1, AnalysisEngine: engine, AnalysisDepth: depth},
			}},
		}
	}
	pos1 := play(gameA, 1, 2, analysis("XG", "4-ply"))
	pos2 := play(gameA, -1, 3, analysis("GNUbg", "2-ply"))
	pos3 := play(gameB, 1, 4, analysis("XG", "XG Roller++"))
	pos4 := play(gameB, -1, 5, nil)

	// PR per match and seat; Alice's seat in B has none.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	for _, r := range []struct {
		match int64
		seat  int
		pr    any
	}{{matchA, 1, 4.0}, {matchA, 2, 9.0}, {matchB, 1, 6.0}, {matchB, 2, nil}} {
		if _, err := raw.Exec(`INSERT INTO match_stats (match_id, seat, pr) VALUES (?, ?, ?)`, r.match, r.seat, r.pr); err != nil {
			t.Fatalf("insert match_stats: %v", err)
		}
	}

	find := func(f domain.SearchFilters) []int64 {
		var ids []int64
		for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
			if err != nil {
				t.Fatalf("Find(%+v): %v", f, err)
			}
			ids = append(ids, pos.ID)
		}
		slices.Sort(ids)
		return ids
	}
	for _, c := range []struct {
		name string
		f    domain.SearchFilters
		want []int64
	}{
		{"pl either seat", domain.SearchFilters{PlayerFilter: `pl"alice"`}, []int64{pos1, pos2, pos3, pos4}},
		{"pl wildcard", domain.SearchFilters{PlayerFilter: `pl"CAR*"`}, []int64{pos3, pos4}},
		{"pl is literal outside the joker", domain.SearchFilters{PlayerFilter: `pl"Al_ce"`}, nil},
		{"pl bare name", domain.SearchFilters{PlayerFilter: "Bob"}, []int64{pos1, pos2}},
		{"pl! own decisions", domain.SearchFilters{PlayerFilter: `pl!"Alice"`}, []int64{pos1, pos4}},
		{"pl! second seat", domain.SearchFilters{PlayerFilter: `pl!"bob"`}, []int64{pos2}},
		{"op alone is either seat", domain.SearchFilters{OpponentFilter: `op"Bob"`}, []int64{pos1, pos2}},
		{"pl and op", domain.SearchFilters{PlayerFilter: `pl"Alice"`, OpponentFilter: `op"Carol"`}, []int64{pos3, pos4}},
		{"pl and op wrong pair", domain.SearchFilters{PlayerFilter: `pl"Bob"`, OpponentFilter: `op"Carol"`}, nil},
		{"pl! and op", domain.SearchFilters{PlayerFilter: `pl!"Alice"`, OpponentFilter: `op"Carol"`}, []int64{pos4}},
		{"wildcards on both", domain.SearchFilters{PlayerFilter: `pl"Al*"`, OpponentFilter: `op"b*"`}, []int64{pos1, pos2}},
		{"tournament by name", domain.SearchFilters{TournamentNameFilter: `tn"open*"`}, []int64{pos1, pos2}},
		{"tournament unknown", domain.SearchFilters{TournamentNameFilter: `tn"Closed"`}, nil},
		{"round", domain.SearchFilters{RoundFilter: "3"}, []int64{pos1, pos2}},
		{"round list and joker", domain.SearchFilters{RoundFilter: "9;fin*"}, []int64{pos3, pos4}},
		{"match length", domain.SearchFilters{MatchLengthFilter: "ml:7"}, []int64{pos1, pos2}},
		{"match length range", domain.SearchFilters{MatchLengthFilter: "ml:5,9"}, []int64{pos1, pos2, pos3, pos4}},
		{"match length min", domain.SearchFilters{MatchLengthFilter: "ml>6"}, []int64{pos1, pos2}},
		{"match length max", domain.SearchFilters{MatchLengthFilter: "ml<6"}, []int64{pos3, pos4}},
		{"match date year", domain.SearchFilters{MatchDateFilter: "md:2024"}, []int64{pos1, pos2}},
		{"match date span covers last month", domain.SearchFilters{MatchDateFilter: "md:2023-01..2023-11"}, []int64{pos3, pos4}},
		{"match date day", domain.SearchFilters{MatchDateFilter: "md:2024-03-15"}, []int64{pos1, pos2}},
		{"match date before", domain.SearchFilters{MatchDateFilter: "md<2023-12"}, []int64{pos3, pos4}},
		{"match date after", domain.SearchFilters{MatchDateFilter: "md>2024-04"}, nil},
		{"match date unreadable", domain.SearchFilters{MatchDateFilter: "md:2024-13"}, nil},
		{"pr high, seat of the decider", domain.SearchFilters{PlayerPRFilter: "pr>8"}, []int64{pos2}},
		{"pr low", domain.SearchFilters{PlayerPRFilter: "pr<5"}, []int64{pos1}},
		{"pr range, no PR excluded", domain.SearchFilters{PlayerPRFilter: "pr4,6"}, []int64{pos1, pos3}},
		{"pr with seat", domain.SearchFilters{PlayerPRFilter: "pr<5", PlayerFilter: `pl!"Bob"`}, nil},
		{"engine", domain.SearchFilters{AnalysisProvenanceFilter: "xg"}, []int64{pos1, pos3}},
		{"engine and depth", domain.SearchFilters{AnalysisProvenanceFilter: "xg;4ply"}, []int64{pos1}},
		{"engines are alternatives", domain.SearchFilters{AnalysisProvenanceFilter: "xg;gnubg"}, []int64{pos1, pos2, pos3}},
		{"depth at least", domain.SearchFilters{AnalysisProvenanceFilter: "3ply+"}, []int64{pos1, pos3}},
		{"rollout depth", domain.SearchFilters{AnalysisProvenanceFilter: "rollout"}, []int64{pos3}},
		{"engine and depth must both hold", domain.SearchFilters{AnalysisProvenanceFilter: "gnubg;4ply"}, nil},
		{"combined with a board-free filter", domain.SearchFilters{PlayerFilter: `pl!"Alice"`, MatchLengthFilter: "ml:5"}, []int64{pos4}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := find(c.f); !reflect.DeepEqual(got, c.want) && !(len(got) == 0 && len(c.want) == 0) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
