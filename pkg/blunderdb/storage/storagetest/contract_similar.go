// Contract cases for the "positions like this one" scan.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testSimilarIsExactAndOrdered pins the two promises the contract makes about
// Similar, both of which an approximate index would break: the neighbours come
// back NEAREST FIRST, and the scan is exhaustive — so a position that is
// closer is never missed (issue #293).
//
// It also pins the one exclusion: a position is not its own neighbour. Asking
// "what is like this?" and being handed the thing itself is a non-answer, and
// it is the kind of thing a query written against the whole table gets wrong.
func testSimilarIsExactAndOrdered(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	// One reference and three variations, at growing distance from it: one
	// checker moved one pip, then three, then six.
	base := similarityBoard(map[int]int{13: 5, 8: 5, 6: 5})
	near := similarityBoard(map[int]int{13: 4, 12: 1, 8: 5, 6: 5})
	middle := similarityBoard(map[int]int{13: 4, 10: 1, 8: 5, 6: 5})
	far := similarityBoard(map[int]int{13: 4, 7: 1, 8: 5, 6: 5})

	baseID, err := ps.Save(ctx, "", &base)
	if err != nil {
		t.Fatalf("Save base: %v", err)
	}
	nearID, err := ps.Save(ctx, "", &near)
	if err != nil {
		t.Fatalf("Save near: %v", err)
	}
	middleID, err := ps.Save(ctx, "", &middle)
	if err != nil {
		t.Fatalf("Save middle: %v", err)
	}
	if _, err := ps.Save(ctx, "", &far); err != nil {
		t.Fatalf("Save far: %v", err)
	}

	base.ID = baseID
	got, err := ps.Similar(ctx, "", &base, 2)
	if err != nil {
		t.Fatalf("Similar: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Similar returned %d neighbours, want 2", len(got))
	}
	if got[0].Position.ID != nearID || got[1].Position.ID != middleID {
		t.Errorf("neighbours must come back nearest first: got %d then %d, want %d then %d",
			got[0].Position.ID, got[1].Position.ID, nearID, middleID)
	}
	if got[0].Distance >= got[1].Distance {
		t.Errorf("distances must grow: got %d then %d", got[0].Distance, got[1].Distance)
	}
	for _, n := range got {
		if n.Position.ID == baseID {
			t.Error("a position is not its own neighbour")
		}
	}
}

// similarityBoard builds a money-game checker position from Black's points,
// with White standing clear of every point Black uses.
func similarityBoard(black map[int]int) domain.Position {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	for pt, n := range black {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.Black}
	}
	for _, pt := range []int{17, 19, 21} {
		p.Board.Points[pt] = domain.Point{Checkers: 5, Color: domain.White}
	}
	p.PlayerOnRoll = domain.Black
	p.Dice = [2]int{3, 1}
	p.Score = [2]int{-1, -1}
	return p
}
