package server

import (
	"encoding/json"
	"iter"
	"net/http"
)

// ndjsonContentType is the media type for newline-delimited JSON streams.
const ndjsonContentType = "application/x-ndjson"

// streamSeq2 writes seq as NDJSON, flushing after each record. An error before
// the first record becomes a normal error envelope; after it, the status is
// already sent, so the error is a trailing {"error":{...}} line.
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
			_ = enc.Encode(errorEnvelope{Error: errorBodyFor(w, err)})
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
