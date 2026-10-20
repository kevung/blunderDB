package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// "Elsewhere" (ADR-0103) answers, for one board named by its Zobrist hash,
// what other libraries know of it: their analysis, their comments, the matches
// where it was played and how, and their boards nearest to it. The libraries
// are Sources: the tenants of one Storage for the serve daemon (a read set,
// ADR-0063), separate files for the CLI. Both go through ReadSources, the
// fan-out ReadAcross runs on, so a library is read under the same per-source
// tenant context whichever mode asks. Nothing is written anywhere.

// Source is one library a read spans: a Storage and the tenant read in it,
// under the name the caller shows for it (a tenant, a file path).
type Source struct {
	Name    string
	Storage Storage
	Scope   string
}

// SourcesOf names each tenant of tenants as a Source of st.
func SourcesOf(st Storage, tenants ReadTenants) []Source {
	out := make([]Source, len(tenants))
	for i, scope := range tenants {
		out[i] = Source{Name: scope, Storage: st, Scope: scope}
	}
	return out
}

// ReadSources calls read once per source, in order, under a context scoped to
// that source's tenant, and tags each answer with the source's name. The
// first error stops the read and is returned alone: a partial answer would
// pass for the whole set.
func ReadSources[T any](ctx context.Context, sources []Source, read func(ctx context.Context, src Source) (T, error)) ([]Tagged[T], error) {
	if len(sources) > MaxReadTenants {
		return nil, fmt.Errorf("%w: %d sources, at most %d", ErrInvalid, len(sources), MaxReadTenants)
	}
	out := make([]Tagged[T], 0, len(sources))
	for _, src := range sources {
		tctx, err := tenantContext(ctx, src.Scope)
		if err != nil {
			return nil, err
		}
		item, err := read(tctx, src)
		if err != nil {
			return nil, err
		}
		out = append(out, Tagged[T]{Tenant: src.Name, Item: item})
	}
	return out, nil
}

// ZobristOf is the hash a position is stored under (SavePosition), which
// names the same board in every library.
func ZobristOf(p *domain.Position) uint64 {
	if p == nil {
		return 0
	}
	norm := p.NormalizeForStorage()
	return engine.ZobristHash(&norm)
}

// Bounds of one Elsewhere read, per source.
const (
	ElsewhereDefaultNeighbours = 5
	ElsewhereMaxNeighbours     = 50
	ElsewhereMaxMatches        = 50
)

// ElsewhereQuery names the board asked about. Target, when set, is the board
// itself and Zobrist is derived from it; without it a source ranks neighbours
// only around its own copy of the board.
type ElsewhereQuery struct {
	Zobrist uint64           `json:"zobrist"`
	Target  *domain.Position `json:"position,omitempty"`
	// Neighbours is how many nearest boards each source returns (0: the
	// default, negative: none); MaxDistance drops those farther than this many
	// checker-pips (0: no ceiling).
	Neighbours  int `json:"neighbours"`
	MaxDistance int `json:"maxDistance"`
}

// Judge is the library a reader judges with: its thresholds, and the digest of
// its current match equity table ("" for the built-in one). Another library's
// errors are classified with these, never with that library's own.
type Judge struct {
	Settings  LibrarySettings `json:"settings"`
	METDigest string          `json:"metDigest"`
}

// JudgeOf reads the judge of scope in s.
func JudgeOf(ctx context.Context, s Storage, scope string) (Judge, error) {
	settings, err := s.LibrarySettings().Load(ctx, scope)
	if err != nil {
		return Judge{}, err
	}
	cur, err := s.MatchEquityTables().Current(ctx, scope)
	if err != nil {
		return Judge{}, err
	}
	j := Judge{Settings: settings}
	if cur != nil {
		j.METDigest = cur.Digest
	}
	return j, nil
}

// Verdicts of a play read elsewhere, under the reader's thresholds.
const (
	VerdictGood     = "good"
	VerdictError    = "error"
	VerdictBlunder  = "blunder"
	VerdictUnscored = "unscored"
	// VerdictOtherMET is a match-play cost valued with another table than the
	// reader's: shown, never compared (ADR-0068).
	VerdictOtherMET = "other-met"
)

