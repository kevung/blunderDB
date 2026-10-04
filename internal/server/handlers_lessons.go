package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type lessonCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type lessonUpdateReq struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// lessonStepReq is a step to append (LessonID) or to rewrite (ID). A zero
// collectionId or positionId means the step shows none.
type lessonStepReq struct {
	ID           int64  `json:"id"`
	LessonID     int64  `json:"lessonId"`
	Title        string `json:"title"`
	Text         string `json:"text"`
	CollectionID int64  `json:"collectionId"`
	PositionID   int64  `json:"positionId"`
}

// lessonStepDoneReq marks the step ID done, or withdraws the mark.
type lessonStepDoneReq struct {
	ID   int64 `json:"id"`
	Done bool  `json:"done"`
}

type lessonReorderReq struct {
	LessonID int64   `json:"lessonId"`
	StepIDs  []int64 `json:"stepIds"`
}

// lessonRoutes serve the Lessons (ADR-0066): read by any client, written by
// the coach's own tools through the same store as the desktop and the CLI.
func (s *Server) lessonRoutes() []route {
	ls := func() storage.LessonStore { return s.opts.Storage.Lessons() }
	return []route{
		{http.MethodPost, "/v1/lessons.list", rpc(func(ctx context.Context, scope string, _ struct{}) ([]*domain.Lesson, error) {
			return ls().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/lessons.get", rpc(func(ctx context.Context, scope string, req idReq) (*domain.Lesson, error) {
			return ls().Get(ctx, scope, req.ID)
		})},
		// Idempotent: Create has no natural dedup key, so a retry would make a
		// second, identically named Lesson.
		{http.MethodPost, "/v1/lessons.create", s.withIdempotency(rpc(func(ctx context.Context, scope string, req lessonCreateReq) (idResp, error) {
			id, err := ls().Create(ctx, scope, req.Name, req.Description)
			return idResp{ID: id}, err
		}))},
		{http.MethodPost, "/v1/lessons.update", rpcVoid(func(ctx context.Context, scope string, req lessonUpdateReq) error {
			return ls().Update(ctx, scope, req.ID, req.Name, req.Description)
		})},
		{http.MethodPost, "/v1/lessons.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ls().Delete(ctx, scope, req.ID)
		})},
		// Idempotent for the same reason as create: a retried append would
		// add the step twice.
		{http.MethodPost, "/v1/lessons.addStep", s.withIdempotency(rpc(func(ctx context.Context, scope string, req lessonStepReq) (idResp, error) {
			id, err := ls().AddStep(ctx, scope, req.LessonID, domain.LessonStep{
				Title: req.Title, Text: req.Text, CollectionID: req.CollectionID, PositionID: req.PositionID,
			})
			return idResp{ID: id}, err
		}))},
		{http.MethodPost, "/v1/lessons.updateStep", rpcVoid(func(ctx context.Context, scope string, req lessonStepReq) error {
			return ls().UpdateStep(ctx, scope, domain.LessonStep{
				ID: req.ID, Title: req.Title, Text: req.Text, CollectionID: req.CollectionID, PositionID: req.PositionID,
			})
		})},
		{http.MethodPost, "/v1/lessons.removeStep", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ls().RemoveStep(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/lessons.reorderSteps", rpcVoid(func(ctx context.Context, scope string, req lessonReorderReq) error {
			return ls().ReorderSteps(ctx, scope, req.LessonID, req.StepIDs)
		})},
		// The reader's progress (ADR-0069): written only by this explicit
		// gesture, in the reader's own tenant; get and list write nothing.
		{http.MethodPost, "/v1/lessons.setStepDone", rpcVoid(func(ctx context.Context, scope string, req lessonStepDoneReq) error {
			return ls().SetStepDone(ctx, scope, req.ID, req.Done)
		})},
		{http.MethodPost, "/v1/lessons.doneSteps", rpc(func(ctx context.Context, scope string, req idReq) (map[int64]string, error) {
			return ls().DoneSteps(ctx, scope, req.ID)
		})},
	}
}
