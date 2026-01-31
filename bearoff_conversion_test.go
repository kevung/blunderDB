package main

import (
	"testing"
)

func TestImplicitBearoffConversion(t *testing.T) {
	db := &Database{}

	// Test case 1: Implicit bear-off with to=-1 from home board
	// When from is in home board (1-6) and to=-1, it should be a bear-off
	move1 := [8]int32{4, -1, -1, -1, -1, -1, -1, -1}
	result1 := db.convertXGMoveToString(move1, 1)
	if result1 != "4/off" {
		t.Errorf("Test 1 failed: expected '4/off', got '%s'", result1)
	} else {
		t.Log("Test 1 passed: 4/-1 -> 4/off")
	}

	// Test case 2: Explicit bear-off with to=-2 (should still work)
	move2 := [8]int32{5, -2, -1, -1, -1, -1, -1, -1}
	result2 := db.convertXGMoveToString(move2, 1)
	if result2 != "5/off" {
		t.Errorf("Test 2 failed: expected '5/off', got '%s'", result2)
	} else {
		t.Log("Test 2 passed: 5/-2 -> 5/off")
	}

	// Test case 3: Combined move with implicit bear-off
	// 5/2 5/off should be [5, 2, 5, -1, ...] with -1 converted to bear-off
	move3 := [8]int32{5, 2, 5, -1, -1, -1, -1, -1}
	result3 := db.convertXGMoveToString(move3, 1)
	// After sorting, 5/off should come before 5/2 (same from, off < 2)
	// Or they could be displayed in reverse order depending on sort
	if result3 != "5/off 5/2" && result3 != "5/2 5/off" {
		t.Errorf("Test 3 failed: expected '5/off 5/2' or '5/2 5/off', got '%s'", result3)
	} else {
		t.Logf("Test 3 passed: 5/2 + 5/-1 -> %s", result3)
	}

	// Test case 4: Normal move (not bear-off) should not be affected
	move4 := [8]int32{13, 8, -1, -1, -1, -1, -1, -1}
	result4 := db.convertXGMoveToString(move4, 1)
	if result4 != "13/8" {
		t.Errorf("Test 4 failed: expected '13/8', got '%s'", result4)
	} else {
		t.Log("Test 4 passed: 13/8 stays 13/8")
	}

	// Test case 5: Move from outside home board should NOT convert -1 to bear-off
	// This shouldn't happen in valid data, but let's make sure we don't break anything
	// Note: The code should skip this as invalid (from 13, to -1 is not valid)
	move5 := [8]int32{13, -1, -1, -1, -1, -1, -1, -1}
	result5 := db.convertXGMoveToString(move5, 1)
	// This should be "Cannot Move" because 13/-1 is not a valid move (13 is not in home board)
	if result5 != "Cannot Move" {
		t.Errorf("Test 5 failed: expected 'Cannot Move' for invalid 13/-1, got '%s'", result5)
	} else {
		t.Log("Test 5 passed: 13/-1 is invalid -> Cannot Move")
	}

	// Test case 6: to=0 should also be converted to bear-off when from is in home board
	// This represents 1/off with die 1 (1-1=0)
	move6 := [8]int32{1, 0, -1, -1, -1, -1, -1, -1}
	result6 := db.convertXGMoveToString(move6, 1)
	if result6 != "1/off" {
		t.Errorf("Test 6 failed: expected '1/off', got '%s'", result6)
	} else {
		t.Log("Test 6 passed: 1/0 -> 1/off")
	}

	// Test case 7: Negative destination (like -3, -4) should also be bear-off
	// This represents 2/off with die 5 (2-5=-3)
	move7 := [8]int32{2, -3, -1, -1, -1, -1, -1, -1}
	result7 := db.convertXGMoveToString(move7, 1)
	if result7 != "2/off" {
		t.Errorf("Test 7 failed: expected '2/off', got '%s'", result7)
	} else {
		t.Log("Test 7 passed: 2/-3 -> 2/off")
	}

	// Test case 8: Double bear-off - both 4/off and 3/off
	// This is the actual data from test.xg Game 2 Move 26: [4, -1, 3, -2, -1, -1, -1, -1]
	// First sub-move: 4 -> -1 (implicit bear-off)
	// Second sub-move: 3 -> -2 (explicit bear-off)
	move8 := [8]int32{4, -1, 3, -2, -1, -1, -1, -1}
	result8 := db.convertXGMoveToString(move8, 1)
	// Should show both bear-offs
	if result8 != "4/off 3/off" && result8 != "3/off 4/off" {
		t.Errorf("Test 8 failed: expected '4/off 3/off' or '3/off 4/off', got '%s'", result8)
	} else {
		t.Logf("Test 8 passed: double bear-off [4,-1,3,-2] -> %s", result8)
	}

	// Test case 9: Double implicit bear-off
	// Both sub-moves use implicit bear-off encoding
	move9 := [8]int32{4, -1, 3, -1, -1, -1, -1, -1}
	result9 := db.convertXGMoveToString(move9, 1)
	if result9 != "4/off 3/off" && result9 != "3/off 4/off" {
		t.Errorf("Test 9 failed: expected '4/off 3/off' or '3/off 4/off', got '%s'", result9)
	} else {
		t.Logf("Test 9 passed: double implicit bear-off [4,-1,3,-1] -> %s", result9)
	}
}
