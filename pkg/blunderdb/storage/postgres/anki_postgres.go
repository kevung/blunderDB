package postgres

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type ankiStore struct{ db execer }

var _ storage.AnkiStore = (*ankiStore)(nil)

func (*ankiStore) CreateDeck(context.Context, string, string, string, string, int64, string) (int64, error) {
	return 0, notImpl("Anki", "CreateDeck")
}
func (*ankiStore) ListDecks(context.Context, string) iter.Seq2[*domain.AnkiDeck, error] {
	return errSeq2[domain.AnkiDeck](notImpl("Anki", "ListDecks"))
}
func (*ankiStore) UpdateDeck(context.Context, string, int64, string, string) error {
	return notImpl("Anki", "UpdateDeck")
}
func (*ankiStore) UpdateDeckParams(context.Context, string, int64, float64, float64, bool) error {
	return notImpl("Anki", "UpdateDeckParams")
}
func (*ankiStore) DeleteDeck(context.Context, string, int64) error {
	return notImpl("Anki", "DeleteDeck")
}
func (*ankiStore) ResetDeck(context.Context, string, int64) error {
	return notImpl("Anki", "ResetDeck")
}
func (*ankiStore) Sync(context.Context, string, int64) error {
	return notImpl("Anki", "Sync")
}
func (*ankiStore) SyncWithPositions(context.Context, string, int64, []int64) error {
	return notImpl("Anki", "SyncWithPositions")
}
func (*ankiStore) DeckPositions(context.Context, string, int64) iter.Seq2[*domain.Position, error] {
	return errSeq2[domain.Position](notImpl("Anki", "DeckPositions"))
}
func (*ankiStore) DeckStats(context.Context, string, int64) (*domain.AnkiDeckStats, error) {
	return nil, notImpl("Anki", "DeckStats")
}
func (*ankiStore) NextCard(context.Context, string, int64) (*domain.AnkiReviewCard, error) {
	return nil, notImpl("Anki", "NextCard")
}
func (*ankiStore) ReviewCard(context.Context, string, int64, int) (*domain.AnkiReviewCard, error) {
	return nil, notImpl("Anki", "ReviewCard")
}
