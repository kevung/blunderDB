package document

import (
	"errors"
	"strings"
)

// ReviewItem is an entry an import would refuse as it stands, with what a
// person checks against the diagram before accepting it.
type ReviewItem struct {
	Rank      int    `json:"rank"`
	Reference string `json:"reference"`
	// State is the entry's overall certainty; Checkers and Context are its
	// two parts.
	State    State        `json:"state"`
	Checkers State        `json:"checkers"`
	Context  ContextState `json:"context"`
	// Complete is set for fifteen checkers a side read with no doubt.
	Complete bool `json:"complete"`
	// Note names the doubts of the reading, in the reference's own words for
	// an entry whose context is to verify.
	Note string `json:"note"`
}

// Review lists the entries an import refuses without a decision, in the
// document's order, each with its note.
func Review(r *Releve) []ReviewItem {
	r = r.Upgrade()
	var items []ReviewItem
	for _, e := range r.Entries {
		if r.Importable(e, false) {
			continue
		}
		items = append(items, ReviewItem{
			Rank: e.Rank, Reference: e.Reference(), State: r.StateOf(e),
			Checkers: e.Checkers, Context: e.Context, Complete: e.Complete,
			Note: reviewNote(e),
		})
	}
	return items
}

func reviewNote(e Entry) string {
	if e.Context == ContextToVerify {
		return e.FallbackNote()
	}
	parts := make([]string, 0, len(e.Doubts))
	for _, d := range e.Doubts {
		parts = append(parts, d.Field+" ("+d.Reason+")")
	}
	return strings.Join(parts, "; ")
}

// ConfirmDefaults sets the document's defaults, and the direction of play,
// as confirmed by the person who checked them against the diagrams. A
// confirmation of what the Relevé does not carry is refused.
func ConfirmDefaults(r *Releve, defaults, direction bool) error {
	d := r.Document.Defaults
	if (defaults || direction) && d == nil {
		return errors.New("the relevé carries no defaults to confirm")
	}
	if direction && d.Direction == nil {
		return errors.New("the relevé carries no direction of play to confirm")
	}
	if defaults {
		d.Confirmed = true
	}
	if direction {
		d.Direction.Confirmed = true
	}
	return r.Validate()
}
