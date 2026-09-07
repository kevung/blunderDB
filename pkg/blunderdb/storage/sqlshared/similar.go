package sqlshared

import (
	"container/heap"
	"context"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// « Des positions comme celle-ci » (#293, fiche J.3 ; révisé par l'ADR-0043),
// côté stockage.
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
//
// # La classe se dit en SQL, et c'est ce qui garde le scan linéaire
//
// L'ADR-0043 range la classe d'équivalence — même type de décision, même
// régime pour un videau, un autre match — DANS la requête plutôt qu'après le
// calcul de la distance. Deux raisons : le tas borné ne doit pas se remplir de
// candidats qu'on jettera, et surtout le décodage de `state` est le vrai coût
// du balayage. Filtrer sur des colonnes indexables avant de décoder, c'est
// payer la classe une fois par ligne au lieu d'une fois par vecteur.

// Similar returns the neighbours of target inside opts' class, nearest first,
// excluding target itself.
func Similar(ctx context.Context, db Execer, scope string, target *domain.Position, opts storage.SimilarOptions) ([]storage.SimilarPosition, error) {
	if target == nil || opts.Limit <= 0 {
		return nil, nil
	}
	tenant, args := db.TenantFilter("p", scope)
	wanted := engine.BuildSimilarityVector(target)

	var where strings.Builder
	where.WriteString(tenant)
	if opts.DecisionType != nil {
		where.WriteString(" AND p.decision_type = ?")
		args = append(args, *opts.DecisionType)
	}
	if opts.Money != nil {
		// The same test as domain.Position.IsMoney, in SQL: both away scores
		// at the -1 sentinel. similar_regime_test.go holds the two forms
		// together, because a regime that means one thing in Go and another in
		// SQL would split a class in silence.
		if *opts.Money {
			where.WriteString(" AND p.score_1 < 0 AND p.score_2 < 0")
		} else {
			where.WriteString(" AND (p.score_1 >= 0 OR p.score_2 >= 0)")
		}
	}
	// "Another match": the plies around the target are its closest structures
	// and never its neighbours. Resolved from the target rather than asked of
	// the caller — it is part of the class, not a choice (ADR-0043 rule 1).
	excluded, err := MatchesOfPosition(ctx, db, scope, target.ID)
	if err != nil {
		return nil, err
	}
	if len(excluded) > 0 {
		// Said once per row against move(position_id), which is indexed. The
		// list is the matches the TARGET was met in — one or two in practice —
		// so the IN stays tiny whatever the library holds.
		where.WriteString(` AND NOT EXISTS (SELECT 1 FROM move mv JOIN game g ON g.id = mv.game_id
			WHERE mv.position_id = p.id AND g.match_id IN (` + Placeholders(len(excluded)) + `))`)
		for _, id := range excluded {
			args = append(args, id)
		}
	}

	rows, err := db.Query(ctx,
		`SELECT p.id, p.state, p.player_on_roll FROM position p WHERE `+where.String()+` ORDER BY p.id`, args...)
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
		// The ceiling drops a neighbour outright rather than ranking it last:
		// a ranking that finds nothing close comes back empty, and says so,
		// instead of handing over the least distant of the unrelated.
		if opts.MaxDistance > 0 && d > opts.MaxDistance {
			continue
		}
		if h.Len() < opts.Limit {
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

// MatchesOfPosition lists the matches a stored position was met in, which is
// what SimilarOptions.ExcludeMatchIDs wants.
//
// Plural, and that is the point: positions are deduplicated by Zobrist hash,
// so one row is reached by the moves of every match that played through it.
// The plies around it are equally uninformative in each of them.
//
// A position that belongs to no match — imported on its own, or a board the
// user has merely drawn — returns nothing, and then nothing is excluded.
func MatchesOfPosition(ctx context.Context, db Execer, scope string, positionID int64) ([]int64, error) {
	if positionID <= 0 {
		return nil, nil
	}
	tenant, args := db.TenantFilter("mv", scope)
	args = append(args, positionID)
	rows, err := db.Query(ctx,
		`SELECT DISTINCT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id
		 WHERE `+tenant+` AND mv.position_id = ?`, args...)
	if err != nil {
		return nil, errf(db, "list the matches a position was met in", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id *int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(db, "list the matches a position was met in", err)
		}
		if id != nil {
			out = append(out, *id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errf(db, "list the matches a position was met in", err)
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
