package tests

import (
	"fmt"
	"sort"
	"testing"
)

// Copie de la fonction formatPoint de db.go pour test isolé
func formatPoint(p int32) string {
	if p == 25 {
		return "bar"
	} else if p == -2 {
		return "off"
	} else if p >= 1 && p <= 24 {
		return fmt.Sprintf("%d", p)
	}
	return fmt.Sprintf("?%d", p)
}

// Copie simplifiée de la fonction convertXGMoveToString pour test isolé
func convertXGMoveToStringTest(playedMove [8]int32, activePlayer int32) string {
	_ = activePlayer

	var fromPts []int32
	var toPts []int32
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		if from == -1 || to == -1 {
			break
		}
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}
		fromPts = append(fromPts, from)
		toPts = append(toPts, to)
	}

	if len(fromPts) == 0 {
		return "Cannot Move"
	}

	type moveItem struct {
		from int32
		to   int32
	}
	items := make([]moveItem, len(fromPts))
	for i := range fromPts {
		items[i] = moveItem{from: fromPts[i], to: toPts[i]}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	type groupedMove struct {
		from  int32
		to    int32
		count int
	}
	var grouped []groupedMove
	for _, item := range items {
		found := false
		for i := range grouped {
			if grouped[i].from == item.from && grouped[i].to == item.to {
				grouped[i].count++
				found = true
				break
			}
		}
		if !found {
			grouped = append(grouped, groupedMove{from: item.from, to: item.to, count: 1})
		}
	}

	var result string
	for i, g := range grouped {
		if i > 0 {
			result += " "
		}
		if g.count > 1 {
			result += fmt.Sprintf("%s/%s(%d)", formatPoint(g.from), formatPoint(g.to), g.count)
		} else {
			result += fmt.Sprintf("%s/%s", formatPoint(g.from), formatPoint(g.to))
		}
	}

	return result
}

func TestBearoffConversion(t *testing.T) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertXGMoveToStringTest(tc.input, 0)
			if result != tc.expected {
				t.Errorf("convertXGMoveToStringTest(%v) = %q, expected %q", tc.input, result, tc.expected)
			} else {
				t.Logf("PASS: %q", result)
			}
		})
	}
}

func TestFormatPointFunction(t *testing.T) {
	testCases := []struct {
		input    int32
		expected string
	}{
		{-2, "off"},
		{25, "bar"},
		{6, "6"},
		{24, "24"},
		{1, "1"},
	}

	for _, tc := range testCases {
		result := formatPoint(tc.input)
		if result != tc.expected {
			t.Errorf("formatPoint(%d) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
