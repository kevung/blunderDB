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
	// TableSettings are the tables that carry properties, by number; a table
	// absent here is an ordinary table (ADR-0058).
	TableSettings []TableSetting `json:"tableSettings"`
	// EventRooms are the rooms each member Tournament may play in, by
	// Tournament id. A member absent or with an empty list plays on every
	// table: rooms are labels on tables, not entities (ADR-0058 §2, §5).
	EventRooms map[int64][]string `json:"eventRooms"`
}

// TableSetting holds the properties of one table (ADR-0058). The number is
// its identity: the director, the CLI and the players say "table 23", and a
// number is unique among the tables of its owner.
type TableSetting struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Room     string `json:"room"`
	Reserved bool   `json:"reserved"`
	// AssignedTo names the persons the table is kept for; the name is the
	// link (ADR-0056 §3). Never nil once read back.
	AssignedTo []string `json:"assignedTo"`
}

// TrashRencontrePayload is a deleted Rencontre: the room and the Tournaments
// it held, so a restore gives the room back with its members.
type TrashRencontrePayload struct {
	Rencontre Rencontre `json:"rencontre"`
}
