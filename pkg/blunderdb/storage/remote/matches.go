package remote

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func (s matchStore) Save(ctx context.Context, scope string, m *domain.Match) (int64, error) {
	var r idResp
	if err := s.c.call(ctx, scope, "matches.save", struct {
		Match *domain.Match `json:"match"`
	}{m}, &r); err != nil {
		return 0, err
	}
	m.ID = r.ID
	return r.ID, nil
}

func (s matchStore) FindByHash(ctx context.Context, scope string, hash, canonicalHash string) (int64, bool, error) {
	var r existsResp
	err := s.c.call(ctx, scope, "matches.findByHash", struct {
		Hash          string `json:"hash"`
		CanonicalHash string `json:"canonicalHash"`
	}{hash, canonicalHash}, &r)
	return r.ID, r.Found, err
}

func (s matchStore) MatchesByPosition(ctx context.Context, scope string, positionID int64) ([]int64, error) {
	var r struct {
		MatchIDs []int64 `json:"matchIds"`
	}
	err := s.c.call(ctx, scope, "matches.byPosition", positionIDReq{positionID}, &r)
	return r.MatchIDs, err
}

func (s matchStore) Get(ctx context.Context, scope string, id int64) (*domain.Match, error) {
	var m *domain.Match
	if err := s.c.call(ctx, scope, "matches.get", idReq{id}, &m); err != nil {
		return nil, err
	}
	return m, nil
}

type matchListReq struct {
	PlayerName         string   `json:"playerName,omitempty"`
	PlayerNameContains string   `json:"playerNameContains,omitempty"`
	Text               string   `json:"text,omitempty"`
	Unassigned         bool     `json:"unassigned,omitempty"`
	TournamentIDs      *[]int64 `json:"tournamentIds,omitempty"`
	IDs                *[]int64 `json:"ids,omitempty"`
	DateFrom           string   `json:"dateFrom,omitempty"`
	DateTo             string   `json:"dateTo,omitempty"`
	MatchLength        *[]int   `json:"matchLength,omitempty"`
	Sort               string   `json:"sort,omitempty"`
	Limit              int      `json:"limit,omitempty"`
	Offset             int      `json:"offset,omitempty"`
}

// matchListBody maps the options onto matches.list. A slice goes as a
// pointer so an empty one ("no id at all") is not confused with an absent one
// ("any id"). PlayerSpellings has no field there, and dropping it would widen
// the answer silently.
func matchListBody(method string, o storage.MatchListOpts) (matchListReq, error) {
	if len(o.PlayerSpellings) > 0 {
		return matchListReq{}, unsupported(method + " with PlayerSpellings")
	}
	return matchListReq{
		PlayerName: o.PlayerName, PlayerNameContains: o.PlayerNameContains, Text: o.Text,
		Unassigned: o.Unassigned, TournamentIDs: present(o.TournamentIDs), IDs: present(o.IDs),
		DateFrom: o.DateFrom, DateTo: o.DateTo, MatchLength: present(o.MatchLength),
		Sort: o.Sort, Limit: o.Limit, Offset: o.Offset,
	}, nil
}

func (s matchStore) List(ctx context.Context, scope string, opts storage.MatchListOpts) iter.Seq2[*domain.Match, error] {
	body, err := matchListBody("MatchStore.List", opts)
	if err != nil {
		return func(yield func(*domain.Match, error) bool) { yield(nil, err) }
	}
	return stream[*domain.Match](ctx, s.c, scope, "matches.list", body)
}

func (s matchStore) Count(ctx context.Context, scope string, opts storage.MatchListOpts) (int, error) {
	body, err := matchListBody("MatchStore.Count", opts)
	if err != nil {
		return 0, err
	}
	var r struct {
		Count int `json:"count"`
	}
	err = s.c.call(ctx, scope, "matches.count", body, &r)
	return r.Count, err
}

func (s matchStore) Update(ctx context.Context, scope string, id int64, player1Name, player2Name, matchDate string) error {
	return s.c.call(ctx, scope, "matches.update", struct {
		ID          int64  `json:"id"`
		Player1Name string `json:"player1Name"`
		Player2Name string `json:"player2Name"`
		MatchDate   string `json:"matchDate"`
	}{id, player1Name, player2Name, matchDate}, nil)
}

func (s matchStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	return s.c.call(ctx, scope, "matches.updateComment", struct {
		ID      int64  `json:"id"`
		Comment string `json:"comment"`
	}{id, comment}, nil)
}

func (s matchStore) SetVideoSource(ctx context.Context, scope string, id int64, source string) error {
	return s.c.call(ctx, scope, "matches.setVideoSource", struct {
		ID     int64  `json:"id"`
		Source string `json:"source"`
	}{id, source}, nil)
}

func (s matchStore) DeleteCascade(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "matches.delete", idReq{id}, nil)
}

