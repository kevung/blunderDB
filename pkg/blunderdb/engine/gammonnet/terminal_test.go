// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The terminal valuation (#188). Before these tests no position in the whole
// suite had a play that ended the game: terminalValue was at 0 % coverage,
// valueSweep never took its isOver branch, and the one thing that stays
// unexecuted by construction — the match-referential value of a finished
// game, where a backgammon at 2-away is a match and a single game is not —
// was the one thing a wrong port would get plausibly wrong.
//
// Three boards, one per stake. White is on roll with two checkers left, on
// the ace point (index 0) and the five point (index 4), thirteen borne off;
// a 6-2 gives exactly two legal plays, 5/off 1/off which ends the game and
// 5/3 3/off which does not — so every search below has one terminal and one
// live candidate, in the same ranking. (A 4-2 would not do: 5/1 1/off and
// 5/3 3/off both leave one checker on the ace point, and the generator
// rightly keeps one play per resulting board.) Black differs in what the win is
// worth: three checkers off (a plain game), none off (a gammon), none off
// and one on the bar (a backgammon).
//
// The same three boards are the last three cases of the ADR-0023 gold
// corpus (buildSearchCubeCorpus's terminalCases), so the C reference values
// them too — see testdata/gold/README.md.
func terminalBoards() [3]Position {
	white := func() Position {
		var p Position
		p.Points[0] = 1
		p.Points[4] = 1
		p.Off[White] = 13
		p.Turn = White
		return p
	}

	single := white()
	single.Points[20] = -6
	single.Points[21] = -6
	single.Off[Black] = 3

	gammon := white()
	gammon.Points[19] = -5
	gammon.Points[20] = -5
	gammon.Points[21] = -5

	backgammon := white()
	backgammon.Points[19] = -5
	backgammon.Points[20] = -5
	backgammon.Points[21] = -4
	backgammon.Bar[Black] = 1

	return [3]Position{single, gammon, backgammon}
}

// terminalDice is the roll every terminal board is searched with.
const terminalD1, terminalD2 = 6, 2

// terminalState is the score the three boards are valued at in the gold
// corpus and in the tests below, as White (the winner) sees it: 4-away
// against 3-away, cube centred at 1. Chosen so the three stakes land on
// three distinct rows of the MET — 3-away/3-away after a single game,
// 2-away/3-away after a gammon, 1-away/3-away (Crawford) after a backgammon
// — where 2-away/2-away, say, would price the gammon and the backgammon the
// same.
var terminalState = MatchState{AwayOnRoll: 4, AwayOpponent: 3, Cube: 1}

// finished plays the game-ending 5/off 1/off on a terminal board and returns
// the resulting position, turn switched to the loser as a Play's Result is.
func finished(board Position) Position {
	p := board
	p.Points[0] = 0
	p.Points[4] = 0
	p.Off[White] = NumCheckers
	p.Turn = Black
	return p
}

// The boards are what they claim to be: valid, not over, and the play that
// ends the game is worth 1, 2 and 3 points respectively.
func TestTerminalBoardsAreValidAndStaked(t *testing.T) {
	for i, board := range terminalBoards() {
		if !board.Valid() {
			t.Fatalf("board %d is not structurally valid", i)
		}
		if board.isOver() {
			t.Fatalf("board %d is already over", i)
		}
		if got := gameValue(&board); got != -1 {
			t.Fatalf("board %d: gameValue on a live position = %d, want -1", i, got)
		}
		done := finished(board)
		if !done.Valid() || !done.isOver() {
			t.Fatalf("board %d: the finishing play does not end the game", i)
		}
		if got, want := gameValue(&done), i+1; got != want {
			t.Fatalf("board %d: stake %d, want %d", i, got, want)
		}
		if done.winner() != White {
			t.Fatalf("board %d: winner %d, want White", i, done.winner())
		}
	}
}

// metWin is the winner's MWC from the winner's own state, straight from the
// MET — computed here without metAfter so the test does not merely compare
// terminalValue with itself.
func metWin(state MatchState, stake int) float64 {
	matchTo := state.AwayOnRoll
	if state.AwayOpponent > matchTo {
		matchTo = state.AwayOpponent
	}
	return engine.GnuBGGetME(matchTo-state.AwayOnRoll, matchTo-state.AwayOpponent, matchTo, 0, stake, 0, state.Crawford)
}

