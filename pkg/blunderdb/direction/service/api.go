package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The Direction surface the frontend and the CLI both call (ADR-0047): thin calls into package
// direction, no rule here. NOTHING returned is a translated string: the engine's codes are
// rendered by the frontend.

// DirectionSummary is what a list of directed tournaments shows without replaying any of them.
type DirectionSummary struct {
	TournamentID  int64  `json:"tournamentId"`
	State         string `json:"state"`
	EngineVersion string `json:"engineVersion"`
	OutputDir     string `json:"outputDir"`
	UpdatedAt     string `json:"updatedAt"`
	// RencontreName is the room this Tournament plays in, empty when none (ADR-0056). Named, not
	// just an id: `tournament list` is read without a GUI to cross-reference it against.
	RencontreName string `json:"rencontreName,omitempty"`
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
	// Infos says, for every entrant engaged in no phase yet, where they will enter — a free
	// bye of the phase under way, a later phase open to all, or nowhere. Derived at every call.
	Infos      []tournoi.Info   `json:"infos,omitempty"`
	Running    []*tournoi.Match `json:"running"`
	Phase      int              `json:"phase"`
	Finished   bool             `json:"finished"`
	EventCount int              `json:"eventCount"`
	// RencontreID is the room this Tournament plays in, 0 when none (ADR-0056).
	RencontreID int64 `json:"rencontreId"`
	// BusyTables are the tables the sister events of the Rencontre play on right now: the
	// proposals above already avoid them. Replayed, never stored.
	BusyTables []int `json:"busyTables"`
	// Elsewhere says, by Participant id, where each one who plays in a sister event sits right
	// now — a pair as soon as one of its members does. The proposals already hold them back; a
	// manual pairing of one is accepted and shown with this seat. Replayed, never stored.
	Elsewhere map[string]direction.Seat `json:"elsewhere,omitempty"`
	// Pairs gives the two persons behind each doubles Participant, by Participant id; empty for
	// a singles event (ADR-0056 §4).
	Pairs map[string][]PairMember `json:"pairs,omitempty"`
	// TableSettings are the effective table properties — the Rencontre's when the Tournament
	// plays in one, its own otherwise (ADR-0058 §3) — by number.
	TableSettings []domain.TableSetting `json:"tableSettings"`
	// Rooms are the rooms the Tournament may play in; empty means every table.
	Rooms []string `json:"rooms"`
}

// ListDirections names the directed tournaments of this database.
func (d *Service) ListDirections(ctx context.Context) ([]DirectionSummary, error) {
	recs, err := d.dirStore().ListDirections(ctx)
	if err != nil {
		return nil, err
	}
	names := map[int64]string{} // Rencontre id -> name, fetched once each
	out := make([]DirectionSummary, 0, len(recs))
	for _, r := range recs {
		s := DirectionSummary{
			TournamentID: r.TournamentID, State: string(r.State),
			EngineVersion: r.EngineVersion, OutputDir: r.OutputDir,
			UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
		}
		if rid, err := d.st.Rencontres().Of(ctx, d.scope, r.TournamentID); err == nil && rid != 0 {
			if name, ok := names[rid]; ok {
				s.RencontreName = name
			} else if rc, err := d.st.Rencontres().Get(ctx, d.scope, rid); err == nil {
				names[rid] = rc.Name
				s.RencontreName = rc.Name
			}
		}
		out = append(out, s)
	}
	return out, nil
}

// CreateDirection starts directing a Tournament that has none. The configuration arrives as the
// engine's own JSON so the frontend composes it without this file knowing every format option.
// A seed of 0 means "pick one" (and record it, so the draw stays reproducible).
func (d *Service) CreateDirection(ctx context.Context, tournamentID int64, configJSON string, seed int64) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	cfg, err := parseDirectionConfig(configJSON)
	if err != nil {
		return err
	}
	_, err = direction.Create(ctx, d.dirStore(), tournamentID, cfg, seed, time.Now())
	return err
}

// GetDirection replays a Tournament's Direction and returns everything the panel shows.
func (d *Service) GetDirection(ctx context.Context, tournamentID int64) (*DirectionView, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
	v.RencontreID, _ = d.RencontreOf(ctx, tournamentID)
	if v.Pairs, err = d.Pairs(ctx, tournamentID); err != nil {
		return nil, err
	}
	plan, err := d.planFor(ctx, tournamentID, cfg)
	if err != nil {
		return nil, err
	}
	v.TableSettings, v.Rooms = plan.Settings, plan.Rooms
	if st := dir.State(); st != nil {
		// Proposed at the WALL CLOCK, not the journal's last timestamp: a micro-round's
		// deadline and a break's warning depend on the current time, not on the last result.
		room := d.roomAround(ctx, tournamentID, dir)
		ext := room.external()
		v.BusyTables = ext.BusyTables
		if len(room.players) > 0 {
			v.Elsewhere = make(map[string]direction.Seat, len(room.players))
			for id, seat := range room.players {
				v.Elsewhere[string(id)] = seat
			}
		}
		v.Proposals = room.propose(dir, time.Now())
		v.Warnings = dir.Warnings()
		v.Ranking = dir.Ranking()
		v.Running = st.Running()
		v.Phase = st.Current
		v.Finished = st.Finished
		v.Infos = st.Infos
		for _, id := range st.Order {
			if p := st.Players[id]; p != nil {
				v.Players = append(v.Players, *p)
			}
		}
	}
	return v, nil
}

