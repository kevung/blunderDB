package parser

import (
	"errors"
	"regexp"
	"sync"
)

// ErrEmpty / ErrNoXGID mirror the JS parser's thrown errors so callers (the
// server) can map them to a 4xx response.
var (
	errEmpty  = errors.New("parser: empty or invalid input")
	errNoXGID = errors.New("parser: XGID not found in the content")

	// ErrUnrecognisedAnalysis is returned when the text plainly carries an
	// analysis block (a numbered move list, win-chance breakdowns) that none
	// of the language markers matched, so nothing could be read from it.
	// Before it existed the parser returned an empty analysis and no error,
	// and a Spanish or Russian XG export was saved as a bare position with
	// no one told. XG ships in English, German, French, Spanish, Japanese,
	// Greek and Russian (docs/recherche/P9-formats-de-fichiers.md); the last
	// three are not recognised yet, for want of a verified sample.
	ErrUnrecognisedAnalysis = errors.New("parser: analysis block not recognised — XG language not supported? (supported: English, French, German, Japanese)")
)

// regexCache compiles each pattern once and reuses it. Patterns are static
// strings, so the cache is bounded; a mutex keeps it safe under the concurrent,
// lock-free ParsePositionText calls.
type regexCache struct {
	mu sync.Mutex
	m  map[string]*regexp.Regexp
}

func newRegexCache() *regexCache {
	return &regexCache{m: make(map[string]*regexp.Regexp)}
}

func (c *regexCache) get(pattern string) *regexp.Regexp {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.m[pattern]; ok {
		return r
	}
	r := regexp.MustCompile(pattern)
	c.m[pattern] = r
	return r
}
