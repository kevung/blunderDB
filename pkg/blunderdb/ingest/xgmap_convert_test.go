package ingest

import "testing"

// Moved from the legacy tests/bearoff_unit_test.go and tests/xg_import_test.go
// (root "tests" package, invisible to coverage since it built no code): both
// duplicated convertXGMoveToString (and, for bearoff_unit_test.go, formatPoint)
// as local copies instead of exercising the real ingest code, so a change here
// could drift from what it claimed to test — and xg_import_test.go's cases
// were never actually asserted (t.Logf only). This exercises the real
// convertXGMoveToString directly, with real assertions.
func TestConvertXGMoveToString(t *testing.T) {
	testCases := []struct {
		name     string
		input    [8]int32
		expected string
	}{
		{
			name:     "Simple bear-off: 6/off",
			input:    [8]int32{6, -2, -1, -1, -1, -1, -1, -1},
			expected: "6/off",
		},
		{
			name:     "Double bear-off: 4/off 3/off",
			input:    [8]int32{4, -2, 3, -2, -1, -1, -1, -1},
			expected: "4/off 3/off",
		},
		{
			name:     "Bear-off and move: 5/off 4/1",
			input:    [8]int32{5, -2, 4, 1, -1, -1, -1, -1},
			expected: "5/off 4/1",
		},
		{
			name:     "Simple move: 24/23 13/8",
			input:    [8]int32{13, 8, 24, 23, -1, -1, -1, -1},
			expected: "24/23 13/8",
		},
		{
			name:     "Bar entry: bar/23",
			input:    [8]int32{25, 23, -1, -1, -1, -1, -1, -1},
			expected: "bar/23",
		},
		// Moved from the legacy tests/xg_import_test.go: exercises mergeSlides,
		// merged from a real match transcription (anonymised — the original
		// carried real player names).
		{
			name:     "Doublet slide 11: 24/20",
			input:    [8]int32{24, 23, 23, 22, 22, 21, 21, 20},
			expected: "24/20",
		},
		{
			name:     "Grouped doublet 22: 18/16(2) 6/4(2)",
			input:    [8]int32{18, 16, 18, 16, 6, 4, 6, 4},
			expected: "18/16(2) 6/4(2)",
		},
		{
			name:     "Cannot move",
			input:    [8]int32{-1, -1, -1, -1, -1, -1, -1, -1},
			expected: "Cannot Move",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertXGMoveToString(tc.input)
			if result != tc.expected {
				t.Errorf("convertXGMoveToString(%v) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}
