package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/openingbook"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The exercise « which play at this score? » over HTTP: the book is data
// embedded in the binary, not a tenant's rows, so every tenant draws from and
// is judged against the same book. A client records its sessions through
// /v1/training.save like any other exercise.

type openingBookStatusReq struct{}

func (s *Server) openingBook() (*openingbook.Book, error) {
	if s.opts.OpeningBook != nil {
		return s.opts.OpeningBook, nil
	}
	return openingbook.Default()
}

func (s *Server) openingBookRoutes() []route {
	invalid := func(err error) error {
		if err == nil {
			return nil
		}
		return fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	return []route{
		{http.MethodPost, "/v1/openingbook.status", rpc(func(_ context.Context, _ string, _ openingBookStatusReq) (openingbook.Status, error) {
			b, err := s.openingBook()
			if err != nil {
				return openingbook.Status{}, err
			}
			st := b.Status()
			if st.Scores == nil {
				st.Scores = []string{}
			}
			return st, nil
		})},
		{http.MethodPost, "/v1/openingbook.question", rpc(func(_ context.Context, _ string, req openingbook.QuestionRequest) (openingbook.Question, error) {
			b, err := s.openingBook()
			if err != nil {
				return openingbook.Question{}, err
			}
			q, err := b.Draw(req)
			return q, invalid(err)
		})},
		{http.MethodPost, "/v1/openingbook.grade", rpc(func(_ context.Context, _ string, req openingbook.Answer) (openingbook.Verdict, error) {
			b, err := s.openingBook()
			if err != nil {
				return openingbook.Verdict{}, err
			}
			v, err := b.Grade(req)
			return v, invalid(err)
		})},
	}
}
