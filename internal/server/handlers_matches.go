package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type matchSaveReq struct {
	Match *domain.Match `json:"match"`
}

type matchUpdateReq struct {
	ID          int64  `json:"id"`
	Player1Name string `json:"player1Name"`
	Player2Name string `json:"player2Name"`
	MatchDate   string `json:"matchDate"`
}

type matchCommentReq struct {
	ID      int64  `json:"id"`
	Comment string `json:"comment"`
}

type mergePlayersReq struct {
	Names     []string `json:"names"`
	Canonical string   `json:"canonical"`
}

type lastVisitedReq struct {
	ID            int64 `json:"id"`
	PositionIndex int   `json:"positionIndex"`
}

type gameReq struct {
	Game *domain.Game `json:"game"`
}

type moveReq struct {
	Move *domain.Move `json:"move"`
}

type matchIDReq struct {
	MatchID int64 `json:"matchId"`
}

type gameIDReq struct {
	GameID int64 `json:"gameId"`
}

func (s *Server) matchRoutes() []route {
	ms := func() storage.MatchStore { return s.opts.Storage.Matches() }
	return []route{
		{http.MethodPost, "/v1/matches.save", rpc(func(ctx context.Context, scope string, req matchSaveReq) (idResp, error) {
			id, err := ms().Save(ctx, scope, req.Match)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/matches.get", rpc(func(ctx context.Context, scope string, req idReq) (*domain.Match, error) {
			return ms().Get(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/matches.list", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterMatches {
			return ms().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/matches.update", rpcVoid(func(ctx context.Context, scope string, req matchUpdateReq) error {
			return ms().Update(ctx, scope, req.ID, req.Player1Name, req.Player2Name, req.MatchDate)
		})},
		{http.MethodPost, "/v1/matches.updateComment", rpcVoid(func(ctx context.Context, scope string, req matchCommentReq) error {
			return ms().UpdateComment(ctx, scope, req.ID, req.Comment)
		})},
		{http.MethodPost, "/v1/matches.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ms().DeleteCascade(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/matches.swapPlayers", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ms().SwapPlayers(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/matches.mergePlayers", rpcVoid(func(ctx context.Context, scope string, req mergePlayersReq) error {
			return ms().MergePlayers(ctx, scope, req.Names, req.Canonical)
		})},
		{http.MethodPost, "/v1/matches.setLastVisitedPosition", rpcVoid(func(ctx context.Context, scope string, req lastVisitedReq) error {
			return ms().SetLastVisitedPosition(ctx, scope, req.ID, req.PositionIndex)
		})},
		{http.MethodPost, "/v1/matches.lastVisited", rpc(func(ctx context.Context, scope string, _ struct{}) (*domain.Match, error) {
			return ms().LastVisited(ctx, scope)
		})},
		{http.MethodPost, "/v1/matches.createGame", rpc(func(ctx context.Context, scope string, req gameReq) (idResp, error) {
			id, err := ms().CreateGame(ctx, scope, req.Game)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/matches.games", rpcStream(func(ctx context.Context, scope string, req matchIDReq) iterGames {
			return ms().Games(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/matches.createMove", rpc(func(ctx context.Context, scope string, req moveReq) (idResp, error) {
			id, err := ms().CreateMove(ctx, scope, req.Move)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/matches.moves", rpcStream(func(ctx context.Context, scope string, req gameIDReq) iterMoves {
			return ms().Moves(ctx, scope, req.GameID)
		})},
		{http.MethodPost, "/v1/matches.movesByMatch", rpcStream(func(ctx context.Context, scope string, req matchIDReq) iterMoves {
			return ms().MovesByMatch(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/matches.movePositions", rpcStream(func(ctx context.Context, scope string, req matchIDReq) iterMovePos {
			return ms().MovePositions(ctx, scope, req.MatchID)
		})},
	}
}
