package remote

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func (s lessonStore) Create(ctx context.Context, scope string, name, description string) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "lessons.create", nameDescReq{Name: name, Description: description}, &r)
	return r.ID, err
}

func (s lessonStore) Get(ctx context.Context, scope string, id int64) (*domain.Lesson, error) {
	var l *domain.Lesson
	if err := s.c.call(ctx, scope, "lessons.get", idReq{id}, &l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s lessonStore) List(ctx context.Context, scope string) ([]*domain.Lesson, error) {
	var ls []*domain.Lesson
	err := s.c.call(ctx, scope, "lessons.list", nil, &ls)
	return ls, err
}

func (s lessonStore) Update(ctx context.Context, scope string, id int64, name, description string) error {
	return s.c.call(ctx, scope, "lessons.update", nameDescReq{id, name, description}, nil)
}

func (s lessonStore) Delete(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "lessons.delete", idReq{id}, nil)
}

type lessonStepReq struct {
	ID           int64  `json:"id,omitempty"`
	LessonID     int64  `json:"lessonId,omitempty"`
	Title        string `json:"title"`
	Text         string `json:"text"`
	CollectionID int64  `json:"collectionId"`
	PositionID   int64  `json:"positionId"`
}

func (s lessonStore) AddStep(ctx context.Context, scope string, lessonID int64, step domain.LessonStep) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "lessons.addStep", lessonStepReq{
		LessonID: lessonID, Title: step.Title, Text: step.Text, CollectionID: step.CollectionID, PositionID: step.PositionID,
	}, &r)
	return r.ID, err
}

func (s lessonStore) UpdateStep(ctx context.Context, scope string, step domain.LessonStep) error {
	return s.c.call(ctx, scope, "lessons.updateStep", lessonStepReq{
		ID: step.ID, Title: step.Title, Text: step.Text, CollectionID: step.CollectionID, PositionID: step.PositionID,
	}, nil)
}

func (s lessonStore) RemoveStep(ctx context.Context, scope string, stepID int64) error {
	return s.c.call(ctx, scope, "lessons.removeStep", idReq{stepID}, nil)
}

func (s lessonStore) ReorderSteps(ctx context.Context, scope string, lessonID int64, stepIDs []int64) error {
	return s.c.call(ctx, scope, "lessons.reorderSteps", struct {
		LessonID int64   `json:"lessonId"`
		StepIDs  []int64 `json:"stepIds"`
	}{lessonID, stepIDs}, nil)
}

func (s lessonStore) SetStepDone(ctx context.Context, scope string, stepID int64, done bool) error {
	return s.c.call(ctx, scope, "lessons.setStepDone", struct {
		ID   int64 `json:"id"`
		Done bool  `json:"done"`
	}{stepID, done}, nil)
}

func (s lessonStore) DoneSteps(ctx context.Context, scope string, lessonID int64) (map[int64]string, error) {
	var m map[int64]string
	err := s.c.call(ctx, scope, "lessons.doneSteps", idReq{lessonID}, &m)
	return m, err
}

func (s librarySettingsStore) Load(ctx context.Context, scope string) (storage.LibrarySettings, error) {
	var ls storage.LibrarySettings
	err := s.c.call(ctx, scope, "librarySettings.load", nil, &ls)
	return ls, err
}

func (s librarySettingsStore) Save(ctx context.Context, scope string, settings storage.LibrarySettings) error {
	return s.c.call(ctx, scope, "librarySettings.save", settings, nil)
}

func (s metadataStore) Version(ctx context.Context, scope string) (string, error) {
	var r versionResp
	err := s.c.call(ctx, scope, "metadata.version", nil, &r)
	return r.Version, err
}

func (s metadataStore) Counts(ctx context.Context, scope string) (storage.Counts, error) {
	var c storage.Counts
	err := s.c.call(ctx, scope, "metadata.counts", nil, &c)
	return c, err
}

func (s metadataStore) EstimatedCounts(ctx context.Context, scope string, exactBelow int) (storage.CountsEstimate, error) {
	var c storage.CountsEstimate
	err := s.c.call(ctx, scope, "metadata.countsEstimate", struct {
		ExactBelow int `json:"exact_below"`
	}{exactBelow}, &c)
	return c, err
}

func (s trainingStore) Save(ctx context.Context, scope string, session storage.TrainingSession) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "training.save", session, &r)
	return r.ID, err
}

func (s trainingStore) Sessions(ctx context.Context, scope, exercise string, limit int) ([]storage.TrainingSession, error) {
	var out []storage.TrainingSession
	err := s.c.call(ctx, scope, "training.sessions", struct {
		Exercise string `json:"exercise"`
		Limit    int    `json:"limit"`
	}{exercise, limit}, &out)
	return out, err
}

func (s trainingStore) NumberStats(ctx context.Context, scope, exercise string) ([]storage.TrainingNumberStat, error) {
	var out []storage.TrainingNumberStat
	err := s.c.call(ctx, scope, "training.numberStats", struct {
		Exercise string `json:"exercise"`
	}{exercise}, &out)
	return out, err
}

func (s trainingStore) Missed(ctx context.Context, scope string, filter storage.TrainingMissedFilter) ([]int64, error) {
	var out []int64
	err := s.c.call(ctx, scope, "training.missed", filter, &out)
	return out, err
}
