package postgres

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type analysisStore struct{ db execer }

var _ storage.AnalysisStore = (*analysisStore)(nil)

func (*analysisStore) Save(context.Context, string, int64, *domain.PositionAnalysis) error {
	return notImpl("Analysis", "Save")
}
func (*analysisStore) Load(context.Context, string, int64) (*domain.PositionAnalysis, error) {
	return nil, notImpl("Analysis", "Load")
}
func (*analysisStore) Delete(context.Context, string, int64) error {
	return notImpl("Analysis", "Delete")
}
