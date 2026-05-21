package sqlite

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type matchStore struct{ db execer }

var _ storage.MatchStore = (*matchStore)(nil)

func (*matchStore) Save(context.Context, string, *domain.Match) (int64, error) {
	return 0, notImpl("Match", "Save")
}
func (*matchStore) Get(context.Context, string, int64) (*domain.Match, error) {
	return nil, notImpl("Match", "Get")
}
func (*matchStore) List(context.Context, string) iter.Seq2[*domain.Match, error] {
	return errSeq2[domain.Match](notImpl("Match", "List"))
}
func (*matchStore) Update(context.Context, string, int64, string, string, string) error {
	return notImpl("Match", "Update")
}
func (*matchStore) UpdateComment(context.Context, string, int64, string) error {
	return notImpl("Match", "UpdateComment")
}
func (*matchStore) DeleteCascade(context.Context, storage.Tx, string, int64) error {
	return notImpl("Match", "DeleteCascade")
}
func (*matchStore) SwapPlayers(context.Context, string, int64) error {
	return notImpl("Match", "SwapPlayers")
}
func (*matchStore) MergePlayers(context.Context, string, []string, string) error {
	return notImpl("Match", "MergePlayers")
}
func (*matchStore) SetLastVisitedPosition(context.Context, string, int64, int) error {
	return notImpl("Match", "SetLastVisitedPosition")
}
func (*matchStore) LastVisited(context.Context, string) (*domain.Match, error) {
	return nil, notImpl("Match", "LastVisited")
}
func (*matchStore) CreateGame(context.Context, string, *domain.Game) (int64, error) {
	return 0, notImpl("Match", "CreateGame")
}
func (*matchStore) Games(context.Context, string, int64) iter.Seq2[*domain.Game, error] {
	return errSeq2[domain.Game](notImpl("Match", "Games"))
}
func (*matchStore) CreateMove(context.Context, string, *domain.Move) (int64, error) {
	return 0, notImpl("Match", "CreateMove")
}
func (*matchStore) Moves(context.Context, string, int64) iter.Seq2[*domain.Move, error] {
	return errSeq2[domain.Move](notImpl("Match", "Moves"))
}
func (*matchStore) MovePositions(context.Context, string, int64) iter.Seq2[*domain.MatchMovePosition, error] {
	return errSeq2[domain.MatchMovePosition](notImpl("Match", "MovePositions"))
}