// Verdict classifies a cost in millipoints with j's thresholds.
func (j Judge) Verdict(errorMP *int32) string {
	switch {
	case errorMP == nil:
		return VerdictUnscored
	case int(*errorMP) >= j.Settings.BlunderThresholdMP:
		return VerdictBlunder
	case int(*errorMP) >= j.Settings.ErrorThresholdMP:
		return VerdictError
	}
	return VerdictGood
}

// ElsewherePlay is one time the board was played in a source's match.
type ElsewherePlay struct {
	MatchID     int64    `json:"matchId"`
	Player1     string   `json:"player1"`
	Player2     string   `json:"player2"`
	Event       string   `json:"event"`
	MatchLength int32    `json:"matchLength"`
	MatchDate   string   `json:"matchDate,omitempty"`
	MoveNumber  int32    `json:"moveNumber"`
	Player      int32    `json:"player"`
	MoveType    string   `json:"moveType"`
	Played      string   `json:"played"`
	ErrorMP     *int32   `json:"errorMP,omitempty"`
	Verdict     string   `json:"verdict"`
	Dice        [2]int32 `json:"dice"`
}

// ElsewhereNeighbour is one of a source's boards near the one asked about.
type ElsewhereNeighbour struct {
	Zobrist  uint64          `json:"zobrist"`
	Distance int             `json:"distance"`
	Position domain.Position `json:"position"`
}

// ElsewhereReport is what one source knows of the board. PositionID is 0 when
// it does not hold the board; its neighbours are listed all the same.
type ElsewhereReport struct {
	PositionID int64                    `json:"positionId"`
	Analysis   *domain.PositionAnalysis `json:"analysis,omitempty"`
	// METName names the table the analysis was valued with ("" for the
	// built-in one); OtherMET says it is not the reader's, at a match score.
	METName  string                 `json:"metName,omitempty"`
	OtherMET bool                   `json:"otherMet"`
	Comments []*domain.CommentEntry `json:"comments"`
	Plays    []ElsewherePlay        `json:"plays"`
	// MatchesTotal counts the matches that played the board; Plays reads
	// the ElsewhereMaxMatches most recently imported of them, and
	// PlaysTruncated says when that left some out.
	MatchesTotal   int                  `json:"matchesTotal"`
	PlaysTruncated bool                 `json:"playsTruncated"`
	Neighbours     []ElsewhereNeighbour `json:"neighbours"`
}

// Elsewhere reads, from every source in order, what it knows of the board q
// names, judged with judge. A source that holds nothing of it still has a
// report, empty but for its neighbours.
func Elsewhere(ctx context.Context, sources []Source, q ElsewhereQuery, judge Judge) ([]Tagged[ElsewhereReport], error) {
	if q.Target != nil {
		q.Zobrist = ZobristOf(q.Target)
	}
	if q.Zobrist == 0 {
		return nil, fmt.Errorf("%w: elsewhere needs a zobrist hash or a position", ErrInvalid)
	}
	if q.MaxDistance < 0 {
		return nil, fmt.Errorf("%w: maxDistance %d is negative", ErrInvalid, q.MaxDistance)
	}
	switch {
	case q.Neighbours == 0:
		q.Neighbours = ElsewhereDefaultNeighbours
	case q.Neighbours > ElsewhereMaxNeighbours:
		return nil, fmt.Errorf("%w: %d neighbours, at most %d", ErrInvalid, q.Neighbours, ElsewhereMaxNeighbours)
	}
	return ReadSources(ctx, sources, func(ctx context.Context, src Source) (ElsewhereReport, error) {
		return elsewhereIn(ctx, src.Storage, src.Scope, q, judge)
	})
}

