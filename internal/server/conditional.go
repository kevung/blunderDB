package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Conditional reads (ADR-0057 rule 4): a read route answers with an ETag, and a client that
// sends it back in If-None-Match gets 304 with no body while nothing it reads has changed.
// The wall page of a Rencontre polls this way, and a 304 costs no replay of any Direction.
//
// The tag condenses the version of what the route reads (a journal's length, a record, a
// Rencontre row — service.DirectionVersion and its siblings), the request itself and the
// minute. The minute is in it because proposals, pages and the clock also depend on the time
// of the reading: a tag lives one minute at most, so a polling client sees a deadline or a
// break move within that minute even when no gesture is made. The tag is weak — a page also
// carries its render time, so two answers under one tag are equivalent, not byte-identical.

// readKey names what a conditional read depends on: one Direction, one Rencontre, or the
// whole scope.
type readKey struct {
	tournamentID int64
	rencontreID  int64
	scopeWide    bool
}

// keyedReq is implemented by every request type of a conditional read route.
type keyedReq interface {
	readKey() readKey
}

// versionFunc returns the version token of what a key names under a scope.
type versionFunc func(ctx context.Context, scope string, k readKey) (string, error)

// rpcRead is rpc for a read route: the same JSON in and out, plus ETag / If-None-Match. When
// the version cannot be read — no such Direction, another tenant's id — no tag is set and the
// handler runs, so the error is the handler's own (404), never a 304 on nothing.
func rpcRead[Req keyedReq, Resp any](version versionFunc, fn func(ctx context.Context, scope string, req Req) (Resp, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		scope := scopeOf(r)
		if v, err := version(r.Context(), scope, req.readKey()); err == nil {
			tag := readETag(r.URL.Path, scope, req, v)
			if etagMatches(r.Header.Get("If-None-Match"), tag) {
				w.Header().Set("ETag", tag)
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w = &etagWriter{ResponseWriter: w, tag: tag}
		}
		resp, err := fn(r.Context(), scope, req)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		writeJSONResp(w, resp)
	}
}

// readETag is the weak tag of one read: route, scope, request and version (minute included).
func readETag(path, scope string, req any, version string) string {
	h := sha256.New()
	body, _ := json.Marshal(req)
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s", path, scope, body, version)
	return `W/"` + hex.EncodeToString(h.Sum(nil))[:32] + `"`
}

// etagMatches applies If-None-Match's weak comparison (RFC 9110 §13.1.2): any listed tag
// whose opaque part equals ours, or "*".
func etagMatches(header, tag string) bool {
	if header == "" {
		return false
	}
	want := strings.TrimPrefix(tag, "W/")
	for _, t := range strings.Split(header, ",") {
		t = strings.TrimSpace(t)
		if t == "*" || strings.TrimPrefix(t, "W/") == want {
			return true
		}
	}
	return false
}

// etagWriter sets the tag on a 200 only: an error answer carries no validator.
type etagWriter struct {
	http.ResponseWriter
	tag string
}

func (e *etagWriter) WriteHeader(code int) {
	if code == http.StatusOK {
		e.Header().Set("ETag", e.tag)
	}
	e.ResponseWriter.WriteHeader(code)
}
