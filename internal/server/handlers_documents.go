package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/document"
)

// documentImportReleveReq carries the Relevé of a document, as the JSON object
// `blunderdb document check` validates, and the caller's choices.
type documentImportReleveReq struct {
	Releve json.RawMessage `json:"releve"`
	// Collection names the Collection; the document's title when empty.
	Collection string `json:"collection"`
	// Force imports entries neither sure nor verified.
	Force bool `json:"force"`
}

// documentRoutes serve the import of a document's Relevé (ADR-0117). Reading a
// document is not served: the daemon receives the Relevé a desktop produced.
func (s *Server) documentRoutes() []route {
	return []route{
		// Idempotent by construction: a second import of the same Relevé
		// writes nothing new.
		{http.MethodPost, "/v1/documents.importReleve", rpc(func(ctx context.Context, scope string, req documentImportReleveReq) (document.ImportResult, error) {
			r, err := document.Decode(req.Releve)
			if err != nil {
				return document.ImportResult{}, err
			}
			return document.Import(ctx, s.opts.Storage, scope, r, document.ImportOptions{Collection: req.Collection, Force: req.Force})
		})},
	}
}
