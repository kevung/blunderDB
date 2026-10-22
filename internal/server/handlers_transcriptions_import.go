package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// transcriptionImportReq is a draft's source: the text of a .mat, or a
// transcription document (header and actions) — exactly one of the two.
type transcriptionImportReq struct {
	MAT      string               `json:"mat,omitempty"`
	Document *transcript.Document `json:"document,omitempty"`
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
			switch {
			case req.MAT != "" && req.Document != nil:
				return nil, fmt.Errorf("transcriptions.import takes mat or document, not both: %w", storage.ErrInvalid)
			case req.Document != nil:
				return s.transcripts().ImportDocument(ctx, scope, *req.Document)
			case req.MAT != "":
				return s.transcripts().ImportMAT(ctx, scope, req.MAT)
			}
			return nil, fmt.Errorf("transcriptions.import needs mat or document: %w", storage.ErrInvalid)
		}))},
	}
}
