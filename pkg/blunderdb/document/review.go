package document

import (
	"errors"
	"slices"
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
	// Held is set when the import with the options asked for still refuses
	// the entry; Fallback when it takes it under its position's context.
	Held     bool `json:"held"`
	Fallback bool `json:"fallback"`
	// Fields names what to check against the diagram, as codes an interface
	// translates: the context fields of an entry whose context is to verify
	// ("context" when none is named), else the fields of the reading's doubts.
	Fields []string `json:"fields"`
	// Note is the same, in English prose, for the command line.
	Note string `json:"note"`
}

// ReviewResult is what a person decides on before importing a Relevé.
type ReviewResult struct {
	Items []ReviewItem `json:"items"`
	// Refused is the list the import with the options asked for refuses
	// (Refused); the import is possible when it is empty.
	Refused []int `json:"refused"`
	// FallbackRanks are the ranks the fallback would take, whether or not it
	// is asked for: it offers nothing when empty.
	FallbackRanks []int `json:"fallbackRanks"`
}

// Review lists the entries an import refuses without a decision, in the
// document's order, each with its fields to check, and what the import with
// opts would still refuse.
func Review(r *Releve, opts ImportOptions) ReviewResult {
	r = r.Upgrade()
	res := ReviewResult{Items: []ReviewItem{}, Refused: Refused(r, opts), FallbackRanks: FallbackRanks(r, opts.AcceptComplete)}
	held := map[int]bool{}
	for _, k := range res.Refused {
		held[k] = true
	}
	taken := map[int]bool{}
	if opts.Fallback {
		for _, k := range res.FallbackRanks {
			taken[k] = true
		}
	}
	for _, e := range r.Entries {
		if r.Importable(e, false) {
			continue
		}
		fields, note := reviewFields(e)
		res.Items = append(res.Items, ReviewItem{
			Rank: e.Rank, Reference: r.ReferenceOf(e), State: r.StateOf(e),
			Checkers: e.Checkers, Context: e.Context, Complete: e.Complete,
			Held: held[e.Rank], Fallback: taken[e.Rank], Fields: fields, Note: note,
		})
	}
	return res
}

// doubtCategory is the closed list of the fields a reader gives a doubt
// (Entry.Doubts), each with the category an interface names it by: what to
// check against the diagram. A reader that emits another field fails
// TestDoubtFieldsAreClosed.
var doubtCategory = map[string]string{
	"point": "checkers", "bar": "checkers", "checkers": "checkers",
	"frame": "diagram", "board": "diagram", "position": "diagram",
	"orientation": "orientation", "pips": "pips", "context": "context",
	"cube": "cube", "dice": "dice", "score": "score", "turn": "turn",
}

// categories maps doubt fields to their categories, once each, in order.
func categories(fields []string) []string {
	out := []string{}
	for _, f := range fields {
		c, ok := doubtCategory[f]
		if !ok {
			c = f
		}
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

func reviewFields(e Entry) ([]string, string) {
	if e.Context == ContextToVerify {
		var fields []string
		for _, d := range e.Doubts {
			if contextField[d.Field] && d != ContextDoubt {
				fields = append(fields, d.Field)
			}
		}
		if len(fields) == 0 {
			fields = []string{"context"}
		}
		return categories(fields), e.FallbackNote()
	}
	fields := []string{}
	parts := make([]string, 0, len(e.Doubts))
	for _, d := range e.Doubts {
		fields = append(fields, d.Field)
		parts = append(parts, d.Field+" ("+d.Reason+")")
	}
	return categories(fields), strings.Join(parts, "; ")
}

// ConfirmDefaults marks the document's defaults, and the direction of play
// when it carries one, as confirmed by the person who checked them against
// the diagrams: one gesture for both. A Relevé without defaults is refused.
func ConfirmDefaults(r *Releve) error {
	d := r.Document.Defaults
	if d == nil {
		return errors.New("the relevé carries no defaults to confirm")
	}
	d.Confirmed = true
	if d.Direction != nil {
		d.Direction.Confirmed = true
	}
	return r.Validate()
}
