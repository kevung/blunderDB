package sqlshared

// The effective date of a tournament is the date the user entered, else the
// day of the earliest of all the tournament's matches, else empty. It is
// derived on read and never written (ADR-0007), so a filter on the matches
// never moves it.
//
// It is read through one aggregate of the match table joined to the
// tournaments, never a subquery per tournament: match.tournament_id has no
// index, and a correlated subquery scans every match once per tournament.

// TournamentEffectiveDate is the effective date of the tournament aliased t,
// over the aggregate TournamentMatchAgg joins as tma.
const TournamentEffectiveDate = `COALESCE(NULLIF(t.date,''), tma.first_day, '')`

// TournamentMatchCount is the number of matches of the tournament aliased t,
// over the same aggregate.
const TournamentMatchCount = `COALESCE(tma.match_count, 0)`

// TournamentMatchAgg renders the LEFT JOIN of the per-tournament match
// aggregate (alias tma) onto the tournament aliased t. filter confines the
// matches aggregated (alias tm) — their tenant, written in the caller's
// placeholder style. perTenant groups and joins on the tenant too, for the
// backend that has one, so a tournament only counts its own tenant's matches.
func TournamentMatchAgg(dateText func(col string) string, filter string, perTenant bool) string {
	tenantCol, tenantOn := "", ""
	if perTenant {
		tenantCol = ", tm.tenant_id"
		tenantOn = " AND tma.tenant_id = t.tenant_id"
	}
	return ` LEFT JOIN (SELECT tm.tournament_id` + tenantCol + `, COUNT(*) AS match_count,
	   MIN(NULLIF(` + dateText("tm.match_date") + `,'')) AS first_day
	   FROM match tm WHERE ` + filter + ` AND tm.tournament_id IS NOT NULL
	   GROUP BY tm.tournament_id` + tenantCol + `) tma
	 ON tma.tournament_id = t.id` + tenantOn
}

// tournamentMatchAgg is TournamentMatchAgg in sqlshared's placeholder style,
// with its arguments, which precede the WHERE clause's. narrow, when not
// empty, confines the matches aggregated further, with its own arguments.
func tournamentMatchAgg(d Dialect, scope, narrow string, narrowArgs ...any) (string, []any) {
	filter, args := d.TenantFilter("tm", scope)
	if narrow != "" {
		filter += " AND " + narrow
		args = append(args, narrowArgs...)
	}
	cols, _ := d.TenantColumns(scope)
	return TournamentMatchAgg(d.DateText, filter, len(cols) > 0), args
}

// tournamentChronology orders tournaments by their effective date eff,
// oldest first, undated last; creation breaks ties.
func tournamentChronology(eff string) string {
	return ` ORDER BY ` + eff + ` = '', ` + eff + `, t.created_at, t.id`
}
