package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// The transcription family (ADR-0057): a draft is typed gesture by gesture
// through transcription.Service, the same one the desktop uses. The reads are
// always served; the gestures only under Options.Transcription (`serve
// --transcription`), and every gesture that writes names the revision it was
// typed against in If-Match.

// defaultTranscriptionTTL is how long an idle session keeps its undo stack.
const defaultTranscriptionTTL = 30 * time.Minute

type transcriptionIDReq struct {
	ID int64 `json:"id"`
}

type transcriptionCreateReq struct {
	Header transcript.Header `json:"header"`
}

// transcriptionSessionReq names a draft and the session the client holds.
// Over HTTP the session is required (see sessionOf); under `call` it may be
// left out, one process per call, no undo between calls.
type transcriptionSessionReq struct {
	ID        int64  `json:"id"`
	SessionID string `json:"sessionId"`
}

type transcriptionApplyReq struct {
	ID        int64              `json:"id"`
	SessionID string             `json:"sessionId"`
	Gesture   transcript.Gesture `json:"gesture"`
}

// transcriptionEditResp is the draft opened on a Match and what opening it
// dropped: a client shows Losses when Losses.Lossy.
type transcriptionEditResp struct {
	State  *transcription.State  `json:"state"`
	Losses *transcription.Losses `json:"losses"`
}

// transcriptionMATResp carries a draft's .mat text and its default file name.
type transcriptionMATResp struct {
	Text     string `json:"text"`
	Filename string `json:"filename"`
}

// transcripts is the server's transcription service, made on first use.
func (s *Server) transcripts() *transcription.Service {
	s.transcriptsOnce.Do(func() {
		ttl := s.opts.TranscriptionTTL
		if ttl <= 0 {
			ttl = defaultTranscriptionTTL
		}
		o := transcription.Options{TTL: ttl}
		if s.eventsEnabled() {
			o.Events = s.events
		}
		s.transcriptSvc = transcription.New(s.opts.Storage, o)
	})
	return s.transcriptSvc
}

