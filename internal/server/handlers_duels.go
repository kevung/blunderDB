package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The Duel family (ADR-0072 rule 11, on the pattern of ADR-0057): a Duel is
// played one Action at a time through duel.Service, the same one the desktop
// uses. The reads are always served; the gestures only under Options.Duel
// (`serve --duel`), and every gesture on an existing Duel names the revision it
// was decided against in If-Match. The daemon authenticates nobody (ADR-0005):
// an Action names its Side, and the client answers for who plays for whom.
// The seed leaves by no route — State carries its fingerprint — until the
// Match that ended the Duel reveals it.

type duelIDReq struct {
	ID int64 `json:"id"`
}

// duelPlayReq is a Play as a client names it. The Side is a pointer so that
// one left out is told from player 1: an Action names its Side.
type duelPlayReq struct {
	Side  *int                 `json:"side"`
	Kind  duel.PlayKind        `json:"kind"`
	Steps []domain.CheckerStep `json:"steps,omitempty"`
	Level int                  `json:"level,omitempty"`
}

type duelActReq struct {
	ID   int64       `json:"id"`
	Play duelPlayReq `json:"play"`
}

// duelForfeitReq names the Side giving the match up, 0 player 1 or 1 player
// 2; required, since a forfeit answers no Decision that would name it.
type duelForfeitReq struct {
	ID   int64 `json:"id"`
	Side *int  `json:"side"`
}

type duelContributeReq struct {
	ID           int64  `json:"id"`
	Side         *int   `json:"side"`
	Contribution string `json:"contribution"`
}

type duelStopReq struct {
	ID int64 `json:"id"`
	// Keep writes a money session's Match as it stands, and is refused for a
	// match in points (duel.Service.Stop); false throws the draft away.
	Keep bool `json:"keep"`
}

// duels is the server's Duel service, made on first use.
func (s *Server) duels() *duel.Service {
	s.duelOnce.Do(func() {
		sides := s.opts.DuelSides
		if sides == nil {
			sides = duel.Resolve
		}
		s.duelSvc = duel.New(s.opts.Storage, duel.Options{Sides: sides})
	})
	return s.duelSvc
}

// isDuelRefusal reports an error that is the client's request: an Action the
// rules refuse, a side or a Cadence this build does not offer.
func isDuelRefusal(err error) bool {
	var r *transcript.Refusal
	return errors.As(err, &r) || errors.Is(err, duel.ErrUnknownSide) || errors.Is(err, duel.ErrUnknownLevel) || errors.Is(err, duel.ErrInvalidCadence)
}

// announce tells the subscribers of the tenant that a Duel moved.
func (s *Server) announce(scope string, id int64, st *duel.State, err error) {
	if err != nil || st == nil || !s.eventsEnabled() || !s.publisher.Wants(scope) {
		return
	}
	ev := events.Event{Scope: scope, Kind: events.KindDuel, DuelID: id, Revision: st.Revision}
	if st.Ended != nil {
		ev.Removed, ev.MatchID = true, st.Ended.MatchID
	}
	s.publisher.Publish(ev)
}

// duelGesture runs a gesture on the Duel id: it is made the open one first
// when it is not — a daemon that restarted, a Duel in suspense — and the
// result is announced.
func (s *Server) duelGesture(ctx context.Context, scope string, id int64, fn func(rev int64) (*duel.State, error)) (*duel.State, error) {
	rev, _ := ctx.Value(ifMatchKey{}).(int64)
	rev, err := s.duels().Ensure(ctx, scope, id, rev)
	if err != nil {
		return nil, err
	}
	st, err := fn(rev)
	s.announce(scope, id, st, err)
	return st, err
}