// terminalValue at a score is 2×MWC−1 of the game just finished, from the
// point of view of the player on turn — the loser, in the search's own
// convention — and with no state it is terminalEquity in money points.
func TestTerminalValueAtScoreIsTheMETOfTheStake(t *testing.T) {
	boards := terminalBoards()
	loserState := terminalState.Swap() // the position's own Turn is Black, the loser

	var atScore [3]float64
	for i, board := range boards {
		done := finished(board)
		stake := i + 1

		// Money: the stake itself, negated because Turn names the loser.
		if got := terminalEquity(&done); got != -float64(stake) {
			t.Errorf("stake %d: terminalEquity = %v, want %v", stake, got, -float64(stake))
		}
		if got := terminalValue(&done, nil); got != terminalEquity(&done) {
			t.Errorf("stake %d: terminalValue(nil) = %v, want terminalEquity %v", stake, got, terminalEquity(&done))
		}

		// Match: the loser's MWC is one minus the winner's, read from the
		// winner's row of the table. The MET is stored in float32 and the
		// two readings (the loser's own cell, one minus the winner's) round
		// independently, so they agree to float32 and not to the bit.
		want := 2*(1-metWin(terminalState, stake)) - 1
		got := terminalValue(&done, &loserState)
		if math.Abs(got-want) > 1e-6 {
			t.Errorf("stake %d: terminalValue at 3-away/4-away = %v, want %v", stake, got, want)
		}
		// A lost game values in [-1, 0]: 0 exactly for the single game,
		// which lands on the table's diagonal (3-away/3-away).
		if got > 0 || got < -1 {
			t.Errorf("stake %d: a lost game must value in [-1, 0], got %v", stake, got)
		}
		atScore[i] = got

		// Antisymmetry: seen from the winner, with the state swapped back,
		// the same finished game is worth the negation.
		fromWinner := done
		fromWinner.Turn = White
		if v := terminalValue(&fromWinner, &terminalState); math.Abs(v+got) > 1e-6 {
			t.Errorf("stake %d: winner's value %v is not the negation of the loser's %v", stake, v, got)
		}
		if v := terminalEquity(&fromWinner); v != float64(stake) {
			t.Errorf("stake %d: winner's money equity %v, want +%d", stake, v, stake)
		}
	}
	// Losing more points is worse — the stake reaches the MET.
	if !(atScore[0] > atScore[1] && atScore[1] > atScore[2]) {
		t.Errorf("values are not decreasing in the stake: single %v, gammon %v, backgammon %v", atScore[0], atScore[1], atScore[2])
	}
}

// The score decides what a stake buys: at DMP every stake is the match; at
// 2-away a gammon is the match and a single game is not; at cube 2 a single
// game is worth what a gammon is at cube 1.
func TestTerminalValueSaturatesWhereTheStakeWinsTheMatch(t *testing.T) {
	boards := terminalBoards()
	for i, board := range boards {
		done := finished(board)
		done.Turn = White // valued from the winner's side, so a win reads positive
		stake := i + 1

		dmp := MatchState{AwayOnRoll: 1, AwayOpponent: 1, Cube: 1, Crawford: true}
		if got := terminalValue(&done, &dmp); got != 1 {
			t.Errorf("stake %d at DMP: %v, want exactly +1", stake, got)
		}

		twoAway := MatchState{AwayOnRoll: 2, AwayOpponent: 5, Cube: 1}
		got := terminalValue(&done, &twoAway)
		if stake >= 2 && got != 1 {
			t.Errorf("stake %d at 2-away: %v, want exactly +1 (the gammon is the match)", stake, got)
		}
		if stake == 1 && got >= 1 {
			t.Errorf("a single game at 2-away/5-away is not the match, got %v", got)
		}
	}

	single, gammon := finished(boards[0]), finished(boards[1])
	single.Turn, gammon.Turn = White, White
	cube2 := MatchState{AwayOnRoll: 7, AwayOpponent: 5, Cube: 2}
	cube1 := MatchState{AwayOnRoll: 7, AwayOpponent: 5, Cube: 1}
	if a, b := terminalValue(&single, &cube2), terminalValue(&gammon, &cube1); a != b {
		t.Errorf("a single game at cube 2 (%v) must be worth a gammon at cube 1 (%v)", a, b)
	}

	// A live position has no terminal value, on either scale.
	live := boards[0]
	if got := terminalValue(&live, &terminalState); got != 0 {
		t.Errorf("terminalValue on a live position = %v, want 0", got)
	}
	if got := terminalEquity(&live); got != 0 {
		t.Errorf("terminalEquity on a live position = %v, want 0", got)
	}
}

