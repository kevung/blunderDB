package sqlshared

// badges_from_table.go — the match and tournament badges and the players
// table's MWC7 read from match_stats. Each seat row carries the counted
// decisions' error, their number and their MWC loss with its per-game
// spread, which is all the direct passes in stats.go and player_mwc7.go
// derive from the decisions; the arithmetic after the sums is the same
// function on both paths.

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// matchBadgesFromTable is matchBadgesDirect over match_stats. A match
// without a counted decision on either seat has no badge, as there.
func (s *StatsStore) matchBadgesFromTable(ctx context.Context, scope string, matchIDs []int64) (map[int64]storage.MatchBadge, error) {
	tenant, targs := s.DB.TenantFilter("ms", scope)
	q := `SELECT ms.match_id, ms.seat, ms.decisions, ms.error_mp, ms.mwc_loss, COALESCE(ms.mwc_games, 0),
		COALESCE(ms.mwc_game_ss, 0), COALESCE(m.match_length, 0)` + matchStatsJoin + ` WHERE ` + tenant
	type seatRow struct {
		decisions int
		errMP     int64
		mwc       *float64
		games     int
		ss        float64
	}
	type matchRow struct {
		seats  [2]seatRow
		length int
	}
	acc := map[int64]*matchRow{}
	collect := func(query string, args []any) error {
		return scanEach(ctx, s.DB, query, args, func(r Rows) error {
			var id int64
			var seat, length int
			var sr seatRow
			if err := r.Scan(&id, &seat, &sr.decisions, &sr.errMP, &sr.mwc, &sr.games, &sr.ss, &length); err != nil {
				return err
			}
			m := acc[id]
			if m == nil {
				m = &matchRow{length: length}
				acc[id] = m
			}
			m.seats[seat-1] = sr
			return nil
		})
	}
	if len(matchIDs) == 0 {
		if err := collect(q, targs); err != nil {
			return nil, fmt.Errorf("MatchBadges (table): %w", err)
		}
	}
	for start := 0; start < len(matchIDs); start += matchStatsBatch {
		chunk := matchIDs[start:min(start+matchStatsBatch, len(matchIDs))]
		if err := collect(q+` AND ms.match_id IN (`+Placeholders(len(chunk))+`)`, append(append([]any{}, targs...), int64Args(chunk)...)); err != nil {
			return nil, fmt.Errorf("MatchBadges (table): %w", err)
		}
	}

	out := make(map[int64]storage.MatchBadge, len(acc))
	for id, m := range acc {
		if m.seats[0].decisions == 0 && m.seats[1].decisions == 0 {
			continue
		}
		var b storage.MatchBadge
		for seat, sr := range m.seats {
			var loss float64
			var elo domain.MWC7
			if sr.mwc != nil {
				loss = *sr.mwc
				elo = domain.MatchMWC7FromSpread(loss, m.length, sr.games, sr.ss)
			}
			if seat == 0 {
				b.PR, b.MWCLoss, b.MWC7 = pr(sr.errMP, sr.decisions), loss, elo
			} else {
				b.PR2, b.MWCLoss2, b.MWC7P2 = pr(sr.errMP, sr.decisions), loss, elo
			}
		}
		out[id] = b
	}
	return out, nil
}

// tournamentBadgesFromTable is tournamentBadgesDirect over match_stats: the
// same per (tournament, player) accumulators, one seat row at a time.
func (s *StatsStore) tournamentBadgesFromTable(ctx context.Context, scope, main string, aliases storage.AliasMap) (map[int64]storage.TournamentBadge, error) {
	tenant, targs := s.DB.TenantFilter("ms", scope)
	acc := make(map[int64]map[string]*storage.TournamentPlayerAcc)
	err := scanEach(ctx, s.DB,
		`SELECT m.tournament_id, ms.match_id, ms.decisions, ms.error_mp, ms.mwc_loss, COALESCE(m.match_length, 0),
			COALESCE(CASE WHEN ms.seat = 1 THEN m.player1_name ELSE m.player2_name END, '')`+
			matchStatsJoin+` WHERE `+tenant+` AND m.tournament_id IS NOT NULL AND ms.decisions > 0`,
		targs, func(r Rows) error {
			var tournamentID, matchID, errMP int64
			var decisions, length int
			var mwc *float64
			var name string
			if err := r.Scan(&tournamentID, &matchID, &decisions, &errMP, &mwc, &length, &name); err != nil {
				return err
			}
			byPlayer := acc[tournamentID]
			if byPlayer == nil {
				byPlayer = make(map[string]*storage.TournamentPlayerAcc)
				acc[tournamentID] = byPlayer
			}
			a := byPlayer[name]
			if a == nil {
				a = &storage.TournamentPlayerAcc{Matches: make(map[int64]struct{}),
					MatchMWC: make(map[int64]float64), MatchLength: make(map[int64]int)}
				byPlayer[name] = a
			}
			a.SumErr += errMP
			a.Cnt += decisions
			a.Matches[matchID] = struct{}{}
			if mwc != nil {
				a.MWC += *mwc
				a.MatchMWC[matchID] += *mwc
				a.MatchLength[matchID] = length
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("TournamentBadges (table): %w", err)
	}
	out := make(map[int64]storage.TournamentBadge, len(acc))
	for tournamentID, byPlayer := range acc {
		out[tournamentID] = storage.PickReferencePlayer(byPlayer, main, aliases)
	}
	return out, nil
}

// playerMWC7FromTable is playerMWC7 over match_stats, for the filters
// fromMatchStats admits: the same (match, player) units, pooled in the same
// order. Its MET predicate is the match's, as the PR beside it read from the
// same table.
func (s *StatsStore) playerMWC7FromTable(ctx context.Context, scope string, f storage.StatsFilter, aliases storage.AliasMap) (map[string]domain.MWC7, error) {
	where, args := s.matchStatsWhere(scope, f, true)
	losses := map[playerMWC7Unit]float64{}
	err := scanEach(ctx, s.DB,
		`SELECT COALESCE(CASE WHEN ms.seat = 1 THEN m.player1_name ELSE m.player2_name END, ''), ms.match_id,
			ms.mwc_loss, COALESCE(m.match_length, 0)`+matchStatsJoin+where+` AND ms.mwc_loss IS NOT NULL`,
		args, func(r Rows) error {
			var name string
			var match int64
			var loss float64
			var length int
			if err := r.Scan(&name, &match, &loss, &length); err != nil {
				return err
			}
			if name == "" || !domain.MWC7Defined(length) {
				return nil
			}
			losses[playerMWC7Unit{match, aliases.Canonical(name), length}] += loss
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("PlayerTable MWC7 (table): %w", err)
	}
	return poolPlayerMWC7(losses), nil
}
