package storage

import "context"

// PositionListFields is what a table of positions shows beside the board, for
// one position, read for a whole page at once.
type PositionListFields struct {
	PositionID int64 `json:"positionId"`
	// Analysed is false when the position has no analysis: PlayedErrorMP is
	// then nil and means "no analysis", not "no error".
	Analysed bool `json:"analysed"`
	// PlayedErrorMP is the equity the played decision lost, in millipoints on
	// the position's own scale (money points at money, normalised equity at a
	// match score): the cube error of a cube decision, the checker error of a
	// checker play. Nil without analysis, or when the play is none of the
	// analysed candidates. A position reached by several matches is scored on
	// the one play its analysis names, as the error filter of the search is.
	PlayedErrorMP *int64 `json:"playedErrorMp"`
	// Date is the day (YYYY-MM-DD, UTC) of the earliest match that reaches the
	// position, else the day it entered the library; MatchDated says which.
	Date       string `json:"date"`
	MatchDated bool   `json:"matchDated"`
	// Collections holds the position, in the library's collection order;
	// empty, never nil, when none does.
	Collections []CollectionRef `json:"collections"`
	// Decks are the study decks with a card of the position, in the order
	// decks were created (the order the decks are listed in); empty, never
	// nil, when none does.
	Decks []DeckRef `json:"decks"`
}

// CollectionRef names a collection without its contents.
type CollectionRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// DeckRef names a study deck without its cards.
type DeckRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ListFieldsStore reads the list fields of a page of positions in a number of
// queries that does not grow with the page.
type ListFieldsStore interface {
	// ListFields answers in the order of ids; an id naming no position of
	// the scope is skipped, a repeated id answered each time.
	ListFields(ctx context.Context, scope string, ids []int64) ([]PositionListFields, error)
}

// ListFieldsProvider is implemented by the Stores of a backend that answers
// list fields (the remote client does not).
type ListFieldsProvider interface {
	PositionListFields() ListFieldsStore
}
