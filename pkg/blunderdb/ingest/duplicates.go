package ingest

import (
	"context"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// diceMatch is a stored match reduced to what the duplicate search compares:
// one string per game, one byte per roll (diceGameKey).
type diceMatch struct {
	id      int64
	players string
	p1, p2  string
	length  int
	initial [2]int
	hash    string
	games   []string
}

// FindDuplicateSuspects lists the pairs of stored matches whose dice say they
// are probably one match: the same dice under other player names
// (DuplicateSameDice), or one match's dice continuing another's — a match
// truncated then completed (DuplicateLonger). Nothing is merged. A match
// whose dice_hash was never computed (imported before the column existed)
// gets it on the way, so later imports can recognise it; the second result
// counts those.
func FindDuplicateSuspects(ctx context.Context, ms storage.MatchStore, scope string) ([]domain.DuplicateSuspect, int, error) {
	var all []diceMatch
	backfill := map[int64]string{}
	for md, err := range ms.DiceSequences(ctx, scope) {
		if err != nil {
			return nil, 0, err
		}
		h := DiceMatchHash(md.Length, md.Initial, md.Games)
		if md.DiceHash != h {
			backfill[md.ID] = h
		}
		if h == "" {
			continue // no real roll: nothing to recognise it by
		}
		dm := diceMatch{id: md.ID, p1: md.Player1, p2: md.Player2, length: md.Length, hash: h,
			players: md.Player1 + " – " + md.Player2}
		if len(md.Initial) > 0 {
			dm.initial = md.Initial[0]
			if dm.initial[0] > dm.initial[1] {
				dm.initial[0], dm.initial[1] = dm.initial[1], dm.initial[0]
			}
		}
		for _, g := range md.Games {
			dm.games = append(dm.games, diceGameKey(g))
		}
		all = append(all, dm)
	}
	// Written after the read: the stream holds the connection while it runs.
	ids := make([]int64, 0, len(backfill))
	for id := range backfill {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if err := ms.SetDiceHash(ctx, scope, id, backfill[id]); err != nil {
			return nil, 0, err
		}
	}
	return diceSuspects(all), len(ids), nil
}

// diceGroupRolls is how many opening rolls of the first game key a group: a
// truncated match keeps them, so a match and its longer version meet in one
// group, and the pairwise comparison stays within a handful of matches.
const diceGroupRolls = 3

// diceSuspects pairs the matches of all, in id order.
func diceSuspects(all []diceMatch) []domain.DuplicateSuspect {
	type key struct {
		length  int
		initial [2]int
		opening string
	}
	groups := map[key][]int{}
	var order []key
	for i, m := range all {
		if len(m.games) == 0 || m.games[0] == "" {
			continue
		}
		op := m.games[0]
		if len(op) > diceGroupRolls {
			op = op[:diceGroupRolls]
		}
		k := key{m.length, m.initial, op}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], i)
	}
	var out []domain.DuplicateSuspect
	for _, k := range order {
		idx := groups[k]
		for a := 0; a < len(idx); a++ {
			for b := a + 1; b < len(idx); b++ {
				x, y := &all[idx[a]], &all[idx[b]]
				switch {
				case x.hash == y.hash:
					if !samePlayers(x.p1, x.p2, y.p1, y.p2) {
						out = append(out, domain.DuplicateSuspect{Kind: domain.DuplicateSameDice,
							MatchID: y.id, OtherID: x.id, Players: y.players, OtherPlayers: x.players})
					}
				case dicePrefix(x.games, y.games):
					out = append(out, domain.DuplicateSuspect{Kind: domain.DuplicateLonger,
						MatchID: y.id, OtherID: x.id, Players: y.players, OtherPlayers: x.players})
				case dicePrefix(y.games, x.games):
					out = append(out, domain.DuplicateSuspect{Kind: domain.DuplicateLonger,
						MatchID: x.id, OtherID: y.id, Players: x.players, OtherPlayers: y.players})
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OtherID != out[j].OtherID {
			return out[i].OtherID < out[j].OtherID
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}

// dicePrefix reports whether short is a truncation of long: every game but
// its last equal to long's, its last game's dice a prefix of long's game at
// the same place, and long strictly longer.
func dicePrefix(short, long []string) bool {
	if len(short) == 0 || len(short) > len(long) {
		return false
	}
	last := len(short) - 1
	for i := 0; i < last; i++ {
		if short[i] != long[i] {
			return false
		}
	}
	if !strings.HasPrefix(long[last], short[last]) {
		return false
	}
	return len(short) < len(long) || len(short[last]) < len(long[last])
}
