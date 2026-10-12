package domain

import (
	"encoding/json"
	"strconv"
	"strings"
)

// SearchSourceIDs returns the position ids a search deck's source stores.
// The source is a JSON document `{command, position, ids, payload|library}`
// whose `ids` are the positions matched when the deck was made; an older
// database may still hold a bare comma-separated id list, read whole. A
// document that is neither yields no id.
func SearchSourceIDs(source string) []int64 {
	if doc, ok := SearchSourceDocument(source); ok {
		var ids []int64
		if raw, ok := doc["ids"]; ok {
			_ = json.Unmarshal(raw, &ids)
		}
		return ids
	}
	var ids []int64
	for tok := range strings.SplitSeq(source, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(tok), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// SearchSourceDocument reads a search deck's source as its JSON document,
// keyed by field, its values left raw so a rewrite keeps every field it does
// not touch. ok is false for the legacy comma-separated form.
func SearchSourceDocument(source string) (map[string]json.RawMessage, bool) {
	s := strings.TrimSpace(source)
	if !strings.HasPrefix(s, "{") {
		return nil, false
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		return nil, false
	}
	return doc, true
}
