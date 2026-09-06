package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The tag vocabulary, desktop/CLI side (issue #265, fiche I.9).
//
// A tag is a `#word` inside a comment. Nothing declares one, no table holds
// one, and that is deliberate: the vocabulary is the user's own prose, and
// forcing a declaration before a tag can be used would turn a habit into
// paperwork. What was missing was the other half — being able to SEE the
// vocabulary one has actually built, and to click a tag rather than remember
// how it was spelt.

// Tags returns every tag used in this database with the number of positions
// carrying it, most used first. Delegates to the Storage contract, so the
// desktop, the CLI and the daemon all count the same way.
func (d *Database) Tags() ([]domain.TagCount, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Comments().Tags(context.Background(), "")
}

// RecommendedTags is the vocabulary the comment editor suggests at '#'. It is
// a constant of the domain, exposed here only so the frontend can read it
// through the same binding as everything else rather than keeping a second
// copy of the list.
func (d *Database) RecommendedTags() []string {
	return domain.RecommendedTags
}
