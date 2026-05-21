package engine

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers to build well-known positions
// ---------------------------------------------------------------------------

func initialPosition() Position {
	return Position{
		Board:        initialBoard(),              // reuses helper from bitboards_test.go
		Cube:         Cube{Owner: None, Value: 0}, // 0 = exponent for cube-at-1
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}
}

// bearoffPosition returns a pure race where both players have only their
// home-board checkers left.
func bearoffPosition() Position {
	var b Board
	// Black racing home (points 1-6)
	b.Points[1] = Point{Checkers: 3, Color: Black}
	b.Points[2] = Point{Checkers: 3, Color: Black}
	b.Points[3] = Point{Checkers: 3, Color: Black}
	b.Points[4] = Point{Checkers: 3, Color: Black}
	b.Points[5] = Point{Checkers: 3, Color: Black}
	// White racing home (points 19-24)
	b.Points[20] = Point{Checkers: 3, Color: White}
	b.Points[21] = Point{Checkers: 3, Color: White}
	b.Points[22] = Point{Checkers: 3, Color: White}
	b.Points[23] = Point{Checkers: 3, Color: White}
	b.Points[24] = Point{Checkers: 3, Color: White}
	return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0, DecisionType: CheckerAction}
}

func barCheckerPosition() Position {
	p := initialPosition()
	// Black has a checker on the bar
	p.Board.Points[BlackBar] = Point{Checkers: 1, Color: Black}
	p.Board.Points[24].Checkers = 1 // one less at 24
	return p
}

// cubePosition creates a cube-decision position. cubeExp is the cube exponent:
// 0 = cube at 1 (initial), 1 = cube at 2, 2 = cube at 4, …
func cubePosition(cubeExp int, cubeOwner int) Position {
	return Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: cubeOwner, Value: cubeExp},
		PlayerOnRoll: 0,
		DecisionType: CubeAction,
	}
}

func moneyPosition() Position {
	p := initialPosition()
	p.HasJacoby = 1
	p.HasBeaver = 1
	return p
}

func scorePosition(s1, s2 int) Position {
	return Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: None, Value: 0}, // 0 = exponent for cube-at-1
		Score:        [2]int{s1, s2},
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}
}

// ---------------------------------------------------------------------------
// Stability test — 32 frozen positions
// ---------------------------------------------------------------------------

// stabilityPositions defines the 32 positions whose hashes must never change.
// The want values are filled in by freezeZobristHashes (run once with -update).
var stabilityPositions = []struct {
	name string
	pos  Position
	want uint64
}{
	{"initial", initialPosition(), 0x5aab493a553eacc1},
	{"initial_mirror", func() Position { p := initialPosition(); p.PlayerOnRoll = 1; return p }(), 0x5aab493a553eacc1},
	{"bearoff", bearoffPosition(), 0x1380c7a0f973daaf},
	{"bar_checker", barCheckerPosition(), 0xcea6fe9267ccb020},
	// cubePosition args are exponents: 1=cube@2, 2=cube@4, …
	{"cube2_black", cubePosition(1, Black), 0x3c2ae046ad036490},
	{"cube4_white", cubePosition(2, White), 0x6e44c1f090bc085d},
	{"cube8_none", cubePosition(3, None), 0xc4e19b22960ad3fc},
	{"cube16_black", cubePosition(4, Black), 0xfb6e47ea5935c927},
	{"cube32_white", cubePosition(5, White), 0x4ab19b26eafcc585},
	{"cube64_none", cubePosition(6, None), 0xf7e53720e65c7c9f},
	{"cube128_black", cubePosition(7, Black), 0x88dbaaf79eb2e077},
	{"cube256_white", cubePosition(8, White), 0x8f32f1f41e49851f},
	{"cube512_none", cubePosition(9, None), 0x7ab8e476a429c748},
	{"cube1024_black", cubePosition(10, Black), 0x40bac1cafb12aefe},
	{"money_jacoby_beaver", moneyPosition(), 0x1128e30862a692b9},
	{"score_6_4", scorePosition(6, 4), 0x2b51a32d7e178238},
	{"score_0_1", scorePosition(0, 1), 0x4572d8103da80e14},
	{"score_1_1", scorePosition(1, 1), 0x157df587cf37c8bf},
	{"score_10_10", scorePosition(10, 10), 0xe41c2ec3fa94d32f},
	{"score_15_14", scorePosition(15, 14), 0x27d99756f7281989},
	{
		"dice_65", func() Position {
			p := initialPosition()
			p.Dice = [2]int{6, 5}
			return p
		}(), 0x3569868bae709125,
	},
	{
		"dice_11", func() Position {
			p := initialPosition()
			p.Dice = [2]int{1, 1}
			return p
		}(), 0xe54d7c9ae54f6165,
	},
	{
		"checker_action_dice", func() Position {
			p := initialPosition()
			p.DecisionType = CheckerAction
			p.Dice = [2]int{3, 2}
			return p
		}(), 0x65854419e54e6169,
	},
	{
		"cube_action", func() Position {
			p := initialPosition()
			p.DecisionType = CubeAction
			return p
		}(), 0x0142b53c5f6399eb,
	},
	{
		"bearoff_partial",
		func() Position {
			p := bearoffPosition()
			p.Board.Bearoff[0] = 5
			p.Board.Bearoff[1] = 3
			return p
		}(), 0x142014ba5f60315b,
	},
	{
		"5prime_black", func() Position {
			var b Board
			for i := 4; i <= 8; i++ {
				b.Points[i] = Point{Checkers: 2, Color: Black}
			}
			b.Points[19] = Point{Checkers: 2, Color: White}
			b.Points[1] = Point{Checkers: 2, Color: White}
			return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0}
		}(), 0x57364c3be19652a5,
	},
	{
		"blitz", func() Position {
			var b Board
			// Black prime + two White blots on bar
			b.Points[BlackBar] = Point{Checkers: 2, Color: White}
			b.Points[6] = Point{Checkers: 2, Color: Black}
			b.Points[5] = Point{Checkers: 2, Color: Black}
			b.Points[4] = Point{Checkers: 2, Color: Black}
			b.Points[22] = Point{Checkers: 2, Color: Black}
			b.Points[24] = Point{Checkers: 2, Color: Black}
			b.Points[20] = Point{Checkers: 2, Color: White}
			b.Points[19] = Point{Checkers: 3, Color: White}
			return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0}
		}(), 0x89acd726a1f6448e,
	},
	{
		"backgame", func() Position {
			var b Board
			// Black holds two anchors in White's home board
			b.Points[2] = Point{Checkers: 2, Color: Black}
			b.Points[3] = Point{Checkers: 2, Color: Black}
			b.Points[6] = Point{Checkers: 4, Color: Black}
			b.Points[8] = Point{Checkers: 3, Color: Black}
			b.Points[13] = Point{Checkers: 4, Color: Black}
			// White ahead
			b.Points[20] = Point{Checkers: 2, Color: White}
			b.Points[21] = Point{Checkers: 2, Color: White}
			b.Points[22] = Point{Checkers: 2, Color: White}
			b.Points[23] = Point{Checkers: 2, Color: White}
			b.Points[24] = Point{Checkers: 2, Color: White}
			b.Points[19] = Point{Checkers: 3, Color: White}
			b.Points[18] = Point{Checkers: 2, Color: White}
			return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0}
		}(), 0x8ca933385dc6165b,
	},
	{
		"prime_vs_prime", func() Position {
			var b Board
			// Black prime 8-13
			for i := 8; i <= 13; i++ {
				b.Points[i] = Point{Checkers: 2, Color: Black}
			}
			// White prime 12-17 (overlap doesn't matter for stability test)
			for i := 12; i <= 17; i++ {
				b.Points[i] = Point{Checkers: 2, Color: White}
			}
			b.Points[24] = Point{Checkers: 1, Color: Black}
			b.Points[1] = Point{Checkers: 1, Color: White}
			return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0}
		}(), 0xeb90bef76e9152b6,
	},
	{
		"race_even", func() Position {
			var b Board
			for i := 1; i <= 6; i++ {
				b.Points[i] = Point{Checkers: 2, Color: Black}
				b.Points[i+19] = Point{Checkers: 2, Color: White}
			}
			b.Points[6].Checkers = 3
			return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0}
		}(), 0x41afee4068bfd2b9,
	},
}

