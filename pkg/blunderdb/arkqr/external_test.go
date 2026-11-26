package arkqr

import (
	"encoding/json"
	"math"
	"os"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The measures of the codec on a real library, which the repository does
// not hold. The library is first written out as JSON by TestDumpArchiveCorpus
// (arkqr_test), from a copy of the database opened read-only:
//
//	BLUNDERDB_ARKQR_DUMP=copy.db BLUNDERDB_ARKQR_CORPUS=corpus.json go test ./pkg/blunderdb/arkqr -run Dump
//	BLUNDERDB_ARKQR_CORPUS=corpus.json go test ./pkg/blunderdb/arkqr -run External -v
const externalCorpus = "BLUNDERDB_ARKQR_CORPUS"

func loadExternal(t testing.TB) []Match {
	path := os.Getenv(externalCorpus)
	if path == "" {
		t.Skip(externalCorpus + " unset")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ms []Match
	if err := json.Unmarshal(b, &ms); err != nil {
		t.Fatal(err)
	}
	return ms
}

func replayInfo(x Match) []transcript.ActionInfo { return transcript.Replay(x.Doc, 0).Actions }

// TestExternalSizes codes the library in every version: bytes per match,
// codes, and A4 sheets.
func TestExternalSizes(t *testing.T) {
	ms := loadExternal(t)
	for v := byte(1); v <= FormatVersion; v++ {
		var payloads [][]byte
		var sizes, sevens []int
		total := 0
		for _, x := range ms {
			data, err := encodeVersion(x, v)
			if err != nil {
				t.Fatal(err)
			}
			back, err := Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			if again, _ := encodeVersion(back, v); string(again) != string(data) {
				t.Fatalf("v%d: re-encoding a decoded match changes it", v)
			}
			if x.Doc.Header.MatchLength == 7 {
				sevens = append(sevens, len(data))
			}
			payloads = append(payloads, data)
			sizes = append(sizes, len(data))
			total += len(data)
		}
		sort.Ints(sizes)
		sort.Ints(sevens)
		if len(sevens) > 0 {
			t.Logf("v%d: %d matches in 7 points, median %d bytes", v, len(sevens), sevens[len(sevens)/2])
		}
		frames, _, err := Frames(payloads)
		if err != nil {
			t.Fatal(err)
		}
		codes := make([]Code, len(frames))
		for i, f := range frames {
			codes[i] = Code{Frame: f}
		}
		sheets, err := layout(codes, moduleMM)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("v%d: %d matches, %d bytes, median %d, p90 %d; %d codes, %d sheets, %.2f matches a sheet",
			v, len(ms), total, sizes[len(sizes)/2], sizes[len(sizes)*9/10], len(frames), len(sheets), float64(len(ms))/float64(len(sheets)))
	}
}

// meter is a coder that codes nothing and adds up the cost of each symbol,
// in bits, under the line of the format that codes it.
type meter struct{ cost map[string]float64 }

func (m *meter) decoding() bool { return false }

// meterHelpers are the functions a symbol is coded through, not where.
var meterHelpers = []string{".raw", ".uvar", ".svar", ".flag", ".boolBit", ".(*adaptive).code", ".str", ".count",
	".(*meter).sym", ".rawRel", ".optBoard", ".(*adaptiveFreqs).code", ".gamma.code", ".gamma.signed",
	".(*textModel).code", ".(*models).str", ".engineName", ".rating"}

func (m *meter) sym(fr []uint64, v int) int {
	var tot uint64
	for _, f := range fr {
		tot += f
	}
	pc := make([]uintptr, 16)
	frames := runtime.CallersFrames(pc[:runtime.Callers(2, pc)])
	for {
		f, more := frames.Next()
		if !slices.ContainsFunc(meterHelpers, func(h string) bool { return strings.HasSuffix(f.Function, h) }) {
			m.cost[f.Function[strings.LastIndex(f.Function, ".")+1:]+":"+strconv.Itoa(f.Line)] += math.Log2(float64(tot) / float64(fr[v]))
			break
		}
		if !more {
			break
		}
	}
	return v
}

// TestExternalCosts shows where the bytes of the current version go.
func TestExternalCosts(t *testing.T) {
	ms := loadExternal(t)
	m := &meter{map[string]float64{}}
	for _, x := range ms {
		out := x
		codeMatch(m, FormatVersion, &out, replayInfo(x))
	}
	keys := make([]string, 0, len(m.cost))
	sum := 0.0
	for k, v := range m.cost {
		keys = append(keys, k)
		sum += v
	}
	sort.Slice(keys, func(i, j int) bool { return m.cost[keys[i]] > m.cost[keys[j]] })
	for _, k := range keys[:min(20, len(keys))] {
		t.Logf("%-24s %8.0f bytes %5.1f%%", k, m.cost[k]/8, 100*m.cost[k]/sum)
	}
}
