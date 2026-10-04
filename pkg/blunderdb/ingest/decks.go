package ingest

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// SourceDeck is an Anki deck of a native .db being imported, with its
// members' source position ids. Its review history does not travel: the
// recipient studies the deck afresh.
type SourceDeck struct {
	Deck        domain.AnkiDeck
	PositionIDs []int64
}

// ReadSourceDecks drains a source's decks, then each one's members.
func ReadSourceDecks(ctx context.Context, source storage.Stores, scope string) ([]SourceDeck, error) {
	var out []SourceDeck
	for d, err := range source.Anki().ListDecks(ctx, scope) {
		if err != nil {
			return nil, fmt.Errorf("ingest: list source decks: %w", err)
		}
		out = append(out, SourceDeck{Deck: *d})
	}
	for i := range out {
		for p, err := range source.Anki().DeckPositions(ctx, scope, out[i].Deck.ID) {
			if err != nil {
				return nil, fmt.Errorf("ingest: read source deck %q: %w", out[i].Deck.Name, err)
			}
			out[i].PositionIDs = append(out[i].PositionIDs, p.ID)
		}
	}
	return out, nil
}

// MergeDecks is the one rule both native .db imports apply to Anki decks,
// after MergeCollections, on the terms of MergeLessons: a source deck whose
// name the target already holds is left alone, so importing the same file
// twice changes nothing and a deck the recipient studies keeps its cards and
// their schedule; any other is created, its members remapped through
// targetOf and its collection through the source collection's name. It
// returns the number of decks created.
func MergeDecks(ctx context.Context, tx storage.Stores, scope string, src []SourceDeck, srcCollections []SourceCollection, targetOf map[int64]int64) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	held := map[string]bool{}
	for d, err := range tx.Anki().ListDecks(ctx, scope) {
		if err != nil {
			return 0, err
		}
		held[d.Name] = true
	}
	collName := make(map[int64]string, len(srcCollections))
	for _, sc := range srcCollections {
		collName[sc.Coll.ID] = sc.Coll.Name
	}
	targetColl := map[string]int64{}
	for c, err := range tx.Collections().List(ctx, scope) {
		if err != nil {
			return 0, err
		}
		targetColl[c.Name] = c.ID
	}
	created := 0
	for _, sd := range src {
		if err := ctx.Err(); err != nil {
			return created, err
		}
		d := sd.Deck
		if held[d.Name] {
			continue
		}
		sourceID, sourceCmd := d.SourceID, d.SourceCommand
		switch d.SourceType {
		case domain.AnkiSourceCollection:
			sourceID = 0 // a collection the file does not carry
			if name, ok := collName[d.SourceID]; ok {
				sourceID = targetColl[name]
			}
		case domain.AnkiSourceSearch:
			sourceCmd = remapIDList(sourceCmd, targetOf)
		}
		id, err := tx.Anki().CreateDeck(ctx, scope, d.Name, d.Description, d.SourceType, sourceID, sourceCmd)
		if err != nil {
			return created, fmt.Errorf("ingest: import deck %q: %w", d.Name, err)
		}
		held[d.Name] = true
		var members []int64
		for _, pid := range sd.PositionIDs {
			if t, ok := targetOf[pid]; ok {
				members = append(members, t)
			}
		}
		if len(members) > 0 {
			if err := tx.Anki().SyncWithPositions(ctx, scope, id, members); err != nil {
				return created, fmt.Errorf("ingest: fill deck %q: %w", d.Name, err)
			}
		}
		// A deck of scores holds no position: Sync states its 36 cards here
		// as on any machine (ADR-0042 rule 2).
		if d.SourceType == domain.AnkiSourceScores {
			if err := tx.Anki().Sync(ctx, scope, id); err != nil {
				return created, fmt.Errorf("ingest: fill deck %q: %w", d.Name, err)
			}
		}
		if err := tx.Anki().UpdateDeckParams(ctx, scope, id, d.RequestRetention, d.MaximumInterval, d.EnableFuzz, d.SessionLimit); err != nil {
			return created, fmt.Errorf("ingest: deck %q parameters: %w", d.Name, err)
		}
		created++
	}
	return created, nil
}
