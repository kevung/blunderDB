package direction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// EngineVersion is the Nicomaque tag this build embeds. It is stored with every Direction and
// shown by the credit button, so a replay problem can be traced to the engine that wrote it.
// Keep it in step with the require line in go.mod.
const EngineVersion = "v0.2.0"

// State is where a Direction stands in its life (ADR-0047 § "Le cycle de vie").
type State string

const (
	// StateDraft: no match launched yet. The configuration is rewritten in place and entries
	// come and go freely.
	StateDraft State = "draft"
	// StateRunning: the first match froze the configuration into the created event. From here
	// the configuration only changes through an event.
	StateRunning State = "running"
	// StateFinished: the tournament is closed. Read-only, until a reopening event.
	StateFinished State = "finished"
)

// Record is the row that carries a Direction's own facts. The derived state is not in it.
type Record struct {
	TournamentID  int64
	FormatVersion int
	EngineVersion string
	State         State
	// Config is the configuration being edited, and is only meaningful while the Direction is
	// a draft: once running, the configuration of record is the one in the created event.
	Config    string
	OutputDir string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StoredEvent is one line of the log as persistence hands it back.
type StoredEvent struct {
	Seq     int
	Kind    string
	Time    time.Time
	Payload []byte
}

// Store is everything this package needs from persistence. It is deliberately small, and it is
// the whole reason the package does no SQL: the desktop wrapper and both storage backends
// implement it, so the logic is written once (CLAUDE.md, CLI/GUI/server parity).
//
// AppendEvent must be atomic with respect to the caller's transaction and must REFUSE to
// overwrite an existing sequence number — the log is append-only, and a silent overwrite would
// lose a decision the director made.
type Store interface {
	GetDirection(ctx context.Context, tournamentID int64) (Record, error)
	ListDirections(ctx context.Context) ([]Record, error)
	CreateDirection(ctx context.Context, rec Record) error
	UpdateDirection(ctx context.Context, rec Record) error
	DeleteDirection(ctx context.Context, tournamentID int64) error
	AppendEvent(ctx context.Context, tournamentID int64, ev StoredEvent) error
	LoadEvents(ctx context.Context, tournamentID int64) ([]StoredEvent, error)
}

// ErrNoDirection says this Tournament was never directed — it is a Tournament assembled from
// imported files, which is the ordinary case and not an error condition on its own.
var ErrNoDirection = errors.New("direction: this tournament has no direction")

// ErrFinished says the Direction is closed: it accepts no further decision until it is reopened.
var ErrFinished = errors.New("direction: the tournament is closed")

// Direction is an open Direction: its record, its log, and the state replayed from it.
type Direction struct {
	rec     Record
	store   Store
	journal tournoi.Journal
	st      *tournoi.State
}

// Open reads a Tournament's Direction and replays it. Replaying at every open is the point:
// nothing derived is stored, so a crash mid-tournament costs exactly nothing.
func Open(ctx context.Context, store Store, tournamentID int64) (*Direction, error) {
	rec, err := store.GetDirection(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	stored, err := store.LoadEvents(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("direction %d: loading events: %w", tournamentID, err)
	}
	d := &Direction{rec: rec, store: store}
	for _, se := range stored {
		var ev tournoi.Event
		if err := json.Unmarshal(se.Payload, &ev); err != nil {
			return nil, fmt.Errorf("direction %d: event %d unreadable: %w", tournamentID, se.Seq, err)
		}
		d.journal = append(d.journal, ev)
	}
	if len(d.journal) == 0 {
		// Un brouillon n'a rien écrit : il n'y a pas d'état de tournoi, et il ne faut pas en
		// fabriquer un. Replay sur un journal vide rend un State valide mais VIDE, dont la
		// configuration écraserait celle que le directeur est en train de composer.
		return d, nil
	}
	if d.st, err = tournoi.Replay(d.journal); err != nil {
		return nil, fmt.Errorf("direction %d: replay: %w", tournamentID, err)
	}
	return d, nil
}

// Create starts a Direction on a Tournament that has none. The configuration is kept on the
// record while the Direction is a draft; nothing is written to the log until the tournament
// actually starts, so a director who changes their mind leaves no trace to unwind.
func Create(ctx context.Context, store Store, tournamentID int64, cfg tournoi.Config) (*Direction, error) {
	if _, err := store.GetDirection(ctx, tournamentID); err == nil {
		return nil, fmt.Errorf("direction: tournament %d is already directed", tournamentID)
	} else if !errors.Is(err, ErrNoDirection) {
		return nil, err
	}
	blob, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("direction: configuration: %w", err)
	}
	rec := Record{
		TournamentID:  tournamentID,
		FormatVersion: tournoi.JournalVersion,
		EngineVersion: EngineVersion,
		State:         StateDraft,
		Config:        string(blob),
	}
	if err := store.CreateDirection(ctx, rec); err != nil {
		return nil, err
	}
	return &Direction{rec: rec, store: store, st: nil}, nil
}

// Record returns the Direction's own facts.
func (d *Direction) Record() Record { return d.rec }

// State returns the replayed tournament state, or nil while the Direction is a draft that has
// not started.
func (d *Direction) State() *tournoi.State { return d.st }

// Journal returns the log as it stands. The slice is the Direction's own; do not modify it.
func (d *Direction) Journal() tournoi.Journal { return d.journal }

// Config returns the configuration this Direction runs on: the one in the created event once it
// has started, and the draft being edited before that.
func (d *Direction) Config() (tournoi.Config, error) {
	if d.st != nil {
		return d.st.Config, nil
	}
	var cfg tournoi.Config
	if d.rec.Config == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(d.rec.Config), &cfg)
	return cfg, err
}

