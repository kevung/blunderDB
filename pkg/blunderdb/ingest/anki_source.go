package ingest

import (
	"encoding/json"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// searchSourceKeys are the fields of a search deck's source that travel; any
// other is text nothing here reads, and stays.
var searchSourceKeys = map[string]bool{"ids": true, "command": true, "position": true, "library": true, "payload": true}

// carriedSearchSource moves a search deck's source from one library's
// numbering to another's, for every path a deck travels by — export to a
// file, import from one. Its `ids` are renumbered, a position left behind
// dropping out. Its `command` is rewritten by carriedLivingQuery; one that
// cannot say there what it said here is abandoned with the board it
// searched, and the deck keeps its ids — widening it, or aiming it at other
// rows, would change what the deck asks. Its `position` is the board the
// command was drawn on, written back as a board and nothing else: the id of
// the row it was drawn from names a row of the source, and stays. A stored
// payload naming rows by id is abandoned for the same reason as the command.
// gate is what the export lets a query take out (the zero value on import:
// the file already left). `library` travels as it is: it names the whole
// library, wherever it is. A field outside searchSourceKeys does not travel.
// A legacy comma-separated list is renumbered as a list.
func carriedSearchSource(source string, positions, matches, tournaments map[int64]int64, gate queryGate) string {
	doc, ok := domain.SearchSourceDocument(source)
	if !ok {
		return remapIDList(source, positions)
	}
	for key := range doc {
		if !searchSourceKeys[key] {
			delete(doc, key)
		}
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
			carried, kept = gate.carried(command, positions, matches, tournaments)
		}
		if enc, err := json.Marshal(carried); kept && err == nil {
			doc["command"] = enc
		} else {
			delete(doc, "command")
			delete(doc, "position")
		}
	}
	if raw, ok := doc["position"]; ok {
		if board, ok := carriedBoard(raw); ok {
			doc["position"] = board
		} else {
			delete(doc, "position")
		}
	}
	// Without provenance the payload stays too: it holds the same names as
	// a command, in a form nothing here reads.
	if raw, ok := doc["payload"]; ok && (gate.omitProvenance || domain.SearchPayloadNamesRows(raw)) {
		delete(doc, "payload")
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return remapIDList(source, positions)
	}
	return string(out)
}

// carriedBoard writes the board of a search source back as a board with no
// row id, in the form it was stored in: an object, or a string holding one.
// ok is false for anything that is not a board.
func carriedBoard(raw json.RawMessage) (json.RawMessage, bool) {
	text, inString := "", false
	if json.Unmarshal(raw, &text) == nil {
		raw, inString = json.RawMessage(text), true
	}
	var board domain.Position
	if json.Unmarshal(raw, &board) != nil {
		return nil, false
	}
	board.ID = 0
	out, err := json.Marshal(board)
	if err == nil && inString {
		out, err = json.Marshal(string(out))
	}
	return out, err == nil
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
