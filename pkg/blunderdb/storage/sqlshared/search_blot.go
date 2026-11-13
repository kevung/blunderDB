package sqlshared

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// BitRange is the mask of the bits lo to hi.
func BitRange(lo, hi int) int64 { return (int64(1)<<(hi+1) - 1) &^ (int64(1)<<lo - 1) }

type blotFilter struct {
	filter, prefix string
	// side is the player (1 or 2) whose masks hold the blots.
	side int
}

func blotFilters(f domain.SearchFilters) [4]blotFilter {
	return [4]blotFilter{
		{f.Player1OutfieldBlotFilter, "bo", 1},
		{f.Player2OutfieldBlotFilter, "BO", 2},
		{f.Player1JanBlotFilter, "bj", 1},
		{f.Player2JanBlotFilter, "BJ", 2},
	}
}

// appendBlotClauses refuses a blot filter that reads as nothing, and, when
// inSQL, writes the others as conditions on the stored masks: a blot is a
// point the side occupies (occupancy_N) and has not made (point_mask_N). The
// bounds are parsed integers, written into the text so that a threshold from
// the floor of idx_position_blot_* up is read from that index.
func appendBlotClauses(d Dialect, f domain.SearchFilters, inSQL bool, where *strings.Builder) error {
	for _, b := range blotFilters(f) {
		if b.filter == "" {
			continue
		}
		lo, hi, hasLo, hasHi, ok := domain.ParseBlotFilter(b.filter, b.prefix)
		if !ok {
			return fmt.Errorf("%w: blot filter %q: write %s>N (N or more), %s<N (N or fewer) or %sA,B (between A and B)",
				storage.ErrInvalid, b.filter, b.prefix, b.prefix, b.prefix)
		}
		if !inSQL {
			continue
		}
		z := domain.BlotZoneOf(b.prefix)
		count := d.BitsSet(fmt.Sprintf("p.occupancy_%d & ~p.point_mask_%d", b.side, b.side), z.Lo, z.Hi)
		if hasLo {
			fmt.Fprintf(where, " AND %s >= %d", count, lo)
		}
		if hasHi {
			fmt.Fprintf(where, " AND %s <= %d", count, hi)
		}
	}
	return nil
}
