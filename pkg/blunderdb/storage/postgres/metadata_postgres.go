package postgres

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type metadataStore struct{ db execer }

var _ storage.MetadataStore = (*metadataStore)(nil)

func (*metadataStore) Version(context.Context, string) (string, error) {
	return "", notImpl("Metadata", "Version")
}
func (*metadataStore) SetVersion(context.Context, string, string) error {
	return notImpl("Metadata", "SetVersion")
}
func (*metadataStore) Load(context.Context, string) (map[string]string, error) {
	return nil, notImpl("Metadata", "Load")
}
func (*metadataStore) Save(context.Context, string, map[string]string) error {
	return notImpl("Metadata", "Save")
}
func (*metadataStore) Counts(context.Context, string) (storage.Counts, error) {
	return storage.Counts{}, notImpl("Metadata", "Counts")
}
