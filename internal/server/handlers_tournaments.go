package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type tournamentCreateReq struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	Location string `json:"location"`
}

type tournamentUpdateReq struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Date     string `json:"date"`
	Location string `json:"location"`
}

type tournamentCommentReq struct {
	ID      int64  `json:"id"`
	Comment string `json:"comment"`
}

type tournamentMatchReq struct {
	TournamentID int64 `json:"tournamentId"`
	MatchID      int64 `json:"matchId"`
}

type setMatchByNameReq struct {
	MatchID        int64  `json:"matchId"`
	TournamentName string `json:"tournamentName"`
}

type reorderMatchesReq struct {
	TournamentID int64   `json:"tournamentId"`
	MatchIDs     []int64 `json:"matchIds"`
}

type tournamentIDReq struct {
	TournamentID int64 `json:"tournamentId"`
}

func (s *Server) tournamentRoutes() []route {
	ts := func() storage.TournamentStore { return s.opts.Storage.Tournaments() }
	return []route{
		{http.MethodPost, "/v1/tournaments.create", rpc(func(ctx context.Context, scope string, req tournamentCreateReq) (idResp, error) {
			id, err := ts().Create(ctx, scope, req.Name, req.Date, req.Location)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/tournaments.get", rpc(func(ctx context.Context, scope string, req idReq) (*domain.Tournament, error) {
			return ts().Get(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/tournaments.list", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterTours {
			return ts().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/tournaments.update", rpcVoid(func(ctx context.Context, scope string, req tournamentUpdateReq) error {
			return ts().Update(ctx, scope, req.ID, req.Name, req.Date, req.Location)
		})},
		{http.MethodPost, "/v1/tournaments.updateComment", rpcVoid(func(ctx context.Context, scope string, req tournamentCommentReq) error {
			return ts().UpdateComment(ctx, scope, req.ID, req.Comment)
		})},
		{http.MethodPost, "/v1/tournaments.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ts().Delete(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/tournaments.addMatch", rpcVoid(func(ctx context.Context, scope string, req tournamentMatchReq) error {
			return ts().AddMatch(ctx, scope, req.TournamentID, req.MatchID)
		})},
		{http.MethodPost, "/v1/tournaments.removeMatch", rpcVoid(func(ctx context.Context, scope string, req matchIDReq) error {
			return ts().RemoveMatch(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/tournaments.setMatchByName", rpcVoid(func(ctx context.Context, scope string, req setMatchByNameReq) error {
			return ts().SetMatchByName(ctx, scope, req.MatchID, req.TournamentName)
		})},
		{http.MethodPost, "/v1/tournaments.reorderMatches", rpcVoid(func(ctx context.Context, scope string, req reorderMatchesReq) error {
			return ts().ReorderMatches(ctx, scope, req.TournamentID, req.MatchIDs)
		})},
		{http.MethodPost, "/v1/tournaments.matches", rpcStream(func(ctx context.Context, scope string, req tournamentIDReq) iterMatches {
			return ts().Matches(ctx, scope, req.TournamentID)
		})},
		{http.MethodPost, "/v1/tournaments.tournamentOf", rpc(func(ctx context.Context, scope string, req matchIDReq) (*domain.Tournament, error) {
			return ts().TournamentOf(ctx, scope, req.MatchID)
		})},
	}
}
