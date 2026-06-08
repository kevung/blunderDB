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
