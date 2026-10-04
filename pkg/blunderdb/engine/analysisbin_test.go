package engine

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

// demoAnalysisBlobs returns every analysis blob of the embedded demo
// database, as an earlier release wrote them (JSON in zstd).
func demoAnalysisBlobs(t testing.TB) map[int64][]byte {
	t.Helper()
	gz, err := os.Open(filepath.Join("..", "..", "..", "internal", "gui", "demo.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	zr, err := gzip.NewReader(gz)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "demo.db")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(f, zr); err != nil {
		t.Fatal(err)
	}
	f.Close()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT position_id, data FROM analysis`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64][]byte{}
	for rows.Next() {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			t.Fatal(err)
		}
		out[id] = data
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out) < 100 {
		t.Fatalf("demo database has only %d analyses", len(out))
	}
	return out
}

// requireExactRoundTrip encodes a in the binary format at both levels and
// requires the very same value back: DeepEqual (floats bit for bit, nil and
// empty slices apart, time zones) and the same JSON.
func requireExactRoundTrip(t *testing.T, name string, a domain.PositionAnalysis) {
	t.Helper()
	want, err := json.Marshal(&a)
	if err != nil {
		t.Fatalf("%s: marshal: %v", name, err)
	}
	fast, err := EncodeAnalysisForStorage(&a)
	if err != nil {
		t.Fatalf("%s: encode: %v", name, err)
	}
	compact, err := CompactAnalysisData(fast)
	if err != nil {
		t.Fatalf("%s: compact: %v", name, err)
	}
	for level, blob := range map[string][]byte{"level 7": fast, "level 19": compact} {
		if !isBinaryBlob(blob) || NeedsRecompression(blob) {
			t.Fatalf("%s %s: not a binary blob", name, level)
		}
		back, err := DecodeAnalysisFromStorage(blob)
		if err != nil {
			t.Fatalf("%s %s: decode: %v", name, level, err)
		}
		if !reflect.DeepEqual(back, a) {
			t.Fatalf("%s %s: round trip differs\n got %+v\nwant %+v", name, level, back, a)
		}
		got, _ := json.Marshal(&back)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s %s: JSON differs\n got %s\nwant %s", name, level, got, want)
		}
	}
}

// TestBinaryRoundTripOnDemoAnalyses re-encodes every real analysis of the
// demo database (XG, GNUbg-style gammonNet and BGBlitz) and requires each to
// come back exactly as the legacy JSON blob decoded.
func TestBinaryRoundTripOnDemoAnalyses(t *testing.T) {
	var legacy, binary int
	for id, blob := range demoAnalysisBlobs(t) {
		a, err := DecodeAnalysisFromStorage(blob)
		if err != nil {
			t.Fatalf("position %d: legacy decode: %v", id, err)
		}
		// The legacy JSON itself, not only the struct: the binary blob must
		// marshal back to the very bytes the old blob held.
		stored, err := inflateLegacyAnalysis(blob)
		if err != nil {
			t.Fatal(err)
		}
		requireExactRoundTrip(t, "demo", a)
		fresh, _ := EncodeAnalysisForStorage(&a)
		viaJSON, err := DecompressAnalysisData(fresh)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(viaJSON, stored) {
			t.Fatalf("position %d: binary blob does not marshal back to the stored JSON", id)
		}
		legacy += len(blob)
		binary += len(fresh)
	}
	t.Logf("demo: legacy %d bytes, binary %d bytes (%.1f %%)", legacy, binary, 100*float64(binary)/float64(legacy))
}

// TestBinaryRoundTripOnEdgeValues covers what the demo database lacks:
// rollouts, every cube analysis, nil against empty slices, values no scale
// reproduces, negative zero, NaN, odd time zones and a zero time.
func TestBinaryRoundTripOnEdgeValues(t *testing.T) {
	tokyo := time.FixedZone("", 9*3600)
	odd := time.FixedZone("", -(3*3600 + 30*60 + 17))
	eqErr := -0.123
	inexact := 1.0 / 3
	cube := domain.DoublingCubeAnalysis{
		AnalysisDepth: "Rollout", AnalysisEngine: "a new engine",
		PlayerWinChances: 66.67, OpponentWinChances: 33.33, PlayerGammonChances: inexact,
		CubelessNoDoubleEquity: math.Copysign(0, -1), CubefulDoubleTakeError: -1e-300,
		BestCubeAction: "Too good to double, pass", WrongTakePercentage: 12.34,
	}
	a := domain.PositionAnalysis{
		PositionID: -7, XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10",
		Player1: "Ünïcode ♟", Player2: strings.Repeat("x", 300),
		AnalysisType: "CheckerMove", AnalysisEngineVersion: "XG",
		DoublingCubeAnalysis: &cube,
		AllCubeAnalyses:      []domain.DoublingCubeAnalysis{cube, {}},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Index: 0, AnalysisDepth: "4-ply", AnalysisEngine: "XG", Move: "13/7 8/7", Equity: 0.123, EquityError: new(float64), PlayerWinChance: 51.2, OpponentWinChance: 48.8},
			{Index: 5, AnalysisDepth: "4-ply", AnalysisEngine: "XG", Move: "24/18 13/11", Equity: 0.0, EquityError: &eqErr, PlayerWinChance: 50, OpponentWinChance: 49.99},
			{Index: 2, Move: "13/7 8/7", Equity: math.NaN(), PlayerWinChance: math.Inf(1)},
		}},
		PlayedMove: "13/7 8/7", PlayedCubeAction: "",
		PlayedMoves: []string{"13/7 8/7", "13/7 8/7"}, PlayedCubeActions: []string{},
		Rollouts: []domain.RolloutAnalysis{{
			AnalysisEngine: "gammonNet", AnalysisDepth: "Rollout", Signature: "sig-1", Kind: domain.RolloutKindCube,
			Settings: domain.RolloutSettings{Truncation: 11, MinGames: 324, MaxGames: 1296, JSDLimit: 2.33, Ply: 2, Candidates: 5, Seed: math.MaxUint64},
			Games:    1296, Stop: "jsd", CubefulBias: true,
			Candidates: []domain.RolloutCandidate{{Move: "Double, Take", Equity: 0.512, StdErr: 0.0042, CI95: 0.0082, Games: 1296, JSD: 3.1, PlayerWinChance: 70.12, OpponentWinChance: 29.88}},
			JSDDouble:  1.5, Date: time.Date(2026, 10, 4, 9, 8, 7, 123456789, odd),
		}, {Kind: domain.RolloutKindMoves, Candidates: []domain.RolloutCandidate{}, Date: time.Time{}}},
		CreationDate:     time.Date(2025, 6, 23, 10, 0, 0, 1, tokyo),
		LastModifiedDate: time.Date(2025, 6, 23, 9, 59, 59, 999999999, time.UTC),
	}
	// DeepEqual needs NaN-free values; JSON cannot hold NaN at all. The
	// NaN/Inf case is checked on the payload bits below.
	a.CheckerAnalysis.Moves[2].Equity = 7
	a.CheckerAnalysis.Moves[2].PlayerWinChance = 1e308
	requireExactRoundTrip(t, "edge", a)

	// Times as encoding/json leaves them: Local when the offset is the local
	// zone's, UTC at offset 0.
	local := time.Date(2025, 1, 2, 3, 4, 5, 6, time.Local)
	b := domain.PositionAnalysis{CreationDate: local, LastModifiedDate: local.Add(time.Hour)}
	requireExactRoundTrip(t, "local", b)
	requireExactRoundTrip(t, "zero", domain.PositionAnalysis{})
	requireExactRoundTrip(t, "nil moves", domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{}})
	requireExactRoundTrip(t, "empty moves", domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{}}})

	nan := domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Equity: math.NaN(), PlayerWinChance: math.Inf(-1)}}}}
	back, err := UnmarshalAnalysisBinary(MarshalAnalysisBinary(&nan))
	if err != nil {
		t.Fatal(err)
	}
	m := back.CheckerAnalysis.Moves[0]
	if math.Float64bits(m.Equity) != math.Float64bits(nan.CheckerAnalysis.Moves[0].Equity) || !math.IsInf(m.PlayerWinChance, -1) {
		t.Errorf("NaN/Inf not preserved: %v %v", m.Equity, m.PlayerWinChance)
	}
}

// fieldPaths lists every leaf field reachable from t, as the binary codec has
// to know them.
func fieldPaths(t reflect.Type, prefix string, out *[]string) {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice:
		fieldPaths(t.Elem(), prefix+"[]", out)
		return
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			*out = append(*out, prefix+":time")
			return
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			fieldPaths(f.Type, prefix+"."+f.Name, out)
		}
		return
	}
	*out = append(*out, prefix+":"+t.Kind().String())
}

// TestBinaryFormatCoversEveryField fails when a field is added to (or removed
// from) PositionAnalysis or what it holds: the binary payload names its fields
// by position, so a new field must be written by MarshalAnalysisBinary — and,
// stored rows being forever, under a new format version (ADR-0070) — before
// this list is updated.
func TestBinaryFormatCoversEveryField(t *testing.T) {
	var got []string
	fieldPaths(reflect.TypeOf(domain.PositionAnalysis{}), "", &got)
	sort.Strings(got)
	list := strings.Join(got, "\n")
	sum := sha256.Sum256([]byte(list))
	// The 93 leaf fields format version 1 writes, by the hash of their list.
	const want = "7132471539e17044"
	if hex.EncodeToString(sum[:8]) != want {
		t.Fatalf("PositionAnalysis changed shape (%d leaf fields, hash %x); binary format version 1 knows:\n%s",
			len(got), sum[:8], list)
	}
}

// TestBinarySeedStringsAreUnique: a duplicate would make two indexes for one
// string; harmless, but a sign the table was edited — which is a new version.
func TestBinarySeedStringsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range binarySeedStrings {
		if seen[s] {
			t.Errorf("duplicate seed string %q", s)
		}
		seen[s] = true
	}
}

// TestBinaryBlobOfUnknownVersionIsRefused: a newer release's blob is an
// error, never a misread.
func TestBinaryBlobOfUnknownVersionIsRefused(t *testing.T) {
	blob, _ := EncodeAnalysisForStorage(&domain.PositionAnalysis{XGID: "x"})
	blob[1] = binFormatVersion + 1
	if _, err := DecodeAnalysisFromStorage(blob); err == nil || !strings.Contains(err.Error(), "unknown binary format") {
		t.Fatalf("unknown version accepted: %v", err)
	}
}

// binaryZeroBomb is a binary blob whose frame inflates to n zero bytes.
func binaryZeroBomb(n int64) []byte {
	frame := zstdZeroBomb(n)
	return append([]byte{binHeaderTag, binFormatVersion}, frame[len(zstdMagic):]...)
}

func TestBinaryBlobRefusesADecompressionBomb(t *testing.T) {
	bomb := binaryZeroBomb(1 << 30)
	start := time.Now()
	_, err := DecodeAnalysisFromStorage(bomb)
	elapsed := time.Since(start)
	if !errors.Is(err, ErrAnalysisTooLarge) {
		t.Fatalf("expected ErrAnalysisTooLarge, got %v", err)
	}
	if elapsed > time.Second && !raceEnabled {
		t.Fatalf("refusing a 1 GiB binary bomb took %s, expected under a second", elapsed)
	}
}

// legacyZstdJSON writes JSON the way releases before ADR-0070 did.
func legacyZstdJSON(t testing.TB, js []byte) []byte {
	t.Helper()
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderDict(analysisZstdDict), zstd.WithEncoderCRC(false))
	if err != nil {
		t.Fatal(err)
	}
	return enc.EncodeAll(js, nil)
}

// TestLegacyBlobsUpgradeToBinary: every legacy format reads, and the
// upgrade paths turn it into the binary format with the same content.
func TestLegacyBlobsUpgradeToBinary(t *testing.T) {
	js := []byte(`{"positionId":3,"xgid":"XGID=a","player1":"A","player2":"B","analysisType":"","analysisEngineVersion":"","creationDate":"2025-01-01T00:00:00Z","lastModifiedDate":"2025-01-01T00:00:00Z"}`)
	for name, blob := range map[string][]byte{
		"raw":  js,
		"zlib": zlibCompressForFuzzSeed(js),
		"zstd": legacyZstdJSON(t, js),
	} {
		if !NeedsRecompression(blob) || !NeedsCompaction(blob) {
			t.Fatalf("%s: legacy blob not flagged for upgrade", name)
		}
		for step, up := range map[string]func([]byte) ([]byte, error){"recompress": RecompressAnalysisData, "compact": CompactAnalysisData} {
			out, err := up(blob)
			if err != nil {
				t.Fatalf("%s %s: %v", name, step, err)
			}
			got, err := DecompressAnalysisData(out)
			if err != nil || !bytes.Equal(got, js) || !isBinaryBlob(out) {
				t.Fatalf("%s %s: %s, %v", name, step, got, err)
			}
		}
	}
}

// FuzzUnmarshalAnalysisBinary feeds hostile payloads straight to the
// binary decoder (no zstd in front, so the fuzzer reaches the structure).
// Contract: never panics; the elements a payload declares are bounded by its
// length and by maxBinaryElements, so allocation follows the input; and
// whatever decodes re-encodes to bytes that decode to the same value.
func FuzzUnmarshalAnalysisBinary(f *testing.F) {
	f.Add([]byte{})
	f.Add(MarshalAnalysisBinary(&domain.PositionAnalysis{}))
	f.Add(MarshalAnalysisBinary(&domain.PositionAnalysis{
		XGID: "XGID=x", DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{PlayerWinChances: 51.2, OpponentWinChances: 48.8},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "8/2 6/2", Equity: 0.25}, {Move: "13/7", Equity: 0.1, EquityError: new(float64)}}},
		PlayedMoves:     []string{"8/2 6/2"},
		Rollouts:        []domain.RolloutAnalysis{{Candidates: []domain.RolloutCandidate{{Equity: 1}}}},
	}))
	f.Add([]byte{0x03, 0x00, 0xff, 0xff, 0xff, 0xff, 0x0f})       // flags, then a huge string length
	f.Add([]byte{0x02, 0x00, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0}) // checker with a huge count next
	f.Fuzz(func(t *testing.T, payload []byte) {
		a, err := UnmarshalAnalysisBinary(payload)
		if err != nil {
			return
		}
		elements := len(a.AllCubeAnalyses) + len(a.PlayedMoves) + len(a.PlayedCubeActions) + len(a.Rollouts)
		if a.CheckerAnalysis != nil {
			elements += len(a.CheckerAnalysis.Moves)
		}
		for _, r := range a.Rollouts {
			elements += len(r.Candidates)
		}
		if elements > len(payload) || elements > maxBinaryElements {
			t.Fatalf("%d elements from a %d-byte payload", elements, len(payload))
		}
		again := MarshalAnalysisBinary(&a)
		b, err := UnmarshalAnalysisBinary(again)
		if err != nil {
			t.Fatalf("re-encoded payload does not decode: %v", err)
		}
		if !bytes.Equal(MarshalAnalysisBinary(&b), again) {
			t.Fatal("decode(encode(x)) is not x")
		}
	})
}
