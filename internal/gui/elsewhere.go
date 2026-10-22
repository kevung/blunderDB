package gui

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ElsewhereGroupReport is what one waiting library group knows of a board:
// storage.ElsewhereReport's fields, named one by one rather than embedded, so
// that the binding generator emits every type they use.
type ElsewhereGroupReport struct {
	Group          int                          `json:"group"`
	Path           string                       `json:"path"`
	PositionID     int64                        `json:"positionId"`
	Analysis       *domain.PositionAnalysis     `json:"analysis,omitempty"`
	METName        string                       `json:"metName,omitempty"`
	OtherMET       bool                         `json:"otherMet"`
	Comments       []*domain.CommentEntry       `json:"comments"`
	Plays          []storage.ElsewherePlay      `json:"plays"`
	MatchesTotal   int                          `json:"matchesTotal"`
	PlaysTruncated bool                         `json:"playsTruncated"`
	Neighbours     []storage.ElsewhereNeighbour `json:"neighbours"`
}

func groupReport(group int, path string, r storage.ElsewhereReport) ElsewhereGroupReport {
	return ElsewhereGroupReport{
		Group: group, Path: path, PositionID: r.PositionID, Analysis: r.Analysis,
		METName: r.METName, OtherMET: r.OtherMET, Comments: r.Comments, Plays: r.Plays,
		MatchesTotal: r.MatchesTotal, PlaysTruncated: r.PlaysTruncated, Neighbours: r.Neighbours,
	}
}

// ElsewhereView answers the desktop's "Elsewhere" view: the board's hash, the
// judge it was read with (the active library's thresholds and table) and one
// report per other open group, in group order.
type ElsewhereView struct {
	Zobrist uint64                 `json:"zobrist"`
	Judge   storage.Judge          `json:"judge"`
	Groups  []ElsewhereGroupReport `json:"groups"`
}

// ElsewhereInGroups reads, from every library group but the active one, what
// it knows of pos (ADR-0101, ADR-0103): the same storage.Elsewhere the CLI and
// the daemon run, over the Storage of each waiting handle. Neighbours is how
// many nearest boards each group returns (0: the service default, negative:
// none). The group lock is held throughout, so no group closes or swaps its
// handle under the read; nothing is written anywhere. A job writing a waiting
// group meanwhile does not block it: the file is in WAL mode and the read takes
// its own pooled connection, so each statement sees the last committed state
// (the report's parts may straddle a commit, never a half-written one).
func (a *App) ElsewhereInGroups(pos database.Position, neighbours int) (*ElsewhereView, error) {
	if a.db == nil {
		return nil, errors.New("no database handle")
	}
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	here, _ := database.CurrentStore(a.db)
	if here == nil {
		return nil, errors.New("no database open")
	}
	ctx := context.Background()
	judge, err := storage.JudgeOf(ctx, here, "")
	if err != nil {
		return nil, err
	}
	sources, paths := a.parkedSourcesLocked()
	// The hash names the board in every library; an id of the active one
	// would name another row elsewhere.
	target := pos
	target.ID = 0
	view := &ElsewhereView{Zobrist: storage.ZobristOf(&target), Judge: judge, Groups: []ElsewhereGroupReport{}}
	if len(sources) == 0 {
		return view, nil
	}
	found, err := storage.Elsewhere(ctx, sources, storage.ElsewhereQuery{Target: &target, Neighbours: neighbours}, judge)
	if err != nil {
		return nil, err
	}
	for _, r := range found {
		id, _ := strconv.Atoi(r.Tenant)
		view.Groups = append(view.Groups, groupReport(id, paths[r.Tenant], r.Item))
	}
	return view, nil
}

// parkedSourcesLocked names every waiting library group as a Source, in group
// order, with each one's file path. The caller holds a.libs.mu.
func (a *App) parkedSourcesLocked() ([]storage.Source, map[string]string) {
	ids := make([]int, 0, len(a.libs.parked))
	for id := range a.libs.parked {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	paths := map[string]string{}
	sources := make([]storage.Source, 0, len(ids))
	for _, id := range ids {
		d := a.libs.parked[id]
		st, _ := database.CurrentStore(d)
		if st == nil {
			continue
		}
		name := strconv.Itoa(id)
		paths[name] = database.OpenPath(d)
		sources = append(sources, storage.Source{Name: name, Storage: st})
	}
	return sources, paths
}

// ElsewhereObjectGroup is whether one waiting library group holds an object
// (storage.ObjectPresence, its fields named for the binding generator): ID 0
// when it does not.
type ElsewhereObjectGroup struct {
	Group int    `json:"group"`
	Path  string `json:"path"`
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ElsewhereObjectInGroups reads, from every library group but the active
// one, whether it holds object id of kind ("match", "tournament",
// "collection", "lesson") of the active library: storage.ElsewhereObject, as
// the CLI's elsewhere and the daemon's across.objectElsewhere run it, under
// the key a copy merges the object on. Nothing is written anywhere.
func (a *App) ElsewhereObjectInGroups(kind string, id int64) ([]ElsewhereObjectGroup, error) {
	if a.db == nil {
		return nil, errors.New("no database handle")
	}
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	here, _ := database.CurrentStore(a.db)
	if here == nil {
		return nil, errors.New("no database open")
	}
	ctx := context.Background()
	q, err := storage.ObjectKey(ctx, here, "", kind, id)
	if err != nil {
		return nil, err
	}
	out := []ElsewhereObjectGroup{}
	sources, paths := a.parkedSourcesLocked()
	if len(sources) == 0 {
		return out, nil
	}
	found, err := storage.ElsewhereObject(ctx, sources, q)
	if err != nil {
		return nil, err
	}
	for _, r := range found {
		g, _ := strconv.Atoi(r.Tenant)
		out = append(out, ElsewhereObjectGroup{Group: g, Path: paths[r.Tenant], ID: r.Item.ID, Name: r.Item.Name, Count: r.Item.Count})
	}
	return out, nil
}
