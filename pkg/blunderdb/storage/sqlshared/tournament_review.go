package sqlshared

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// reviewMatchRow is one match as the tournament review lists it.
type reviewMatchRow struct {
	id         int64
	p1, p2     string
	length     int
	day        string
	tournament int64
}

// TournamentReview gathers a player's matches of the tournament and of the
// usual window, reads each through MatchDecisionLosses — the badges'
// decisions and conversion — and the study plan of the player's errors in the
// tournament, and hands them to storage.BuildTournamentReview (ADR-0081).
func (s *StatsStore) TournamentReview(ctx context.Context, scope string, tournamentID int64, player string) (storage.TournamentReview, error) {
	tenant, args := s.DB.TenantFilter("t", scope)
	var name, date string
	found := false
	err := scanEach(ctx, s.DB, `SELECT t.name, COALESCE(t.date, '') FROM tournament t WHERE `+tenant+` AND t.id = ?`,
		append(args, tournamentID), func(r Rows) error {
			found = true
			return r.Scan(&name, &date)
		})
	if err != nil {
		return storage.TournamentReview{}, errf(s.DB, "TournamentReview tournament", err)
	}
	if !found {
		return storage.TournamentReview{}, storage.ErrNotFound
	}

	mtenant, margs := s.DB.TenantFilter("m", scope)
	cols := `SELECT m.id, COALESCE(m.player1_name, ''), COALESCE(m.player2_name, ''), COALESCE(m.match_length, 0), ` +
		s.DB.DateText("m.match_date") + `, COALESCE(m.tournament_id, 0) FROM match m WHERE ` + mtenant
	scanMatch := func(out *[]reviewMatchRow) func(Rows) error {
		return func(r Rows) error {
			var m reviewMatchRow
			if err := r.Scan(&m.id, &m.p1, &m.p2, &m.length, &m.day, &m.tournament); err != nil {
				return err
			}
			*out = append(*out, m)
			return nil
		}
	}
	var inTournament []reviewMatchRow
	// Rounds run in the tournament's order, ties by date then by id.
	if err := scanEach(ctx, s.DB, cols+` AND m.tournament_id = ?
		ORDER BY COALESCE(m.tournament_sort_order, 0), m.match_date, m.id`,
		append(append([]any{}, margs...), tournamentID), scanMatch(&inTournament)); err != nil {
		return storage.TournamentReview{}, errf(s.DB, "TournamentReview matches", err)
	}

	if player == "" {
		player = mostFrequentPlayer(inTournament)
	}
	review := storage.TournamentReview{TournamentID: tournamentID, Name: name, Date: date, Player: player}
	if player == "" {
		return storage.BuildTournamentReview(review, nil, nil, nil), nil
	}
	filter, err := s.withPlayerAliases(ctx, scope, storage.StatsFilter{PlayerName: player, DecisionType: -1})
	if err != nil {
		return storage.TournamentReview{}, errf(s.DB, "TournamentReview aliases", err)
	}
	names := map[string]bool{}
	for _, n := range storage.PlayerNameSet(filter) {
		names[n] = true
	}

	ref := reviewDay(date)
	if ref == "" {
		for _, m := range inTournament {
			if d := reviewDay(m.day); d != "" && (ref == "" || d < ref) {
				ref = d
			}
		}
	}
	var usualRows []reviewMatchRow
	if ref != "" {
		day, _ := time.Parse(time.DateOnly, ref)
		review.Usual.From = day.AddDate(0, 0, -storage.TournamentUsualDays).Format(time.DateOnly)
		review.Usual.To = ref
		var all []reviewMatchRow
		clause, pargs := playerFilterClause(storage.PlayerNameSet(filter), false)
		if err := scanEach(ctx, s.DB, cols+` AND m.match_length BETWEEN 1 AND 64 AND `+clause+` ORDER BY m.match_date, m.id`,
			append(append([]any{}, margs...), pargs...), scanMatch(&all)); err != nil {
			return storage.TournamentReview{}, errf(s.DB, "TournamentReview usual", err)
		}
		for _, m := range all {
			if d := reviewDay(m.day); m.tournament != tournamentID && d >= review.Usual.From && d != "" && d < ref {
				usualRows = append(usualRows, m)
			}
		}
	}

	matches, err := s.reviewMatches(ctx, scope, inTournament, names)
	if err != nil {
		return storage.TournamentReview{}, err
	}
	usual, err := s.reviewMatches(ctx, scope, usualRows, names)
	if err != nil {
		return storage.TournamentReview{}, err
	}
	plan, err := s.StudyPlan(ctx, scope, storage.StatsFilter{PlayerName: player, TournamentIDs: []int64{tournamentID}, DecisionType: -1})
	if err != nil {
		return storage.TournamentReview{}, err
	}
	return storage.BuildTournamentReview(review, matches, usual, plan), nil
}

// reviewMatches reads the decisions of the rows the player (any of names)
// played, the player's seat first found as player 1.
func (s *StatsStore) reviewMatches(ctx context.Context, scope string, rows []reviewMatchRow, names map[string]bool) ([]storage.ReviewMatch, error) {
	var out []storage.ReviewMatch
	for _, m := range rows {
		seat, opponent := 0, m.p2
		switch {
		case names[m.p1]:
		case names[m.p2]:
			seat, opponent = 1, m.p1
		default:
			continue
		}
		decisions, err := s.MatchDecisionLosses(ctx, scope, m.id)
		if err != nil {
			return nil, err
		}
		out = append(out, storage.ReviewMatch{MatchID: m.id, Opponent: opponent, Date: reviewDay(m.day),
			Length: m.length, Seat: seat, Decisions: decisions})
	}
	return out, nil
}

// mostFrequentPlayer is the name in the most matches, ties to the first in
// alphabetical order.
func mostFrequentPlayer(rows []reviewMatchRow) string {
	count := map[string]int{}
	for _, m := range rows {
		for _, n := range []string{m.p1, m.p2} {
			if n != "" {
				count[n]++
			}
		}
	}
	names := make([]string, 0, len(count))
	for n := range count {
		names = append(names, n)
	}
	slices.SortFunc(names, func(a, b string) int { return cmp.Or(cmp.Compare(count[b], count[a]), cmp.Compare(a, b)) })
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// reviewDay is the YYYY-MM-DD day a date text starts with, "" when it does
// not start with one: a tournament date is typed by hand.
func reviewDay(text string) string {
	if len(text) < 10 {
		return ""
	}
	if _, err := time.Parse(time.DateOnly, text[:10]); err != nil {
		return ""
	}
	return text[:10]
}
