package sqlshared

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// pipBander is implemented by an Execer that indexes the positions in pip order
// within bands of black's farthest checker (PostgreSQL, idx_position_pip_band).
type pipBander interface {
	// PipBand is the band of the row aliased alias: 0 when black has no checker on
	// the board, n + 1 when its farthest stands on point n.
	PipBand(alias string) string
}

// pipBandBands is the last band: a checker on the bar, point 25.
const pipBandBands = 26

// blackFloor is what a structure asks of black's checkers, read off its template.
type blackFloor struct {
	// on holds the points the template puts a black checker on.
	on [pipBandBands]bool
	// top is the farthest of them, 0 for none.
	top int
	// pips is the least pip count a row the masks keep can have: on each point of
	// the template stand at least min(Checkers, 2) black checkers.
	pips int
}

func newBlackFloor(b domain.Board) *blackFloor {
	f := &blackFloor{}
	for i, p := range b.Points {
		if i == 0 || p.Checkers <= 0 || p.Color != domain.Black {
			continue
		}
		f.on[i], f.top = true, i
		// The masks hold one checker (occupancy) or two (made point) on a point, never
		// three: a floor counting the template's third checker would drop rows the
		// read without bands keeps, and the count that offsets its tail would no
		// longer describe what was read. The Go phase checks the rest.
		f.pips += min(p.Checkers, 2) * i
	}
	if f.top == 0 {
		return nil
	}
	return f
}

// band is the least pip count of a position of band b the structure keeps: its
// farthest checker adds its point unless the template already counts one there.
func (f *blackFloor) band(b int) int {
	if n := b - 1; !f.on[n] {
		return f.pips + n
	}
	return f.pips
}

// pipBandSpan is how far above the least pip count the first read looks; each read
// that comes short doubles it, the last has no ceiling.
const pipBandSpan, pipBandReads = 16, 5

// pipBands: atomic so a test can compare the reads with and without (SetPipBands).
var pipBands atomic.Bool

func init() { pipBands.Store(true) }

// SetPipBands turns the read of a structure in pip order band by band on or off and
// returns a func restoring the previous state. For tests only.
func SetPipBands(on bool) (restore func()) {
	old := pipBands.Swap(on)
	return func() { pipBands.Store(old) }
}

// pipBandWindow reads a window of a structure in ascending pips, band by band.
//
// Read in pip order across the tenant, a structure meets its first match after
// every bear-off and race position below it: millions of rows tested against
// masks for one page. Its own black checkers bound both what bands it can keep
// (a checker on point n puts the farthest one at n or beyond) and how few pips
// each band can have. Each band is read in pip order from that floor, and
// merged; every band must then give its first row, and the one whose matches
// start late would be read to them: a ceiling on the pips stops each band, and
// rises — doubling its span above the floor — until the window is full, the last
// read without it. A row under the ceiling is ranked against every row under
// it, so a full window is the answer.
//
// Each band's read keeps to the leader: the masks have no statistics, the
// planner expects to read the whole band and starts workers for every one of
// them, which costs more than the reads. pg_backend_pid() is parallel
// restricted and true: a scan filtered by it runs in no worker.
func (s *SearchStore) pipBandWindow(ctx context.Context, f *blackFloor, from, filter string, args []any, limit, offset int, keys map[int64]any) (ids []int64, read bool, err error) {
	b, ok := s.DB.(pipBander)
	if f == nil || !ok || !pipBands.Load() || limit <= 0 {
		return nil, false, nil
	}
	want := limit + offset
	band := b.PipBand("p")
	floor := f.band(f.top + 1)
	for i := f.top + 2; i <= pipBandBands; i++ {
		floor = min(floor, f.band(i))
	}
	span := pipBandSpan
	for r := 1; ; r++ {
		last := r == pipBandReads
		var q strings.Builder
		var qArgs []any
		q.WriteString("(")
		for i := f.top + 1; i <= pipBandBands; i++ {
			if i > f.top+1 {
				q.WriteString(" UNION ALL ")
			}
			q.WriteString("(SELECT p.id, p.pip_1 FROM " + from + " WHERE " + filter + " AND pg_backend_pid() <> 0 AND " +
				band + " = " + strconv.Itoa(i) + " AND p.pip_1 >= " + strconv.Itoa(f.band(i)))
			if !last {
				q.WriteString(" AND p.pip_1 <= " + strconv.Itoa(floor+span))
			}
			q.WriteString(" ORDER BY p.pip_1, p.id LIMIT " + strconv.Itoa(want) + ")")
			qArgs = append(qArgs, args...)
		}
		q.WriteString(") p")
		got := map[int64]any{}
		ids, err = s.readIDs(ctx, q.String(), "1=1", "p.pip_1 ASC, p.id", "p.pip_1", qArgs, limit, offset, got)
		if err != nil {
			return nil, true, err
		}
		if last || len(ids) == limit {
			for id, k := range got {
				if keys != nil {
					keys[id] = k
				}
			}
			return ids, true, nil
		}
		span *= 2
	}
}
