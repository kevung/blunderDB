package server

import (
	"context"
	"fmt"
	"net/http"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Direction and the Rencontre, read (ADR-0057). Every route here is a read of the
// Direction service — the one the desktop and the CLI call — bound to the caller's tenant, and
// every one is conditional (conditional.go). The gestures are not served here: they write,
// and the daemon opens them only on request (rule 5).
//
// Left out on purpose: the routes that write a file on the server's disk (WriteDirectionPage,
// WriteRencontrePage, the output folders) — a remote client reads pageHtml instead, and the
// outputDir fields are emptied, a path of the server's disk being no client's business — and
// SetDirectionStrings, a catalogue the desktop pushes into its own process. The pages render
// in the engine's own language, French.

// directionReq names one directed Tournament.
type directionReq struct {
	TournamentID int64 `json:"tournamentId"`
}

func (r directionReq) readKey() readKey { return readKey{tournamentID: r.TournamentID} }

// directionHistoryReq filters a Direction's history by player and/or match ("" for all).
type directionHistoryReq struct {
	TournamentID int64  `json:"tournamentId"`
	Player       string `json:"player"`
	Match        string `json:"match"`
}

func (r directionHistoryReq) readKey() readKey { return readKey{tournamentID: r.TournamentID} }

// directionRoundReq names one batch of pairings of a Direction.
type directionRoundReq struct {
	TournamentID int64 `json:"tournamentId"`
	Round        int   `json:"round"`
}

func (r directionRoundReq) readKey() readKey { return readKey{tournamentID: r.TournamentID} }

func (r directionRoundReq) validate() error {
	if r.Round < 0 {
		return fmt.Errorf("%w: round %d is negative", storage.ErrInvalid, r.Round)
	}
	return nil
}

// directionSlotsReq names the Direction whose Slots are read; their drafts are part of what
// the answer depends on.
type directionSlotsReq struct {
	TournamentID int64 `json:"tournamentId"`
}

func (r directionSlotsReq) readKey() readKey {
	return readKey{tournamentID: r.TournamentID, drafts: true}
}

// rencontreReq names one Rencontre.
type rencontreReq struct {
	ID int64 `json:"id"`
}

func (r rencontreReq) readKey() readKey { return readKey{rencontreID: r.ID} }

// scopeReq is the empty request of a read over the whole tenant.
type scopeReq struct{}

func (scopeReq) readKey() readKey { return readKey{scopeWide: true} }

// htmlResp carries a rendered page: a self-contained HTML document.
type htmlResp struct {
	HTML string `json:"html"`
}

// csvResp carries a CSV document.
type csvResp struct {
	CSV string `json:"csv"`
}

// directionService is the Direction service of one tenant, over the server's shared memory.
func (s *Server) directionService(scope string) *service.Service {
	return service.New(s.opts.Storage, scope, s.direction)
}

// readVersion is the version of what k names, with the minute folded in (see conditional.go).
func (s *Server) readVersion(ctx context.Context, scope string, k readKey) (string, error) {
	svc := s.directionService(scope)
	var v string
	var err error
	switch {
	case k.tournamentID != 0 && k.drafts:
		v, err = svc.SlotsVersion(ctx, k.tournamentID)
	case k.tournamentID != 0:
		v, err = svc.DirectionVersion(ctx, k.tournamentID)
	case k.rencontreID != 0:
		v, err = svc.RencontreVersion(ctx, k.rencontreID)
	case k.scopeWide:
		v, err = svc.ScopeVersion(ctx)
	default:
		return "", fmt.Errorf("%w: no tournament or rencontre named", storage.ErrInvalid)
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s|%d", v, s.opts.now().Unix()/60), nil
}

func (s *Server) directionReadRoutes() []route {
	svc := s.directionService
	v := s.readVersion
	return []route{
		{http.MethodPost, "/v1/directions.list", rpcRead(v, func(ctx context.Context, scope string, _ scopeReq) ([]service.DirectionSummary, error) {
			list, err := svc(scope).ListDirections(ctx)
			for i := range list {
				list[i].OutputDir = ""
			}
			return list, err
		})},
		{http.MethodPost, "/v1/directions.get", rpcRead(v, func(ctx context.Context, scope string, req directionReq) (*service.DirectionView, error) {
			v, err := svc(scope).GetDirection(ctx, req.TournamentID)
			if v != nil {
				v.OutputDir = ""
			}
			return v, err
		})},
		{http.MethodPost, "/v1/directions.participants", rpcRead(v, func(ctx context.Context, scope string, req directionReq) ([]service.ParticipantRow, error) {
			return svc(scope).Participants(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.freeParticipants", rpcRead(v, func(ctx context.Context, scope string, req directionReq) ([]tournoi.Player, error) {
			return svc(scope).FreeParticipants(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.tableGrid", rpcRead(v, func(ctx context.Context, scope string, req directionReq) ([]service.TableCell, error) {
			return svc(scope).TableGrid(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.brackets", rpcRead(v, func(ctx context.Context, scope string, req directionReq) ([]service.BracketPhase, error) {
			return svc(scope).Brackets(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.standings", rpcRead(v, func(ctx context.Context, scope string, req directionReq) (*service.StandingsView, error) {
			return svc(scope).Standings(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.standingsCsv", rpcRead(v, func(ctx context.Context, scope string, req directionReq) (csvResp, error) {
			csv, err := svc(scope).StandingsCSV(ctx, req.TournamentID)
			return csvResp{CSV: csv}, err
		})},
		{http.MethodPost, "/v1/directions.history", rpcRead(v, func(ctx context.Context, scope string, req directionHistoryReq) ([]service.HistoryEntry, error) {
			return svc(scope).History(ctx, req.TournamentID, req.Player, req.Match)
		})},
		{http.MethodPost, "/v1/directions.clock", rpcRead(v, func(ctx context.Context, scope string, req directionReq) (*service.ClockView, error) {
			return svc(scope).Clock(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.slots", rpcRead(v, func(ctx context.Context, scope string, req directionSlotsReq) ([]service.SlotRow, error) {
			return svc(scope).Slots(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.lastDecision", rpcRead(v, func(ctx context.Context, scope string, req directionReq) (*service.LastDecision, error) {
			return svc(scope).LastDecision(ctx, req.TournamentID)
		})},
		{http.MethodPost, "/v1/directions.directory", rpcRead(v, func(ctx context.Context, scope string, _ scopeReq) ([]service.DirectoryEntry, error) {
			return svc(scope).Directory(ctx)
		})},
		{http.MethodPost, "/v1/directions.pageHtml", rpcRead(v, func(ctx context.Context, scope string, req directionReq) (htmlResp, error) {
			html, err := svc(scope).DirectionPageHTML(ctx, req.TournamentID)
			return htmlResp{HTML: html}, err
		})},
		{http.MethodPost, "/v1/directions.pairingSheetHtml", rpcRead(v, func(ctx context.Context, scope string, req directionRoundReq) (htmlResp, error) {
			html, err := svc(scope).DirectionPairingSheetHTML(ctx, req.TournamentID, req.Round)
			return htmlResp{HTML: html}, err
		})},
	}
}

func (s *Server) rencontreReadRoutes() []route {
	svc := s.directionService
	v := s.readVersion
	return []route{
		{http.MethodPost, "/v1/rencontres.list", rpcRead(v, func(ctx context.Context, scope string, _ scopeReq) ([]service.RencontreView, error) {
			list, err := svc(scope).ListRencontres(ctx)
			for i := range list {
				list[i].OutputDir = ""
			}
			return list, err
		})},
		{http.MethodPost, "/v1/rencontres.get", rpcRead(v, func(ctx context.Context, scope string, req rencontreReq) (*service.RencontreView, error) {
			r, err := svc(scope).GetRencontre(ctx, req.ID)
			if r != nil {
				r.OutputDir = ""
			}
			return r, err
		})},
		{http.MethodPost, "/v1/rencontres.pageHtml", rpcRead(v, func(ctx context.Context, scope string, req rencontreReq) (htmlResp, error) {
			html, err := svc(scope).RencontrePageHTML(ctx, req.ID)
			return htmlResp{HTML: html}, err
		})},
		// The room view (Salle) joins here, as one more conditional read keyed by
		// rencontreReq: its version is the Rencontre's, which already covers every member.
	}
}
