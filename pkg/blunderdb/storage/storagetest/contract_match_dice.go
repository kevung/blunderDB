// Contract case for the dice of a match: dice_hash stored and found, and the
// per-game dice streamed back in game and move order.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testMatchDiceSequences(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, DiceHash: "d1"}
	id, err := ms.Save(ctx, "", &m)
	if err != nil {
		t.Fatal(err)
	}
	empty := domain.Match{Player1Name: "Carol", Player2Name: "Dave", MatchLength: 3}
	emptyID, err := ms.Save(ctx, "", &empty)
	if err != nil {
		t.Fatal(err)
	}
	for gn, dice := range [][][2]int32{{{3, 1}, {6, 6}}, {{5, 2}}} {
		gid, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: id, GameNumber: int32(gn + 1), InitialScore: [2]int32{int32(gn), 0}})
		if err != nil {
			t.Fatal(err)
		}
		n := int32(0)
		for _, d := range dice {
			n++
			if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: n, MoveType: "cube", Player: 1}); err != nil {
				t.Fatal(err)
			}
			n++
			if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: n, MoveType: "checker", Player: 1, Dice: d}); err != nil {
				t.Fatal(err)
			}
		}
	}

	got, err := ms.ListByDiceHash(ctx, "", "d1")
	if err != nil || len(got) != 1 || got[0].ID != id || got[0].DiceHash != "d1" {
		t.Fatalf("ListByDiceHash = %+v, %v", got, err)
	}
	if err := ms.SetDiceHash(ctx, "", emptyID, "d2"); err != nil {
		t.Fatal(err)
	}

	var seen []storage.MatchDice
	for md, err := range ms.DiceSequences(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		seen = append(seen, md)
	}
	if len(seen) != 2 {
		t.Fatalf("DiceSequences gave %d matches, want 2", len(seen))
	}
	a, b := seen[0], seen[1]
	if a.ID != id || len(a.Games) != 2 || len(a.Games[0]) != 2 || a.Games[0][1] != [2]int{6, 6} ||
		len(a.Games[1]) != 1 || a.Initial[1] != [2]int{1, 0} || a.Length != 5 || a.Player1 != "Alice" {
		t.Errorf("first match = %+v", a)
	}
	if b.ID != emptyID || len(b.Games) != 0 || b.DiceHash != "d2" {
		t.Errorf("game-less match = %+v", b)
	}
}
