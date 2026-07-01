package server

import (
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Type aliases for the iter.Seq2 shapes streamed as NDJSON by the domain
// handlers. They keep the rpcStream closures' return-type annotations short.
type (
	iterPositions = iter.Seq2[*domain.Position, error]
	iterMatches   = iter.Seq2[*domain.Match, error]
	iterGames     = iter.Seq2[*domain.Game, error]
	iterMoves     = iter.Seq2[*domain.Move, error]
	iterMovePos   = iter.Seq2[*domain.MatchMovePosition, error]
	iterComments  = iter.Seq2[*domain.CommentEntry, error]
	iterColls     = iter.Seq2[*storage.Collection, error]
	iterTours     = iter.Seq2[*domain.Tournament, error]
	iterDecks     = iter.Seq2[*domain.AnkiDeck, error]
	iterReviewLog = iter.Seq2[*domain.AnkiReviewLog, error]
	iterFilters   = iter.Seq2[*storage.Filter, error]
	iterSearchHis = iter.Seq2[*storage.SearchHistory, error]
)
