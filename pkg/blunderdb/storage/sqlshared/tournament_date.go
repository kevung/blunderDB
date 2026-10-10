package sqlshared

// TournamentEffectiveDate renders the SQL expression of a tournament's
// effective date, over a tournament aliased t: the date the user entered,
// else the day of the earliest of all the tournament's matches, else empty.
// It is derived on read and never written (ADR-0007), so a filter on the
// matches never moves it. dateText is the backend's DateText.
func TournamentEffectiveDate(dateText func(col string) string) string {
	return `COALESCE(NULLIF(t.date,''),
	(SELECT MIN(NULLIF(` + dateText("tm.match_date") + `,'')) FROM match tm
	  WHERE tm.tournament_id = t.id), '')`
}

// tournamentChronology orders tournaments by their effective date eff,
// oldest first, undated last; creation breaks ties.
func tournamentChronology(eff string) string {
	return ` ORDER BY ` + eff + ` = '', ` + eff + `, t.created_at, t.id`
}
