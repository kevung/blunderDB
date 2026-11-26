package domain

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

const (
	arkStartCompact  = "[34]65:x2(ySQmLhfA)S3Qm3L3hf3"
	arkStartExpanded = "[34]65:x2(ymhf)m3hf3(SQLA)S3QL3"
)

// arkStart is the opening position, 3-away/4-away, 6-5 to play, the player on
// roll holding the cube at 4.
func arkStart(t *testing.T) Position {
	t.Helper()
	pos, err := DecodeXGID("XGID=-b----E-C---eE---c-e----B-:2:0:1:65:0:0:0:0:10")
	if err != nil {
		t.Fatal(err)
	}
	pos.Score = [2]int{3, 4}
	pos.Cube = Cube{Owner: Black, Value: 2}
	pos.PlayerOnRoll = Black
	return pos
}

func TestArkIDExamples(t *testing.T) {
	pos := arkStart(t)
	got, err := EncodeArkID(&pos)
	if err != nil || got != arkStartCompact {
		t.Fatalf("compact = %q, %v; want %q", got, err, arkStartCompact)
	}
	got, err = EncodeArkIDExpanded(&pos)
	if err != nil || got != arkStartExpanded {
		t.Fatalf("expanded = %q, %v; want %q", got, err, arkStartExpanded)
	}
	for _, in := range []string{arkStartCompact, arkStartExpanded, "ArkID=" + arkStartCompact, "(SQLA)S3QL3(ymhf)m3hf3"} {
		// the reader takes any order only inside a group of the same shape; the
		// last spelling puts the opponent first, which is still one Position.
		if strings.HasPrefix(in, "(") {
			in = "[34]65:x2" + in
		}
		back, err := DecodeArkID(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if !reflect.DeepEqual(back, pos) {
			t.Errorf("%q decoded to %+v, want %+v", in, back, pos)
		}
	}
}

func TestArkIDScoreForms(t *testing.T) {
	cases := map[string][2]int{
		"[$]":     {Unlimited, Unlimited},
		"[11-9]":  {11, 9},
		"[14c]":   {Crawford, 4},
		"[14]":    {PostCrawford, 4},
		"[2-13]":  {2, 13},
		"[11c]":   {Crawford, Crawford},
		"[41c]":   {4, Crawford},
		"[12-1c]": {12, Crawford},
	}
	for in, want := range cases {
		pos, err := DecodeArkID(in + "65:")
		if err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if pos.Score != want {
			t.Errorf("%s = %v, want %v", in, pos.Score, want)
		}
		back, err := EncodeArkID(&pos)
		if err != nil {
			t.Errorf("%s encode: %v", in, err)
			continue
		}
		if !strings.HasPrefix(back, in) {
			t.Errorf("%s re-encoded as %s", in, back)
		}
	}
}

func TestArkIDCubeDecisionAndCube(t *testing.T) {
	pos, err := DecodeArkID("[$]:X(mf)m3f3")
	if err != nil {
		t.Fatal(err)
	}
	if pos.DecisionType != CubeAction || pos.Cube.Owner != White || pos.Cube.Value != 1 {
		t.Fatalf("got %+v", pos)
	}
	if got, _ := EncodeArkID(&pos); got != "[$]:X(mf)m3f3" {
		t.Errorf("re-encoded as %s", got)
	}
	pos, err = DecodeArkID("[$]31:x0(mf)m3f3")
	if err != nil || pos.Cube.Owner != Black || pos.Cube.Value != 0 {
		t.Fatalf("x0: %+v, %v", pos, err)
	}
}

func TestArkIDBar(t *testing.T) {
	in := "[$]31:z2(mf)m3f3ZC"
	pos, err := DecodeArkID(in)
	if err != nil {
		t.Fatal(err)
	}
	if p := pos.Board.Points[25]; p.Checkers != 2 || p.Color != Black {
		t.Errorf("on-roll bar = %+v", p)
	}
	if p := pos.Board.Points[0]; p.Checkers != 1 || p.Color != White {
		t.Errorf("opponent bar = %+v", p)
	}
	got, _ := EncodeArkID(&pos)
	if want := "[$]31:(mf)z2m3f3CZ"; got != want {
		t.Errorf("re-encoded as %s, want %s", got, want)
	}
}

func TestArkIDErrors(t *testing.T) {
	bad := []string{
		"",
		"65:x",
		"[34",
		"[34]65",
		"[0$]65:",
		"[3]65:",
		"[35c]65:",
		"[34]75:",
		"[34]6:",
		"[34]65:x11",
		"[34]65:(f)f3(F)",     // a point held by both players
		"[34]65:fF",           // both players, one checker each
		"[34]65:(fm)f13m3",    // 15 + ... more than 15 for one player
		"[34]65:f3",           // three checkers need a gate
		"[34]65:(f)f(f)",      // a gate twice
		"[34]65:(z)",          // the bar is not a gate
		"[34]65:(x)",          // X is the cube
		"[34]65:(f",           // unclosed
		"[34]65:f)",           // unopened
		"[34]65:f0",           // count zero
		"[34]65:ff",           // a point twice
		"[34]65:z2z",          // a bar twice
		"[34]65:@",            // unknown character
		"[34]65:(fmhy)f13m13", // too many
	}
	for _, in := range bad {
		if _, err := DecodeArkID(in); !errors.Is(err, ErrInvalidArkID) {
			t.Errorf("%q: err = %v, want ErrInvalidArkID", in, err)
		}
	}
}

func TestArkIDRejectsMoreThan15(t *testing.T) {
	// 5 + 5 + 5 + 3 lowercase checkers
	if _, err := DecodeArkID("[$]:(mfhy)m3f3h3y"); !errors.Is(err, ErrInvalidArkID) {
		t.Errorf("err = %v", err)
	}
}

func TestArkIDLooksLike(t *testing.T) {
	for in, want := range map[string]bool{
		arkStartCompact:                       true,
		"  ArkID=" + arkStartExpanded:         true,
		"XGID=-b----E-C---eE---c-e----B-:0:0": false,
		"[note] hello":                        false,
		"[34]no colon":                        false,
		"4HPwATDgc/ABMA:MAEAAAAAAAAA":         false,
	} {
		if got := LooksLikeArkID(in); got != want {
			t.Errorf("LooksLikeArkID(%q) = %v", in, got)
		}
	}
}

// The GNUbg corpus draws boards, cubes and scores at random: every one must
// survive a round trip, and the written form must be stable.
func TestArkIDRoundTripCorpus(t *testing.T) {
	raw, err := os.ReadFile("testdata/gnubgid_reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []gnubgCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		pos := gnubgPosition(c)
		if pos.Cube.Owner == None && pos.DecisionType == CheckerAction {
			// gnubg draws centred cubes of any value; a real position has none.
			pos.Cube.Value = 0
		}
		id, err := EncodeArkID(&pos)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		back, err := DecodeArkID(id)
		if err != nil {
			t.Fatalf("%s: %q: %v", c.Name, id, err)
		}
		want := pos.NormalizeForStorage()
		want.HasJacoby, want.HasBeaver, want.MaxCube, want.ID = 0, 0, 0, 0
		for i, p := range want.Board.Points {
			if p.Checkers <= 0 {
				want.Board.Points[i] = Point{Color: None}
			}
		}
		want.DecisionType = CheckerAction
		if want.Dice[0] < 1 {
			want.DecisionType = CubeAction
		}
		want.Dice = [2]int{max(want.Dice[0], want.Dice[1]), min(want.Dice[0], want.Dice[1])}
		if !reflect.DeepEqual(back, want) {
			t.Errorf("%s: %q decoded to %+v, want %+v", c.Name, id, back, want)
		}
		again, _ := EncodeArkID(&back)
		expanded, _ := EncodeArkIDExpanded(&pos)
		fromExpanded, err := DecodeArkID(expanded)
		if err != nil {
			t.Fatalf("%s: %q: %v", c.Name, expanded, err)
		}
		third, _ := EncodeArkID(&fromExpanded)
		if again != id || third != id {
			t.Errorf("%s: not canonical: %q %q %q", c.Name, id, again, third)
		}
	}
}

func FuzzDecodeArkID(f *testing.F) {
	for _, s := range []string{arkStartCompact, arkStartExpanded, "[$]:", "[11-9]31:X(a)a3"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		pos, err := DecodeArkID(in)
		if err != nil {
			return
		}
		id, err := EncodeArkID(&pos)
		if err != nil {
			t.Fatalf("%q decoded but does not encode: %v", in, err)
		}
		back, err := DecodeArkID(id)
		if err != nil || !reflect.DeepEqual(back, pos) {
			t.Fatalf("%q -> %q does not round trip: %v", in, id, err)
		}
	})
}

