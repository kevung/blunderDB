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
	// Absent or null means "no session limit"; 0 is a limit that serves no
	// card. A plain int would collapse the two (ADR-0026 rule 3).
	SessionLimit *int `json:"sessionLimit"`
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

// pageLimit implements pagedReq (handlers_rpc.go): rpc/rpcStream enforce
// maxPageSize on it before anki.reviewLog ever runs.
func (r reviewLogReq) pageLimit() int { return r.Limit }

type forecastReq struct {
	DeckID int64 `json:"deckId"`
	Days   int   `json:"days"`
}

// retentionReq carries no `apply`: the route reads (ADR-0026 rule 5). The old
// anki.optimizeParams, which wrote a tuned target back, is gone rather than
// deprecated — left in place, the verb invites someone to rebuild the
// write-back.
type retentionReq struct {
	DeckID int64 `json:"deckId"`
}

type cardIDReq struct {
	CardID int64 `json:"cardId"`
}

type suspendCardReq struct {
	CardID    int64 `json:"cardId"`
	Suspended bool  `json:"suspended"`
}

// linkedCardReq asks for the other half of a cube decision (#276).
type linkedCardReq struct {
	DeckID int64 `json:"deckId"`
	CardID int64 `json:"cardId"`
}

// reviewsByGameTypeReq asks how much was studied since an ISO date (#275).
type reviewsByGameTypeReq struct {
	Since string `json:"since"`
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
			return as().UpdateDeckParams(ctx, scope, req.ID, req.RequestRetention, req.MaximumInterval, req.EnableFuzz, req.SessionLimit)
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
		// Wrapped with withIdempotency (#236): ReviewCard applies a spaced-
		// repetition rating and advances scheduling state on every call —
		// unlike positions.save, calling it twice is not a no-op, it grades
		// the same review twice. A client retrying a dropped response with
		// the same Idempotency-Key gets the first attempt's result replayed
		// instead of a second, phantom review.
		{http.MethodPost, "/v1/anki.reviewCard", s.withIdempotency(rpc(func(ctx context.Context, scope string, req reviewCardReq) (*domain.AnkiReviewCard, error) {
			return as().ReviewCard(ctx, scope, req.CardID, req.Rating)
		}))},
		{http.MethodPost, "/v1/anki.reviewLog", rpcStream(func(ctx context.Context, scope string, req reviewLogReq) iterReviewLog {
			return as().ReviewLog(ctx, scope, req.DeckID, req.Limit)
		})},
		// Combien de POSITIONS ont été révisées depuis une date, par plan de
		// jeu (#275). Des positions et non des révisions : une carte revue
		// quatre fois est une position étudiée, et compter les répétitions
		// ferait passer un mois de bachotage pour un mois de couverture.
		// L'autre moitié d'une décision de videau (#276), quand elle est dans
		// le même paquet et due. Le lien est dérivé des données de match, pas
		// stocké : une colonne serait une seconde copie d'un fait qu'un
		// réimport peut changer.
		{http.MethodPost, "/v1/anki.linkedCard", rpc(func(ctx context.Context, scope string, req linkedCardReq) (*domain.AnkiReviewCard, error) {
			return as().LinkedCard(ctx, scope, req.DeckID, req.CardID)
		})},
		{http.MethodPost, "/v1/anki.reviewsByGameType", rpc(func(ctx context.Context, scope string, req reviewsByGameTypeReq) (map[string]int, error) {
			return as().ReviewsByGameType(ctx, scope, req.Since)
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
		{http.MethodPost, "/v1/anki.retention", rpc(func(ctx context.Context, scope string, req retentionReq) (*domain.AnkiRetention, error) {
			return as().Retention(ctx, scope, req.DeckID)
		})},
	}
}
