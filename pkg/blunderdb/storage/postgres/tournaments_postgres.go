package postgres

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type tournamentStore struct{ db execer }

var _ storage.TournamentStore = (*tournamentStore)(nil)

func (*tournamentStore) Create(context.Context, string, string, string, string) (int64, error) {
	return 0, notImpl("Tournament", "Create")
}
func (*tournamentStore) Get(context.Context, string, int64) (*domain.Tournament, error) {
	return nil, notImpl("Tournament", "Get")
}
func (*tournamentStore) List(context.Context, string) iter.Seq2[*domain.Tournament, error] {
	return errSeq2[domain.Tournament](notImpl("Tournament", "List"))
}
func (*tournamentStore) Update(context.Context, string, int64, string, string, string) error {
	return notImpl("Tournament", "Update")
}
func (*tournamentStore) UpdateComment(context.Context, string, int64, string) error {
	return notImpl("Tournament", "UpdateComment")
}
func (*tournamentStore) Delete(context.Context, string, int64) error {
	return notImpl("Tournament", "Delete")
}
func (*tournamentStore) AddMatch(context.Context, string, int64, int64) error {
	return notImpl("Tournament", "AddMatch")
}
func (*tournamentStore) RemoveMatch(context.Context, string, int64) error {
	return notImpl("Tournament", "RemoveMatch")
}
func (*tournamentStore) SetMatchByName(context.Context, string, int64, string) error {
	return notImpl("Tournament", "SetMatchByName")
}
func (*tournamentStore) ReorderMatches(context.Context, string, int64, []int64) error {
	return notImpl("Tournament", "ReorderMatches")
}
func (*tournamentStore) Matches(context.Context, string, int64) iter.Seq2[*domain.Match, error] {
	return errSeq2[domain.Match](notImpl("Tournament", "Matches"))
}
func (*tournamentStore) TournamentOf(context.Context, string, int64) (*domain.Tournament, error) {
	return nil, notImpl("Tournament", "TournamentOf")
}
