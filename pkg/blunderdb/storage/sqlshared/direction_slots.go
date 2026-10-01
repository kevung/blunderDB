package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Slots and pairs: where a directed Tournament meets the library (ADR-0047)
// and the persons behind a doubles Participant (ADR-0056 §4). Every
// statement is confined to the scope's tenant.

// FilledSlots returns the Matches of the Tournament that fill a Slot. The
// final score of a Match is the last score its games reach, never past its
// length: a gammon at 5-5 in a 7-point match ends at 7.
// A game's points go to its Game.Winner (1 = player 1, -1 = player 2, see
// domain.WinnerPlayer1); an unfinished game adds nothing.
func (s *DirectionStore) FilledSlots(ctx context.Context, scope string, tournamentID int64) ([]storage.FilledSlot, error) {
	tenant, targs := s.DB.TenantFilter("m", scope)
	rows, err := s.DB.Query(ctx, `
		SELECT m.direction_match_id, m.id, COALESCE(m.player1_name,''), COALESCE(m.player2_name,''),
		       COALESCE(m.match_length,0),
		       (SELECT MAX(g.initial_score_1 + CASE WHEN g.points_won > 0 AND g.winner = 1 THEN g.points_won ELSE 0 END) FROM game g WHERE g.match_id = m.id),
		       (SELECT MAX(g.initial_score_2 + CASE WHEN g.points_won > 0 AND g.winner = -1 THEN g.points_won ELSE 0 END) FROM game g WHERE g.match_id = m.id)
		  FROM match m
		 WHERE `+tenant+` AND m.tournament_id = ? AND m.direction_match_id <> ''
		 ORDER BY m.id`, append(targs, tournamentID)...)
	if err != nil {
		return nil, errf(s.DB, "filled slots", err)
	}
	defer rows.Close()
	var out []storage.FilledSlot
	for rows.Next() {
		var (
			f      storage.FilledSlot
			s1, s2 *int64
		)
		if err := rows.Scan(&f.SlotID, &f.MatchID, &f.Player1, &f.Player2, &f.Length, &s1, &s2); err != nil {
			return nil, errf(s.DB, "filled slots", err)
		}
		if s1 != nil && s2 != nil {
			f.Score1, f.Score2, f.HasScore = capScore(int(*s1), f.Length), capScore(int(*s2), f.Length), true
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "filled slots", err)
	}
	return out, nil
}

// capScore stops a score at the match length; a money session (length 0)
// has none.
func capScore(score, length int) int {
	if length > 0 && score > length {
		return length
	}
	return score
}

// UnattachedMatches returns the Matches of the Tournament that fill no Slot.
func (s *DirectionStore) UnattachedMatches(ctx context.Context, scope string, tournamentID int64) ([]storage.SlotCandidate, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `
		SELECT id, COALESCE(player1_name,''), COALESCE(player2_name,''),
		       COALESCE(match_length,0), `+s.DB.DateText("match_date")+`
		  FROM match
		 WHERE `+tenant+` AND tournament_id = ? AND (direction_match_id IS NULL OR direction_match_id = '')
		 ORDER BY id`, append(targs, tournamentID)...)
	if err != nil {
		return nil, errf(s.DB, "unattached matches", err)
	}
	defer rows.Close()
	var out []storage.SlotCandidate
	for rows.Next() {
		var c storage.SlotCandidate
		if err := rows.Scan(&c.MatchID, &c.Player1, &c.Player2, &c.Length, &c.Date); err != nil {
			return nil, errf(s.DB, "unattached matches", err)
		}
		// SQLite hands back the stored timestamp text; the contract is the day.
		if len(c.Date) > len("2006-01-02") {
			c.Date = c.Date[:len("2006-01-02")]
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "unattached matches", err)
	}
	return out, nil
}

