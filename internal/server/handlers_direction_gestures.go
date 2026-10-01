package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The gestures of a Direction and of a Rencontre (ADR-0057): every write the desktop makes on a
// directed Tournament or on a room, through the same service, bound to the caller's tenant.
// They are served only under `serve --direction` (rule 5) — and always under `call`, a local
// process — and every one but the two creations states its version in If-Match (gesture.go).
// A creation names nothing that exists yet, so it has no version to state; Idempotency-Key
// keeps a resent one from creating twice, and it does the same on every gesture.
//
// Every Direction gesture answers with the DirectionView as it now stands — the one the
// desktop shows — and every Rencontre gesture with the RencontreView. The display pages are
// the service's business: it rewrites them after each gesture, in whatever folder the database
// names, so a gesture sent here updates the wall of a room the desktop set up.
//
// The engine's own JSON (a configuration, an action, a list of players, of members, of breaks)
// travels as a JSON value, not as a string holding one.

// The requests that name one Direction.

type directionGestureReq struct {
	TournamentID int64 `json:"tournamentId"`
}

func (r directionGestureReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionCreateReq struct {
	TournamentID int64           `json:"tournamentId"`
	Config       json.RawMessage `json:"config"`
	Seed         int64           `json:"seed"`
}

type directionConfigReq struct {
	TournamentID int64           `json:"tournamentId"`
	Config       json.RawMessage `json:"config"`
}

func (r directionConfigReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionPlayersReq struct {
	TournamentID int64           `json:"tournamentId"`
	Players      json.RawMessage `json:"players"`
}

func (r directionPlayersReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

// directionAddParticipantReq enters one player; section and key, when given, seat a latecomer
// on a free bye slot (directions.freeSlots is not served: the desktop's latecomer dialog reads it).
type directionAddParticipantReq struct {
	TournamentID int64   `json:"tournamentId"`
	Name         string  `json:"name"`
	Club         string  `json:"club"`
	Rating       float64 `json:"rating"`
	Section      string  `json:"section"`
	Key          string  `json:"key"`
}

func (r directionAddParticipantReq) gestureKey() readKey {
	return readKey{tournamentID: r.TournamentID}
}

type directionUpdateParticipantReq struct {
	TournamentID int64   `json:"tournamentId"`
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Club         string  `json:"club"`
	Rating       float64 `json:"rating"`
}

func (r directionUpdateParticipantReq) gestureKey() readKey {
	return readKey{tournamentID: r.TournamentID}
}

type directionParticipantReq struct {
	TournamentID int64  `json:"tournamentId"`
	ID           string `json:"id"`
}

func (r directionParticipantReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionWithdrawReq struct {
	TournamentID int64  `json:"tournamentId"`
	ID           string `json:"id"`
	AfterCurrent bool   `json:"afterCurrent"`
}

func (r directionWithdrawReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionAbsentReq struct {
	TournamentID int64  `json:"tournamentId"`
	ID           string `json:"id"`
	Until        string `json:"until"`
	Round        int    `json:"round"`
}

func (r directionAbsentReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionAddPairReq struct {
	TournamentID int64           `json:"tournamentId"`
	Members      json.RawMessage `json:"members"`
	Rating       float64         `json:"rating"`
}

func (r directionAddPairReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionUpdatePairReq struct {
	TournamentID int64           `json:"tournamentId"`
	ID           string          `json:"id"`
	Members      json.RawMessage `json:"members"`
	Rating       float64         `json:"rating"`
}

func (r directionUpdatePairReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionProposalReq struct {
	TournamentID int64           `json:"tournamentId"`
	Action       json.RawMessage `json:"action"`
}

func (r directionProposalReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionStartMatchReq struct {
	TournamentID int64  `json:"tournamentId"`
	A            string `json:"a"`
	B            string `json:"b"`
	Length       int    `json:"length"`
	Table        int    `json:"table"`
}

func (r directionStartMatchReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionResultReq struct {
	TournamentID int64  `json:"tournamentId"`
	MatchID      string `json:"matchId"`
	Winner       string `json:"winner"`
	ScoreA       int    `json:"scoreA"`
	ScoreB       int    `json:"scoreB"`
	Note         string `json:"note"`
}

func (r directionResultReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionForfeitReq struct {
	TournamentID int64  `json:"tournamentId"`
	MatchID      string `json:"matchId"`
	Winner       string `json:"winner"`
	Note         string `json:"note"`
}

func (r directionForfeitReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionMoveReq struct {
	TournamentID int64  `json:"tournamentId"`
	MatchID      string `json:"matchId"`
	Table        int    `json:"table"`
}

func (r directionMoveReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionMatchReq struct {
	TournamentID int64  `json:"tournamentId"`
	MatchID      string `json:"matchId"`
}

func (r directionMatchReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

type directionNoteReq struct {
	TournamentID int64  `json:"tournamentId"`
	Text         string `json:"text"`
}

func (r directionNoteReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

// directionSlotReq names a Slot of a Direction and, to attach, the Match that fills it.
type directionSlotReq struct {
	TournamentID int64  `json:"tournamentId"`
	SlotID       string `json:"slotId"`
	MatchID      int64  `json:"matchId"`
}

func (r directionSlotReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

// The requests that name one Rencontre.

type rencontreCreateReq struct {
	Name     string `json:"name"`
	StartsOn string `json:"startsOn"`
	EndsOn   string `json:"endsOn"`
	Tables   int    `json:"tables"`
}

type rencontreUpdateReq struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	StartsOn string `json:"startsOn"`
	EndsOn   string `json:"endsOn"`
	Tables   int    `json:"tables"`
}

func (r rencontreUpdateReq) gestureKey() readKey { return readKey{rencontreID: r.ID} }

// rencontreAttachReq puts the directed Tournament tournamentId in the room id; the version is
// the room's.
type rencontreAttachReq struct {
	ID           int64 `json:"id"`
	TournamentID int64 `json:"tournamentId"`
}

func (r rencontreAttachReq) gestureKey() readKey { return readKey{rencontreID: r.ID} }

type rencontreGestureReq struct {
	ID int64 `json:"id"`
}

func (r rencontreGestureReq) gestureKey() readKey { return readKey{rencontreID: r.ID} }

type rencontreTableReq struct {
	ID    int64 `json:"id"`
	Table int   `json:"table"`
	Out   bool  `json:"out"`
}

func (r rencontreTableReq) gestureKey() readKey { return readKey{rencontreID: r.ID} }

type rencontreBreaksReq struct {
	ID     int64           `json:"id"`
	Breaks json.RawMessage `json:"breaks"`
}

func (r rencontreBreaksReq) gestureKey() readKey { return readKey{rencontreID: r.ID} }

type rencontreTablesReq struct {
	ID            int64                 `json:"id"`
	TableSettings []domain.TableSetting `json:"tableSettings"`
}

func (r rencontreTablesReq) gestureKey() readKey { return readKey{rencontreID: r.ID} }

type rencontreRoomsReq struct {
	ID           int64    `json:"id"`
	TournamentID int64    `json:"tournamentId"`
	Rooms        []string `json:"rooms"`
}

func (r rencontreRoomsReq) gestureKey() readKey { return readKey{rencontreID: r.ID} }

type directionTablesReq struct {
	TournamentID  int64                 `json:"tournamentId"`
	TableSettings []domain.TableSetting `json:"tableSettings"`
}

func (r directionTablesReq) gestureKey() readKey { return readKey{tournamentID: r.TournamentID} }

// trashResp names the trash entry a deleted Rencontre went to.
type trashResp struct {
	TrashID int64 `json:"trashId"`
}

// rawText is the engine JSON a service method takes as a string; absent is "".
func rawText(m json.RawMessage) string { return string(m) }

// bare hides the output folder: a path of the server's disk is no client's business.
func bare(v *service.DirectionView, err error) (*service.DirectionView, error) {
	if v != nil {
		v.OutputDir = ""
	}
	return v, err
}

// bareRoom is bare for a Rencontre.
func bareRoom(r *service.RencontreView, err error) (*service.RencontreView, error) {
	if r != nil {
		r.OutputDir = ""
	}
	return r, err
}

// thenView returns the Direction as it stands once a gesture that answers nothing succeeded,
// and the version of that very view as the answer's Direction-Version (service.ViewAfter).
func thenView(ctx context.Context, svc *service.Service, tournamentID int64, err error) (*service.DirectionView, error) {
	if err != nil {
		return nil, err
	}
	return bare(svc.ViewAfter(ctx, tournamentID))
}

// directionGestureRoutes is empty unless the daemon was started with --direction.
func (s *Server) directionGestureRoutes() []route {
	if !s.opts.EnableDirection {
		return nil
	}
	svc := s.directionService
	return []route{
		{http.MethodPost, "/v1/directions.create", s.withIdempotency(rpc(func(ctx context.Context, scope string, req directionCreateReq) (*service.DirectionView, error) {
			if req.TournamentID == 0 {
				return nil, errNoTarget("tournamentId")
			}
			d := svc(scope)
			v, err := thenView(ctx, d, req.TournamentID, d.CreateDirection(ctx, req.TournamentID, rawText(req.Config), req.Seed))
			return v, gestureError(err)
		}))},
		{http.MethodPost, "/v1/directions.previewConfig", rpc(func(ctx context.Context, scope string, req directionConfigReq) (*service.ConfigPreview, error) {
			p, err := svc(scope).PreviewDirectionConfig(ctx, req.TournamentID, rawText(req.Config))
			return p, gestureError(err)
		})},
		{http.MethodPost, "/v1/directions.setConfig", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionConfigReq) (*service.DirectionView, error) {
			d := svc(scope)
			return thenView(ctx, d, req.TournamentID, d.SetDirectionConfig(ctx, req.TournamentID, rawText(req.Config)))
		}))},
		{http.MethodPost, "/v1/directions.enterParticipants", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionPlayersReq) (*service.DirectionView, error) {
			d := svc(scope)
			return thenView(ctx, d, req.TournamentID, d.EnterParticipants(ctx, req.TournamentID, rawText(req.Players)))
		}))},
		{http.MethodPost, "/v1/directions.addParticipant", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionAddParticipantReq) (*service.DirectionView, error) {
			if req.Section != "" || req.Key != "" {
				return bare(svc(scope).AddParticipantAtSlot(ctx, req.TournamentID, req.Name, req.Club, req.Rating, req.Section, req.Key))
			}
			return bare(svc(scope).AddParticipant(ctx, req.TournamentID, req.Name, req.Club, req.Rating))
		}))},
		{http.MethodPost, "/v1/directions.updateParticipant", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionUpdateParticipantReq) (*service.DirectionView, error) {
			return bare(svc(scope).UpdateParticipant(ctx, req.TournamentID, req.ID, req.Name, req.Club, req.Rating))
		}))},
		{http.MethodPost, "/v1/directions.withdraw", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionWithdrawReq) (*service.DirectionView, error) {
			return bare(svc(scope).WithdrawParticipant(ctx, req.TournamentID, req.ID, req.AfterCurrent))
		}))},
		{http.MethodPost, "/v1/directions.reinstate", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionParticipantReq) (*service.DirectionView, error) {
			return bare(svc(scope).ReinstateParticipant(ctx, req.TournamentID, req.ID))
		}))},
		{http.MethodPost, "/v1/directions.makeAbsent", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionAbsentReq) (*service.DirectionView, error) {
			return bare(svc(scope).MakeParticipantAbsent(ctx, req.TournamentID, req.ID, req.Until, req.Round))
		}))},
		{http.MethodPost, "/v1/directions.makeAvailable", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionParticipantReq) (*service.DirectionView, error) {
			return bare(svc(scope).MakeParticipantAvailable(ctx, req.TournamentID, req.ID))
		}))},
		{http.MethodPost, "/v1/directions.addPair", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionAddPairReq) (*service.DirectionView, error) {
			return bare(svc(scope).AddPair(ctx, req.TournamentID, rawText(req.Members), req.Rating))
		}))},
		{http.MethodPost, "/v1/directions.updatePair", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionUpdatePairReq) (*service.DirectionView, error) {
			return bare(svc(scope).UpdatePair(ctx, req.TournamentID, req.ID, rawText(req.Members), req.Rating))
		}))},
		{http.MethodPost, "/v1/directions.confirmProposal", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionProposalReq) (*service.DirectionView, error) {
			return bare(svc(scope).ConfirmProposal(ctx, req.TournamentID, rawText(req.Action)))
		}))},
		{http.MethodPost, "/v1/directions.confirmAllProposals", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionGestureReq) (*service.DirectionView, error) {
			return bare(svc(scope).ConfirmAllProposals(ctx, req.TournamentID))
		}))},
		{http.MethodPost, "/v1/directions.startMatch", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionStartMatchReq) (*service.DirectionView, error) {
			return bare(svc(scope).StartMatchManually(ctx, req.TournamentID, req.A, req.B, req.Length, req.Table))
		}))},
		{http.MethodPost, "/v1/directions.enterResult", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionResultReq) (*service.DirectionView, error) {
			return bare(svc(scope).EnterResult(ctx, req.TournamentID, req.MatchID, req.Winner, req.ScoreA, req.ScoreB, req.Note))
		}))},
		{http.MethodPost, "/v1/directions.enterForfeit", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionForfeitReq) (*service.DirectionView, error) {
			return bare(svc(scope).EnterForfeit(ctx, req.TournamentID, req.MatchID, req.Winner, req.Note))
		}))},
		{http.MethodPost, "/v1/directions.moveMatchToTable", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionMoveReq) (*service.DirectionView, error) {
			return bare(svc(scope).MoveMatchToTable(ctx, req.TournamentID, req.MatchID, req.Table))
		}))},
		{http.MethodPost, "/v1/directions.cancelMatch", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionMatchReq) (*service.DirectionView, error) {
			return bare(svc(scope).CancelMatch(ctx, req.TournamentID, req.MatchID))
		}))},
		{http.MethodPost, "/v1/directions.correctResult", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionResultReq) (*service.DirectionView, error) {
			return bare(svc(scope).CorrectResult(ctx, req.TournamentID, req.MatchID, req.Winner, req.ScoreA, req.ScoreB, req.Note))
		}))},
		{http.MethodPost, "/v1/directions.close", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionGestureReq) (*service.DirectionView, error) {
			return bare(svc(scope).CloseDirection(ctx, req.TournamentID))
		}))},
		{http.MethodPost, "/v1/directions.reopen", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionGestureReq) (*service.DirectionView, error) {
			return bare(svc(scope).ReopenDirection(ctx, req.TournamentID))
		}))},
		{http.MethodPost, "/v1/directions.addNote", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionNoteReq) (*service.DirectionView, error) {
			return bare(svc(scope).AddDirectionNote(ctx, req.TournamentID, req.Text))
		}))},
		{http.MethodPost, "/v1/directions.attachMatch", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionSlotReq) (*service.DirectionView, error) {
			d := svc(scope)
			return thenView(ctx, d, req.TournamentID, d.AttachMatchToSlot(ctx, req.TournamentID, req.SlotID, req.MatchID))
		}))},
		{http.MethodPost, "/v1/directions.detachMatch", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionSlotReq) (*service.DirectionView, error) {
			d := svc(scope)
			return thenView(ctx, d, req.TournamentID, d.DetachMatchFromSlot(ctx, req.TournamentID, req.SlotID))
		}))},
	}
}

