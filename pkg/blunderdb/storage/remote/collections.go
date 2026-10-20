package remote

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type collPositionReq struct {
	CollectionID int64 `json:"collectionId"`
	PositionID   int64 `json:"positionId"`
}

type collPositionsReq struct {
	CollectionID int64   `json:"collectionId"`
	PositionIDs  []int64 `json:"positionIds"`
}

type collectionPositionsReq struct {
	CollectionID int64 `json:"collectionId"`
	Limit        int   `json:"limit"`
	Offset       int   `json:"offset"`
}

type nameDescReq struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s collectionStore) Create(ctx context.Context, scope string, name, description string) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "collections.create", nameDescReq{Name: name, Description: description}, &r)
	return r.ID, err
}

func (s collectionStore) Get(ctx context.Context, scope string, id int64) (*storage.Collection, error) {
	var col *storage.Collection
	if err := s.c.call(ctx, scope, "collections.get", idReq{id}, &col); err != nil {
		return nil, err
	}
	return col, nil
}

func (s collectionStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Collection, error] {
	return stream[*storage.Collection](ctx, s.c, scope, "collections.list", nil)
}

func (s collectionStore) Update(ctx context.Context, scope string, id int64, name, description string) error {
	return s.c.call(ctx, scope, "collections.update", nameDescReq{id, name, description}, nil)
}

func (s collectionStore) SetFilterQuery(ctx context.Context, scope string, id int64, query string) error {
	return s.c.call(ctx, scope, "collections.setFilter", struct {
		ID    int64  `json:"id"`
		Query string `json:"query"`
	}{id, query}, nil)
}

func (s collectionStore) Delete(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "collections.delete", idReq{id}, nil)
}

func (s collectionStore) Reorder(ctx context.Context, scope string, collectionIDs []int64) error {
	return s.c.call(ctx, scope, "collections.reorder", struct {
		CollectionIDs []int64 `json:"collectionIds"`
	}{collectionIDs}, nil)
}

func (s collectionStore) AddPosition(ctx context.Context, scope string, collectionID, positionID int64) error {
	return s.c.call(ctx, scope, "collections.addPosition", collPositionReq{collectionID, positionID}, nil)
}

func (s collectionStore) AddPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	return s.c.call(ctx, scope, "collections.addPositions", collPositionsReq{collectionID, positionIDs}, nil)
}

func (s collectionStore) RemovePosition(ctx context.Context, scope string, collectionID, positionID int64) error {
	return s.c.call(ctx, scope, "collections.removePosition", collPositionReq{collectionID, positionID}, nil)
}

func (s collectionStore) RemovePositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	return s.c.call(ctx, scope, "collections.removePositions", collPositionsReq{collectionID, positionIDs}, nil)
}

func (s collectionStore) ReorderPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	return s.c.call(ctx, scope, "collections.reorderPositions", collPositionsReq{collectionID, positionIDs}, nil)
}

func (s collectionStore) MovePosition(ctx context.Context, scope string, fromCollectionID, toCollectionID, positionID int64) error {
	return s.c.call(ctx, scope, "collections.movePosition", struct {
		FromCollectionID int64 `json:"fromCollectionId"`
		ToCollectionID   int64 `json:"toCollectionId"`
		PositionID       int64 `json:"positionId"`
	}{fromCollectionID, toCollectionID, positionID}, nil)
}

func (s collectionStore) CopyPosition(ctx context.Context, scope string, toCollectionID, positionID int64) error {
	return s.c.call(ctx, scope, "collections.copyPosition", struct {
		ToCollectionID int64 `json:"toCollectionId"`
		PositionID     int64 `json:"positionId"`
	}{toCollectionID, positionID}, nil)
}

func (s collectionStore) Positions(ctx context.Context, scope string, collectionID int64, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return stream[*domain.Position](ctx, s.c, scope, "collections.positions", collectionPositionsReq{collectionID, opts.Limit, opts.Offset})
}

func (s collectionStore) PositionIDs(ctx context.Context, scope string, collectionID int64, opts storage.ListOpts) ([]int64, error) {
	var ids []int64
	err := s.c.call(ctx, scope, "collections.positionIds", collectionPositionsReq{collectionID, opts.Limit, opts.Offset}, &ids)
	return ids, err
}

func (s collectionStore) CountPositions(ctx context.Context, scope string, collectionID int64) (int, error) {
	var n int
	err := s.c.call(ctx, scope, "collections.countPositions", collectionPositionsReq{CollectionID: collectionID}, &n)
	return n, err
}

