package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The Direction surface the frontend and the CLI both call (ADR-0047, CLAUDE.md CLI/GUI/server
// parity). Everything here is a thin call into package direction: no rule lives in this file.
//
// One shape runs through all of it: NOTHING returned here is a translated string. Labels,
// ranking notes, warnings and wait reasons are the engine's codes, and the frontend renders
// them in the user's language. blunderDB speaks nine.

// DirectionSummary is what a list of directed tournaments shows without replaying any of them.
type DirectionSummary struct {
	TournamentID  int64  `json:"tournamentId"`
	State         string `json:"state"`
	EngineVersion string `json:"engineVersion"`
	OutputDir     string `json:"outputDir"`
	UpdatedAt     string `json:"updatedAt"`
}

// DirectionView is a replayed Direction as the panel needs it. The derived parts — proposals,
// standings, warnings — are recomputed at every call and never stored.
type DirectionView struct {
	TournamentID  int64             `json:"tournamentId"`
	State         string            `json:"state"`
	EngineVersion string            `json:"engineVersion"`
	OutputDir     string            `json:"outputDir"`
	Config        tournoi.Config    `json:"config"`
	Proposals     []tournoi.Action  `json:"proposals"`
	Warnings      []tournoi.Warning `json:"warnings"`
	Ranking       []tournoi.Rank    `json:"ranking"`
	Players       []tournoi.Player  `json:"players"`
	Running       []*tournoi.Match  `json:"running"`
	Phase         int               `json:"phase"`
	Finished      bool              `json:"finished"`
	EventCount    int               `json:"eventCount"`
}

// ListDirections names the directed tournaments of this database.
func (d *Database) ListDirections() ([]DirectionSummary, error) {
	recs, err := d.DirectionStore().ListDirections(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]DirectionSummary, 0, len(recs))
	for _, r := range recs {
		out = append(out, DirectionSummary{
			TournamentID: r.TournamentID, State: string(r.State),
			EngineVersion: r.EngineVersion, OutputDir: r.OutputDir,
			UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}

// CreateDirection starts directing a Tournament that has none. The configuration arrives as the
// engine's own JSON so the frontend composes it without this file knowing every format option.
func (d *Database) CreateDirection(tournamentID int64, configJSON string) error {
	cfg, err := parseDirectionConfig(configJSON)
	if err != nil {
		return err
	}
	_, err = direction.Create(context.Background(), d.DirectionStore(), tournamentID, cfg)
	return err
}

// GetDirection replays a Tournament's Direction and returns everything the panel shows.
func (d *Database) GetDirection(tournamentID int64) (*DirectionView, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	cfg, err := dir.Config()
	if err != nil {
		return nil, err
	}
	rec := dir.Record()
	v := &DirectionView{
		TournamentID: rec.TournamentID, State: string(rec.State),
		EngineVersion: rec.EngineVersion, OutputDir: rec.OutputDir,
		Config: cfg, EventCount: len(dir.Journal()),
	}
	if st := dir.State(); st != nil {
		v.Proposals = dir.Propose()
		v.Warnings = dir.Warnings()
		v.Ranking = dir.Ranking()
		v.Running = st.Running()
		v.Phase = st.Current
		v.Finished = st.Finished
		for _, id := range st.Order {
			if p := st.Players[id]; p != nil {
				v.Players = append(v.Players, *p)
			}
		}
	}
	return v, nil
}

// HasDirection says whether a Tournament is directed, without replaying it.
func (d *Database) HasDirection(tournamentID int64) (bool, error) {
	_, err := d.DirectionStore().GetDirection(context.Background(), tournamentID)
	if errors.Is(err, direction.ErrNoDirection) {
		return false, nil
	}
	return err == nil, err
}

// SetDirectionConfig rewrites the draft configuration. Refused once the tournament has started:
// from there a change is an event, so what the director decided stays readable in order.
func (d *Database) SetDirectionConfig(tournamentID int64, configJSON string) error {
	cfg, err := parseDirectionConfig(configJSON)
	if err != nil {
		return err
	}
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return err
	}
	return dir.SetConfig(context.Background(), cfg)
}

// SetDirectionOutputDir remembers where the standalone display page is written.
func (d *Database) SetDirectionOutputDir(tournamentID int64, dir string) error {
	dd, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return err
	}
	return dd.SetOutputDir(context.Background(), dir)
}

// StartDirection writes the created event and the entries, freezing the configuration.
func (d *Database) StartDirection(tournamentID int64, seed int64, playersJSON string) error {
	var players []tournoi.Player
	if playersJSON != "" {
		if err := json.Unmarshal([]byte(playersJSON), &players); err != nil {
			return fmt.Errorf("entries: %w", err)
		}
	}
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return err
	}
	if seed == 0 {
		// A draw must be reproducible, so the seed is recorded — but nobody should have to
		// invent one. The clock is as good a source as any and it is written in the log.
		seed = time.Now().UnixNano()
	}
	return dir.Start(context.Background(), seed, time.Now(), players)
}

// DeleteDirection removes a Direction and its log. The Tournament and its Matches stay; the
// Matches merely lose the Slot they filled (ADR-0047).
func (d *Database) DeleteDirection(tournamentID int64) error {
	return d.DirectionStore().DeleteDirection(context.Background(), tournamentID)
}

// parseDirectionConfig reads and validates a configuration coming from the frontend.
func parseDirectionConfig(configJSON string) (tournoi.Config, error) {
	var cfg tournoi.Config
	if configJSON == "" {
		return cfg, fmt.Errorf("direction: empty configuration")
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return cfg, fmt.Errorf("direction: configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}