func elsewhereIn(ctx context.Context, s Storage, scope string, q ElsewhereQuery, judge Judge) (ElsewhereReport, error) {
	rep := ElsewhereReport{Comments: []*domain.CommentEntry{}, Plays: []ElsewherePlay{}, Neighbours: []ElsewhereNeighbour{}}
	id, found, err := s.Positions().Exists(ctx, scope, q.Zobrist)
	if err != nil {
		return rep, err
	}
	var held *domain.Position
	if found {
		rep.PositionID = id
		if held, err = s.Positions().Load(ctx, scope, id); err != nil {
			return rep, err
		}
		if err := elsewhereKnown(ctx, s, scope, held, judge, &rep); err != nil {
			return rep, err
		}
	}
	if q.Neighbours < 0 {
		return rep, nil
	}
	f := domain.SearchFilters{LikeFilter: true, LikeMaxDistance: q.MaxDistance}
	switch {
	case held != nil:
		// Its own copy, so the plies of its matches around the board are
		// left out as for any ranking there.
		f.LikeTargetID = id
	case q.Target != nil:
		// An id of the asking library names nothing here.
		f.LikeTargetBoard = *q.Target
		f.LikeTargetBoard.ID = 0
	default:
		return rep, nil
	}
	// One more than asked: the board itself ranks first when held.
	ranked, err := s.Search().Rank(ctx, scope, f, ListOpts{Limit: q.Neighbours + 1})
	if err != nil {
		return rep, err
	}
	for _, n := range ranked {
		z := ZobristOf(&n.Position)
		if z == q.Zobrist || len(rep.Neighbours) == q.Neighbours {
			continue
		}
		rep.Neighbours = append(rep.Neighbours, ElsewhereNeighbour{Zobrist: z, Distance: n.Distance, Position: n.Position})
	}
	return rep, nil
}

// elsewhereKnown fills what scope holds on its copy of the board.
func elsewhereKnown(ctx context.Context, s Storage, scope string, held *domain.Position, judge Judge, rep *ElsewhereReport) error {
	ana, err := s.Analyses().Load(ctx, scope, held.ID)
	switch {
	case errors.Is(err, ErrNotFound):
	case err != nil:
		return err
	default:
		rep.Analysis = ana
		digest, name, err := analysisTable(ctx, s, scope, held.ID)
		if err != nil {
			return err
		}
		rep.METName = name
		rep.OtherMET = !held.IsMoney() && digest != judge.METDigest
	}

	comments, err := s.Comments().ByPositions(ctx, scope, []int64{held.ID})
	if err != nil {
		return err
	}
	if c := comments[held.ID]; c != nil {
		rep.Comments = c
	}

	matchIDs, err := s.Matches().MatchesByPosition(ctx, scope, held.ID)
	if err != nil {
		return err
	}
	rep.MatchesTotal = len(matchIDs)
	if len(matchIDs) > ElsewhereMaxMatches {
		// Ids ascend with imports: the last ones are the latest imported,
		// read without loading every header of a board played thousands
		// of times.
		matchIDs = matchIDs[len(matchIDs)-ElsewhereMaxMatches:]
		rep.PlaysTruncated = true
	}
	money := held.IsMoney()
	for _, mid := range matchIDs {
		m, err := s.Matches().Get(ctx, scope, mid)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		// The match read scores each play from the analysis; the read by
		// position does not.
		for mv, err := range s.Matches().MovesByMatch(ctx, scope, mid) {
			if err != nil {
				return err
			}
			if mv.PositionID != held.ID {
				continue
			}
			play := ElsewherePlay{
				MatchID: m.ID, Player1: m.Player1Name, Player2: m.Player2Name, Event: m.Event,
				MatchLength: m.MatchLength, MoveNumber: mv.MoveNumber, Player: mv.Player,
				MoveType: mv.MoveType, Played: mv.CheckerMove, ErrorMP: mv.ErrorMP, Dice: mv.Dice,
				Verdict: judge.Verdict(mv.ErrorMP),
			}
			if mv.MoveType == "cube" {
				play.Played = mv.CubeAction
			}
			if !m.MatchDate.IsZero() {
				play.MatchDate = m.MatchDate.Format("2006-01-02")
			}
			if rep.OtherMET && !money && mv.ErrorMP != nil {
				play.Verdict = VerdictOtherMET
			}
			rep.Plays = append(rep.Plays, play)
		}
	}
	return nil
}

// analysisTable returns the digest and name of the table positionID's
// analysis was valued with, both "" for the built-in one.
func analysisTable(ctx context.Context, s Storage, scope string, positionID int64) (digest, name string, err error) {
	metID, err := s.MatchEquityTables().OfAnalysis(ctx, scope, positionID)
	if err != nil || metID == 0 {
		return "", "", err
	}
	t, err := s.MatchEquityTables().Load(ctx, scope, metID)
	if err != nil {
		return "", "", err
	}
	return t.Digest, t.Name, nil
}
