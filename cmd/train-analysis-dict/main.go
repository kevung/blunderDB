// train-analysis-dict (re)generates the shared zstd dictionary embedded at
// pkg/blunderdb/engine/analysis_dict.bin, used to compress the analysis JSON
// blob (see pkg/blunderdb/engine/analysiscodec.go and ADR-0030 / #180). It is
// a dev-time asset generator, the same role scripts/build-demo-db.sh plays
// for the demo database — not part of the shipped binary, not a CLI
// subcommand.
//
// The corpus is real analysis data already in this repository: every match
// fixture under testdata/ (re-imported into a scratch database so their
// analyses get encoded exactly like production data) plus the positions in
// the embedded demo database (internal/gui/demo.db.gz). No user data ever
// leaves the machine; nothing here talks to the network.
//
// Method (see docs/recherche/P11-compression-blobs.md, "Protocole de mesure
// reproductible"): the corpus is split 80/train 20/test by a deterministic
// hash of a synthetic per-blob id, the dictionary is trained on the train
// split only with the reference `zstd --train` CLI (offline; nothing at
// runtime depends on the presence of a zstd binary — the shipped codec reads
// the trained bytes with the pure-Go github.com/klauspost/compress/zstd),
// and the reported compression ratio is measured on the held-out test split
// only, so the number printed is not inflated by measuring a dictionary on
// the very data it memorised.
//
// Usage:
//
//	go run ./cmd/train-analysis-dict [--dict-size 32768] [--out pkg/blunderdb/engine/analysis_dict.bin]
//
// Requires the `zstd` CLI (Debian/Ubuntu/Arch package `zstd`) on PATH.
// Regenerate only when the corpus changes meaningfully (new large fixtures,
// a PositionAnalysis field change) — not on every commit.
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "train-analysis-dict:", err)
		os.Exit(1)
	}
}

func run() error {
	dictSize := flag.Int("dict-size", 32768, "trained dictionary size in bytes (zstd --maxdict)")
	out := flag.String("out", "pkg/blunderdb/engine/analysis_dict.bin", "where to write the trained dictionary")
	testdataDir := flag.String("testdata", "testdata", "directory of match/position fixtures to import for the corpus")
	demoGz := flag.String("demo", "internal/gui/demo.db.gz", "gzip-compressed demo database to fold into the corpus")
	level := flag.Int("level", 19, "zstd level to measure the resulting dictionary at (informational only)")
	flag.Parse()

	if _, err := exec.LookPath("zstd"); err != nil {
		return fmt.Errorf("the zstd CLI is required to train the dictionary (offline only, not a runtime dependency): %w", err)
	}

	blobs, err := collectCorpus(*testdataDir, *demoGz)
	if err != nil {
		return err
	}
	if len(blobs) < 100 {
		return fmt.Errorf("corpus too small (%d blobs): zstd --train refuses under ~100 samples", len(blobs))
	}
	fmt.Printf("corpus: %d real analysis blobs\n", len(blobs))

	train, test := splitCorpus(blobs)
	fmt.Printf("split: %d train / %d test\n", len(train), len(test))

	workDir, err := os.MkdirTemp("", "train-analysis-dict-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	trainDir := filepath.Join(workDir, "train")
	if err := writeCorpusFiles(trainDir, train); err != nil {
		return err
	}

	dictPath := filepath.Join(workDir, "dict.bin")
	if err := trainDictionary(trainDir, dictPath, *dictSize); err != nil {
		return err
	}
	dict, err := os.ReadFile(dictPath)
	if err != nil {
		return err
	}

	if err := measure(dict, test, *level); err != nil {
		return err
	}

	if err := os.WriteFile(*out, dict, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", *out, err)
	}
	fmt.Printf("wrote %s (%d bytes)\n", *out, len(dict))
	return nil
}

// collectCorpus decompresses every real analysis JSON blob reachable from
// the repo's own fixtures: match files under testdataDir (re-imported into a
// scratch database) and the embedded demo database.
func collectCorpus(testdataDir, demoGz string) ([][]byte, error) {
	var blobs [][]byte

	tmp, err := os.MkdirTemp("", "train-analysis-dict-corpus-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	scratchDB := filepath.Join(tmp, "corpus.db")
	d := database.NewDatabase()
	if err := d.SetupDatabase(scratchDB); err != nil {
		return nil, fmt.Errorf("scratch database: %w", err)
	}
	if err := importMatchFixtures(d, testdataDir); err != nil {
		d.Close()
		return nil, err
	}
	d.Close()

	fromScratch, err := extractAnalysisBlobs(scratchDB)
	if err != nil {
		return nil, err
	}
	blobs = append(blobs, fromScratch...)

	if demoGz != "" {
		demoPath, cleanup, err := decompressDemoDB(demoGz, tmp)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		fromDemo, err := extractAnalysisBlobs(demoPath)
		if err != nil {
			return nil, err
		}
		blobs = append(blobs, fromDemo...)
	}

	return blobs, nil
}

// importMatchFixtures walks dir and imports every recognised match file
// (mirrors internal/cli's importMatch file-type dispatch) into d. A file that
// fails to import (e.g. testdata/bgf_positions/*.txt, which are excerpt
// snippets rather than full match files) is skipped: this tool wants volume
// of real analysis data, not a strict import audit.
func importMatchFixtures(d *database.Database, dir string) error {
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".xg":
			_, _ = d.ImportXGMatch(path)
		case ".sgf", ".mat":
			_, _ = d.ImportGnuBGMatch(path)
		case ".bgf":
			_, _ = d.ImportBGFMatch(path)
		case ".xgp":
			_, _ = d.ImportXGPPosition(path)
		}
		return nil
	})
}