// valueSweep values a game-ending play through terminalValue, in the state
// the resulting position's own mover sees — the swapped one — and the
// search hands that value out unchanged at every ply: a finished game has no
// deeper ply to search.
func TestSearchValuesATerminalPlayByTheMET(t *testing.T) {
	boards := terminalBoards()
	out := make([]Candidate, MaxPlays)
	for i, board := range boards {
		stake := i + 1
		wantEquity := 2*metWin(terminalState, stake) - 1

		var atPly [3]float64
		for ply := 0; ply <= 2; ply++ {
			cfg := DefaultConfig(ply)
			cfg.UseMatch, cfg.Match = true, terminalState
			cfg.UseCube, cfg.CubeOwner, cfg.CubeX = true, CubeCentred, DefaultEfficiency(CubeCentred)
			s, err := NewSearcher(cfg)
			if err != nil {
				t.Fatal(err)
			}
			n, err := s.Plays(&board, terminalD1, terminalD2, out)
			if err != nil {
				t.Fatal(err)
			}
			if n != 2 {
				t.Fatalf("stake %d, %d-ply: %d plays, want 2 (5/off 1/off and 5/3 3/off)", stake, ply, n)
			}
			terminal := -1
			for c := 0; c < n; c++ {
				if out[c].Play.Result.isOver() {
					if terminal >= 0 {
						t.Fatalf("stake %d, %d-ply: two terminal plays", stake, ply)
					}
					terminal = c
				}
			}
			if terminal < 0 {
				t.Fatalf("stake %d, %d-ply: no terminal play among %d", stake, ply, n)
			}
			res := out[terminal].Play.Result
			if res != finished(board) {
				t.Fatalf("stake %d, %d-ply: the terminal play does not end on the expected board", stake, ply)
			}
			got := out[terminal].Equity
			swapped := terminalState.Swap()
			if want := -terminalValue(&res, &swapped); got != want {
				t.Errorf("stake %d, %d-ply: terminal candidate equity %v, want -terminalValue %v", stake, ply, got, want)
			}
			if math.Abs(got-wantEquity) > 1e-6 {
				t.Errorf("stake %d, %d-ply: terminal candidate equity %v, want 2×MWC−1 = %v", stake, ply, got, wantEquity)
			}
			atPly[ply] = got
		}
		if atPly[0] != atPly[1] || atPly[1] != atPly[2] {
			t.Errorf("stake %d: a finished game's value moved with the ply: %v", stake, atPly)
		}
	}
}

// Money, cubeless: the same sweep values the finished game in points, and a
// backgammon outranks a gammon outranks a single game in the ranking itself.
func TestSearchValuesATerminalPlayInMoneyPoints(t *testing.T) {
	boards := terminalBoards()
	out := make([]Candidate, MaxPlays)
	s, err := NewSearcher(DefaultConfig(0))
	if err != nil {
		t.Fatal(err)
	}
	for i, board := range boards {
		stake := float64(i + 1)
		n, err := s.Plays(&board, terminalD1, terminalD2, out)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for c := 0; c < n; c++ {
			if !out[c].Play.Result.isOver() {
				continue
			}
			found = true
			if out[c].Equity != stake {
				t.Errorf("stake %v: terminal candidate equity %v in money", stake, out[c].Equity)
			}
		}
		if !found {
			t.Fatalf("stake %v: no terminal play", stake)
		}
	}
}
