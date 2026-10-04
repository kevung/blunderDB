package ingest

import (
	"log/slog"
	"math"
	"reflect"
)

// dropNonFiniteAnalyses removes from g every analysis fragment holding a NaN
// or an infinity, and returns how many it removed. JSON has no spelling for
// those values: one left in a fragment makes the stored blob unencodable and
// refuses the whole match, where the decision's analysis alone is unusable.
// The move and its position are kept; only that decision goes unanalysed.
func dropNonFiniteAnalyses(g *MatchGraph) int {
	dropped := 0
	for gi := range g.Games {
		for mi := range g.Games[gi].Moves {
			mg := &g.Games[gi].Moves[mi]
			kept := mg.Analyses[:0]
			for _, a := range mg.Analyses {
				if a != nil && hasNonFinite(reflect.ValueOf(a)) {
					dropped++
					slog.Warn("import: analysis left out, it holds a value that is not a finite number",
						"game", g.Games[gi].Game.GameNumber, "move", mg.Move.MoveNumber)
					continue
				}
				kept = append(kept, a)
			}
			clear(mg.Analyses[len(kept):])
			mg.Analyses = kept
		}
	}
	return dropped
}

// hasNonFinite reports whether a float anywhere under v is NaN or infinite.
// It walks the value rather than naming fields, so a field added to an
// analysis later is covered without being listed here.
func hasNonFinite(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		return math.IsNaN(f) || math.IsInf(f, 0)
	case reflect.Pointer, reflect.Interface:
		return !v.IsNil() && hasNonFinite(v.Elem())
	case reflect.Struct:
		for i := range v.NumField() {
			if hasNonFinite(v.Field(i)) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if hasNonFinite(v.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		for it := v.MapRange(); it.Next(); {
			if hasNonFinite(it.Value()) {
				return true
			}
		}
	}
	return false
}