// HasDirection says whether a Tournament is directed, without replaying it.
func (d *Service) HasDirection(ctx context.Context, tournamentID int64) (bool, error) {
	_, err := d.dirStore().GetDirection(ctx, tournamentID)
	if errors.Is(err, direction.ErrNoDirection) {
		return false, nil
	}
	return err == nil, err
}

// SetDirectionConfig installs a configuration, in preparation and in the middle of a tournament
// alike. It is always an event, so decisions stay readable in order. PreviewDirectionConfig
// shows the engine's refusal BEFORE the click; the refusal returned here is the same *tournoi.ConfigRefusal.
func (d *Service) SetDirectionConfig(ctx context.Context, tournamentID int64, configJSON string) error {
	if err := d.setDirectionConfig(ctx, tournamentID, configJSON); err != nil {
		return err
	}
	// A configuration a room shares changes every sister's page, not only this one.
	d.writeRoomPages(context.WithoutCancel(ctx), tournamentID)
	return nil
}

func (d *Service) setDirectionConfig(ctx context.Context, tournamentID int64, configJSON string) (err error) {
	d, release, err := d.lockRoom(ctx, tournamentID, 0)
	if err != nil {
		return err
	}
	defer release(&err)
	cfg, err := parseDirectionConfig(configJSON)
	if err != nil {
		return err
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	if rid, _ := d.RencontreOf(ctx, tournamentID); rid != 0 {
		if cur, err := dir.Config(); err == nil && !direction.SameRoom(cur, direction.RoomOf(cfg)) {
			if err := cfg.Validate(); err != nil {
				return direction.Refused(err)
			}
			return d.setMemberConfig(ctx, rid, tournamentID, cfg)
		}
	}
	return dir.SetConfig(ctx, cfg)
}

// SetDirectionOutputDir remembers where the standalone display page is written.
func (d *Service) SetDirectionOutputDir(ctx context.Context, tournamentID int64, dir string) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dd, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	return dd.SetOutputDir(ctx, dir)
}

// EnterParticipants records several entries at once — what an import or "take last time's
// entrants" produces. Entering is possible in preparation and afterwards alike.
func (d *Service) EnterParticipants(ctx context.Context, tournamentID int64, playersJSON string) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	var players []tournoi.Player
	if playersJSON == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(playersJSON), &players); err != nil {
		return direction.Refused(fmt.Errorf("entries: %w", err))
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	st := dir.State()
	if st == nil {
		return direction.ErrNoDirection
	}
	taken := map[string]bool{}
	for id := range st.Players {
		taken[string(id)] = true
	}
	now := time.Now()
	for _, p := range players {
		if p.ID == "" {
			p.ID = tournoi.PlayerID(participantID(p.Name, taken))
		}
		taken[string(p.ID)] = true
		if err := dir.Enter(ctx, p, now); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDirection removes a Direction and its log. The Tournament and its Matches stay; the
// Matches merely lose the Slot they filled (ADR-0047).
func (d *Service) DeleteDirection(ctx context.Context, tournamentID int64) (err error) {
	d, release, err := d.lockRoom(ctx, tournamentID, 0)
	if err != nil {
		return err
	}
	defer release(&err)
	return d.dirStore().DeleteDirection(ctx, tournamentID)
}

// parseDirectionConfig reads and validates a configuration coming from the frontend.
func parseDirectionConfig(configJSON string) (tournoi.Config, error) {
	var cfg tournoi.Config
	if configJSON == "" {
		return cfg, direction.Refusef("direction: empty configuration")
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return cfg, direction.Refused(fmt.Errorf("direction: configuration: %w", err))
	}
	if err := cfg.Validate(); err != nil {
		return cfg, direction.Refused(err)
	}
	return cfg, nil
}

// DirectionJournalJSON gives a Direction's raw event journal: the whole truth, from which
// everything else is replayed. Readable with the engine alone — an exit, not a lock-in.
func (d *Service) DirectionJournalJSON(ctx context.Context, tournamentID int64) (string, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return "", err
	}
	blob, err := json.MarshalIndent(dir.Journal(), "", "  ")
	if err != nil {
		return "", fmt.Errorf("direction: journal: %w", err)
	}
	return string(blob) + "\n", nil
}
