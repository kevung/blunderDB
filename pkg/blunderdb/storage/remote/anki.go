package remote

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type deckIDReq struct {
	DeckID int64 `json:"deckId"`
}

type cardIDReq struct {
	CardID int64 `json:"cardId"`
}

type deckFilterReq struct {
	DeckID     int64  `json:"deckId"`
	Filter     string `json:"filter"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	PositionID int64  `json:"positionId,omitempty"`
}

// indexOf reads a route that answers a position's rank, or -1 when the
// position is not there.
func (c *client) indexOf(ctx context.Context, scope, method string, req any) (int, bool, error) {
	var i int
	if err := c.call(ctx, scope, method, req, &i); err != nil {
		return 0, false, err
	}
	if i < 0 {
		return 0, false, nil
	}
	return i, true, nil
}

func (s ankiStore) CreateDeck(ctx context.Context, scope string, name, description, sourceType string, sourceID int64, sourceCommand string) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "anki.createDeck", struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		SourceType    string `json:"sourceType"`
		SourceID      int64  `json:"sourceId"`
		SourceCommand string `json:"sourceCommand"`
	}{name, description, sourceType, sourceID, sourceCommand}, &r)
	return r.ID, err
}

func (s ankiStore) ListDecks(ctx context.Context, scope string) iter.Seq2[*domain.AnkiDeck, error] {
	return stream[*domain.AnkiDeck](ctx, s.c, scope, "anki.listDecks", nil)
}

func (s ankiStore) UpdateDeck(ctx context.Context, scope string, id int64, name, description string) error {
	return s.c.call(ctx, scope, "anki.updateDeck", nameDescReq{id, name, description}, nil)
}

func (s ankiStore) UpdateDeckParams(ctx context.Context, scope string, id int64, requestRetention, maximumInterval float64, enableFuzz bool, sessionLimit *int) error {
	return s.c.call(ctx, scope, "anki.updateDeckParams", struct {
		ID               int64   `json:"id"`
		RequestRetention float64 `json:"requestRetention"`
		MaximumInterval  float64 `json:"maximumInterval"`
		EnableFuzz       bool    `json:"enableFuzz"`
		SessionLimit     *int    `json:"sessionLimit"`
	}{id, requestRetention, maximumInterval, enableFuzz, sessionLimit}, nil)
}

func (s ankiStore) DeleteDeck(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "anki.deleteDeck", idReq{id}, nil)
}

func (s ankiStore) ResetDeck(ctx context.Context, scope string, deckID int64) error {
	return s.c.call(ctx, scope, "anki.resetDeck", deckIDReq{deckID}, nil)
}

func (s ankiStore) SyncWithPositions(ctx context.Context, scope string, deckID int64, positionIDs []int64) error {
	return s.c.call(ctx, scope, "anki.syncWithPositions", struct {
		DeckID      int64   `json:"deckId"`
		PositionIDs []int64 `json:"positionIds"`
	}{deckID, positionIDs}, nil)
}

func (s ankiStore) DeckPositions(ctx context.Context, scope string, deckID int64) iter.Seq2[*domain.Position, error] {
	return stream[*domain.Position](ctx, s.c, scope, "anki.deckPositions", deckIDReq{deckID})
}

func (s ankiStore) DeckPositionIDs(ctx context.Context, scope string, deckID int64, opts storage.ListOpts) ([]int64, error) {
	var ids []int64
	err := s.c.call(ctx, scope, "anki.deckPositionIds", struct {
		DeckID int64 `json:"deckId"`
		Limit  int   `json:"limit"`
		Offset int   `json:"offset"`
	}{deckID, opts.Limit, opts.Offset}, &ids)
	return ids, err
}

func (s ankiStore) DeckPositionCount(ctx context.Context, scope string, deckID int64) (int, error) {
	var n int
	err := s.c.call(ctx, scope, "anki.deckPositionCount", deckIDReq{deckID}, &n)
	return n, err
}

func (s ankiStore) IndexOfDeckPosition(ctx context.Context, scope string, deckID, positionID int64) (int, bool, error) {
	return s.c.indexOf(ctx, scope, "anki.indexOfDeckPosition", struct {
		DeckID     int64 `json:"deckId"`
		PositionID int64 `json:"positionId"`
	}{deckID, positionID})
}

func (s ankiStore) FilteredDeckPositionIDs(ctx context.Context, scope string, deckID int64, filter domain.AnkiCardFilter, opts storage.ListOpts) ([]int64, error) {
	var ids []int64
	err := s.c.call(ctx, scope, "anki.filteredPositionIds", deckFilterReq{DeckID: deckID, Filter: string(filter), Limit: opts.Limit, Offset: opts.Offset}, &ids)
	return ids, err
}

func (s ankiStore) FilteredDeckPositionCount(ctx context.Context, scope string, deckID int64, filter domain.AnkiCardFilter) (int, error) {
	var n int
	err := s.c.call(ctx, scope, "anki.filteredPositionCount", deckFilterReq{DeckID: deckID, Filter: string(filter)}, &n)
	return n, err
}

func (s ankiStore) IndexOfFilteredDeckPosition(ctx context.Context, scope string, deckID int64, filter domain.AnkiCardFilter, positionID int64) (int, bool, error) {
	return s.c.indexOf(ctx, scope, "anki.indexOfFilteredPosition", deckFilterReq{DeckID: deckID, Filter: string(filter), PositionID: positionID})
}

func (s ankiStore) DeckStats(ctx context.Context, scope string, deckID int64) (*domain.AnkiDeckStats, error) {
	var r *domain.AnkiDeckStats
	if err := s.c.call(ctx, scope, "anki.deckStats", deckIDReq{deckID}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s ankiStore) Forecast(ctx context.Context, scope string, deckID int64, days int) ([]domain.AnkiForecastDay, error) {
	var r []domain.AnkiForecastDay
	err := s.c.call(ctx, scope, "anki.forecast", struct {
		DeckID int64 `json:"deckId"`
		Days   int   `json:"days"`
	}{deckID, days}, &r)
	return r, err
}

func (s ankiStore) ReviewsByGameType(ctx context.Context, scope string, since string) (map[string]int, error) {
	var r map[string]int
	err := s.c.call(ctx, scope, "anki.reviewsByGameType", struct {
		Since string `json:"since"`
	}{since}, &r)
	return r, err
}

// card reads a route answering one card, or null for none.
func (c *client) card(ctx context.Context, scope, method string, req any) (*domain.AnkiReviewCard, error) {
	var r *domain.AnkiReviewCard
	if err := c.call(ctx, scope, method, req, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s ankiStore) LinkedCard(ctx context.Context, scope string, deckID, cardID int64) (*domain.AnkiReviewCard, error) {
	return s.c.card(ctx, scope, "anki.linkedCard", struct {
		DeckID int64 `json:"deckId"`
		CardID int64 `json:"cardId"`
	}{deckID, cardID})
}

func (s ankiStore) NextCard(ctx context.Context, scope string, deckID int64) (*domain.AnkiReviewCard, error) {
	return s.c.card(ctx, scope, "anki.nextCard", deckIDReq{deckID})
}

func (s ankiStore) ReviewCard(ctx context.Context, scope string, cardID int64, rating int) (*domain.AnkiReviewCard, error) {
	return s.c.card(ctx, scope, "anki.reviewCard", struct {
		CardID int64 `json:"cardId"`
		Rating int   `json:"rating"`
	}{cardID, rating})
}

func (s ankiStore) UndoReview(ctx context.Context, scope string, cardID int64) (*domain.AnkiReviewCard, error) {
	return s.c.card(ctx, scope, "anki.undoReview", cardIDReq{cardID})
}

func (s ankiStore) SetCardSuspended(ctx context.Context, scope string, cardID int64, suspended bool) error {
	return s.c.call(ctx, scope, "anki.suspendCard", struct {
		CardID    int64 `json:"cardId"`
		Suspended bool  `json:"suspended"`
	}{cardID, suspended}, nil)
}

func (s ankiStore) BuryCard(ctx context.Context, scope string, cardID int64) error {
	return s.c.call(ctx, scope, "anki.buryCard", cardIDReq{cardID}, nil)
}

func (s ankiStore) RemoveCard(ctx context.Context, scope string, cardID int64) error {
	return s.c.call(ctx, scope, "anki.removeCard", cardIDReq{cardID}, nil)
}

func (s ankiStore) ReviewLog(ctx context.Context, scope string, deckID int64, limit int) iter.Seq2[*domain.AnkiReviewLog, error] {
	return stream[*domain.AnkiReviewLog](ctx, s.c, scope, "anki.reviewLog", struct {
		DeckID int64 `json:"deckId"`
		Limit  int   `json:"limit"`
	}{deckID, limit})
}

func (s ankiStore) Retention(ctx context.Context, scope string, deckID int64) (*domain.AnkiRetention, error) {
	var r *domain.AnkiRetention
	if err := s.c.call(ctx, scope, "anki.retention", deckIDReq{deckID}, &r); err != nil {
		return nil, err
	}
	return r, nil
}