func (s matchStore) SwapPlayers(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "matches.swapPlayers", idReq{id}, nil)
}

func (s matchStore) MergePlayers(ctx context.Context, scope string, names []string, canonical string) error {
	return s.c.call(ctx, scope, "matches.mergePlayers", struct {
		Names     []string `json:"names"`
		Canonical string   `json:"canonical"`
	}{names, canonical}, nil)
}

func (s matchStore) SetLastVisitedPosition(ctx context.Context, scope string, id int64, positionIndex int) error {
	return s.c.call(ctx, scope, "matches.setLastVisitedPosition", struct {
		ID            int64 `json:"id"`
		PositionIndex int   `json:"positionIndex"`
	}{id, positionIndex}, nil)
}

func (s matchStore) LastVisited(ctx context.Context, scope string) (*domain.Match, error) {
	var m *domain.Match
	if err := s.c.call(ctx, scope, "matches.lastVisited", nil, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s matchStore) CreateGame(ctx context.Context, scope string, g *domain.Game) (int64, error) {
	var r idResp
	if err := s.c.call(ctx, scope, "matches.createGame", struct {
		Game *domain.Game `json:"game"`
	}{g}, &r); err != nil {
		return 0, err
	}
	g.ID = r.ID
	return r.ID, nil
}

func (s matchStore) Games(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Game, error] {
	return stream[*domain.Game](ctx, s.c, scope, "matches.games", matchIDReq{matchID})
}

func (s matchStore) CreateMove(ctx context.Context, scope string, mv *domain.Move) (int64, error) {
	var r idResp
	if err := s.c.call(ctx, scope, "matches.createMove", struct {
		Move *domain.Move `json:"move"`
	}{mv}, &r); err != nil {
		return 0, err
	}
	mv.ID = r.ID
	return r.ID, nil
}

// CreateMoves has no batch route: it sends matches.createMove once per move,
// so a failure midway leaves the moves before it written.
func (s matchStore) CreateMoves(ctx context.Context, scope string, mvs []*domain.Move) error {
	for _, mv := range mvs {
		if _, err := s.CreateMove(ctx, scope, mv); err != nil {
			return err
		}
	}
	return nil
}

func (s matchStore) Moves(ctx context.Context, scope string, gameID int64) iter.Seq2[*domain.Move, error] {
	return stream[*domain.Move](ctx, s.c, scope, "matches.moves", struct {
		GameID int64 `json:"gameId"`
	}{gameID})
}

// RecordedMoves reads matches.moves, the only /v1 route for a game's moves,
// and drops the cost the daemon computed: same answer as the local
// backends, without their saving.
func (s matchStore) RecordedMoves(ctx context.Context, scope string, gameID int64) iter.Seq2[*domain.Move, error] {
	return func(yield func(*domain.Move, error) bool) {
		for mv, err := range s.Moves(ctx, scope, gameID) {
			if mv != nil {
				mv.ErrorMP = nil
			}
			if !yield(mv, err) {
				return
			}
		}
	}
}

// LibraryMoves has no /v1 route of its own: it walks List, then each match's
// games and their RecordedMoves, the order the local backends read in one
// join. Each level is collected before the next is read, so no two streams
// of the client are open at once.
func (s matchStore) LibraryMoves(ctx context.Context, scope string) iter.Seq2[*domain.LibraryMove, error] {
	return func(yield func(*domain.LibraryMove, error) bool) {
		var matchIDs []int64
		for m, err := range s.List(ctx, scope, storage.MatchListOpts{}) {
			if err != nil {
				yield(nil, err)
				return
			}
			matchIDs = append(matchIDs, m.ID)
		}
		for _, matchID := range matchIDs {
			var games []*domain.Game
			for g, err := range s.Games(ctx, scope, matchID) {
				if err != nil {
					yield(nil, err)
					return
				}
				games = append(games, g)
			}
			for _, g := range games {
				var moves []*domain.Move
				for mv, err := range s.RecordedMoves(ctx, scope, g.ID) {
					if err != nil {
						yield(nil, err)
						return
					}
					moves = append(moves, mv)
				}
				for _, mv := range moves {
					if !yield(&domain.LibraryMove{Move: *mv, MatchID: matchID, GameNumber: g.GameNumber}, nil) {
						return
					}
				}
			}
		}
	}
}

func (s matchStore) MovesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Move, error] {
	return stream[*domain.Move](ctx, s.c, scope, "matches.movesByMatch", matchIDReq{matchID})
}

func (s matchStore) MovePositions(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MatchMovePosition, error] {
	return stream[*domain.MatchMovePosition](ctx, s.c, scope, "matches.movePositions", matchIDReq{matchID})
}

// present is nil for a nil slice and a pointer to it otherwise.
func present[T any](s []T) *[]T {
	if s == nil {
		return nil
	}
	return &s
}
