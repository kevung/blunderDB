package ingest

import (
	"os"
	"strings"
	"testing"
)

func TestMapGnuBGText_TruncatedMATIsRefused(t *testing.T) {
	b, err := os.ReadFile("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.mat")
	if err != nil {
		t.Fatal(err)
	}
	full := string(b)
	if _, err := MapGnuBGText(full); err != nil {
		t.Fatalf("whole file refused: %v", err)
	}
	for _, f := range []float64{0.3, 0.5, 0.7, 0.9} {
		g, err := MapGnuBGText(full[:int(float64(len(full))*f)])
		if err == nil || g != nil {
			t.Errorf("cut at %.0f%%: want an error and no graph, got err=%v", f*100, err)
		}
	}
}

func TestMapGnuBG_TruncatedMATFileIsRefused(t *testing.T) {
	b, err := os.ReadFile("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.mat")
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/cut.mat"
	if err := os.WriteFile(path, b[:len(b)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := MapGnuBG(path); err == nil {
		t.Fatal("truncated .mat file accepted")
	}
}

func TestMapBGFTextPosition_UnreadableTextIsRefused(t *testing.T) {
	for _, txt := range []string{"", "ceci ne contient ni match ni position", "a\nb\nc\n"} {
		if g, err := MapBGFTextPositionText(txt); err == nil || g != nil {
			t.Errorf("%q: want an error, got err=%v", txt, err)
		}
	}
	path := t.TempDir() + "/x.txt"
	if err := os.WriteFile(path, []byte(strings.Repeat("rien\n", 5)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := MapBGFTextPosition(path); err == nil {
		t.Fatal("unrelated .txt file accepted")
	}
}
