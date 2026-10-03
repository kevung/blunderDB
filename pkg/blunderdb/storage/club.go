package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The club and coach reads (ADR-0065) are compositions of single-tenant reads,
// run once per tenant of a read set by ReadAcross / StreamAcross (ADR-0063):
// no store takes several scopes, so row-level security filters each call as
// any other. This file holds the per-tenant halves and the merge that only
// makes sense once every tenant has answered.

// MaxZobristLookups bounds the hashes one comment lookup takes: a whole match
// is a few hundred positions.
const MaxZobristLookups = 1000

// ZobristComment is one comment of a tenant, joined to the board it was
// written on by that board's Zobrist hash: the position id names the board in
// its own tenant only, the hash names it in every tenant.
type ZobristComment struct {
	Zobrist    uint64               `json:"zobrist"`
	PositionID int64                `json:"positionId"`
	Comment    *domain.CommentEntry `json:"comment"`
}

// CommentsByZobrist returns scope's comments on the boards named by hashes,
// in the order of hashes and oldest first within a board, in two reads (the
// boards, then their comments) whatever the number of hashes. A hash scope
// does not hold is skipped; a repeated hash is read once.
func CommentsByZobrist(ctx context.Context, s Storage, scope string, hashes []uint64) ([]ZobristComment, error) {
	if len(hashes) > MaxZobristLookups {
		return nil, fmt.Errorf("%w: %d hashes, at most %d", ErrInvalid, len(hashes), MaxZobristLookups)
	}
	seen := make(map[uint64]bool, len(hashes))
	unique := make([]uint64, 0, len(hashes))
	for _, z := range hashes {
		if !seen[z] {
			seen[z] = true
			unique = append(unique, z)
		}
	}
	out := []ZobristComment{}
	if len(unique) == 0 {
		return out, nil
	}
	byHash, err := s.Positions().ExistsMany(ctx, scope, unique)
	if err != nil {
		return nil, err
	}
	if len(byHash) == 0 {
		return out, nil
	}
	ids := make([]int64, 0, len(byHash))
	for _, z := range unique {
		if id, ok := byHash[z]; ok {
			ids = append(ids, id)
		}
	}
	byPos, err := s.Comments().ByPositions(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	for _, z := range unique {
		id, ok := byHash[z]
		if !ok {
			continue
		}
		for _, c := range byPos[id] {
			out = append(out, ZobristComment{Zobrist: z, PositionID: id, Comment: c})
		}
	}
	return out, nil
}

// ClubPlayer names one player of one tenant. A name identifies a person
// within the tenant whose matches spell it so, not across tenants: two
// students who each face a "Paul" have not met the same Paul.
type ClubPlayer struct {
	Tenant string `json:"tenant"`
	Name   string `json:"name"`
}

// ClubRow is one line of a club ranking: a tenant's player table row, with the
// tenant it was read from and its rank among the rows kept.
type ClubRow struct {
	Tenant string `json:"tenant"`
	// Rank is 1 for the best PR; rows of equal PR share a rank. A row with no
	// counted decision has rank 0: its PR measures nothing, so it is listed
	// after the ranked rows and not placed among them.
	Rank   int       `json:"rank"`
	Player PlayerRow `json:"player"`
}

// ClubRanking merges the player tables of several tenants into one ranking,
// best PR first. Rows are never merged across tenants (see ClubPlayer). When
// only is not empty, a row is kept only if its (tenant, name) is listed, the
// name compared case- and space-insensitively; rows with fewer than
// minDecisions counted decisions are dropped.
func ClubRanking(tables []Tagged[[]PlayerRow], only []ClubPlayer, minDecisions int) []ClubRow {
	key := func(tenant, name string) string {
		return tenant + "\x00" + strings.Join(strings.Fields(strings.ToLower(name)), " ")
	}
	keep := make(map[string]bool, len(only))
	for _, p := range only {
		keep[key(p.Tenant, p.Name)] = true
	}
	rows := []ClubRow{}
	for _, tg := range tables {
		for _, r := range tg.Item {
			if len(keep) > 0 && !keep[key(tg.Tenant, r.Name)] {
				continue
			}
			if r.Decisions < minDecisions {
				continue
			}
			rows = append(rows, ClubRow{Tenant: tg.Tenant, Player: r})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i].Player, rows[j].Player
		if (a.Decisions == 0) != (b.Decisions == 0) {
			return b.Decisions == 0
		}
		if a.Decisions != 0 && a.PR != b.PR {
			return a.PR < b.PR
		}
		// Same name in two tenants: the stable sort keeps the read set's order.
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	for i := range rows {
		switch {
		case rows[i].Player.Decisions == 0:
			rows[i].Rank = 0
		case i > 0 && rows[i-1].Player.Decisions != 0 && rows[i-1].Player.PR == rows[i].Player.PR:
			rows[i].Rank = rows[i-1].Rank
		default:
			rows[i].Rank = i + 1
		}
	}
	return rows
}