func (s *Server) duelRoutes() []route {
	rs := []route{
		{http.MethodPost, "/v1/duels.list", rpc(func(ctx context.Context, scope string, _ struct{}) ([]duel.Summary, error) {
			return s.duels().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/duels.get", s.duelGetHandler},
		// The origin of a Match played here, its seed revealed; null for a
		// Match not played here, 404 for no Match. A read: served without --duel.
		{http.MethodPost, "/v1/matches.origin", rpc(func(ctx context.Context, scope string, req matchIDReq) (*duel.Origin, error) {
			return duel.ReadOrigin(ctx, s.opts.Storage, scope, req.MatchID)
		})},
	}
	if !s.opts.Duel {
		return rs
	}
	gesture := func(h http.HandlerFunc) http.HandlerFunc { return s.withIdempotency(s.withIfMatch(h)) }
	return append(rs, []route{
		{http.MethodPost, "/v1/duels.create", s.withIdempotency(rpc(func(ctx context.Context, scope string, req duel.Settings) (*duel.State, error) {
			// A Duel driven one process per call keeps no clock between calls.
			if s.opts.SessionPerCall && req.Cadence != nil {
				return nil, fmt.Errorf("a Duel played one call per process has no Cadence: %w", storage.ErrInvalid)
			}
			st, err := s.duels().Create(ctx, scope, req)
			if st != nil {
				s.announce(scope, st.ID, st, err)
			}
			return st, err
		}))},
		{http.MethodPost, "/v1/duels.open", gesture(rpc(func(ctx context.Context, scope string, req duelIDReq) (*duel.State, error) {
			rev, _ := ctx.Value(ifMatchKey{}).(int64)
			st, err := s.duels().OpenAt(ctx, scope, req.ID, rev)
			s.announce(scope, req.ID, st, err)
			return st, err
		}))},
		{http.MethodPost, "/v1/duels.act", gesture(rpc(func(ctx context.Context, scope string, req duelActReq) (*duel.State, error) {
			if req.Play.Side == nil || (*req.Play.Side != 0 && *req.Play.Side != 1) {
				return nil, fmt.Errorf("play.side is required, 0 for player 1 or 1 for player 2: %w", storage.ErrInvalid)
			}
			p := duel.Play{Side: *req.Play.Side, Kind: req.Play.Kind, Steps: req.Play.Steps, Level: req.Play.Level}
			return s.duelGesture(ctx, scope, req.ID, func(rev int64) (*duel.State, error) {
				return s.duels().Play(ctx, scope, req.ID, rev, p)
			})
		}))},
		{http.MethodPost, "/v1/duels.flag", gesture(rpc(func(ctx context.Context, scope string, req duelIDReq) (*duel.State, error) {
			return s.duelGesture(ctx, scope, req.ID, func(int64) (*duel.State, error) {
				return s.duels().Flag(ctx, scope, req.ID)
			})
		}))},
		{http.MethodPost, "/v1/duels.suspend", gesture(rpc(func(ctx context.Context, scope string, req duelIDReq) (okResp, error) {
			rev, _ := ctx.Value(ifMatchKey{}).(int64)
			if _, err := s.duels().Ensure(ctx, scope, req.ID, rev); err != nil {
				return okResp{}, err
			}
			if err := s.duels().Suspend(ctx, scope, req.ID); err != nil {
				return okResp{}, err
			}
			if s.eventsEnabled() && s.publisher.Wants(scope) {
				s.publisher.Publish(events.Event{Scope: scope, Kind: events.KindDuel, DuelID: req.ID})
			}
			return okResp{OK: true}, nil
		}))},
		{http.MethodPost, "/v1/duels.stop", gesture(rpc(func(ctx context.Context, scope string, req duelStopReq) (*duel.State, error) {
			return s.duelGesture(ctx, scope, req.ID, func(rev int64) (*duel.State, error) {
				return s.duels().Stop(ctx, scope, req.ID, rev, req.Keep)
			})
		}))},
		{http.MethodPost, "/v1/duels.forfeit", gesture(rpc(func(ctx context.Context, scope string, req duelForfeitReq) (*duel.State, error) {
			if req.Side == nil || (*req.Side != 0 && *req.Side != 1) {
				return nil, fmt.Errorf("side is required, 0 for player 1 or 1 for player 2: %w", storage.ErrInvalid)
			}
			return s.duelGesture(ctx, scope, req.ID, func(rev int64) (*duel.State, error) {
				return s.duels().Forfeit(ctx, scope, req.ID, rev, *req.Side)
			})
		}))},
		{http.MethodPost, "/v1/duels.contribute", gesture(rpc(func(ctx context.Context, scope string, req duelContributeReq) (*duel.State, error) {
			if req.Side == nil || (*req.Side != 0 && *req.Side != 1) {
				return nil, fmt.Errorf("side is required, 0 for player 1 or 1 for player 2: %w", storage.ErrInvalid)
			}
			return s.duelGesture(ctx, scope, req.ID, func(rev int64) (*duel.State, error) {
				return s.duels().Contribute(ctx, scope, req.ID, rev, *req.Side, req.Contribution)
			})
		}))},
		{http.MethodPost, "/v1/duels.discard", gesture(rpc(func(ctx context.Context, scope string, req duelIDReq) (*duel.State, error) {
			return s.duelGesture(ctx, scope, req.ID, func(rev int64) (*duel.State, error) {
				return s.duels().Stop(ctx, scope, req.ID, rev, false)
			})
		}))},
	}...)
}

// duelGetHandler serves POST /v1/duels.get {id}: the Duel read from its row,
// with its revision as ETag; an If-None-Match naming that revision answers 304
// without a body.
func (s *Server) duelGetHandler(w http.ResponseWriter, r *http.Request) {
	var req duelIDReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid request body", err)
		return
	}
	st, err := s.duels().Get(r.Context(), scopeOf(r), req.ID)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	w.Header().Set("ETag", revisionETag(st.Revision))
	if n, ok := parseRevision(r.Header.Get("If-None-Match")); ok && n == st.Revision {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSONResp(w, st)
}
