package domain

// MatchResultOf is the result of a match of the given length (0 for money)
// whose games are listed in playing order: the score after the last game,
// and the player who reached the length, WinnerUnfinished when none did. A
// match without games has no result: nil and WinnerUnfinished.
func MatchResultOf(length int32, games []Game) (*[2]int32, int32) {
	if len(games) == 0 {
		return nil, WinnerUnfinished
	}
	last := games[len(games)-1]
	final := last.InitialScore
	switch last.Winner {
	case WinnerPlayer1:
		final[0] += last.PointsWon
	case WinnerPlayer2:
		final[1] += last.PointsWon
	}
	return &final, MatchWinner(length, final)
}

// MatchWinner names the player whose score reached length, WinnerUnfinished
// when neither did or the match has no length.
func MatchWinner(length int32, final [2]int32) int32 {
	switch {
	case length <= 0:
		return WinnerUnfinished
	case final[0] >= length:
		return WinnerPlayer1
	case final[1] >= length:
		return WinnerPlayer2
	}
	return WinnerUnfinished
}
