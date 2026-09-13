package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// Compress gzips a response when the client asked for it and the payload is
// worth compressing (G.9, #237).
//
// The NDJSON listings are the reason: they are long, extremely repetitive —
// the same field names on every line — and typically read over a network the
// daemon does not control. gzip cuts them by an order of magnitude for a cost
// that does not show up next to the SQL underneath.
//
// Three properties this has to keep, and which a naive wrapper loses:
//
//   - **the stream stays a stream.** streamSeq2 flushes after every record so a
//     client sees rows as they come; the gzip writer is flushed on the same
//     beat, and Flush travels through. A compressor that only flushed at Close
//     would turn an incremental listing into one late block.
//   - **the decision is made on the response, not the request.** The
//     Content-Type is only known once the handler writes its header, so the
//     choice happens there: NDJSON and JSON are compressed, an already-
//     compressed body (a .db export, a .dbx container) is not — gzipping a
//     zstd blob spends CPU to grow it.
//   - **ResponseController keeps working.** server.go sets a per-request
//     read/write deadline through it; Unwrap is what lets it find the real
//     writer underneath.
func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}
		cw := &compressWriter{ResponseWriter: w}
		defer cw.Close()
		next.ServeHTTP(cw, r)
	})
}

// acceptsGzip reports whether the header offers gzip with a non-zero quality.
// "gzip;q=0" is a client saying explicitly that it does not want it.
func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(fields[0]), "gzip") {
			continue
		}
		for _, p := range fields[1:] {
			if q := strings.TrimSpace(p); strings.HasPrefix(q, "q=") {
				return strings.TrimPrefix(q, "q=") != "0" && strings.TrimPrefix(q, "q=") != "0.0"
			}
		}
		return true
	}
	return false
}

// compressibleTypes are the media types worth gzipping. Everything else — an
// exported database, an encrypted container, an image — is either already
// compressed or too small to matter.
var compressibleTypes = map[string]bool{
	"application/x-ndjson": true,
	"application/json":     true,
	"text/plain":           true,
}

type compressWriter struct {
	http.ResponseWriter
	gz     *gzip.Writer
	once   sync.Once
	closed bool
}

// Unwrap lets http.ResponseController reach the underlying writer (deadlines,
// hijacking): without it, every SetWriteDeadline through this wrapper fails.
func (c *compressWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }

// decide runs once, on the first WriteHeader or Write: by then the handler has
// set its Content-Type, which is what the choice rests on.
func (c *compressWriter) decide() {
	c.once.Do(func() {
		mediaType := c.Header().Get("Content-Type")
		if i := strings.IndexByte(mediaType, ';'); i >= 0 {
			mediaType = mediaType[:i]
		}
		if !compressibleTypes[strings.TrimSpace(mediaType)] {
			return
		}
		// Content-Length would describe the uncompressed body; drop it and let
		// the transfer be chunked.
		c.Header().Del("Content-Length")
		c.Header().Set("Content-Encoding", "gzip")
		c.Header().Add("Vary", "Accept-Encoding")
		// The level is not a tuning knob, it is what keeps the first property
		// above worth having. A listing is flushed after every record, and
		// from Go 1.27 the rewritten encoders behind levels 1-6 (and so
		// DefaultCompression) no longer find matches across those flushes: a
		// flush of fewer than 128 bytes is written as a Huffman-only (or
		// stored) block and resets the encoder's history (deflateFast in
		// compress/flate), a record is ~70 bytes, and the "compressed"
		// listing comes out LARGER than the plain one —
		// 108-115 % on 1.27.1, against 13-18 % on 1.26.8 (measured 2026-09-13,
		// #412). Levels 7-9 kept the chained encoder, which does: 13-20 % on
		// 1.27.1. On the 1.26 this repository builds with today, the default
		// still works and level 7 only matches it (13-17 %), for more CPU on
		// varied records — 5.3 µs against 3.4 µs per record, best of five.
		// That is the price of not shipping listings that grow the day
		// GO_VERSION moves to 1.27 (#409); level 9 buys no ratio for up to six
		// times the CPU. TestCompress_GzipsARepetitiveListing is what notices if
		// a future release moves that line again.
		c.gz, _ = gzip.NewWriterLevel(c.ResponseWriter, gzipLevel)
	})
}

// gzipLevel: see decide. NewWriterLevel only errors on a level outside
// [-2, 9], which a constant cannot be.
const gzipLevel = 7

func (c *compressWriter) WriteHeader(status int) {
	c.decide()
	c.ResponseWriter.WriteHeader(status)
}

func (c *compressWriter) Write(b []byte) (int, error) {
	// A handler that writes without WriteHeader implies 200, exactly as
	// net/http does — and must reach the same decision, not skip it.
	c.decide()
	if c.gz != nil {
		return c.gz.Write(b)
	}
	return c.ResponseWriter.Write(b)
}

// Flush pushes the compressor's buffer and then the underlying writer's, so a
// streamed listing stays incremental through the compressor.
func (c *compressWriter) Flush() {
	if c.gz != nil {
		_ = c.gz.Flush()
	}
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Close finishes the gzip stream. Idempotent: the deferred call in Compress is
// the only one, but a double Close on a gzip.Writer is an error worth not
// risking.
func (c *compressWriter) Close() {
	if c.gz != nil && !c.closed {
		c.closed = true
		_ = c.gz.Close()
	}
}
