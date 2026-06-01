package server

import (
	"encoding/json"
	"iter"
	"net/http"
)

// ndjsonContentType is the media type for newline-delimited JSON streams.
const ndjsonContentType = "application/x-ndjson"

// streamSeq2 writes an iter.Seq2[T, error] as an NDJSON stream, flushing after
// each record so clients receive rows incrementally over HTTP/1.1 chunked
// transfer. If an error is yielded before any record is written, it is turned
// into a normal error envelope; once streaming has begun the status code is
// already sent, so a mid-stream error is reported as a trailing
// {"error":{...}} line instead.
func streamSeq2[T any](w http.ResponseWriter, seq iter.Seq2[T, error]) {
	flusher, _ := w.(http.Flusher)
	enc := json.NewEncoder(w)
	wroteHeader := false

	for v, err := range seq {
		if err != nil {
			if !wroteHeader {
				writeStorageError(w, err)
				return
			}
			_ = enc.Encode(errorEnvelope{Error: errorBody{
				Code:    codeForErr(err),
				Message: err.Error(),
			}})
			if flusher != nil {
				flusher.Flush()
			}
			return
		}
		if !wroteHeader {
			w.Header().Set("Content-Type", ndjsonContentType)
			w.WriteHeader(http.StatusOK)
			wroteHeader = true
		}
		if encErr := enc.Encode(v); encErr != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}

	// An empty stream still returns 200 with an NDJSON content type and no body.
	if !wroteHeader {
		w.Header().Set("Content-Type", ndjsonContentType)
		w.WriteHeader(http.StatusOK)
	}
}
