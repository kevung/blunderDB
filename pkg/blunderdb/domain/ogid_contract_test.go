package domain

import (
	"encoding/json"
	"os"
	"testing"
)

// The OGID contract (issue #260, fiche I.4).
//
// Every case pairs an OGID with the XGID of the SAME physical position, and
// this test asserts that the two decoders agree. Pinning the new reader
// against the old one — rather than against a table of expectations somebody
// typed out — is what makes the corpus a contract instead of a transcription
// of one person's reading of a specification.
//
// The strings themselves come from the reference implementation (AnkiGammon
// 1.8.1), over positions dumped from testdata/test.xg and testdata/test.mat.
// That provenance is the point: the fiche forbade writing this reader before
// real samples existed, because the format it originally named turned out not
// to exist at all.

type ogidCorpus struct {
	Cases []struct {
		XGID string `json:"xgid"`
		OGID string `json:"ogid"`
	} `json:"cases"`
	// Reference holds what AnkiGammon's own parse_ogid returned for each
	// string (see the corpus's _reference_comment).
	Reference []struct {
		OGID          string  `json:"ogid"`
		White         [26]int `json:"white"`
		Black         [26]int `json:"black"`
		CubeOwner     string  `json:"cube_owner"`
		CubeValue     int     `json:"cube_value"`
		Dice          *[2]int `json:"dice"`
		OnRoll        *string `json:"on_roll"`
		ScoreWhite    *int    `json:"score_white"`
		ScoreBlack    *int    `json:"score_black"`
		MatchLength   *int    `json:"match_length"`
		MatchModifier *string `json:"match_modifier"`
		XGID          string  `json:"xgid"`
	} `json:"reference"`
}

func loadOGIDCorpus(t *testing.T) ogidCorpus {
	t.Helper()
	raw, err := os.ReadFile("testdata/ogid_corpus.json")
	if err != nil {
		raw, err = os.ReadFile("../../../testdata/ogid_corpus.json")
	}
	if err != nil {
		t.Fatalf("reading the corpus: %v", err)
	}
	var corpus ogidCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("parsing the corpus: %v", err)
	}
	if len(corpus.Cases) == 0 || len(corpus.Reference) == 0 {
		t.Fatal("the corpus is empty")
	}
	return corpus
}

// ogidXGIDPairs is every OGID of the corpus with the XGID of the same
// position: the cases, then the reference entries.
func ogidXGIDPairs(corpus ogidCorpus) [][2]string {
	var pairs [][2]string
	for _, c := range corpus.Cases {
		pairs = append(pairs, [2]string{c.OGID, c.XGID})
	}
	for _, r := range corpus.Reference {
		pairs = append(pairs, [2]string{r.OGID, r.XGID})
	}
	return pairs
}

func TestDecodeOGIDMatchesDecodeXGID(t *testing.T) {
	corpus := loadOGIDCorpus(t)

	for _, pair := range ogidXGIDPairs(corpus) {
		c := struct{ OGID, XGID string }{pair[0], pair[1]}
		want, err := DecodeXGID(c.XGID)
		if err != nil {
			t.Fatalf("DecodeXGID(%s): %v", c.XGID, err)
		}
		got, err := DecodeOGID(c.OGID)
		if err != nil {
			t.Fatalf("DecodeOGID(%s): %v", c.OGID, err)
		}

		if got.Board.Points != want.Board.Points {
			t.Errorf("board differs\n OGID %s\n XGID %s\n  got %v\n want %v",
				c.OGID, c.XGID, got.Board.Points, want.Board.Points)
			continue
		}
		if got.Board.Bearoff != want.Board.Bearoff {
			t.Errorf("bearoff differs for %s: got %v, want %v", c.OGID, got.Board.Bearoff, want.Board.Bearoff)
		}
		if got.Cube != want.Cube {
			t.Errorf("cube differs for %s: got %+v, want %+v", c.OGID, got.Cube, want.Cube)
		}
		if got.Dice != want.Dice {
			t.Errorf("dice differ for %s: got %v, want %v", c.OGID, got.Dice, want.Dice)
		}
		if got.PlayerOnRoll != want.PlayerOnRoll {
			t.Errorf("player on roll differs for %s: got %d, want %d", c.OGID, got.PlayerOnRoll, want.PlayerOnRoll)
		}
		if got.Score != want.Score {
			t.Errorf("score differs for %s: got %v, want %v", c.OGID, got.Score, want.Score)
		}
		if got.DecisionType != want.DecisionType {
			t.Errorf("decision type differs for %s: got %d, want %d", c.OGID, got.DecisionType, want.DecisionType)
		}
	}
}

