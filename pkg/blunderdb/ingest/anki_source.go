package ingest

import (
	"encoding/json"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// payloadIDFields are the fields of a stored search payload that name rows
// by id. The payload is the frontend's filter set, replayed as is: nothing
// here can rewrite it the way the command bar's grammar is rewritten.
var payloadIDFields = []string{"likeTargetId", "positionIDsFilter", "matchIDsFilter", "tournamentIDsFilter", "restrictToPositionIDs"}

// carriedSearchSource moves a search deck's source from one library's
// numbering to another's, for every path a deck travels by — export to a
// file, import from one. Its `ids` are renumbered, a position left behind
// dropping out. Its `command` is rewritten by carriedLivingQuery; one that
// cannot say there what it said here is abandoned with the board it
// searched, and the deck keeps its ids — widening it, or aiming it at other
// rows, would change what the deck asks. A stored payload naming rows by id
// is abandoned for the same reason. `library` travels as it is: it names the
// whole library, wherever it is. A legacy comma-separated list is renumbered
// as a list.
func carriedSearchSource(source string, positions, matches, tournaments map[int64]int64) string {
	doc, ok := domain.SearchSourceDocument(source)
	if !ok {
		return remapIDList(source, positions)
	}
	ids := []int64{}
	for _, id := range domain.SearchSourceIDs(source) {
		if to, ok := positions[id]; ok {
			ids = append(ids, to)
		}
	}
	if raw, err := json.Marshal(ids); err == nil {
		doc["ids"] = raw
	}
	if raw, ok := doc["command"]; ok {
		var command string
		carried, kept := "", false
		if json.Unmarshal(raw, &command) == nil {
			carried, kept = carriedLivingQuery(command, positions, matches, tournaments)
		}
		if enc, err := json.Marshal(carried); kept && err == nil {
			doc["command"] = enc
		} else {
			delete(doc, "command")
			delete(doc, "position")
		}
	}
	if raw, ok := doc["payload"]; ok && payloadNamesRows(raw) {
		delete(doc, "payload")
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return remapIDList(source, positions)
	}
	return string(out)
}

// payloadNamesRows reports whether a stored search payload names a row by
// id; one that cannot be read is treated as naming some.
func payloadNamesRows(raw json.RawMessage) bool {
	var p map[string]any
	if err := json.Unmarshal(raw, &p); err != nil {
		return true
	}
	for _, k := range payloadIDFields {
		switch v := p[k].(type) {
		case nil:
		case string:
			if v != "" {
				return true
			}
		case float64:
			if v != 0 {
				return true
			}
		default:
			return true
		}
	}
	return false
}