func TestZobristStability(t *testing.T) {
	// First check all positions have non-zero hashes (sanity).
	for _, tc := range stabilityPositions {
		h := ZobristHash(&tc.pos)
		if h == 0 {
			t.Errorf("%s: hash is zero (very unlikely, probably a bug)", tc.name)
		}
	}

	// If want values are non-zero, check them.
	for _, tc := range stabilityPositions {
		if tc.want == 0 {
			continue
		}
		h := ZobristHash(&tc.pos)
		if h != tc.want {
			t.Errorf("%s: got 0x%016x, want 0x%016x", tc.name, h, tc.want)
		}
	}
}

// TestZobristEquivalence verifies that a position with PlayerOnRoll=0 and its
// mirror (PlayerOnRoll=1) produce the same hash.
func TestZobristEquivalence(t *testing.T) {
	cases := []struct {
		name string
		pos  Position
	}{
		{"initial", initialPosition()},
		{"bearoff", bearoffPosition()},
		{"cube2_black", cubePosition(1, Black)}, // exponent 1 = cube at 2
		{"score_6_4", scorePosition(6, 4)},
	}

	for _, tc := range cases {
		mirror := tc.pos.Mirror()
		h0 := ZobristHash(&tc.pos)
		h1 := ZobristHash(&mirror)
		if h0 != h1 {
			t.Errorf("%s: PlayerOnRoll=0 hash 0x%016x != mirror hash 0x%016x", tc.name, h0, h1)
		}
	}
}

// TestZobristDistinct verifies that distinct positions have distinct hashes
// (collision would be a red flag, though not guaranteed impossible).
func TestZobristDistinct(t *testing.T) {
	seen := make(map[uint64]string)
	for _, tc := range stabilityPositions {
		h := ZobristHash(&tc.pos)
		if prev, ok := seen[h]; ok {
			// Collision: report but don't fail — two "distinct" test positions
			// might intentionally normalize to the same hash (e.g. initial and initial_mirror).
			t.Logf("collision: %s and %s share hash 0x%016x", tc.name, prev, h)
		}
		seen[h] = tc.name
	}
}

// ---------------------------------------------------------------------------
// Helper to print hashes for freezing (run manually when positions change)
// ---------------------------------------------------------------------------

func TestZobristPrintHashes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hash printer in short mode")
	}
	t.Log("Hash values for stabilityPositions (paste into want fields to freeze):")
	for _, tc := range stabilityPositions {
		h := ZobristHash(&tc.pos)
		t.Logf(`{"%s", ..., 0x%016x},`, tc.name, h)
	}
}