// TestDecodeOGIDMatchesReference holds the decoder to the reference
// implementation's own reading of each string, field by field — the check
// that does not go through XGID at all, and so the one that settles the
// orientation question (White on roll on an asymmetric board is in there).
func TestDecodeOGIDMatchesReference(t *testing.T) {
	corpus := loadOGIDCorpus(t)
	for _, r := range corpus.Reference {
		got, err := DecodeOGID(r.OGID)
		if err != nil {
			t.Fatalf("DecodeOGID(%s): %v", r.OGID, err)
		}
		var onBoard [2]int
		for i := 0; i < 26; i++ {
			want := Point{Checkers: 0, Color: None}
			switch {
			case r.White[i] > 0:
				want = Point{Checkers: r.White[i], Color: White}
			case r.Black[i] > 0:
				want = Point{Checkers: r.Black[i], Color: Black}
			}
			if got.Board.Points[i] != want {
				t.Errorf("%s: point %d = %+v, reference %+v", r.OGID, i, got.Board.Points[i], want)
			}
			onBoard[White] += r.White[i]
			onBoard[Black] += r.Black[i]
		}
		if got.Board.Bearoff != [2]int{15 - onBoard[0], 15 - onBoard[1]} {
			t.Errorf("%s: bearoff %v, reference on board %v", r.OGID, got.Board.Bearoff, onBoard)
		}

		wantOwner := map[string]int{"N": None, "W": White, "B": Black}[r.CubeOwner]
		if got.Cube.Owner != wantOwner || 1<<got.Cube.Value != r.CubeValue {
			t.Errorf("%s: cube %+v, reference owner %s value %d", r.OGID, got.Cube, r.CubeOwner, r.CubeValue)
		}

		if r.Dice != nil {
			if got.Dice != *r.Dice || got.DecisionType != CheckerAction {
				t.Errorf("%s: dice %v (type %d), reference %v", r.OGID, got.Dice, got.DecisionType, *r.Dice)
			}
		} else if got.Dice != [2]int{} || got.DecisionType != CubeAction {
			t.Errorf("%s: dice %v (type %d), reference none: a cube decision", r.OGID, got.Dice, got.DecisionType)
		}

		// No turn: the reference's own input dialog falls back to White.
		wantRoll := White
		if r.OnRoll != nil && *r.OnRoll == "B" {
			wantRoll = Black
		}
		if got.PlayerOnRoll != wantRoll {
			t.Errorf("%s: on roll %d, want %d", r.OGID, got.PlayerOnRoll, wantRoll)
		}

		// The reference reads Crawford exactly from the "C" modifier; the one
		// departure is the 1-point match, always its own Crawford game (#411).
		wantScore := [2]int{Unlimited, Unlimited}
		if r.MatchLength != nil && *r.MatchLength > 0 && r.ScoreWhite != nil && r.ScoreBlack != nil {
			crawford := (r.MatchModifier != nil && *r.MatchModifier == "C") || *r.MatchLength == 1
			wantScore = AwayScoresWithCrawford(*r.MatchLength, *r.ScoreBlack, *r.ScoreWhite, crawford)
		}
		if got.Score != wantScore {
			t.Errorf("%s: away score %v, want %v", r.OGID, got.Score, wantScore)
		}
	}
}

// The canonical example — the one AnkiGammon's README gives — spelled out, so
// that the reading of every field is visible in one place and not only as a
// comparison against another decoder.
func TestDecodeOGIDCanonicalExample(t *testing.T) {
	pos, err := DecodeOGID("cccccggggg:ddddiiiiii:N0N:63:W:IW:4:3:7:1:15")
	if err != nil {
		t.Fatal(err)
	}
	if pos.Board.Points[12] != (Point{5, White}) || pos.Board.Points[16] != (Point{5, White}) ||
		pos.Board.Points[13] != (Point{4, Black}) || pos.Board.Points[18] != (Point{6, Black}) {
		t.Errorf("board: %v", pos.Board.Points)
	}
	if pos.Board.Bearoff != [2]int{5, 5} {
		t.Errorf("bearoff %v, want 5 off each", pos.Board.Bearoff)
	}
	if pos.Cube != (Cube{Value: 0, Owner: None}) {
		t.Errorf("cube %+v, want centred at 1", pos.Cube)
	}
	if pos.Dice != [2]int{6, 3} || pos.DecisionType != CheckerAction || pos.PlayerOnRoll != White {
		t.Errorf("dice %v type %d on roll %d, want White to play 63", pos.Dice, pos.DecisionType, pos.PlayerOnRoll)
	}
	// White 4, Black 3 in a 7-point match: Black is 4 away, White 3.
	if pos.Score != [2]int{4, 3} {
		t.Errorf("away score %v, want [4 3] (Black, White)", pos.Score)
	}
}

