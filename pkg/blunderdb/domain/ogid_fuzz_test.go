package domain

import (
	"errors"
	"strings"
	"testing"
)

// FuzzDecodeOGID holds the OGID reader to the contract of every identifier a
// user pastes or a CLI argument carries: no panic, an ErrInvalidOGID-wrapped
// error for what it refuses, and — for what it accepts — a spelling stable
// under one more decode/encode, so the same position always writes the same
// identifier. DecodePositionID, which picks OGID or XGID from the text, is
// driven with the same input.
func FuzzDecodeOGID(f *testing.F) {
	seeds := []string{
		"",
		"OGID=",
		"11ccccchhhjjjjj:66666888dddddoo:N0N:51:B::0:0:7:",
		"12cccchhhhjjjjj:66666888dddddoo:N0N::B::0:0:7:",
		"cccccggggg:ddddiiiiii:N0N:63:W:IW:4:3:7:1:15",
		"OGID=cccccggggg:ddddiiiiii:W9D:63:W:IW:4:3:7C:1:15",
		"::::::::::",
		"0:p:N0N::W::0:0:0:",
		strings.Repeat("0", 40) + ":" + strings.Repeat("p", 40) + ":N0N::W::0:0:1:",
		"XGID=a--aB-BBA--acDa-Ab-db---BA:0:0:1:64:2:0:0:13:10",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = DecodePositionID(s)

		pos, err := DecodeOGID(s)
		if err != nil {
			return
		}
		// A cube past 512 has no OGID spelling (EncodeOGID's doc); the
		// reader never produces one, so this guard only documents the edge.
		if pos.Cube.Value > 9 {
			return
		}
		first := EncodeOGID(&pos)
		again, err := DecodeOGID(first)
		if err != nil {
			t.Fatalf("DecodeOGID(%q) accepted, but its re-encoding %q is refused: %v", s, first, err)
		}
		if second := EncodeOGID(&again); second != first {
			t.Fatalf("OGID spelling unstable for input %q:\n first  %q\n second %q", s, first, second)
		}
	})
}

// A score that has already reached the match length leaves no decision to
// store; accepting it produced a negative away score.
func TestDecodeOGID_RefusesAWonScore(t *testing.T) {
	for _, s := range []string{
		"::000::::7:7:1",
		"cccccggggg:ddddiiiiii:N0N:63:W::7:3:7:",
		"cccccggggg:ddddiiiiii:N0N:63:W::-1:3:7:",
	} {
		if _, err := DecodeOGID(s); !errors.Is(err, ErrInvalidOGID) {
			t.Errorf("DecodeOGID(%q) err = %v, want ErrInvalidOGID", s, err)
		}
	}
	if _, err := DecodeOGID("cccccggggg:ddddiiiiii:N0N:63:W::6:3:7:"); err != nil {
		t.Errorf("6-3 in a 7-point match is a live score: %v", err)
	}
}