// rencontreGestureRoutes is empty unless the daemon was started with --direction.
func (s *Server) rencontreGestureRoutes() []route {
	if !s.opts.EnableDirection {
		return nil
	}
	svc := s.directionService
	return []route{
		{http.MethodPost, "/v1/rencontres.create", s.withIdempotency(rpc(func(ctx context.Context, scope string, req rencontreCreateReq) (*service.RencontreView, error) {
			r, err := bareRoom(svc(scope).CreateRencontre(ctx, req.Name, req.StartsOn, req.EndsOn, req.Tables))
			return r, gestureError(err)
		}))},
		{http.MethodPost, "/v1/rencontres.update", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req rencontreUpdateReq) (*service.RencontreView, error) {
			return bareRoom(svc(scope).UpdateRencontre(ctx, req.ID, req.Name, req.StartsOn, req.EndsOn, req.Tables))
		}))},
		{http.MethodPost, "/v1/rencontres.attach", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req rencontreAttachReq) (*service.RencontreView, error) {
			return bareRoom(svc(scope).AttachToRencontre(ctx, req.TournamentID, req.ID))
		}))},
		// detach names the Tournament leaving its room: the version is its own, the room's.
		{http.MethodPost, "/v1/rencontres.detach", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionGestureReq) (*service.DirectionView, error) {
			d := svc(scope)
			return thenView(ctx, d, req.TournamentID, d.DetachFromRencontre(ctx, req.TournamentID))
		}))},
		{http.MethodPost, "/v1/rencontres.trash", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req rencontreGestureReq) (trashResp, error) {
			id, err := svc(scope).TrashRencontre(ctx, req.ID)
			return trashResp{TrashID: id}, err
		}))},
		{http.MethodPost, "/v1/rencontres.setTableOutOfService", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req rencontreTableReq) (*service.RencontreView, error) {
			return bareRoom(svc(scope).SetRencontreTableOutOfService(ctx, req.ID, req.Table, req.Out))
		}))},
		{http.MethodPost, "/v1/rencontres.setBreaks", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req rencontreBreaksReq) (*service.RencontreView, error) {
			return bareRoom(svc(scope).SetRencontreBreaks(ctx, req.ID, rawText(req.Breaks)))
		}))},
		{http.MethodPost, "/v1/rencontres.setTables", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req rencontreTablesReq) (*service.RencontreView, error) {
			return bareRoom(svc(scope).SetRencontreTables(ctx, req.ID, req.TableSettings))
		}))},
		{http.MethodPost, "/v1/rencontres.setEventRooms", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req rencontreRoomsReq) (*service.RencontreView, error) {
			return bareRoom(svc(scope).SetEventRooms(ctx, req.ID, req.TournamentID, req.Rooms))
		}))},
		{http.MethodPost, "/v1/directions.setTables", s.withIdempotency(rpcGesture(s, func(ctx context.Context, scope string, req directionTablesReq) (*service.DirectionView, error) {
			return bare(svc(scope).SetDirectionTables(ctx, req.TournamentID, req.TableSettings))
		}))},
	}
}
