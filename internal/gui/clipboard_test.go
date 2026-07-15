package gui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// chooseClipboardRung is the pure Fallback policy for the image-clipboard host
// capability. Because it is pure (facts in, rung out, no I/O), every combination
// of facts is exercised here without touching a real host — this is where the
// risk of the capability lives, per docs/adr/0004.
func TestChooseClipboardRung(t *testing.T) {
	cases := []struct {
		name string
		in   clipboardFacts
		want clipboardRung
	}{
		{
			name: "wayland with wl-copy prefers wl-copy",
			in:   clipboardFacts{HasWlCopy: true, IsWayland: true},
			want: rungWlCopy,
		},
		{
			name: "wayland with both tools still prefers wl-copy",
			in:   clipboardFacts{HasXclip: true, HasWlCopy: true, IsWayland: true},
			want: rungWlCopy,
		},
		{
			name: "wayland with only xclip falls back to xclip (XWayland)",
			in:   clipboardFacts{HasXclip: true, IsWayland: true},
			want: rungXclip,
		},
		{
			name: "wayland with no tool falls back to file",
			in:   clipboardFacts{IsWayland: true},
			want: rungFile,
		},
		{
			name: "x11 with xclip prefers xclip",
			in:   clipboardFacts{HasXclip: true},
			want: rungXclip,
		},
		{
			name: "x11 with both tools prefers xclip",
			in:   clipboardFacts{HasXclip: true, HasWlCopy: true},
			want: rungXclip,
		},
		{
			name: "x11 with only wl-copy falls back to wl-copy",
			in:   clipboardFacts{HasWlCopy: true},
			want: rungWlCopy,
		},
		{
			name: "x11 with no tool falls back to file",
			in:   clipboardFacts{},
			want: rungFile,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := chooseClipboardRung(tc.in); got != tc.want {
				t.Errorf("chooseClipboardRung(%+v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestChooseClipboardRungExhaustive guards the intent that the file rung is the
// only outcome when no clipboard tool is present, on either session type.
func TestChooseClipboardRungExhaustive(t *testing.T) {
	for _, wayland := range []bool{false, true} {
		for _, xclip := range []bool{false, true} {
			for _, wl := range []bool{false, true} {
				facts := clipboardFacts{HasXclip: xclip, HasWlCopy: wl, IsWayland: wayland}
				got := chooseClipboardRung(facts)
				hasTool := xclip || wl
				if !hasTool && got != rungFile {
					t.Errorf("no tool present but rung %d (facts %+v); want file", got, facts)
				}
				if hasTool && got == rungFile {
					t.Errorf("tool present but chose file rung (facts %+v)", facts)
				}
			}
		}
	}
}

func TestWriteBoardImage(t *testing.T) {
	dir := t.TempDir()
	png := []byte("\x89PNG\r\n\x1a\nfake-png-bytes")

	path, err := writeBoardImage(dir, png)
	if err != nil {
		t.Fatalf("writeBoardImage returned error: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("image written to %s, want inside %s", path, dir)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back saved image: %v", err)
	}
	if !bytes.Equal(got, png) {
		t.Errorf("saved bytes differ from input (%d vs %d bytes)", len(got), len(png))
	}
}
