package storagetest

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// WinnerMigrationGame is one game row as a 2.25.0 library holds it, and the
// winner sqlshared.NormalizeGameWinnerSQL must leave on it. A nil Winner or
// PointsWon is a NULL column.
type WinnerMigrationGame struct {
	S1, S2    int32
	Winner    *int32
	PointsWon *int32
	Want      int32
}

// WinnerMigrationMatch is one match of the fixture: what records its source
// (the file it was imported from, the format of its import batch) and its
// games in order.
type WinnerMigrationMatch struct {
	Name        string
	FilePath    string
	BatchFormat string // "" for no import batch
	Length      int32
	Games       []WinnerMigrationGame
}

// WinnerMigrationCases is the fixture both backends run the 2.26.0 migration
// on: one match per source and per rule of NormalizeGameWinnerSQL.
func WinnerMigrationCases() []WinnerMigrationMatch {
	const p1, p2, none = domain.WinnerPlayer1, domain.WinnerPlayer2, domain.WinnerUnfinished
	v := func(n int32) *int32 { return &n }
	g := func(s1, s2, w, pts, want int32) WinnerMigrationGame {
		return WinnerMigrationGame{S1: s1, S2: s2, Winner: v(w), PointsWon: v(pts), Want: want}
	}
	return []WinnerMigrationMatch{
		// XG: 1 = player 1, -1 = player 2, 0 = unfinished. Already the target.
		{Name: "xg", FilePath: "/m/a.xg", Length: 7, Games: []WinnerMigrationGame{
			g(0, 0, 1, 1, p1), g(1, 0, -1, 2, p2), g(1, 2, -1, 2, p2)}},
		{Name: "xg last unfinished", FilePath: "/m/b.XG", Length: 7, Games: []WinnerMigrationGame{
			g(0, 0, -1, 3, p2), g(0, 3, 0, 0, none)}},
		{Name: "xg by batch, single game", FilePath: "", BatchFormat: "xg", Length: 1, Games: []WinnerMigrationGame{
			g(0, 0, 1, 1, p1)}},
		{Name: "xg money session", FilePath: "/m/money.xg", Length: 0, Games: []WinnerMigrationGame{
			g(0, 0, 1, 2, p1), g(0, 0, -1, 1, p2), g(0, 0, 0, 0, none)}},
		// gnubg: 0 = player 1, 1 = player 2, -1 = unfinished.
		{Name: "mat", FilePath: "/m/c.mat", Length: 7, Games: []WinnerMigrationGame{
			g(0, 0, 0, 1, p1), g(1, 0, 1, 2, p2), g(1, 2, 0, 3, p1)}},
		{Name: "sgf last won by player 2", FilePath: "/m/d.sgf", BatchFormat: "gnubg", Length: 5, Games: []WinnerMigrationGame{
			g(0, 0, 0, 2, p1), g(2, 0, 1, 4, p2)}},
		{Name: "sgf last unfinished", FilePath: "/m/e.sgf", Length: 5, Games: []WinnerMigrationGame{
			g(0, 0, 1, 1, p2), g(0, 1, -1, 0, none)}},
		{Name: "transcription, player 2", FilePath: "", Length: 3, Games: []WinnerMigrationGame{
			g(0, 0, 1, 3, p2)}},
		{Name: "transcription, player 1", FilePath: "", Length: 3, Games: []WinnerMigrationGame{
			g(0, 0, 0, 3, p1)}},
		// The games the scores settle name the encoding over the file name:
		// an .xg match saved again from a transcription holds gnubg's values.
		{Name: "xg file, gnubg values", FilePath: "/m/f.xg", Length: 7, Games: []WinnerMigrationGame{
			g(0, 0, 0, 1, p1), g(1, 0, 1, 1, p2)}},
		// BGF stored 0 for every game: the scores settle all but the last.
		{Name: "bgf", FilePath: "/m/g.bgf", BatchFormat: "bgf", Length: 5, Games: []WinnerMigrationGame{
			g(0, 0, 0, 2, p2), g(0, 2, 0, 1, p1), g(1, 2, 0, 4, none)}},
		// A gnubg match whose players were swapped: the swap negated 0 and 1
		// as if they were XG's. The scores repair what they settle; the last
		// game, which nothing settles, reads as unfinished.
		{Name: "swapped mat", FilePath: "/m/h.mat", Length: 7, Games: []WinnerMigrationGame{
			g(0, 0, 0, 1, p2), g(0, 1, -1, 2, none)}},
		{Name: "nulls", FilePath: "/m/i.mat", Length: 7, Games: []WinnerMigrationGame{
			{S1: 0, S2: 0, Winner: nil, PointsWon: nil, Want: none},
			{S1: 0, S2: 0, Winner: v(0), PointsWon: nil, Want: none}}},
	}
}
