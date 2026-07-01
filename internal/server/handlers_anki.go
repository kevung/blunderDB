package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type deckCreateReq struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	SourceType    string `json:"sourceType"`
	SourceID      int64  `json:"sourceId"`
	SourceCommand string `json:"sourceCommand"`
}

type deckUpdateReq struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type deckParamsReq struct {
	ID               int64   `json:"id"`
	RequestRetention float64 `json:"requestRetention"`
	MaximumInterval  float64 `json:"maximumInterval"`
	EnableFuzz       bool    `json:"enableFuzz"`
}

type deckIDReq struct {
	DeckID int64 `json:"deckId"`
}

type deckSyncPositionsReq struct {
	DeckID      int64   `json:"deckId"`
	PositionIDs []int64 `json:"positionIds"`
}

type reviewCardReq struct {
	CardID int64 `json:"cardId"`
	Rating int   `json:"rating"`
}

type reviewLogReq struct {
	DeckID int64 `json:"deckId"`
	Limit  int   `json:"limit"`
}

type forecastReq struct {
	DeckID int64 `json:"deckId"`
	Days   int   `json:"days"`
}

type optimizeReq struct {
	DeckID int64 `json:"deckId"`
	Apply  bool  `json:"apply"`
}

type cardIDReq struct {
	CardID int64 `json:"cardId"`
}

type suspendCardReq struct {
	CardID    int64 `json:"cardId"`
	Suspended bool  `json:"suspended"`
}

func (s *Server) ankiRoutes() []route {
	as := func() storage.AnkiStore { return s.opts.Storage.Anki() }
	return []route{
		{http.MethodPost, "/v1/anki.createDeck", rpc(func(ctx context.Context, scope string, req deckCreateReq) (idResp, error) {
			id, err := as().CreateDeck(ctx, scope, req.Name, req.Description, req.SourceType, req.SourceID, req.SourceCommand)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/anki.listDecks", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterDecks {
			return as().ListDecks(ctx, scope)
		})},
		{http.MethodPost, "/v1/anki.updateDeck", rpcVoid(func(ctx context.Context, scope string, req deckUpdateReq) error {
			return as().UpdateDeck(ctx, scope, req.ID, req.Name, req.Description)
		})},
		{http.MethodPost, "/v1/anki.updateDeckParams", rpcVoid(func(ctx context.Context, scope string, req deckParamsReq) error {
			return as().UpdateDeckParams(ctx, scope, req.ID, req.RequestRetention, req.MaximumInterval, req.EnableFuzz)
		})},
		{http.MethodPost, "/v1/anki.deleteDeck", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return as().DeleteDeck(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/anki.resetDeck", rpcVoid(func(ctx context.Context, scope string, req deckIDReq) error {
			return as().ResetDeck(ctx, scope, req.DeckID)
		})},
		{http.MethodPost, "/v1/anki.sync", rpcVoid(func(ctx context.Context, scope string, req deckIDReq) error {
			return as().Sync(ctx, scope, req.DeckID)
		})},
		{http.MethodPost, "/v1/anki.syncWithPositions", rpcVoid(func(ctx context.Context, scope string, req deckSyncPositionsReq) error {
			return as().SyncWithPositions(ctx, scope, req.DeckID, req.PositionIDs)
		})},
		{http.MethodPost, "/v1/anki.deckPositions", rpcStream(func(ctx context.Context, scope string, req deckIDReq) iterPositions {
			return as().DeckPositions(ctx, scope, req.DeckID)
		})},
		{http.MethodPost, "/v1/anki.deckStats", rpc(func(ctx context.Context, scope string, req deckIDReq) (*domain.AnkiDeckStats, error) {
			return as().DeckStats(ctx, scope, req.DeckID)
		})},
		{http.MethodPost, "/v1/anki.nextCard", rpc(func(ctx context.Context, scope string, req deckIDReq) (*domain.AnkiReviewCard, error) {
			return as().NextCard(ctx, scope, req.DeckID)
		})},
		{http.MethodPost, "/v1/anki.reviewCard", rpc(func(ctx context.Context, scope string, req reviewCardReq) (*domain.AnkiReviewCard, error) {
			return as().ReviewCard(ctx, scope, req.CardID, req.Rating)
		})},
		{http.MethodPost, "/v1/anki.reviewLog", rpcStream(func(ctx context.Context, scope string, req reviewLogReq) iterReviewLog {
			return as().ReviewLog(ctx, scope, req.DeckID, req.Limit)
		})},
		{http.MethodPost, "/v1/anki.forecast", rpc(func(ctx context.Context, scope string, req forecastReq) ([]domain.AnkiForecastDay, error) {
			return as().Forecast(ctx, scope, req.DeckID, req.Days)
		})},
		{http.MethodPost, "/v1/anki.suspendCard", rpcVoid(func(ctx context.Context, scope string, req suspendCardReq) error {
			return as().SetCardSuspended(ctx, scope, req.CardID, req.Suspended)
		})},
		{http.MethodPost, "/v1/anki.buryCard", rpcVoid(func(ctx context.Context, scope string, req cardIDReq) error {
			return as().BuryCard(ctx, scope, req.CardID)
		})},
		{http.MethodPost, "/v1/anki.removeCard", rpcVoid(func(ctx context.Context, scope string, req cardIDReq) error {
			return as().RemoveCard(ctx, scope, req.CardID)
		})},
		{http.MethodPost, "/v1/anki.optimizeParams", rpc(func(ctx context.Context, scope string, req optimizeReq) (*domain.AnkiOptimizeResult, error) {
			return as().OptimizeParams(ctx, scope, req.DeckID, req.Apply)
		})},
	}
}
