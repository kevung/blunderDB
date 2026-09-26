package domain

// Rencontre is the room several directed Tournaments share (ADR-0056): a
// festival's main event, speed and doubles, run by one director at the same
// tables. It owns the facts about the room — how many tables, where its wall
// page goes. What happens in the room (a table out of service, a break) is
// an event in the log of every member Direction, never a column here.
type Rencontre struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	StartsOn  string `json:"startsOn"`
	EndsOn    string `json:"endsOn"`
	Tables    int    `json:"tables"`
	OutputDir string `json:"outputDir"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	// TournamentIDs are the member Tournaments, in id order. Membership is
	// tournament.rencontre_id: a Tournament is in at most one Rencontre.
	TournamentIDs []int64 `json:"tournamentIds"`
}

// TrashRencontrePayload is a deleted Rencontre: the room and the Tournaments
// it held, so a restore gives the room back with its members.
type TrashRencontrePayload struct {
	Rencontre Rencontre `json:"rencontre"`
}
