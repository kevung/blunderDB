package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The tag vocabulary, desktop/CLI side. A tag is a `#word` inside a comment;
// deliberately nothing declares one and no table holds one.

// Tags returns every tag used in this database with the number of positions
// carrying it, most used first. Delegates to the Storage contract, so the
// desktop, the CLI and the daemon all count the same way.
func (d *Database) Tags() ([]domain.TagCount, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Comments().Tags(context.Background(), "")
}

// RecommendedTags is the vocabulary the comment editor suggests at '#'. It is
// a domain constant, exposed so the frontend keeps no second copy.
func (d *Database) RecommendedTags() []string {
	return domain.RecommendedTags
}
