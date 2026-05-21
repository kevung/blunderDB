package sqlite

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type sessionStore struct{ db execer }

var _ storage.SessionStore = (*sessionStore)(nil)

func (*sessionStore) Save(context.Context, string, storage.SessionState) error {
	return notImpl("Session", "Save")
}
func (*sessionStore) Load(context.Context, string) (*storage.SessionState, error) {
	return nil, notImpl("Session", "Load")
}
func (*sessionStore) Clear(context.Context, string) error {
	return notImpl("Session", "Clear")
}
