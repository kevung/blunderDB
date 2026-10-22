package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// transcriptionImportReq is a draft's source: the text of a .mat (`mat`), or
// a transcription document's Actions (`actions`) with its optional header —
// exactly one of the two. FormatVersion and NextScore let a stored draft's
// document be posted as it is.
type transcriptionImportReq struct {
	MAT           string              `json:"mat,omitempty"`
	Header        *transcript.Header  `json:"header,omitempty"`
	Actions       []transcript.Action `json:"actions,omitempty"`
	FormatVersion int                 `json:"format_version,omitempty"`
	NextScore     *[2]int             `json:"next_score,omitempty"`
}

// transcriptionImportRoutes opens a draft on a document written elsewhere. It
// is a gesture: it writes a row, so it is served only under
// Options.Transcription, and an Idempotency-Key keeps a resent request from
// opening two drafts.
func (s *Server) transcriptionImportRoutes() []route {
	if !s.opts.Transcription {
		return nil
	}
	return []route{
		{http.MethodPost, "/v1/transcriptions.import", s.withIdempotency(rpc(func(ctx context.Context, scope string, req transcriptionImportReq) (*transcription.State, error) {
			isDoc := req.Actions != nil || req.Header != nil
			switch {
			case req.MAT != "" && isDoc:
				return nil, fmt.Errorf("transcriptions.import takes mat or actions, not both: %w", storage.ErrInvalid)
			case req.MAT != "":
				return s.transcripts().ImportMAT(ctx, scope, req.MAT)
			case req.Actions != nil:
				doc := transcript.Document{FormatVersion: req.FormatVersion, Actions: req.Actions, NextScore: req.NextScore}
				if req.Header != nil {
					doc.Header = *req.Header
				}
				return s.transcripts().ImportDocument(ctx, scope, doc)
			}
			return nil, fmt.Errorf("transcriptions.import needs mat or actions: %w", storage.ErrInvalid)
		}))},
	}
}
