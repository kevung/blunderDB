package arkqr

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// An archive lives on paper for years: every format version this package
// ever wrote must still decode. testdata/golden_v<N>.json holds the streams
// version N wrote for the matches goldenMatches builds; they are written once,
// when the version is frozen, and never again.
//
//	BLUNDERDB_ARKQR_GOLDEN_WRITE=<N> go test ./pkg/blunderdb/arkqr -run Golden

// timedMatch is a match recorded on video: durations and timecodes on every
// play, a header as the transcription screen fills it, source metadata.
func timedMatch(t *testing.T) Match {
	doc := loadCorpus(t)[1].doc
	acts := append([]transcript.Action(nil), doc.Actions...)
	ms := func(v int64) *int64 { return &v }
	clock := int64(47_120)
	for i := range acts {
		a := &acts[i]
		if a.Kind != transcript.KindChecker && a.Kind != transcript.KindDance {
			continue
		}
		if i%3 == 0 {
			cube := int64(1_500 + 97*i%2_000)
			a.CubeDecisionMS = ms(cube)
			clock += cube
		}
		a.RollTickMS = ms(clock)
		if a.Kind == transcript.KindChecker {
			d := int64(3_000 + 1_337*i%11_000)
			a.DecisionMS = ms(d)
			clock += d
		}
		if i%7 != 0 {
			a.TickMS = ms(clock)
		}
		clock += int64(250 * (i % 4))
	}
	doc.Actions = acts
	h := &doc.Header
	h.Player1, h.Player2, h.Transcriber = "Kévin Unger", "Nicolas Harmand", "Kévin Unger"
	h.Event, h.Location, h.Round = "Match d'entrainement", "Bar les Arènes", "1"
	h.VideoSource = "2026-11-24-lesArenes-Kevin-Nico-7p.mp4"
	elo, exp := 1612.0, 230
	return Match{Doc: doc, Elo: [2]*float64{&elo, &elo}, Experience: [2]*int{&exp, nil}, EngineVersion: "eXtreme Gammon, file format 32"}
}

func goldenMatches(t *testing.T) map[string]Match {
	out := map[string]Match{
		"degenerate": degenerateMatch(t),
		"empty":      {Doc: transcript.New(5)},
		"timed":      timedMatch(t),
	}
	elo, exp := 1834.25, 412
	out["source"] = Match{Doc: loadCorpus(t)[0].doc, Elo: [2]*float64{&elo, nil}, Experience: [2]*int{nil, &exp}, EngineVersion: "eXtreme Gammon 2.19"}
	for _, cm := range loadCorpus(t) {
		out[cm.name] = Match{Doc: cm.doc}
	}
	return out
}

func TestDecodeGolden(t *testing.T) {
	matches := goldenMatches(t)
	if v := os.Getenv("BLUNDERDB_ARKQR_GOLDEN_WRITE"); v != "" {
		streams := map[string]string{}
		for name, m := range matches {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > FormatVersion {
				t.Fatalf("version %q", v)
			}
			data, err := encodeVersion(m, byte(n))
			if err != nil {
				t.Fatal(err)
			}
			streams[name] = hex.EncodeToString(data)
		}
		b, _ := json.MarshalIndent(streams, "", "  ")
		if err := os.WriteFile("testdata/golden_v"+v+".json", append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for v := byte(1); v <= FormatVersion; v++ {
		b, err := os.ReadFile("testdata/golden_v" + strconv.Itoa(int(v)) + ".json")
		if err != nil {
			t.Fatalf("version %d: %v", v, err)
		}
		var streams map[string]string
		if err := json.Unmarshal(b, &streams); err != nil {
			t.Fatal(err)
		}
		if len(streams) != len(matches) {
			t.Fatalf("version %d: %d streams for %d matches", v, len(streams), len(matches))
		}
		for name, m := range matches {
			data, err := hex.DecodeString(streams[name])
			if err != nil || len(data) == 0 || data[0] != v {
				t.Fatalf("version %d: %s: bad stream", v, name)
			}
			back, err := Decode(data)
			if err != nil {
				t.Fatalf("version %d: %s: %v", v, name, err)
			}
			checkDecoded(t, name, m, back)
			// The models of a version are frozen: coding again writes the same stream.
			if again, err := encodeVersion(m, v); err != nil || hex.EncodeToString(again) != streams[name] {
				t.Errorf("version %d: %s: the encoder of the version changed (%v)", v, name, err)
			}
		}
	}
}
