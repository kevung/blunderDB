package sqlshared

import (
	"container/heap"
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// « Des positions comme celle-ci » (#293, fiche J.3), côté stockage.
//
// Un BALAYAGE EXHAUSTIF, et c'est une décision, pas un raccourci. Le rapport
// P7 est net : sous ~100 000 vecteurs le scan linéaire donne un rappel PARFAIT
// et rien de plus compliqué ne se justifie — « do not over-engineer a small
// problem ». La dimension ici est 52, pas 768 : l'index approximatif qu'on
// n'écrit pas serait aussi celui qu'on aurait à maintenir cohérent avec chaque
// écriture.
//
// La requête ne lit que `state` et `player_on_roll` : ce sont les deux seules
// colonnes dont la distance dépend. Une base de cent mille positions tient
// donc dans quelques mégaoctets de lecture séquentielle.

// Similar returns the positions closest to target, nearest first, excluding
// the target itself.
func Similar(ctx context.Context, db Execer, scope string, target *domain.Position, limit int) ([]storage.SimilarPosition, error) {
	if target == nil || limit <= 0 {
		return nil, nil
	}
	tenant, targs := db.TenantFilter("", scope)
	wanted := engine.BuildSimilarityVector(target)

	rows, err := db.Query(ctx,
		`SELECT id, state, player_on_roll FROM position WHERE `+tenant+` ORDER BY id`, targs...)
	if err != nil {
		return nil, errf(db, "scan the positions for similarity", err)
	}
	defer rows.Close()

	// A bounded max-heap of the best `limit`: the scan is O(N) and the ranking
	// O(N log k), so a library of a hundred thousand positions costs one pass
	// and a heap of ten entries — not a sort of a hundred thousand.
	h := &farthestFirst{}
	for rows.Next() {
		var id int64
		var state string
		var onRoll *int64
		if err := rows.Scan(&id, &state, &onRoll); err != nil {
			return nil, errf(db, "scan the positions for similarity", err)
		}
		if id == target.ID {
			continue
		}
		p, ok := positionOfState(state)
		if !ok {
			continue
		}
		if onRoll != nil {
			p.PlayerOnRoll = int(*onRoll)
		}
		p.ID = id
		d := engine.SimilarityDistance(wanted, engine.BuildSimilarityVector(&p))
		if h.Len() < limit {
			heap.Push(h, storage.SimilarPosition{Position: p, Distance: d})
			continue
		}
		if d < (*h)[0].Distance {
			(*h)[0] = storage.SimilarPosition{Position: p, Distance: d}
			heap.Fix(h, 0)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errf(db, "scan the positions for similarity", err)
	}

	out := make([]storage.SimilarPosition, h.Len())
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(storage.SimilarPosition)
	}
	return out, nil
}

// farthestFirst is a max-heap on the distance: its root is the worst of the
// candidates kept so far, which is the one a better candidate replaces.
type farthestFirst []storage.SimilarPosition

func (h farthestFirst) Len() int           { return len(h) }
func (h farthestFirst) Less(i, j int) bool { return h[i].Distance > h[j].Distance }
func (h farthestFirst) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *farthestFirst) Push(x any)        { *h = append(*h, x.(storage.SimilarPosition)) }
func (h *farthestFirst) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
