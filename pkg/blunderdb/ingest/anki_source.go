package ingest

import (
	"encoding/json"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

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
	if raw, ok := doc["payload"]; ok && domain.SearchPayloadNamesRows(raw) {
		delete(doc, "payload")
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return remapIDList(source, positions)
	}
	return string(out)
}

// searchSourceLost reports whether carrying a search deck's source dropped
// part of what it said: its command and board, its stored payload, or every
// one of its ids. Such a deck still travels, with its members, but whoever
// opens it cannot rebuild it from its source; the export says so.
func searchSourceLost(source, carried string) bool {
	before, ok := domain.SearchSourceDocument(source)
	if !ok {
		return len(domain.SearchSourceIDs(source)) > 0 && len(domain.SearchSourceIDs(carried)) == 0
	}
	after, ok := domain.SearchSourceDocument(carried)
	if !ok {
		return true
	}
	for _, key := range []string{"command", "payload"} {
		if _, had := before[key]; had {
			if _, has := after[key]; !has {
				return true
			}
		}
	}
	return len(domain.SearchSourceIDs(source)) > 0 && len(domain.SearchSourceIDs(carried)) == 0
}
