package database

import (
	"context"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Statistiques d'étude corrélées au jeu réel.
//
// Ce fichier ne rend PAS un effet ni une corrélation : par plan de jeu, trois
// nombres côte à côte (positions révisées depuis une date, PR avant, PR
// depuis). Rien ici ne contrôle les facteurs confondants, donc aucune
// causalité n'est affirmée ; le rapprochement est celui du lecteur.
//
// Les positions RÉVISÉES sont comptées distinctes (quatre révisions d'une
// carte = une position couverte) ; les DÉCISIONS du PR sont toutes comptées.

// StudyImpactRow is one plan of play, with what was studied and what was
// played on either side of the date.
type StudyImpactRow struct {
	// GameType is the stable token of domain.GameType.
	GameType string `json:"gameType"`
	// Reviewed is how many DISTINCT positions of this plan were revised since
	// the cutoff.
	Reviewed int `json:"reviewed"`
	// PRBefore / PRAfter are the Performance Ratings over the matches played
	// before and since the cutoff, with the decision counts behind them. A PR
	// is unreadable without its count, so neither travels alone.
	PRBefore        float64 `json:"prBefore"`
	DecisionsBefore int     `json:"decisionsBefore"`
	PRAfter         float64 `json:"prAfter"`
	DecisionsAfter  int     `json:"decisionsAfter"`
}

// StudyImpactMinDecisions is the sample below which a PR is not worth reading.
// It is NOT enforced here — the row comes back with its counts and the caller
// greys it — for the same reason storage.MinCellDecisions is not: hiding the
// count would make the figure unauditable.
const StudyImpactMinDecisions = 10

// StudyImpact returns, per plan of play, what was revised over the last
// `days` days and the Performance Rating before and since — three numbers to
// read side by side, never an effect.
func (d *Database) StudyImpact(days int) ([]StudyImpactRow, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	reviewed, err := d.reviewsByGameType(cutoff)
	if err != nil {
		return nil, err
	}

	before, err := d.statsByGameType("", cutoff)
	if err != nil {
		return nil, err
	}
	after, err := d.statsByGameType(cutoff, "")
	if err != nil {
		return nil, err
	}

	// Every plan that appears anywhere gets a row, including one that was
	// revised and never met since: "you studied backgames and have not played
	// one" is an answer, and dropping the row would hide it.
	seen := map[string]bool{}
	var rows []StudyImpactRow
	for _, source := range []map[string]GameTypeStats{before, after} {
		for name := range source {
			seen[name] = true
		}
	}
	for name := range reviewed {
		seen[name] = true
	}
	for name := range seen {
		if name == "unknown" {
			// A database whose plans have never been computed reports
			// everything here; the row would say nothing and `blunderdb
			// repair` is what fixes it.
			continue
		}
		rows = append(rows, StudyImpactRow{
			GameType:        name,
			Reviewed:        reviewed[name],
			PRBefore:        before[name].PR,
			DecisionsBefore: before[name].NumDecisions,
			PRAfter:         after[name].PR,
			DecisionsAfter:  after[name].NumDecisions,
		})
	}
	sortStudyImpact(rows)
	return rows, nil
}

func (d *Database) reviewsByGameType(cutoff string) (map[string]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.store == nil {
		return nil, fmt.Errorf("study impact: %w", storage.ErrInternal)
	}
	return d.store.Anki().ReviewsByGameType(context.Background(), "", cutoff)
}

// statsByGameType runs the ordinary statistics over a date window and keeps
// the per-plan breakdown. Running the SAME computation the Stats tab shows is
// the point: a second PR, computed here, would be a second PR.
func (d *Database) statsByGameType(from, to string) (map[string]GameTypeStats, error) {
	stats, err := d.ComputeStats(StatsFilter{DateFrom: from, DateTo: to, DecisionType: -1})
	if err != nil {
		return nil, fmt.Errorf("study impact: %w", err)
	}
	out := make(map[string]GameTypeStats, len(stats.PerGameType))
	for _, row := range stats.PerGameType {
		out[row.GameType] = row
	}
	return out, nil
}

// sortStudyImpact puts the plans most revised first: the table exists to read
// what was studied against what was played, so what was studied leads.
func sortStudyImpact(rows []StudyImpactRow) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			a, b := rows[j-1], rows[j]
			if a.Reviewed > b.Reviewed || (a.Reviewed == b.Reviewed && a.GameType <= b.GameType) {
				break
			}
			rows[j-1], rows[j] = b, a
		}
	}
}
