package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// goldenZstdBlobs are three analysis blobs exactly as they sit in the
// embedded demo database (internal/gui/demo.db.gz, analysis ids 330, 409 and
// 275 — the smallest, the median and the largest of its 723 rows), written by
// klauspost/compress v1.18.7 with analysis_dict.bin. They are copied here as
// bytes, not regenerated, so a later version of the library, or a new
// assembly decoder on another architecture (arm64 since v1.19), has to read
// what an earlier binary put into a user's file (ADR-0030: a blob names its
// codec, never the library version that wrote it).
//
// The expected output is pinned by length and SHA-256 rather than by the JSON
// itself: the check is byte for byte, the file stays short.
var goldenZstdBlobs = []struct {
	name    string
	blob    string // hex
	jsonLen int
	jsonSHA string
}{
	{
		name:    "demo-330-cube",
		blob:    "28b52ffd4700f29d09422602fd0100040233333037543434393835363536362e3031323639373237312b30323a3030227d08005c15c0fbfc0db30bf07b5a7c3ff90ce5089da9c0965052732d53208ab78f1d",
		jsonLen: 806,
		jsonSHA: "b95e209830214bfdf694e97a56bb35311b34424d2b8081714419e5739efeb677",
	},
	{
		name:    "demo-409",
		blob:    "28b52ffd67f29d09428208bd070022450f13c0910cd1d01bab524a2f7d4a29a5fd9fdb7e8c83613b1d61b27e5566b17d210ef213288174f2c6cb6d8afaaeab8537e60c661040289c4575eb04335543800077a3e8005c1980fbcd24ea356bc0c4d31340b67a1ad2bc2216a42149ac8ba38cb51d725b408aaea470a0949622f920f16195294885a425161200745076410a2427165dac80ca304040024047c44a4582cecab6807179b3fe8d12e2c5f942c5c671c96791377c457cc58ecef7af43952a11becc10640101ac54ab528f43f913a466914f0ee9936fff640750ca8072fa732b505639e70c19f2c4945cfbfe12eaef6cf114f2a0254acfc11b379425133370e96b79a9a74e5be1",
		jsonLen: 2434,
		jsonSHA: "79c69dff7154eb179e343fef229ffeaadcaff38399bc37b23111288b7b8c8a2e",
	},
	{
		name:    "demo-275",
		blob:    "28b52ffd67f29d0942740dad0f0072ce21159007d80ccb0647c46ed9bd53a694137ab7f491042e8520325011a0b7f7f448979bb39fc06ebc1b9cb74adce0d13cc269bdf0ef8e36eeb0f5924f783e8e7870ca6e008340548d3af9f04368388aec12880e333000a2183ded244f89b772005159376b57d15051536c8ab76d93ea951a8c78fba6f2d0c10d36f01c658a7a8a29a614c86f9446d6808aa0c052d5560060240400229aea01fc17c0fd2dbc4e3e272d333a36ee6e2e8eae6178963242ea701d19994c971260a8e7fae40e4aa643c72efa9c731f87a4c79658cad484703e56eb01652479e99ea95b12452d21de840d1917d0b921571c8835775eed17cc17012c5a8f9f1f0741688a832aaedfc928202dbc82a1ad9b41523344ce7277be56305bdce1e50113649236dbd0baa658aea6df244a024de4bb5766eb938943a3113460520b7b75a5e0a0207717329276a1ce2f4f8f601787397ae8b1ff2143b2d460d6623163ef9f83c81462aeb6f6744dda21e4106ef4cca6057d8e1924948de806bfe84248baeb13485443906afa463e603fecee9a2d58ae0ddc80c9cdace904d1a737092866ed1d082266ff2f06f42b9d98e2045b11776a648ceb619ddc4c04055209121f040544a1459ccd70cc84e184833279966f81504e25976b87855d4203402a796ddaed6c669769d9b096505ee50eac4714f0d2032b8436c9d3a54c5ccb1804b913393d",
		jsonLen: 3700,
		jsonSHA: "befd7072eda32cefc6c1cdb76f219c9a0fb3f25bb3ac91357d5ec4624249ed12",
	},
}

// TestGoldenZstdBlobsDecodeByteForByte reads blobs an earlier binary wrote and
// requires the exact JSON back. Only decoding is pinned: the encoder's output
// may change between library versions (ADR-0030 promises nothing about it),
// so a re-encoded blob is required to decode back to the same JSON, not to
// reproduce the stored bytes.
func TestGoldenZstdBlobsDecodeByteForByte(t *testing.T) {
	for _, g := range goldenZstdBlobs {
		t.Run(g.name, func(t *testing.T) {
			blob, err := hex.DecodeString(g.blob)
			if err != nil {
				t.Fatalf("fixture hex: %v", err)
			}
			if !isZstdFrame(blob) {
				t.Fatalf("fixture is not a zstd frame")
			}
			got, err := DecompressAnalysisData(blob)
			if err != nil {
				t.Fatalf("DecompressAnalysisData: %v", err)
			}
			sum := sha256.Sum256(got)
			if len(got) != g.jsonLen || hex.EncodeToString(sum[:]) != g.jsonSHA {
				t.Fatalf("decoded JSON differs: len %d sha %x, want len %d sha %s",
					len(got), sum, g.jsonLen, g.jsonSHA)
			}
			if _, err := DecodeAnalysisFromStorage(blob); err != nil {
				t.Fatalf("DecodeAnalysisFromStorage: %v", err)
			}

			again, err := CompressAnalysisData(got)
			if err != nil {
				t.Fatalf("CompressAnalysisData: %v", err)
			}
			back, err := DecompressAnalysisData(again)
			if err != nil {
				t.Fatalf("re-encoded blob does not decode: %v", err)
			}
			if !bytes.Equal(back, got) {
				t.Fatalf("re-encoded blob decodes to different JSON")
			}
			if !bytes.Equal(again, blob) {
				t.Logf("encoder output changed (%d -> %d bytes); allowed by ADR-0030", len(blob), len(again))
			}
		})
	}
}
