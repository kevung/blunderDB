// Package statsequal compares two statistics results the way the paths that
// compute them can agree: exactly, save the last bit of a float sum.
package statsequal

import (
	"encoding/json"
	"math"
	"strings"
)

// JSON compares two marshalled statistics results field by field:
// exactly, save the MWC sums (any field whose name starts with MWC, and
// everything under it), which add the same float64 losses in another order
// — per decision by date, or per match cell — and so may part at the last
// bit; the direct order is not total either.
func JSON(a, b string) (bool, error) {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false, err
	}
	return sameStatsValue(va, vb, false), nil
}

func sameStatsValue(a, b any, mwc bool) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			if !sameStatsValue(v, y[k], mwc || strings.HasPrefix(k, "MWC")) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !sameStatsValue(x[i], y[i], mwc) {
				return false
			}
		}
		return true
	case float64:
		y, ok := b.(float64)
		if !ok {
			return false
		}
		if mwc {
			return math.Abs(x-y) <= 1e-12*math.Max(1, math.Abs(x))
		}
		return x == y
	}
	return a == b
}
