package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// corruptAnalysisBlobs are stored analysis payloads that cannot be decoded,
// one per failure point of DecodeAnalysisFromStorage: a zlib stream whose
// body is garbage (decompression fails) and a payload that looks like JSON
// but is not (the unmarshal fails).
var corruptAnalysisBlobs = map[string][]byte{
	"truncated zlib": {0x78, 0x9c, 0x01, 0x02, 0x03},
	"invalid JSON":   []byte(`{"xgid": "half-written`),
}

// TestDecodeAnalysesConcurrently_CorruptEntryIsReported pins the worker
// pool's contract on a decode failure: the position lands in failed with its
// error, is absent from decoded, and the others decode normally — so a
// caller can tell "no analysis" from "an analysis nobody can read".
func TestDecodeAnalysesConcurrently_CorruptEntryIsReported(t *testing.T) {
	good, err := EncodeAnalysisForStorage(&domain.PositionAnalysis{XGID: "good"})
	if err != nil {
		t.Fatal(err)
	}
	for name, corrupt := range corruptAnalysisBlobs {
		t.Run(name, func(t *testing.T) {
			decoded, failed := DecodeAnalysesConcurrently(map[int64][]byte{1: good, 2: corrupt, 3: good})
			if len(decoded) != 2 {
				t.Fatalf("len(decoded) = %d, want 2", len(decoded))
			}
			if _, ok := decoded[2]; ok {
				t.Errorf("corrupt entry present in decoded: %+v", decoded[2])
			}
			if failed[2] == nil || len(failed) != 1 {
				t.Errorf("failed = %v, want the corrupt entry alone", failed)
			}
			for _, id := range []int64{1, 3} {
				if decoded[id] == nil || decoded[id].XGID != "good" {
					t.Errorf("decoded[%d] = %+v, want the good analysis", id, decoded[id])
				}
			}
		})
	}
}

func TestDecodeAnalysesConcurrently_Empty(t *testing.T) {
	decoded, failed := DecodeAnalysesConcurrently(nil)
	if len(decoded) != 0 || len(failed) != 0 {
		t.Fatalf("empty input decoded to %v / %v", decoded, failed)
	}
}
