package remote

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func (s filterStore) Save(ctx context.Context, scope string, name, command string) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "filters.save", struct {
		Name    string `json:"name"`
		Command string `json:"command"`
	}{name, command}, &r)
	return r.ID, err
}

func (s filterStore) Update(ctx context.Context, scope string, id int64, name, command string) error {
	return s.c.call(ctx, scope, "filters.update", struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Command string `json:"command"`
	}{id, name, command}, nil)
}

func (s filterStore) Delete(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "filters.delete", idReq{id}, nil)
}

func (s filterStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Filter, error] {
	return stream[*storage.Filter](ctx, s.c, scope, "filters.list", nil)
}

func (s filterStore) SetPinned(ctx context.Context, scope string, id int64, pinned bool) error {
	return s.c.call(ctx, scope, "filters.setPinned", struct {
		ID     int64 `json:"id"`
		Pinned bool  `json:"pinned"`
	}{id, pinned}, nil)
}

func (s filterStore) SaveEditPosition(ctx context.Context, scope string, filterName, editPosition string) error {
	return s.c.call(ctx, scope, "filters.saveEditPosition", struct {
		FilterName   string `json:"filterName"`
		EditPosition string `json:"editPosition"`
	}{filterName, editPosition}, nil)
}

type filterNameReq struct {
	FilterName string `json:"filterName"`
}

func (s filterStore) LoadEditPosition(ctx context.Context, scope string, filterName string) (string, error) {
	var r textResp
	err := s.c.call(ctx, scope, "filters.loadEditPosition", filterNameReq{filterName}, &r)
	return r.Text, err
}

func (s filterStore) SaveExcludePosition(ctx context.Context, scope string, filterName, excludePosition string) error {
	return s.c.call(ctx, scope, "filters.saveExcludePosition", struct {
		FilterName      string `json:"filterName"`
		ExcludePosition string `json:"excludePosition"`
	}{filterName, excludePosition}, nil)
}

func (s filterStore) LoadExcludePosition(ctx context.Context, scope string, filterName string) (string, error) {
	var r textResp
	err := s.c.call(ctx, scope, "filters.loadExcludePosition", filterNameReq{filterName}, &r)
	return r.Text, err
}

func (s sessionStore) Save(ctx context.Context, scope string, state storage.SessionState) error {
	return s.c.call(ctx, scope, "session.save", struct {
		State storage.SessionState `json:"state"`
	}{state}, nil)
}

func (s sessionStore) Load(ctx context.Context, scope string) (*storage.SessionState, error) {
	var st *storage.SessionState
	if err := s.c.call(ctx, scope, "session.load", nil, &st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s sessionStore) Clear(ctx context.Context, scope string) error {
	return s.c.call(ctx, scope, "session.clear", nil, nil)
}

func (s searchHistoryStore) Save(ctx context.Context, scope string, command, position, excludePosition string) error {
	return s.c.call(ctx, scope, "searchHistory.save", struct {
		Command         string `json:"command"`
		Position        string `json:"position"`
		ExcludePosition string `json:"excludePosition"`
	}{command, position, excludePosition}, nil)
}

func (s searchHistoryStore) List(ctx context.Context, scope string) iter.Seq2[*storage.SearchHistory, error] {
	return stream[*storage.SearchHistory](ctx, s.c, scope, "searchHistory.list", nil)
}

func (s searchHistoryStore) DeleteEntry(ctx context.Context, scope string, timestamp int64) error {
	return s.c.call(ctx, scope, "searchHistory.deleteEntry", struct {
		Timestamp int64 `json:"timestamp"`
	}{timestamp}, nil)
}

func (s commandHistoryStore) Save(ctx context.Context, scope string, command string) error {
	return s.c.call(ctx, scope, "history.save", struct {
		Command string `json:"command"`
	}{command}, nil)
}

func (s commandHistoryStore) Load(ctx context.Context, scope string) ([]string, error) {
	var r struct {
		Items []string `json:"items"`
	}
	err := s.c.call(ctx, scope, "history.load", nil, &r)
	return r.Items, err
}

func (s commandHistoryStore) Clear(ctx context.Context, scope string) error {
	return s.c.call(ctx, scope, "history.clear", nil, nil)
}
