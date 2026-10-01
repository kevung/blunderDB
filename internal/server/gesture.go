package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Gestures (ADR-0057 rule 4): every write on a Direction or a Rencontre states the version its
// client last read, in If-Match. No version → 428; a version that moved → 409, with the fresh
// state and its version, so the client reads what changed before it decides again. The
// comparison is the service's, under the gesture's own lock (service.ExpectVersion): of two
// gestures sent on one reading, exactly one applies.
//
// The version is the read's (service.DirectionVersion, or RencontreVersion for a room): every
// read of one Direction or one Rencontre carries it in the Direction-Version header, and every
// gesture answers with the version it left. A Direction that plays in a Rencontre has its room's
// version, so a gesture in a sister moves it too.

// versionHeader carries the version a gesture states in If-Match. It is not the read's ETag:
// that one also covers the request and the minute, and changes when nothing was written.
const versionHeader = "Direction-Version"

// pageWarningHeader reports, once per page, a display page the gesture could not rewrite: the
// gesture applied, the page shows the state before it.
const pageWarningHeader = "Direction-Page-Warning"

// gestureReq is implemented by the request of every versioned gesture: it names the Direction
// (tournamentID) or the Rencontre (rencontreID) whose version If-Match states.
type gestureReq interface {
	gestureKey() readKey
}

// quoteVersion renders a version as the entity-tag If-Match expects back.
func quoteVersion(v string) string { return `"` + v + `"` }

// ifMatchVersion reads the one version If-Match states. "*" states none: it would let a
// gesture apply on whatever the client never read, the very case the header is there to stop.
// A weak tag cannot match (RFC 9110 §13.1.1 compares If-Match strongly) and is refused alike.
func ifMatchVersion(header string) (string, bool) {
	v := strings.TrimSpace(header)
	if v == "" || v == "*" || strings.Contains(v, ",") || strings.HasPrefix(v, "W/") {
		return "", false
	}
	return strings.Trim(v, `"`), true
}

// rpcGesture is rpc for a versioned gesture: If-Match required (428), checked by the service
// under its lock (409 with the fresh state), and the version the gesture left in the answer's
// Direction-Version header.
func rpcGesture[Req gestureReq, Resp any](s *Server, fn func(ctx context.Context, scope string, req Req) (Resp, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		if vr, ok := any(req).(validatedReq); ok {
			if err := vr.validate(); err != nil {
				writeStorageError(w, err)
				return
			}
		}
		want, ok := ifMatchVersion(r.Header.Get("If-Match"))
		if !ok {
			writeErrorDetails(w, CodePreconditionRequired,
				"a gesture states the version it was decided on: send If-Match with the "+versionHeader+" of your last read", nil)
			return
		}
		scope := scopeOf(r)
		ctx := service.CollectPageWarnings(service.ExpectVersion(r.Context(), want))
		resp, err := fn(ctx, scope, req)
		for _, pw := range service.PageWarnings(ctx) {
			w.Header().Add(pageWarningHeader, pageWarningValue(pw))
		}
		if errors.Is(err, service.ErrStale) {
			s.writeStale(w, r.Context(), scope, req.gestureKey())
			return
		}
		if err != nil {
			writeStorageError(w, gestureError(err))
			return
		}
		if v := service.ResultVersion(ctx); v != "" {
			w.Header().Set(versionHeader, quoteVersion(v))
		}
		writeJSONResp(w, resp)
	}
}

// pageWarningValue names the page, not the failure: the error would name the server's folder,
// which a client is never shown.
func pageWarningValue(pw service.PageWarning) string {
	if pw.RencontreID != 0 {
		return "rencontre " + strconv.FormatInt(pw.RencontreID, 10)
	}
	return "tournament " + strconv.FormatInt(pw.TournamentID, 10)
}

// writeStale answers 409 to a gesture decided on a version that moved: the current state of
// what it named, and that state's version, in the error's details and the header.
func (s *Server) writeStale(w http.ResponseWriter, ctx context.Context, scope string, k readKey) {
	svc := s.directionService(scope)
	details := map[string]any{}
	var v string
	var err error
	if k.rencontreID != 0 {
		v, err = svc.RencontreVersion(ctx, k.rencontreID)
		if r, rerr := svc.GetRencontre(ctx, k.rencontreID); rerr == nil {
			r.OutputDir = ""
			details["rencontre"] = r
		}
	} else {
		v, err = svc.DirectionVersion(ctx, k.tournamentID)
		if view, verr := svc.GetDirection(ctx, k.tournamentID); verr == nil {
			view.OutputDir = ""
			details["direction"] = view
		}
	}
	if err == nil {
		details["version"] = v
		w.Header().Set(versionHeader, quoteVersion(v))
	}
	writeErrorDetails(w, CodeConflict,
		"someone else wrote since your version was read: read the state in details and decide again", details)
}

// gestureError classes a gesture's failure. A refusal — the rules', the engine's, a malformed
// configuration (direction.ErrRefused, *tournoi.ConfigRefusal) — is the client's request: a 400
// carrying its message. A sentinel keeps its own code. Anything else is the daemon's failure —
// a lost connection, a closed database — and stays one: a 500 whose message is not shown.
func gestureError(err error) error {
	if err == nil {
		return nil
	}
	var refusal *tournoi.ConfigRefusal
	if (errors.Is(err, direction.ErrRefused) || errors.As(err, &refusal)) && !errors.Is(err, storage.ErrInvalid) {
		return fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	return err
}

// errNoTarget refuses a gesture request that names nothing to version.
func errNoTarget(what string) error {
	return fmt.Errorf("%w: %s is required", storage.ErrInvalid, what)
}