// IndexOfPosition reads collections.indexOfPosition, which answers -1 for a
// position the collection does not hold.
func (s collectionStore) IndexOfPosition(ctx context.Context, scope string, collectionID, positionID int64) (int, bool, error) {
	var i int
	if err := s.c.call(ctx, scope, "collections.indexOfPosition", collPositionReq{collectionID, positionID}, &i); err != nil {
		return 0, false, err
	}
	if i < 0 {
		return 0, false, nil
	}
	return i, true, nil
}

func (s collectionStore) CollectionsOf(ctx context.Context, scope string, positionID int64) iter.Seq2[*storage.Collection, error] {
	return stream[*storage.Collection](ctx, s.c, scope, "collections.collectionsOf", positionIDReq{positionID})
}

func (s collectionStore) PositionIndexMap(ctx context.Context, scope string) (map[int64]int, error) {
	var m map[int64]int
	err := s.c.call(ctx, scope, "collections.positionIndexMap", nil, &m)
	return m, err
}

func (s tournamentStore) Create(ctx context.Context, scope string, name, date, location string) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "tournaments.create", tournamentReq{Name: name, Date: date, Location: location}, &r)
	return r.ID, err
}

type tournamentReq struct {
	ID       int64  `json:"id,omitempty"`
	Name     string `json:"name"`
	Date     string `json:"date"`
	Location string `json:"location"`
}

func (s tournamentStore) Get(ctx context.Context, scope string, id int64) (*domain.Tournament, error) {
	var t *domain.Tournament
	if err := s.c.call(ctx, scope, "tournaments.get", idReq{id}, &t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s tournamentStore) List(ctx context.Context, scope string, opts storage.ListOpts) iter.Seq2[*domain.Tournament, error] {
	return stream[*domain.Tournament](ctx, s.c, scope, "tournaments.list", listReq{opts.Limit, opts.Offset})
}

func (s tournamentStore) Update(ctx context.Context, scope string, id int64, name, date, location string) error {
	return s.c.call(ctx, scope, "tournaments.update", tournamentReq{id, name, date, location}, nil)
}

func (s tournamentStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	return s.c.call(ctx, scope, "tournaments.updateComment", struct {
		ID      int64  `json:"id"`
		Comment string `json:"comment"`
	}{id, comment}, nil)
}

func (s tournamentStore) Delete(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "tournaments.delete", idReq{id}, nil)
}

type matchIDReq struct {
	MatchID int64 `json:"matchId"`
}

func (s tournamentStore) AddMatch(ctx context.Context, scope string, tournamentID, matchID int64) error {
	return s.c.call(ctx, scope, "tournaments.addMatch", struct {
		TournamentID int64 `json:"tournamentId"`
		MatchID      int64 `json:"matchId"`
	}{tournamentID, matchID}, nil)
}

func (s tournamentStore) RemoveMatch(ctx context.Context, scope string, matchID int64) error {
	return s.c.call(ctx, scope, "tournaments.removeMatch", matchIDReq{matchID}, nil)
}

func (s tournamentStore) SetMatchByName(ctx context.Context, scope string, matchID int64, tournamentName string) error {
	return s.c.call(ctx, scope, "tournaments.setMatchByName", struct {
		MatchID        int64  `json:"matchId"`
		TournamentName string `json:"tournamentName"`
	}{matchID, tournamentName}, nil)
}

func (s tournamentStore) ReorderMatches(ctx context.Context, scope string, tournamentID int64, matchIDs []int64) error {
	return s.c.call(ctx, scope, "tournaments.reorderMatches", struct {
		TournamentID int64   `json:"tournamentId"`
		MatchIDs     []int64 `json:"matchIds"`
	}{tournamentID, matchIDs}, nil)
}

func (s tournamentStore) Matches(ctx context.Context, scope string, tournamentID int64) iter.Seq2[*domain.Match, error] {
	return stream[*domain.Match](ctx, s.c, scope, "tournaments.matches", struct {
		TournamentID int64 `json:"tournamentId"`
	}{tournamentID})
}

func (s tournamentStore) TournamentOf(ctx context.Context, scope string, matchID int64) (*domain.Tournament, error) {
	var t *domain.Tournament
	if err := s.c.call(ctx, scope, "tournaments.tournamentOf", matchIDReq{matchID}, &t); err != nil {
		return nil, err
	}
	return t, nil
}
