package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// StudyIDs are the positions to study from the recurring errors of a filter:
// the groups picked by rank (RecurringErrors.StudyPositionIDs), then, when size
// is above zero, a random draw of at most size of them. The one place the GUI,
// the CLI and the daemon agree on which positions "my worst groups" means.
func StudyIDs(ctx context.Context, st Stores, scope string, filter StatsFilter, rank, size int) ([]int64, error) {
	res, err := st.Stats().RecurringErrors(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	ids := res.StudyPositionIDs(rank)
	if size > 0 {
		ids = DrawStudyQuiz(ids, size, nil)
	}
	return ids, nil
}

// CreateStudyDeck makes a search deck of exactly these positions: the ids are
// stored with the deck, as a deck made from a result list.
func CreateStudyDeck(ctx context.Context, st Stores, scope, name string, ids []int64) (int64, error) {
	source, err := json.Marshal(struct {
		IDs []int64 `json:"ids"`
	}{ids})
	if err != nil {
		return 0, fmt.Errorf("marshal deck source: %w", err)
	}
	deckID, err := st.Anki().CreateDeck(ctx, scope, name, "", domain.AnkiSourceSearch, 0, string(source))
	if err != nil {
		return 0, fmt.Errorf("create deck: %w", err)
	}
	if err := st.Anki().SyncWithPositions(ctx, scope, deckID, ids); err != nil {
		return 0, fmt.Errorf("fill deck: %w", err)
	}
	return deckID, nil
}