func (s *Server) transcriptionRoutes() []route {
	rs := []route{
		{http.MethodPost, "/v1/transcriptions.list", rpc(func(ctx context.Context, scope string, _ struct{}) ([]transcription.Summary, error) {
			return s.transcripts().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/transcriptions.get", s.transcriptionGetHandler},
		{http.MethodPost, "/v1/transcriptions.exportMat", rpc(func(ctx context.Context, scope string, req transcriptionIDReq) (transcriptionMATResp, error) {
			text, err := s.transcripts().MAT(ctx, scope, req.ID)
			if err != nil {
				return transcriptionMATResp{}, err
			}
			name, err := s.transcripts().SuggestMATFilename(ctx, scope, req.ID)
			return transcriptionMATResp{Text: text, Filename: name}, err
		})},
		{http.MethodPost, "/v1/transcriptions.losses", rpc(func(ctx context.Context, scope string, req matchIDReq) (*transcription.Losses, error) {
			return s.transcripts().MatchLosses(ctx, scope, req.MatchID)
		})},
	}
	if !s.opts.Transcription {
		return rs
	}
	return append(rs, []route{
		{http.MethodPost, "/v1/transcriptions.create", rpc(func(ctx context.Context, scope string, req transcriptionCreateReq) (*transcription.State, error) {
			return s.transcripts().Create(ctx, scope, req.Header)
		})},
		{http.MethodPost, "/v1/transcriptions.open", rpc(func(ctx context.Context, scope string, req transcriptionIDReq) (*transcription.State, error) {
			return s.transcripts().Open(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/transcriptions.editMatch", rpc(func(ctx context.Context, scope string, req matchIDReq) (transcriptionEditResp, error) {
			st, losses, err := s.transcripts().EditMatch(ctx, scope, req.MatchID)
			return transcriptionEditResp{State: st, Losses: losses}, err
		})},
		{http.MethodPost, "/v1/transcriptions.apply", s.withIdempotency(s.withIfMatch(rpc(func(ctx context.Context, scope string, req transcriptionApplyReq) (*transcription.State, error) {
			if err := s.sessionOf(req.SessionID); err != nil {
				return nil, err
			}
			return s.transcripts().Apply(ctx, scope, req.ID, expectOf(ctx, req.SessionID), req.Gesture)
		})))},
		{http.MethodPost, "/v1/transcriptions.undo", s.withIdempotency(s.withIfMatch(rpc(func(ctx context.Context, scope string, req transcriptionSessionReq) (*transcription.State, error) {
			if err := s.sessionOf(req.SessionID); err != nil {
				return nil, err
			}
			return s.transcripts().Apply(ctx, scope, req.ID, expectOf(ctx, req.SessionID), transcript.Gesture{Kind: transcript.GestureUndo})
		})))},
		{http.MethodPost, "/v1/transcriptions.redo", s.withIdempotency(s.withIfMatch(rpc(func(ctx context.Context, scope string, req transcriptionSessionReq) (*transcription.State, error) {
			if err := s.sessionOf(req.SessionID); err != nil {
				return nil, err
			}
			return s.transcripts().Apply(ctx, scope, req.ID, expectOf(ctx, req.SessionID), transcript.Gesture{Kind: transcript.GestureRedo})
		})))},
		{http.MethodPost, "/v1/transcriptions.close", rpcVoid(func(_ context.Context, scope string, req transcriptionSessionReq) error {
			if err := s.sessionOf(req.SessionID); err != nil {
				return err
			}
			return s.transcripts().Close(scope, req.ID, req.SessionID)
		})},
		{http.MethodPost, "/v1/transcriptions.finish", s.withIdempotency(s.withIfMatch(rpc(func(ctx context.Context, scope string, req transcriptionSessionReq) (*transcription.SaveResult, error) {
			if err := s.sessionOf(req.SessionID); err != nil {
				return nil, err
			}
			return s.transcripts().Finish(ctx, scope, req.ID, expectOf(ctx, req.SessionID))
		})))},
		{http.MethodPost, "/v1/transcriptions.abandon", s.withIdempotency(s.withIfMatch(rpcVoid(func(ctx context.Context, scope string, req transcriptionIDReq) error {
			return s.transcripts().Abandon(ctx, scope, req.ID, expectOf(ctx, ""))
		})))},
	}...)
}

// transcriptionGetHandler serves POST /v1/transcriptions.get {id}: the draft
// read from its row, no session opened, with its revision as ETag; an
// If-None-Match naming that revision answers 304 without a body.
func (s *Server) transcriptionGetHandler(w http.ResponseWriter, r *http.Request) {
	var req transcriptionIDReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid request body", err)
		return
	}
	ctx, scope := r.Context(), scopeOf(r)
	if inm := r.Header.Get("If-None-Match"); inm != "" {
		rev, err := s.transcripts().Revision(ctx, scope, req.ID)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		if n, ok := parseRevision(inm); ok && n == rev {
			w.Header().Set("ETag", revisionETag(rev))
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	st, err := s.transcripts().Get(ctx, scope, req.ID)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	w.Header().Set("ETag", revisionETag(st.Revision))
	writeJSONResp(w, st)
}

type ifMatchKey struct{}

// withIfMatch requires the revision a gesture was typed against (ADR-0057
// rule 4): no If-Match answers 428, an unreadable one 400; the revision
// travels to the handler in the context.
func (s *Server) withIfMatch(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("If-Match")
		if h == "" {
			writeErrorCode(w, CodePreconditionRequired, "If-Match is required: name the revision the gesture was typed against")
			return
		}
		rev, ok := parseRevision(h)
		if !ok {
			writeErrorCode(w, CodeInvalid, "If-Match must name one revision, a positive integer")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ifMatchKey{}, rev)))
	}
}

// sessionOf refuses a session gesture that names no session, unless the
// server runs one session per call (Options.SessionPerCall).
func (s *Server) sessionOf(sessionID string) error {
	if sessionID == "" && !s.opts.SessionPerCall {
		return fmt.Errorf("sessionId is required: name the session transcriptions.open returned: %w", storage.ErrInvalid)
	}
	return nil
}

// expectOf is the expectation a gesture states: its session and its
// If-Match revision.
func expectOf(ctx context.Context, sessionID string) transcription.Expect {
	rev, _ := ctx.Value(ifMatchKey{}).(int64)
	return transcription.Expect{Session: sessionID, Revision: rev}
}

// parseRevision reads an entity tag naming a revision: `"7"`, `W/"7"` or a
// bare 7.
func parseRevision(h string) (int64, bool) {
	h = strings.TrimPrefix(strings.TrimSpace(h), "W/")
	h = strings.Trim(h, `"`)
	n, err := strconv.ParseInt(h, 10, 64)
	return n, err == nil && n > 0
}

func revisionETag(rev int64) string { return `"` + strconv.FormatInt(rev, 10) + `"` }