func TestArkIDOfferedCube(t *testing.T) {
	for in, exp := range map[string]int{"[77]:X?(mf)m3f3": 1, "[77]:X2?(mf)m3f3": 2} {
		pos, err := DecodeArkID(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if !IsResponsePosition(&pos) || pos.Cube.Value != exp || pos.Cube.Owner != None {
			t.Errorf("%q: cube %+v, response %v", in, pos.Cube, IsResponsePosition(&pos))
		}
		if got, err := EncodeArkID(&pos); err != nil || got != in {
			t.Errorf("%q re-encoded as %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"[77]:x?(mf)m3f3", "[77]:X0?(mf)m3f3", "[77]31:X?(mf)m3f3"} {
		if _, err := DecodeArkID(in); !errors.Is(err, ErrInvalidArkID) {
			t.Errorf("%q: err = %v", in, err)
		}
	}
}

func TestArkIDRefusesWhatItWouldLose(t *testing.T) {
	pos := arkStart(t)
	pos.DecisionType, pos.Cube = CubeAction, Cube{Owner: None}
	if _, err := EncodeArkID(&pos); err == nil {
		t.Error("a cube decision with dice was written, losing them")
	}
	pos = arkStart(t)
	pos.Cube = Cube{Owner: None, Value: 2}
	if _, err := EncodeArkID(&pos); err == nil {
		t.Error("a centred cube above 1 on a checker decision was written")
	}
}