// The Crawford modifier is the only thing that tells the Crawford game from
// the games after it, at the same score.
func TestDecodeOGIDCrawfordModifier(t *testing.T) {
	const board = "11jjjjjhhhccccc:ooddddd88866666:N0N:65:W::6:5:"
	for ml, want := range map[string][2]int{
		"7C":   {2, Crawford},     // White 6-5 up, one point away, Crawford game
		"7":    {2, PostCrawford}, // same score, no C: after the Crawford game
		"7L":   {2, PostCrawford}, // L and G say nothing of Crawford
		"7G15": {2, PostCrawford},
	} {
		pos, err := DecodeOGID(board + ml + ":")
		if err != nil {
			t.Fatalf("%s: %v", ml, err)
		}
		if pos.Score != want {
			t.Errorf("match field %q: away %v, want %v", ml, pos.Score, want)
		}
	}
	// A 1-point match is its own Crawford game whatever the modifier (#411).
	pos, err := DecodeOGID("11jjjjjhhhccccc:ooddddd88866666:N0N::W::0:0:1:")
	if err != nil {
		t.Fatal(err)
	}
	if pos.Score != [2]int{Crawford, Crawford} {
		t.Errorf("1-point match: away %v, want [1 1]", pos.Score)
	}
}

// EncodeOGID is DecodeOGID's inverse on everything a Position keeps.
func TestEncodeOGIDRoundTrip(t *testing.T) {
	corpus := loadOGIDCorpus(t)
	for _, pair := range ogidXGIDPairs(corpus) {
		pos, err := DecodeOGID(pair[0])
		if err != nil {
			t.Fatalf("DecodeOGID(%s): %v", pair[0], err)
		}
		enc := EncodeOGID(&pos)
		back, err := DecodeOGID(enc)
		if err != nil {
			t.Fatalf("DecodeOGID(EncodeOGID(%s)) = DecodeOGID(%s): %v", pair[0], enc, err)
		}
		if back != pos {
			t.Errorf("round trip changed the position\n  in  %s\n  out %s\n  %+v\n  %+v", pair[0], enc, pos, back)
		}
		// The same position reached from its XGID encodes the same way: the
		// encoder reads the Position, not the text it came from.
		fromXGID, err := DecodeXGID(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		fromXGID.MaxCube, fromXGID.HasJacoby, fromXGID.HasBeaver = 0, 0, 0 // not in OGID
		if got := EncodeOGID(&fromXGID); got != enc {
			t.Errorf("%s: EncodeOGID from the XGID %q, from the OGID %q", pair[1], got, enc)
		}
	}
}

// The encoder writes what the reference's encode_ogid writes: sorted
// characters, the cube action N, no game state nor move id, the checker count
// left out, and the smallest match the away scores imply (4-3 in a 7-point
// match is 1-0 in a 4-point one: the same distances).
func TestEncodeOGIDSpelling(t *testing.T) {
	for in, want := range map[string]string{
		// AnkiGammon's test_encode_position_only_ogid expects exactly these
		// first three fields for the starting position.
		"11jjjjjhhhccccc:ooddddd88866666:N0N":                   "11ccccchhhjjjjj:66666888dddddoo:N0N::W::0:0::",
		"cccccggggg:ddddiiiiii:N0N:63:W:IW:4:3:7:1:15":          "cccccggggg:ddddiiiiii:N0N:63:W::1:0:4:",
		"jjjjkk:od88866:W2O:43:B:IW:2:1:7:15":                   "jjjjkk:66888do:W2N:43:B::1:0:6:",
		"11jjjjjhhhccccc:ooddddd88866666:N0N:65:W:IW:6:5:7C:42": "11ccccchhhjjjjj:66666888dddddoo:N0N:65:W::1:0:2C:",
		// Post-Crawford 1-away both sides: a 1-point match would say Crawford.
		"11jjjjjhhhccccc:ooddddd88866666:N0N::B::6:6:7:": "11ccccchhhjjjjj:66666888dddddoo:N0N::B::1:1:2:",
		"11jjjjjhhhccccc:ooddddd88866666:N0N::B::0:0:1:": "11ccccchhhjjjjj:66666888dddddoo:N0N::B::0:0:1C:",
	} {
		pos, err := DecodeOGID(in)
		if err != nil {
			t.Fatalf("DecodeOGID(%s): %v", in, err)
		}
		if got := EncodeOGID(&pos); got != want {
			t.Errorf("EncodeOGID(DecodeOGID(%s))\n  = %s\n want %s", in, got, want)
		}
	}
}

// A malformed OGID is refused with a named error, never decoded into a
// plausible-looking board — the failure mode a lax reader produces is a
// position nobody can trace back to anything.
func TestDecodeOGIDRefusesMalformed(t *testing.T) {
	for name, s := range map[string]string{
		"empty":            "",
		"two fields":       "11ccccc:66666",
		"short cube":       "11ccccc:66666:N0",
		"bad character":    "11ccc!c:66666888dddddoo:N0N",
		"too many on one":  "1111111111111111:66666888dddddoo:N0N",
		"colours collided": "1111:1111:N0N",
	} {
		if _, err := DecodeOGID(s); err == nil {
			t.Errorf("%s: expected a refusal for %q", name, s)
		}
	}
}

// The router must never have to ask which format a paste is in.
func TestLooksLikeOGID(t *testing.T) {
	yes := []string{
		"OGID=11ccccchhhjjjjj:66666888dddddoo:N0N::B::0:0:7:",
		"11ccccchhhjjjjj:66666888dddddoo:N0N:51:B::0:0:7:",
		"  ogid=11ccccchhhjjjjj:66666888dddddoo:W2N  ",
	}
	for _, s := range yes {
		if !LooksLikeOGID(s) {
			t.Errorf("LooksLikeOGID(%q) = false, want true", s)
		}
	}
	no := []string{
		"",
		"XGID=-b----E-C---eE---c-e----B-:0:0:1:51:0:0:0:7:0",
		"-b----E-C---eE---c-e----B-:0:0:1:51:0:0:0:7:0", // an XGID's third field is a number
		"just some prose about a position",
	}
	for _, s := range no {
		if LooksLikeOGID(s) {
			t.Errorf("LooksLikeOGID(%q) = true, want false", s)
		}
	}
}

// DecodePositionID routes each identifier to its own reader: the corpus's
// OGID and XGID of one position come back as the same position.
func TestDecodePositionIDRoutesBothFormats(t *testing.T) {
	corpus := loadOGIDCorpus(t)
	for _, pair := range ogidXGIDPairs(corpus) {
		fromOGID, err := DecodePositionID(pair[0])
		if err != nil {
			t.Fatalf("DecodePositionID(%s): %v", pair[0], err)
		}
		fromXGID, err := DecodePositionID(pair[1])
		if err != nil {
			t.Fatalf("DecodePositionID(%s): %v", pair[1], err)
		}
		if fromOGID.Board != fromXGID.Board || fromOGID.Score != fromXGID.Score ||
			fromOGID.Cube != fromXGID.Cube || fromOGID.PlayerOnRoll != fromXGID.PlayerOnRoll {
			t.Errorf("%s and %s decode to different positions", pair[0], pair[1])
		}
	}
	// The two spellings `blunderdb epc --help` shows side by side.
	x, errX := DecodePositionID("XGID=-BBBB----------------bbbb-:0:0:1:00:0:0:0:0:10")
	o, errO := DecodePositionID("llmmnnoo:11223344:N0N::B::0:0::")
	if errX != nil || errO != nil || x.Board != o.Board || x.PlayerOnRoll != o.PlayerOnRoll || x.Score != o.Score {
		t.Errorf("epc's help examples differ: %+v / %+v (%v, %v)", x, o, errX, errO)
	}
	if _, err := DecodePositionID("not an identifier"); err == nil {
		t.Error("DecodePositionID accepted prose")
	}
}