// SetConfig rewrites the draft configuration. It is refused once the tournament has started:
// from there a configuration change is an event, so that what the director decided stays
// readable in order (ADR-0047).
func (d *Direction) SetConfig(ctx context.Context, cfg tournoi.Config) error {
	if d.rec.State != StateDraft {
		return fmt.Errorf("direction: the tournament has started; a configuration change is an event")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	blob, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	d.rec.Config = string(blob)
	return d.store.UpdateDirection(ctx, d.rec)
}

// SetOutputDir remembers where the standalone display page is written.
func (d *Direction) SetOutputDir(ctx context.Context, dir string) error {
	d.rec.OutputDir = dir
	return d.store.UpdateDirection(ctx, d.rec)
}

// Start writes the created event, freezing the draft configuration into the log and moving the
// Direction to running. Entries made before this point are written right after it, in the order
// they were made, so the log is complete on its own.
func (d *Direction) Start(ctx context.Context, seed int64, now time.Time, entries []tournoi.Player) error {
	if d.rec.State != StateDraft {
		return fmt.Errorf("direction: already started")
	}
	cfg, err := d.Config()
	if err != nil {
		return err
	}
	st, created, err := tournoi.New(cfg, seed, now)
	if err != nil {
		return err
	}
	d.st = st
	if err := d.append(ctx, created); err != nil {
		return err
	}
	for _, p := range entries {
		if err := d.Apply(ctx, tournoi.PlayerAddedEvent(p, now)); err != nil {
			return err
		}
	}
	d.rec.State = StateRunning
	return d.store.UpdateDirection(ctx, d.rec)
}

// Propose asks the engine what to do now — at the host's clock.
//
// The engine has no clock of its own and refuses to invent one, so it takes the instant as an
// argument (Nicomaque v0.2.0). It matters: a micro-round's countdown and a break's warning are
// both answers to "what time is it", and calling the clockless Propose() would freeze them at
// the last recorded event. Returns nothing while the Direction is a draft.
func (d *Direction) Propose() []tournoi.Action {
	return d.ProposeAt(time.Now())
}

// ProposeAt asks the engine what to do at a given instant. Tests pass a fixed one so a run is
// reproducible; the panel passes the wall clock.
func (d *Direction) ProposeAt(now time.Time) []tournoi.Action {
	if d.st == nil {
		return nil
	}
	return d.st.ProposeAt(now)
}

// Warnings are the engine's standing complaints: a bracket match played by the wrong players, a
// score beyond the match length. They are shown, never blocking, and they disappear when their
// cause does — not because someone read them.
func (d *Direction) Warnings() []tournoi.Warning {
	if d.st == nil {
		return nil
	}
	return d.st.Warnings
}

// Apply records one decision: it writes the event first, then applies it. The order matters —
// a crash between the two leaves a log that replays to exactly what the director last saw,
// whereas applying first would lose the decision.
func (d *Direction) Apply(ctx context.Context, ev tournoi.Event) error {
	if d.st == nil {
		return fmt.Errorf("direction: the tournament has not started")
	}
	if d.st.Finished && ev.Kind != tournoi.EvNote {
		return ErrFinished
	}
	if err := d.st.Apply(ev); err != nil {
		return err
	}
	return d.append(ctx, ev)
}

// EventFor turns a proposal the director confirmed into the event that records it.
func (d *Direction) EventFor(a tournoi.Action, now time.Time) (tournoi.Event, error) {
	if d.st == nil {
		return tournoi.Event{}, fmt.Errorf("direction: the tournament has not started")
	}
	return d.st.EventFromAction(a, now)
}

// append writes one event at the next sequence number and keeps the in-memory log in step.
func (d *Direction) append(ctx context.Context, ev tournoi.Event) error {
	ev.Seq = len(d.journal)
	blob, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("direction: serialising event %s: %w", ev.Kind, err)
	}
	se := StoredEvent{Seq: ev.Seq, Kind: string(ev.Kind), Time: ev.Time, Payload: blob}
	if err := d.store.AppendEvent(ctx, d.rec.TournamentID, se); err != nil {
		return err
	}
	d.journal = append(d.journal, ev)
	d.rec.UpdatedAt = ev.Time
	return nil
}

// Finish closes the tournament and freezes the final standings.
func (d *Direction) Finish(ctx context.Context, now time.Time) error {
	if d.st == nil {
		return fmt.Errorf("direction: the tournament has not started")
	}
	if d.st.Finished {
		return ErrFinished
	}
	if err := d.Apply(ctx, tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvFinished, Time: now}); err != nil {
		return err
	}
	d.rec.State = StateFinished
	return d.store.UpdateDirection(ctx, d.rec)
}

// Delete removes the Direction and its log. The Tournament and its Matches stay: the Matches
// merely lose the Slot they filled (ADR-0047).
func (d *Direction) Delete(ctx context.Context) error {
	return d.store.DeleteDirection(ctx, d.rec.TournamentID)
}

// Ranking is the current standings. Notes are codes; the frontend renders them.
func (d *Direction) Ranking() []tournoi.Rank {
	if d.st == nil {
		return nil
	}
	return d.st.Ranking()
}
