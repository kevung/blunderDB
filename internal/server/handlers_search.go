package server

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// searchFindReq mirrors storage.ListOpts over the wire (see handlers_positions.go's
// positionListReq): an all-zero Limit/Offset keeps the unbounded scan every
// caller got before pagination was pushed to SQL (B.10, #178).
type searchFindReq struct {
	Filters domain.SearchFilters `json:"filters"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
}

// searchQueryReq is search.find's other door: the query language the
// application's command bar speaks, rather than a hand-assembled filter struct
// (B.18, #186). A client that wants `s cube p>30 E>50` no longer has to know
// which of the forty-five fields those four tokens set.
type searchQueryReq struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// searchIntentReq carries a phrase to translate into search tokens (#283).
type searchIntentReq struct {
	Text string `json:"text"`
}

// searchParseResp is what /v1/search.parse answers: the filters a query
// denotes, and what the grammar could not act on. Exposed on its own so a
// client can show a user what their query means — and validate it — without
// running a scan.
type searchParseResp struct {
	Filters domain.SearchFilters `json:"filters"`
	Diags   []searchQueryDiag    `json:"diags,omitempty"`
	// Canonical is the query re-rendered from the filters: the same query, in
	// the grammar's own canonical token order. Two queries that mean the same
	// thing have the same Canonical, which is what makes a saved search
	// comparable.
	Canonical string `json:"canonical"`
}

type searchQueryDiag struct {
	Kind    string `json:"kind"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

func wireDiags(diags []searchquery.Diag) []searchQueryDiag {
	if len(diags) == 0 {
		return nil
	}
	out := make([]searchQueryDiag, 0, len(diags))
	for _, d := range diags {
		out = append(out, searchQueryDiag{Kind: string(d.Kind), Token: d.Token, Message: d.Message})
	}
	return out
}

// unknownTokens returns the tokens no rule claimed. A query built from a typo
// would otherwise scan the whole database and answer confidently with the wrong
// rows, so search.query refuses it — where search.parse reports it, since
// reporting is what search.parse is for.
func unknownTokens(diags []searchquery.Diag) []string {
	var out []string
	for _, d := range diags {
		if d.Kind == searchquery.DiagUnknown {
			out = append(out, d.Token)
		}
	}
	return out
}

func (s *Server) searchRoutes() []route {
	ss := func() storage.SearchStore { return s.opts.Storage.Search() }
	return []route{
		{http.MethodPost, "/v1/search.find", rpcStream(func(ctx context.Context, scope string, req searchFindReq) iterPositions {
			return ss().Find(ctx, scope, req.Filters, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},

		// search.parse answers what a query means, without touching the
		// storage: filters, canonical form, diagnostics.
		{http.MethodPost, "/v1/search.parse", rpc(func(_ context.Context, _ string, req searchQueryReq) (searchParseResp, error) {
			filters, diags := searchquery.Parse(req.Query)
			return searchParseResp{
				Filters:   filters,
				Diags:     wireDiags(diags),
				Canonical: searchquery.Format(filters),
			}, nil
		})},

		// search.intent translates a phrase into the tokens it means (#283).
		// It answers with the TOKENS, never with rows: the whole point of the
		// layer is that the user sees what was understood before anything is
		// searched.
		{http.MethodPost, "/v1/search.intent", rpc(func(_ context.Context, _ string, req searchIntentReq) (searchquery.Intent, error) {
			return searchquery.TranslateIntent(req.Text), nil
		})},

		// search.query is search.find, addressed in the query language. It
		// streams the same positions; a query carrying a token nothing claimed
		// is refused rather than run.
		{http.MethodPost, "/v1/search.query", s.handleSearchQuery(ss)},
	}
}

// handleSearchQuery decodes the query, refuses an unreadable one with a 400,
// and otherwise streams exactly what search.find would. Written out rather than
// built with rpcStream because the refusal has to happen after decoding and
// before a single row is written — once streamSeq2 has committed a 200, an
// error can only be a trailing line.
func (s *Server) handleSearchQuery(ss func() storage.SearchStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req searchQueryReq
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		if err := checkPageLimit(req); err != nil {
			writeStorageError(w, err)
			return
		}
		filters, diags := searchquery.Parse(req.Query)
		if unknown := unknownTokens(diags); len(unknown) > 0 {
			writeErrorCode(w, CodeInvalid, "unknown token(s) in query: "+strings.Join(unknown, ", "))
			return
		}
		// A no-effect token (`x`, the exclusion structure, which is a board)
		// travels in a header rather than the stream: the body is NDJSON
		// positions and must stay so for every existing client.
		if wire := wireDiags(diags); len(wire) > 0 {
			if encoded, err := json.Marshal(wire); err == nil {
				w.Header().Set("X-BlunderDB-Query-Diagnostics", string(encoded))
			}
		}
		streamSeq2(w, ss().Find(r.Context(), scopeOf(r), filters, storage.ListOpts{Limit: req.Limit, Offset: req.Offset}))
	}
}

var _ iter.Seq2[*domain.Position, error] = iterPositions(nil)