// AttachSlot fills a Slot with a Match. The unique (tournament, slot) index
// refuses a second Match in an occupied Slot. The Tournament must be the
// scope's as much as the Match.
func (s *DirectionStore) AttachSlot(ctx context.Context, scope string, tournamentID int64, slotID string, matchID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	ttenant, ttargs := s.DB.TenantFilter("t", scope)
	args := append([]any{tournamentID, slotID, matchID}, targs...)
	args = append(append(args, tournamentID), ttargs...)
	n, err := s.DB.Exec(ctx, `UPDATE match SET tournament_id = ?, direction_match_id = ? WHERE id = ? AND `+tenant+`
		AND EXISTS (SELECT 1 FROM tournament t WHERE t.id = ? AND `+ttenant+`)`, args...)
	if err != nil {
		return errf(s.DB, "attach slot", err)
	}
	if n == 0 {
		return fmt.Errorf("%s: attach slot: match %d in tournament %d: %w", s.DB.Name(), matchID, tournamentID, storage.ErrNotFound)
	}
	return nil
}

// DetachSlot empties a Slot; the Match keeps its Tournament.
func (s *DirectionStore) DetachSlot(ctx context.Context, scope string, tournamentID int64, slotID string) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx, `UPDATE match SET direction_match_id = '' WHERE tournament_id = ? AND direction_match_id = ? AND `+tenant,
		append([]any{tournamentID, slotID}, targs...)...); err != nil {
		return errf(s.DB, "detach slot", err)
	}
	return nil
}

// SlotOf returns where a Match sits, slotID "" when it fills no Slot.
func (s *DirectionStore) SlotOf(ctx context.Context, scope string, matchID int64) (int64, string, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var (
		tid  *int64
		slot *string
	)
	err := s.DB.QueryRow(ctx, `SELECT tournament_id, direction_match_id FROM match WHERE id = ? AND `+tenant,
		append([]any{matchID}, targs...)...).Scan(&tid, &slot)
	if errors.Is(err, ErrNoRows) {
		return 0, "", fmt.Errorf("%s: slot of match %d: %w", s.DB.Name(), matchID, storage.ErrNotFound)
	}
	if err != nil {
		return 0, "", errf(s.DB, "slot of match", err)
	}
	if tid == nil || slot == nil || *slot == "" {
		return 0, "", nil
	}
	return *tid, *slot, nil
}

// Pairs returns the persons of every pair of the Tournament.
func (s *DirectionStore) Pairs(ctx context.Context, scope string, tournamentID int64) (map[string][]storage.PairMember, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT player_id, name, club, rating FROM direction_pair_member
		WHERE tournament_id = ? AND `+tenant+` ORDER BY player_id, seat`, append([]any{tournamentID}, targs...)...)
	if err != nil {
		return nil, errf(s.DB, "pairs", err)
	}
	defer rows.Close()
	out := map[string][]storage.PairMember{}
	for rows.Next() {
		var (
			id string
			m  storage.PairMember
		)
		if err := rows.Scan(&id, &m.Name, &m.Club, &m.Rating); err != nil {
			return nil, errf(s.DB, "pairs", err)
		}
		out[id] = append(out[id], m)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "pairs", err)
	}
	return out, nil
}

// SetPair replaces the persons behind one Participant, seat by seat.
func (s *DirectionStore) SetPair(ctx context.Context, scope string, tournamentID int64, participantID string, members []storage.PairMember) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `DELETE FROM direction_pair_member WHERE tournament_id = ? AND player_id = ? AND `+tenant,
			append([]any{tournamentID, participantID}, targs...)...); err != nil {
			return errf(tx, "set pair", err)
		}
		for seat, m := range members {
			cols, args := tx.TenantColumns(scope)
			cols = append(cols, "tournament_id", "player_id", "seat", "name", "club", "rating")
			args = append(args, tournamentID, participantID, seat, m.Name, m.Club, m.Rating)
			if _, err := tx.Exec(ctx, `INSERT INTO direction_pair_member (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...); err != nil {
				return errf(tx, "set pair", err)
			}
		}
		return nil
	})
}
