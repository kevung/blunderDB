package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ndjsonHandler writes n NDJSON records, flushing after each — the shape
// streamSeq2 produces.
func ndjsonHandler(n int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < n; i++ {
			_, _ = io.WriteString(w, `{"id":1,"player1Name":"Alice","player2Name":"Bob","matchLength":7}`+"\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})
}

func get(h http.Handler, acceptEncoding string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/matches.list", nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	rec := httptest.NewRecorder()
	Compress(h).ServeHTTP(rec, req)
	return rec
}

func TestCompress_GzipsARepetitiveListing(t *testing.T) {
	plain := get(ndjsonHandler(200), "")
	if plain.Header().Get("Content-Encoding") != "" {
		t.Fatalf("without Accept-Encoding the body must be plain, got %q", plain.Header().Get("Content-Encoding"))
	}

	zipped := get(ndjsonHandler(200), "gzip")
	if zipped.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", zipped.Header().Get("Content-Encoding"))
	}
	if !strings.Contains(zipped.Header().Get("Vary"), "Accept-Encoding") {
		t.Errorf("Vary = %q, must name Accept-Encoding or a cache will serve the wrong body", zipped.Header().Get("Vary"))
	}

	zr, err := gzip.NewReader(bytes.NewReader(zipped.Body.Bytes()))
	if err != nil {
		t.Fatalf("the body is not a gzip stream: %v", err)
	}
	got, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain.Body.Bytes()) {
		t.Error("the decompressed body differs from the plain one")
	}
	// The point of the exercise: these listings repeat their field names.
	if ratio := float64(zipped.Body.Len()) / float64(plain.Body.Len()); ratio > 0.2 {
		t.Errorf("compressed to %.0f%% of the original; a repetitive NDJSON listing should do far better", ratio*100)
	}
}

func TestCompress_LeavesAnAlreadyCompressedBodyAlone(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("an exported .db, already compressed inside"))
	})
	rec := get(h, "gzip")
	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("an octet-stream must not be gzipped: %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != "an exported .db, already compressed inside" {
		t.Errorf("body altered: %q", rec.Body.String())
	}
}

func TestCompress_HonoursAnExplicitRefusal(t *testing.T) {
	rec := get(ndjsonHandler(10), "gzip;q=0")
	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("gzip;q=0 means no: %q", rec.Header().Get("Content-Encoding"))
	}
	if rec := get(ndjsonHandler(10), "br, gzip"); rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("gzip listed among others must be picked up, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec := get(ndjsonHandler(10), "deflate"); rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("no gzip offered, none must be used: %q", rec.Header().Get("Content-Encoding"))
	}
}

// flushProbe counts how many times the layer below is flushed: a compressor
// that only emitted at Close would turn an incremental listing into one late
// block, which is exactly what the streaming routes exist to avoid.
type flushProbe struct {
	http.ResponseWriter
	flushes int
}

func (f *flushProbe) Flush() {
	f.flushes++
	if inner, ok := f.ResponseWriter.(http.Flusher); ok {
		inner.Flush()
	}
}

func TestCompress_KeepsTheStreamIncremental(t *testing.T) {
	probe := &flushProbe{ResponseWriter: httptest.NewRecorder()}
	req := httptest.NewRequest(http.MethodPost, "/v1/matches.list", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	Compress(ndjsonHandler(5)).ServeHTTP(probe, req)

	if probe.flushes != 5 {
		t.Errorf("the underlying writer was flushed %d times, want one per record (5)", probe.flushes)
	}
}

func TestCompress_WriteWithoutWriteHeaderStillDecides(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// no explicit WriteHeader: net/http implies 200
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	rec := get(h, "gzip")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("a handler that skips WriteHeader must still be compressed, got %q", rec.Header().Get("Content-Encoding"))
	}
	zr, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(zr)
	if string(got) != `{"ok":true}` {
		t.Errorf("body = %q", got)
	}
}