// decompressDemoDB gunzips the embedded demo database (same shape as
// internal/gui.PrepareDemoDatabase) to a scratch file under dir.
func decompressDemoDB(gzPath, dir string) (path string, cleanup func(), err error) {
	f, err := os.Open(gzPath)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", nil, err
	}
	defer gz.Close()
	data, err := io.ReadAll(gz)
	if err != nil {
		return "", nil, err
	}
	out := filepath.Join(dir, "demo.db")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", nil, err
	}
	return out, func() {}, nil
}

// extractAnalysisBlobs reads every analysis.data row from the sqlite file at
// path and decompresses it (auto-detecting whichever codec produced it —
// raw JSON or zlib, at the time this tool runs there is no pre-existing
// zstd data) to recover the original PositionAnalysis JSON.
func extractAnalysisBlobs(path string) ([][]byte, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT data FROM analysis WHERE data IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out [][]byte
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		raw, err := engine.DecompressAnalysisData(data)
		if err != nil {
			continue // corrupt/oversized row: not useful training data anyway
		}
		if len(raw) == 0 {
			continue
		}
		out = append(out, raw)
	}
	return out, rows.Err()
}

// splitCorpus splits blobs 80/20 by a deterministic hash of each blob's own
// content (stable across runs and independent of slice order — no seeded
// PRNG, no reliance on map/slice iteration order), so a rerun of this tool
// with an unchanged corpus reproduces the exact same split every time.
func splitCorpus(blobs [][]byte) (train, test [][]byte) {
	for _, b := range blobs {
		sum := sha256.Sum256(b)
		if binary.BigEndian.Uint64(sum[:8])%5 == 0 {
			test = append(test, b)
		} else {
			train = append(train, b)
		}
	}
	return train, test
}

// writeCorpusFiles writes each blob as its own file under dir (zstd --train
// takes a list of sample files), named by content hash so the set is stable.
func writeCorpusFiles(dir string, blobs [][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(blobs))
	for _, b := range blobs {
		sum := sha256.Sum256(b)
		name := fmt.Sprintf("%x.json", sum[:8])
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return err
		}
		names = append(names, name)
	}
	sort.Strings(names) // cosmetic only; zstd --train globs the directory itself
	return nil
}

// trainDictionary shells out to the reference `zstd --train` (fastCover,
// the default — see P11 §"Coût CPU de la formation" for the fastCover vs
// trainFromBuffer trade-off). This is the one place in the whole feature
// that is not pure Go, and it only ever runs on a developer's machine at
// dictionary-generation time; the shipped codec loads the resulting bytes
// with klauspost/compress/zstd, no cgo, no external process.
func trainDictionary(trainDir, dictPath string, dictSize int) error {
	entries, err := os.ReadDir(trainDir)
	if err != nil {
		return err
	}
	samples := make([]string, 0, len(entries))
	for _, e := range entries {
		samples = append(samples, filepath.Join(trainDir, e.Name()))
	}
	args := append([]string{"--train"}, samples...)
	args = append(args, fmt.Sprintf("--maxdict=%d", dictSize), "-o", dictPath, "-f")
	cmd := exec.Command("zstd", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("zstd --train: %w", err)
	}
	return nil
}

// measure reports, on the held-out test split only, the compression ratio
// and per-blob latency the shipped codec (klauspost/compress/zstd, the given
// level, this dictionary) would actually achieve — the number that matters,
// not the ratio zstd --train prints for its own diagnostics.
func measure(dict []byte, test [][]byte, level int) error {
	enc, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)),
		zstd.WithEncoderDict(dict))
	if err != nil {
		return err
	}
	dec, err := zstd.NewReader(nil, zstd.WithDecoderDicts(dict))
	if err != nil {
		return err
	}

	var rawTotal, compressedTotal int
	for _, b := range test {
		c := enc.EncodeAll(b, nil)
		rawTotal += len(b)
		compressedTotal += len(c)
		got, err := dec.DecodeAll(c, nil)
		if err != nil || !bytes.Equal(got, b) {
			return fmt.Errorf("round-trip mismatch on a test blob: %w", err)
		}
	}
	ratio := float64(rawTotal) / float64(compressedTotal)
	fmt.Printf("held-out test set (%d blobs): %d -> %d bytes, ratio %.2fx (%.1f%% reduction)\n",
		len(test), rawTotal, compressedTotal, ratio, 100*(1-1/ratio))
	return nil
}
